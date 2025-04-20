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
	"github.com/samber/lo"
	"github.com/trim21/errgo"

	"app/dto"
	"app/q"
	"app/session"
	"app/templates"
)

type ApiPatchSubject struct {
	CommieMessage    string `json:"commitMessage"`
	ExpectedRevision struct {
		Infobox string `json:"infobox,omitempty"`
		Name    string `json:"name,omitempty"`
		Summary string `json:"summary,omitempty"`
	} `json:"expectedRevision"`
	Subject struct {
		Infobox string `json:"infobox,omitempty"`
		Name    string `json:"name,omitempty"`
		Summary string `json:"summary,omitempty"`
		Nsfw    *bool  `json:"nsfw,omitempty"`
	} `json:"subject"`
}

func (h *handler) handleSubjectReview(w http.ResponseWriter, r *http.Request, patchID uuid.UUID, react string, text string, s *session.Session) error {
	return h.tx(r.Context(), func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)

		p, err := qx.GetSubjectPatchByIDForUpdate(r.Context(), patchID)
		if err != nil {
			return err
		}

		if p.State != PatchStatePending {
			return errors.New("patch is not pending")
		}

		switch react {
		case "comment":
			return h.handleSubjectComment(w, r, qx, p, text, s)
		case "approve":
			return h.handleSubjectApprove(w, r, qx, p, s)
		case "reject":
			return h.handleSubjectReject(w, r, qx, p, s)
		default:
			return nil
		}
	})
}

func (h *handler) handleSubjectApprove(w http.ResponseWriter, r *http.Request, qx *q.Queries, patch q.SubjectPatch, s *session.Session) error {
	var body = ApiPatchSubject{
		CommieMessage: fmt.Sprintf("%s [https://patch.bgm38.tv/subject/%s]", patch.Reason, patch.ID),
	}

	body.ExpectedRevision.Infobox = patch.OriginalInfobox.String
	body.ExpectedRevision.Summary = patch.OriginalSummary.String

	if patch.Name.Valid {
		body.ExpectedRevision.Name = patch.OriginalName
	}
	body.Subject.Name = patch.Name.String
	body.Subject.Infobox = patch.Infobox.String
	body.Subject.Summary = patch.Summary.String
	if patch.Nsfw.Valid {
		body.Subject.Nsfw = lo.ToPtr(patch.Nsfw.Bool)
	}

	resp, err := h.client.R().
		SetHeader("cf-ray", r.Header.Get("cf-ray")).
		SetHeader("Authorization", "Bearer "+s.AccessToken).
		SetBody(body).
		Patch(fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", patch.SubjectID))
	if err != nil {
		return errgo.Wrap(err, "failed to submit patch")
	}

	if resp.StatusCode() >= 500 {
		log.Warn().Int("code", resp.StatusCode()).Msg("failed to submit patch")
		http.Error(w, "failed to submit patch", http.StatusBadGateway)
		return nil
	}

	if resp.StatusCode() >= 300 {
		var errRes dto.ErrorResponse
		if err = json.Unmarshal(resp.Body(), &errRes); err != nil {
			return errgo.Wrap(err, "failed to submit patch")
		}

		if errRes.Code == ErrCodeInvalidToken {
			http.SetCookie(w, &http.Cookie{
				Name:  cookieBackTo,
				Value: fmt.Sprintf("/subject/%s", patch.ID),
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return nil
		}

		if errRes.Code == ErrCodeInvalidWikiSyntax {
			err = qx.RejectSubjectPatch(r.Context(), q.RejectSubjectPatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				ID:           patch.ID,
				RejectReason: fmt.Sprintf("建议包含语法错误，已经自动拒绝:\n %s", errRes.Message),
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/subject/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		if errRes.Code == ErrCodeWikiChanged {
			err = qx.RejectSubjectPatch(r.Context(), q.RejectSubjectPatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateOutdated,
				ID:           patch.ID,
				RejectReason: fmt.Sprintf("已过期\n%s", errRes.Message),
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/subject/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		log.Error().RawJSON("body", resp.Body()).Msg("unexpected response from submit patch")
		return errors.New("failed to submit patch")
	}

	err = qx.AcceptSubjectPatch(context.WithoutCancel(r.Context()), q.AcceptSubjectPatchParams{
		WikiUserID: s.UserID,
		State:      PatchStateAccepted,
		ID:         patch.ID,
	})

	if err != nil {
		return errgo.Wrap(err, "failed to accept patch")
	}

	// Implement subject approval logic here
	http.Redirect(w, r, "/subject/"+patch.ID.String(), http.StatusSeeOther)
	return nil
}

func (h *handler) handleSubjectReject(w http.ResponseWriter, r *http.Request, qx *q.Queries, p q.SubjectPatch, s *session.Session) error {
	err := qx.RejectSubjectPatch(r.Context(), q.RejectSubjectPatchParams{
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

func (h *handler) handleSubjectComment(w http.ResponseWriter, r *http.Request, tx *q.Queries, patch q.SubjectPatch, text string, s *session.Session) error {
	err := tx.CreateComment(r.Context(), q.CreateCommentParams{
		ID:        uuid.Must(uuid.NewV7()),
		PatchID:   patch.ID,
		PatchType: PatchTypeSubject,
		Text:      text,
		FromUser:  s.UserID,
	})
	if err != nil {
		return err
	}

	err = tx.UpdateSubjectPatchCommentCount(r.Context(), patch.ID)
	if err != nil {
		return err
	}

	http.Redirect(w, r, "/subject/"+patch.ID.String(), http.StatusFound)
	return nil
}
