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

type ApiPatchPerson struct {
	CommieMessage    string `json:"commitMessage"`
	AuthorID         int32  `json:"authorID"`
	ExpectedRevision struct {
		Infobox string `json:"infobox,omitempty"`
		Name    string `json:"name,omitempty"`
		Summary string `json:"summary,omitempty"`
	} `json:"expectedRevision"`
	Person struct {
		Infobox string `json:"infobox,omitempty"`
		Name    string `json:"name,omitempty"`
		Summary string `json:"summary,omitempty"`
	} `json:"person"`
}

func (h *handler) handlePersonReview(w http.ResponseWriter, r *http.Request, patchID uuid.UUID, react string, text string, s *session.Session) error {
	return h.tx(r.Context(), func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)

		p, err := qx.GetPersonPatchByIDForUpdate(r.Context(), patchID)
		if err != nil {
			return err
		}

		if p.State != PatchStatePending {
			http.Redirect(w, r, "/person/"+p.ID.String(), http.StatusSeeOther)
			return nil
		}

		switch react {
		case "comment":
			return h.handlePersonComment(w, r, qx, p, text, s)
		case "approve":
			return h.handlePersonApprove(w, r, qx, p, s)
		case "reject":
			return h.handlePersonReject(w, r, qx, p, s)
		default:
			return nil
		}
	})
}

func (h *handler) handlePersonApprove(w http.ResponseWriter, r *http.Request, qx *dal.Queries, patch dal.PersonPatch, s *session.Session) error {
	var body = ApiPatchPerson{
		AuthorID:      patch.FromUserID,
		CommieMessage: fmt.Sprintf("%s [https://patch.bgm38.tv/p/%d]", patch.Reason, patch.NumID),
	}

	body.ExpectedRevision.Infobox = patch.OriginalInfobox.String
	body.ExpectedRevision.Summary = patch.OriginalSummary.String

	if patch.Name.Valid {
		body.ExpectedRevision.Name = patch.OriginalName
	}
	body.Person.Name = patch.Name.String
	body.Person.Infobox = patch.Infobox.String
	body.Person.Summary = patch.Summary.String

	resp, err := h.client.R().
		SetHeader("cf-ray", r.Header.Get("cf-ray")).
		SetHeader("Authorization", "Bearer "+s.AccessToken).
		SetHeader("x-admin-token", h.config.AdminToken).
		SetBody(body).
		Patch(fmt.Sprintf("https://next.bgm.tv/p1/wiki/persons/%d", patch.PersonID))
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
				Value: fmt.Sprintf("/person/%s", patch.ID),
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return nil
		}

		if errRes.Code == ErrCodeInvalidWikiSyntax {
			err = qx.RejectPersonPatch(r.Context(), dal.RejectPersonPatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				ID:           patch.ID,
				RejectReason: fmt.Sprintf("建议包含语法错误，已经自动拒绝:\n %s", errRes.Message),
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/person/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		if errRes.Code == ErrCodeWikiChanged {
			err = qx.RejectPersonPatch(r.Context(), dal.RejectPersonPatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateOutdated,
				ID:           patch.ID,
				RejectReason: fmt.Sprintf("已过期\n%s", errRes.Message),
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/person/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		if errRes.Code == ErrCodeItemLocked {
			err = qx.RejectPersonPatch(r.Context(), dal.RejectPersonPatchParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				ID:           patch.ID,
				RejectReason: "条目已被锁定",
			})
			if err != nil {
				return errgo.Wrap(err, "failed to reject patch")
			}

			http.Redirect(w, r, "/person/"+patch.ID.String(), http.StatusSeeOther)
			return nil
		}

		log.Error().RawJSON("body", resp.Body()).Msg("unexpected response from submit patch")
		return errors.New("failed to submit patch")
	}

	err = qx.AcceptPersonPatch(context.WithoutCancel(r.Context()), dal.AcceptPersonPatchParams{
		WikiUserID: s.UserID,
		State:      PatchStateAccepted,
		ID:         patch.ID,
	})

	if err != nil {
		return errgo.Wrap(err, "failed to accept patch")
	}

	nextID, err := h.q.NextPendingPersonPatch(r.Context(), patch.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Redirect(w, r, "/?type=person", http.StatusSeeOther)
			return nil
		}
		return errgo.Wrap(err, "failed to get next patch")
	}

	http.Redirect(w, r, "/person/"+nextID.String(), http.StatusSeeOther)
	return nil
}

func (h *handler) handlePersonReject(w http.ResponseWriter, r *http.Request, qx *dal.Queries, p dal.PersonPatch, s *session.Session) error {
	err := qx.RejectPersonPatch(r.Context(), dal.RejectPersonPatchParams{
		WikiUserID: s.UserID,
		State:      PatchStateRejected,
		ID:         p.ID,
	})

	if err != nil {
		return templates.Error(r.Method, r.URL.String(), err.Error(), "", "").Render(r.Context(), w)
	}

	http.Redirect(w, r, "/person/"+p.ID.String(), http.StatusFound)
	return nil
}

func (h *handler) handlePersonComment(w http.ResponseWriter, r *http.Request, tx *dal.Queries, patch dal.PersonPatch, text string, s *session.Session) error {
	err := tx.CreateComment(r.Context(), dal.CreateCommentParams{
		ID:        uuid.Must(uuid.NewV7()),
		PatchID:   patch.ID,
		PatchType: PatchTypePerson,
		Text:      text,
		FromUser:  s.UserID,
	})
	if err != nil {
		return err
	}

	err = tx.UpdatePersonPatchCommentCount(r.Context(), patch.ID)
	if err != nil {
		return err
	}

	http.Redirect(w, r, "/person/"+patch.ID.String(), http.StatusFound)
	return nil
}
