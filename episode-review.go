package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/trim21/errgo"

	"app/dal"
	"app/dto"
	"app/session"
	"app/templates"
)

func (h *handler) handleEpisodeReview(w http.ResponseWriter, r *http.Request,
	patchID uuid.UUID, react string, text string, s *session.Session) error {
	return h.tx(r.Context(), func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)

		p, err := qx.GetEpisodePatchByIDForUpdate(r.Context(), patchID)
		if err != nil {
			return err
		}

		if p.State != PatchStatePending {
			http.Redirect(w, r, "/episode/"+p.ID.String(), http.StatusSeeOther)
			return nil
		}

		switch react {
		case "comment":
			return h.handleEpisodeComment(w, r, qx, p, text, s)
		case "approve":
			return h.handleEpisodeApprove(w, r, qx, p, s)
		case "reject":
			return h.handleEpisodeReject(w, r, qx, p, s)
		default:
			return nil
		}
	})
}

func (h *handler) handleEpisodeComment(w http.ResponseWriter, r *http.Request,
	tx *dal.Queries, patch dal.EpisodePatch, text string, s *session.Session) error {
	err := tx.CreateComment(r.Context(), dal.CreateCommentParams{
		ID:        uuid.Must(uuid.NewV7()),
		PatchID:   patch.ID,
		PatchType: PatchTypeEpisode,
		Text:      text,
		FromUser:  s.UserID,
	})
	if err != nil {
		return err
	}

	err = tx.UpdateEpisodePatchCommentCount(r.Context(), patch.ID)
	if err != nil {
		return errgo.Wrap(err, "failed to update ")
	}

	http.Redirect(w, r, "/episode/"+patch.ID.String(), http.StatusFound)
	return nil
}

type ApiExpectedSubject struct {
	Name     *string `json:"name,omitempty"`
	NameCN   *string `json:"nameCN,omitempty"`
	Date     *string `json:"date,omitempty"`
	Duration *string `json:"duration,omitempty"`
	Summary  *string `json:"summary,omitempty"`
}

type ApiEpisode struct {
	Name     *string `json:"name,omitempty"`
	NameCN   *string `json:"nameCN,omitempty"`
	Date     *string `json:"date,omitempty"`
	Duration *string `json:"duration,omitempty"`
	Summary  *string `json:"summary,omitempty"`
}

type ApiUpdateEpisode struct {
	CommieMessage    string             `json:"commitMessage"`
	ExpectedRevision ApiExpectedSubject `json:"expectedRevision"`
	Episode          ApiEpisode         `json:"episode"`
}

func valuePtrIfChanged(val, check string) *string {
	if check == val {
		return nil
	}
	return &val
}

func (h *handler) handleEpisodeApprove(
	w http.ResponseWriter,
	r *http.Request,
	qx *dal.Queries,
	patch dal.EpisodePatch,
	s *session.Session,
) error {
	var body = ApiUpdateEpisode{
		CommieMessage: fmt.Sprintf("%s [https://patch.bgm38.tv/e/%d]", patch.Reason, patch.NumID),
		ExpectedRevision: ApiExpectedSubject{
			Name:     valuePtrIfChanged(patch.OriginalName.String, patch.Name.String),
			NameCN:   valuePtrIfChanged(patch.OriginalNameCn.String, patch.NameCn.String),
			Date:     valuePtrIfChanged(patch.OriginalAirdate.String, patch.Airdate.String),
			Duration: valuePtrIfChanged(patch.OriginalDuration.String, patch.Duration.String),
			Summary:  valuePtrIfChanged(patch.OriginalDescription.String, patch.Description.String),
		},
		Episode: ApiEpisode{
			Name:     valuePtrIfChanged(patch.Name.String, patch.OriginalName.String),
			NameCN:   valuePtrIfChanged(patch.NameCn.String, patch.OriginalNameCn.String),
			Date:     valuePtrIfChanged(patch.Airdate.String, patch.OriginalAirdate.String),
			Duration: valuePtrIfChanged(patch.Duration.String, patch.OriginalDuration.String),
			Summary:  valuePtrIfChanged(patch.Description.String, patch.OriginalDescription.String),
		},
	}

	resp, err := h.client.R().
		SetHeader("cf-ray", r.Header.Get("cf-ray")).
		SetHeader("Authorization", "Bearer "+s.AccessToken).
		SetBody(body).
		Patch(fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", patch.EpisodeID))
	if err != nil {
		return errgo.Wrap(err, "failed to submit subject patch")
	}

	if resp.StatusCode() >= 500 {
		log.Warn().Int("code", resp.StatusCode()).Msg("failed to submit episode patch")
		http.Error(w, "failed to submit patch", http.StatusBadGateway)
		return nil
	}

	if resp.StatusCode() >= 300 {
		var errRes dto.ErrorResponse
		if err = json.Unmarshal(resp.Body(), &errRes); err != nil {
			return errgo.Wrap(err, "failed to submit patch")
		}

		if errRes.Code == ErrCodeInvalidToken {
			needLogin(w, r, fmt.Sprintf("/episode/%s", patch.ID))
			return nil
		}

		if errRes.Code == ErrCodeValidationError {
			err = qx.RejectEpisodePatch(r.Context(), dal.RejectEpisodePatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				ID:           patch.ID,
				RejectReason: fmt.Sprintf("包含错误，已经自动拒绝:\n %s", errRes.Message),
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/episode/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		if errRes.Code == ErrCodeWikiChanged {
			err = qx.RejectEpisodePatch(r.Context(), dal.RejectEpisodePatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateOutdated,
				ID:           patch.ID,
				RejectReason: fmt.Sprintf("已过期\n%s", errRes.Message),
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/episode/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		log.Error().RawJSON("body", resp.Body()).Msg("unexpected response from submit patch")
		return errors.New("failed to submit patch")
	}

	err = qx.AcceptEpisodePatch(context.WithoutCancel(r.Context()), dal.AcceptEpisodePatchParams{
		WikiUserID: s.UserID,
		State:      PatchStateAccepted,
		ID:         patch.ID,
	})

	if err != nil {
		return errgo.Wrap(err, "failed to accept patch")
	}

	nextID, err := h.q.NextPendingEpisodePatch(r.Context(), patch.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Redirect(w, r, "/?type=episode", http.StatusSeeOther)
			return nil
		}
		return errgo.Wrap(err, "failed to get next patch")
	}

	http.Redirect(w, r, "/episode/"+nextID.String(), http.StatusSeeOther)
	return nil
}

func (h *handler) handleEpisodeReject(w http.ResponseWriter, r *http.Request,
	qx *dal.Queries, p dal.EpisodePatch, s *session.Session) error {
	err := qx.RejectEpisodePatch(r.Context(), dal.RejectEpisodePatchParams{
		WikiUserID: s.UserID,
		State:      PatchStateRejected,
		ID:         p.ID,
	})

	if err != nil {
		return templates.Error(r.Method, r.URL.String(), err.Error(), "", "").Render(r.Context(), w)
	}

	http.Redirect(w, r, "/episode/"+p.ID.String(), http.StatusFound)
	return nil
}
