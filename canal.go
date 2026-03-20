package main

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"io"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"github.com/trim21/errgo"

	"app/dal"
)

const canalGroupID = "submit-patch-canal"

const opUpdate = "u"

type debeziumPayload struct {
	After  json.RawMessage `json:"after"`
	Source debeziumSource  `json:"source"`
	Op     string          `json:"op"`
}

type debeziumSource struct {
	Table string `json:"table"`
}

type subjectKey struct {
	ID int32 `json:"subject_id"`
}

type characterKey struct {
	ID int32 `json:"crt_id"`
}

type personKey struct {
	ID int32 `json:"prsn_id"`
}

type subjectAfter struct {
	Name    string `json:"subject_name"`
	Infobox string `json:"field_infobox"`
	Summary string `json:"field_summary"`
}

type characterAfter struct {
	Name    string `json:"crt_name"`
	Infobox string `json:"crt_infobox"`
	Summary string `json:"crt_summary"`
}

type personAfter struct {
	Name    string `json:"prsn_name"`
	Infobox string `json:"prsn_infobox"`
	Summary string `json:"prsn_summary"`
}

func startCanalConsumer(ctx context.Context, cfg Config, h *handler) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.KafkaBroker},
		GroupID:     canalGroupID,
		GroupTopics: cfg.KafkaTopics,
	})
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				return nil
			}
			log.Error().Err(err).Msg("canal: failed to fetch kafka message")
			continue
		}

		if err := handleCanalMessage(ctx, h, msg.Key, msg.Value); err != nil {
			log.Error().Err(err).Str("topic", msg.Topic).Msg("canal: failed to handle message")
			continue
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Error().Err(err).Msg("canal: failed to commit kafka message")
		}
	}
}

func handleCanalMessage(ctx context.Context, h *handler, key, value []byte) error {
	if len(value) == 0 {
		// tombstone event from debezium, ignore
		// https://debezium.io/documentation/reference/stable/connectors/mysql.html#mysql-tombstone-events
		return nil
	}

	var p debeziumPayload
	if err := json.Unmarshal(value, &p); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse debezium payload")
		return nil
	}

	if p.Op != opUpdate {
		return nil
	}

	switch p.Source.Table {
	case "chii_subjects":
		return handleSubjectChange(ctx, h, key, p.After)
	case "chii_characters":
		return handleCharacterChange(ctx, h, key, p.After)
	case "chii_persons":
		return handlePersonChange(ctx, h, key, p.After)
	}

	return nil
}

func handleSubjectChange(ctx context.Context, h *handler, key []byte, afterRaw json.RawMessage) error {
	var k subjectKey
	if err := json.Unmarshal(key, &k); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse subject key")
		return nil
	}
	if k.ID == 0 {
		return nil
	}

	var after subjectAfter
	if err := json.Unmarshal(afterRaw, &after); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse subject after")
		return nil
	}
	after.Name = html.UnescapeString(after.Name)
	after.Infobox = html.UnescapeString(after.Infobox)

	patches, err := h.q.GetPendingSubjectPatchesBySubjectID(ctx, k.ID)
	if err != nil {
		return errgo.Wrap(err, "GetPendingSubjectPatchesBySubjectID")
	}

	for _, patch := range patches {
		if !subjectPatchIsOutdated(patch, after) {
			continue
		}
		if err := h.q.RejectSubjectPatch(ctx, dal.RejectSubjectPatchParams{
			WikiUserID:   0,
			State:        PatchStateOutdated,
			RejectReason: "条目已被修改，建议已过期",
			ID:           patch.ID,
		}); err != nil {
			log.Error().Err(err).Stringer("id", patch.ID).Msg("canal: failed to mark subject patch outdated")
		}
	}
	return nil
}

func subjectPatchIsOutdated(patch dal.GetPendingSubjectPatchesBySubjectIDRow, after subjectAfter) bool {
	// only check a field if the patch actually modifies it (non-null proposed value)
	if patch.Name.Valid && after.Name != patch.OriginalName {
		return true
	}
	if patch.Infobox.Valid && patch.OriginalInfobox.Valid && after.Infobox != patch.OriginalInfobox.String {
		return true
	}
	if patch.Summary.Valid && patch.OriginalSummary.Valid && after.Summary != patch.OriginalSummary.String {
		return true
	}
	return false
}

func handleCharacterChange(ctx context.Context, h *handler, key []byte, afterRaw json.RawMessage) error {
	var k characterKey
	if err := json.Unmarshal(key, &k); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse character key")
		return nil
	}
	if k.ID == 0 {
		return nil
	}

	var after characterAfter
	if err := json.Unmarshal(afterRaw, &after); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse character after")
		return nil
	}
	after.Name = html.UnescapeString(after.Name)
	after.Infobox = html.UnescapeString(after.Infobox)

	patches, err := h.q.GetPendingCharacterPatchesByCharacterID(ctx, k.ID)
	if err != nil {
		return errgo.Wrap(err, "GetPendingCharacterPatchesByCharacterID")
	}

	for _, patch := range patches {
		if !characterPatchIsOutdated(patch, after) {
			continue
		}
		if err := h.q.RejectCharacterPatch(ctx, dal.RejectCharacterPatchParams{
			WikiUserID:   0,
			State:        PatchStateOutdated,
			RejectReason: "角色已被修改，建议已过期",
			ID:           patch.ID,
		}); err != nil {
			log.Error().Err(err).Stringer("id", patch.ID).Msg("canal: failed to mark character patch outdated")
		}
	}
	return nil
}

func characterPatchIsOutdated(patch dal.GetPendingCharacterPatchesByCharacterIDRow, after characterAfter) bool {
	// only check a field if the patch actually modifies it (non-null proposed value)
	if patch.Name.Valid && after.Name != patch.OriginalName {
		return true
	}
	if patch.Infobox.Valid && patch.OriginalInfobox.Valid && after.Infobox != patch.OriginalInfobox.String {
		return true
	}
	if patch.Summary.Valid && patch.OriginalSummary.Valid && after.Summary != patch.OriginalSummary.String {
		return true
	}
	return false
}

func handlePersonChange(ctx context.Context, h *handler, key []byte, afterRaw json.RawMessage) error {
	var k personKey
	if err := json.Unmarshal(key, &k); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse person key")
		return nil
	}
	if k.ID == 0 {
		return nil
	}

	var after personAfter
	if err := json.Unmarshal(afterRaw, &after); err != nil {
		log.Warn().Err(err).Msg("canal: failed to parse person after")
		return nil
	}
	after.Name = html.UnescapeString(after.Name)
	after.Infobox = html.UnescapeString(after.Infobox)

	patches, err := h.q.GetPendingPersonPatchesByPersonID(ctx, k.ID)
	if err != nil {
		return errgo.Wrap(err, "GetPendingPersonPatchesByPersonID")
	}

	for _, patch := range patches {
		if !personPatchIsOutdated(patch, after) {
			continue
		}
		if err := h.q.RejectPersonPatch(ctx, dal.RejectPersonPatchParams{
			WikiUserID:   0,
			State:        PatchStateOutdated,
			RejectReason: "人物已被修改，建议已过期",
			ID:           patch.ID,
		}); err != nil {
			log.Error().Err(err).Stringer("id", patch.ID).Msg("canal: failed to mark person patch outdated")
		}
	}
	return nil
}

func personPatchIsOutdated(patch dal.GetPendingPersonPatchesByPersonIDRow, after personAfter) bool {
	// only check a field if the patch actually modifies it (non-null proposed value)
	if patch.Name.Valid && after.Name != patch.OriginalName {
		return true
	}
	if patch.Infobox.Valid && patch.OriginalInfobox.Valid && after.Infobox != patch.OriginalInfobox.String {
		return true
	}
	if patch.Summary.Valid && patch.OriginalSummary.Valid && after.Summary != patch.OriginalSummary.String {
		return true
	}
	return false
}
