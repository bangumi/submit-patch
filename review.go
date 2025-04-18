package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"

	"app/csrf"
	"app/q"
	"app/session"
	"app/templates"
)

func (h *handler) handleReview(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "must be a valid form", http.StatusBadRequest)
		return nil
	}
	if !csrf.Verify(r, r.PostForm.Get(csrf.FormName)) {
		http.Error(w, "csrf failed", http.StatusBadRequest)
		return nil
	}

	patchID := chi.URLParam(r, "patch_id")
	if patchID == "" {
		http.Error(w, "missing patch id", http.StatusBadRequest)
		return nil
	}

	id, err := uuid.FromString(patchID)
	if err != nil {
		http.Error(w, "invalid patch id, must be uuid", http.StatusBadRequest)
		return nil
	}

	react := r.PostForm.Get("react")
	switch react {
	case "comment", "approve", "reject":
	default:
		http.Error(w, "invalid react type", http.StatusBadRequest)
		return nil
	}

	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return nil
	}

	patchType := chi.URLParam(r, "patch_type")
	switch patchType {
	case "subject":
		return h.handleSubjectReview(w, r, id, react, r.PostForm.Get("text"), s)
	case "episode":
	default:
		http.Error(w, "invalid patch type", http.StatusBadRequest)
		return nil
	}

	return nil
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
			return h.handleSubjectApprove(w, r, p.ID, s)
		case "reject":
			return h.handleSubjectReject(w, r, p, s)
		default:
			return nil
		}
	})
}

func (h *handler) handleSubjectComment(w http.ResponseWriter, r *http.Request, tx *q.Queries, patch q.SubjectPatch, text string, s *session.Session) error {
	err := tx.CreateComment(r.Context(), q.CreateCommentParams{
		ID:        uuid.Must(uuid.NewV7()),
		PatchID:   patch.ID,
		PatchType: q.PatchTypeSubject,
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

func (h *handler) handleEpisodeComment(w http.ResponseWriter, r *http.Request, patchID uuid.UUID, text string, s *session.Session) error {
	err := h.q.CreateComment(r.Context(), q.CreateCommentParams{
		ID:        uuid.Must(uuid.NewV7()),
		PatchID:   patchID,
		PatchType: q.PatchTypeEpisode,
		Text:      text,
		FromUser:  s.UserID,
	})

	if err != nil {
		return err
	}

	http.Redirect(w, r, "/episode/"+patchID.String(), http.StatusFound)
	return nil
}

func (h *handler) handleSubjectApprove(w http.ResponseWriter, r *http.Request, patchID uuid.UUID, s *session.Session) error {
	// Implement subject approval logic here
	return nil
}

func (h *handler) handleEpisodeApprove(w http.ResponseWriter, r *http.Request, patchID uuid.UUID, s *session.Session) error {
	// Implement episode approval logic here
	return nil
}

func (h *handler) handleSubjectReject(w http.ResponseWriter, r *http.Request, p q.SubjectPatch, s *session.Session) error {
	err := h.q.RejectSubjectPatch(r.Context(), q.RejectSubjectPatchParams{
		WikiUserID: s.UserID,
		State:      PatchStateRejected,
		ID:         p.ID,
	})

	if err != nil {
		return templates.Error(r.Method, r.URL.String(), err.Error(), "", "").Render(r.Context(), w)
	}
	// Implement subject rejection logic here
	return nil
}

func (h *handler) handleEpisodeReject(w http.ResponseWriter, r *http.Request, patchID uuid.UUID, s *session.Session) error {
	// Implement episode rejection logic here
	return nil
}
