package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"app/dal"
	notifypb "app/gen/proto/mq/v1"
)

const notifyTopic = "notify.v1"

// Notify types matching old-server and server-private definitions.
const (
	NotifyTypeSubjectPatchAccepted   = 35
	NotifyTypeEpisodePatchAccepted   = 36
	NotifyTypeSubjectPatchRejected   = 37
	NotifyTypeEpisodePatchRejected   = 38
	NotifyTypeSubjectPatchExpired    = 39
	NotifyTypeEpisodePatchExpired    = 40
	NotifyTypeCharacterPatchAccepted = 41
	NotifyTypePersonPatchAccepted    = 42
	NotifyTypeCharacterPatchRejected = 43
	NotifyTypePersonPatchRejected    = 44
	NotifyTypeCharacterPatchExpired  = 45
	NotifyTypePersonPatchExpired     = 46
	NotifyTypeSubjectPatchReply      = 47
	NotifyTypeEpisodePatchReply      = 48
	NotifyTypeCharacterPatchReply    = 49
	NotifyTypePersonPatchReply       = 50
)

const wikiBot = uint32(427613)

func (h *handler) sendNotify(ctx context.Context, mid uint32, userID uint32, senderID uint32, notifyType int32, title string) {
	log.Debug().
		Uint32("mid", mid).
		Uint32("user_id", userID).
		Uint32("sender_id", senderID).
		Int32("notify_type", notifyType).
		Str("title", title).
		Msg("sending notification")

	if h.k == nil {
		return
	}

	msg := &notifypb.Notify{
		Mid:        mid,
		UserId:     userID,
		FromUserId: senderID,
		Type:       notifyType,
		Title:      title,
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal notification")
		return
	}

	key, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal notification key")
		return
	}

	err = h.k.WriteMessages(ctx, kafka.Message{
		Topic: notifyTopic,
		Key:   key,
		Value: data,
	})
	if err != nil {
		log.Error().Err(err).
			Uint32("user_id", userID).
			Int32("notify_type", notifyType).
			Msg("failed to send notification")
	}
}

// Subject patch notifications

func (h *handler) sendNotifySubjectPatchAccepted(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifySubjectPatchAccepted")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeSubjectPatchAccepted, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifySubjectPatchRejected(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifySubjectPatchRejected")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeSubjectPatchRejected, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifySubjectPatchExpired(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifySubjectPatchExpired")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeSubjectPatchExpired, fmt.Sprintf("#%d", numID))
}

// Episode patch notifications

func (h *handler) sendNotifyEpisodePatchAccepted(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyEpisodePatchAccepted")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeEpisodePatchAccepted, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyEpisodePatchRejected(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyEpisodePatchRejected")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeEpisodePatchRejected, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyEpisodePatchExpired(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyEpisodePatchExpired")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeEpisodePatchExpired, fmt.Sprintf("#%d", numID))
}

// Character patch notifications

func (h *handler) sendNotifyCharacterPatchAccepted(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyCharacterPatchAccepted")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeCharacterPatchAccepted, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyCharacterPatchRejected(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyCharacterPatchRejected")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeCharacterPatchRejected, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyCharacterPatchExpired(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyCharacterPatchExpired")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypeCharacterPatchExpired, fmt.Sprintf("#%d", numID))
}

// Person patch notifications

func (h *handler) sendNotifyPersonPatchAccepted(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyPersonPatchAccepted")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypePersonPatchAccepted, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyPersonPatchRejected(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyPersonPatchRejected")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypePersonPatchRejected, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyPersonPatchExpired(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyPersonPatchExpired")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), wikiBot, NotifyTypePersonPatchExpired, fmt.Sprintf("#%d", numID))
}

// Patch reply notification — sends to patch creator and all previous commenters, excluding the commenter.
func (h *handler) sendNotifyPatchReply(ctx context.Context, numID int64, commenterID int32, patchCreatorID int32, comments []dal.GetCommentsRow, notifyType int32) {
	userSet := make(map[int32]struct{})
	userSet[patchCreatorID] = struct{}{}
	for _, c := range comments {
		userSet[c.FromUser] = struct{}{}
	}
	delete(userSet, commenterID)

	for uid := range userSet {
		h.sendNotify(ctx, uint32(numID), uint32(uid), uint32(commenterID), notifyType, fmt.Sprintf("#%d", numID))
	}
}
