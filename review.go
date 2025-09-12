package main

import (
	"fmt"
	"net/http"

	"github.com/gofrs/uuid/v5"

	"app/csrf"
)

func (h *handler) handleReview(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "must be a valid form", http.StatusBadRequest)
		return nil
	}

	if !csrf.Verify(r, r.PostForm.Get(csrf.FormName)) {
		csrf.Clear(w)
		http.Error(w, "csrf failed, please go-back and retry", http.StatusBadRequest)
		return nil
	}

	patchID := r.PathValue("patch-id")
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

	patchType := r.PathValue("patch-type")
	s, err := h.GetFreshSession(w, r, fmt.Sprintf("/%s/%s", patchType, patchID))
	if err != nil {
		return err
	}

	text := r.PostForm.Get("text")

	if patchType == "review" {
		if text == "" {
			return &HttpError{
				StatusCode: http.StatusBadRequest,
				Message:    "review text can't be empty",
			}
		}
	}

	switch patchType {
	case "subject":
		return h.handleSubjectReview(w, r, id, react, text, s)
	case "episode":
		return h.handleEpisodeReview(w, r, id, react, text, s)
	}

	http.Error(w, "invalid patch type", http.StatusBadRequest)
	return nil
}

const ErrCodeValidationError = "REQUEST_VALIDATION_ERROR"
const ErrCodeWikiChanged = "WIKI_CHANGED"
const ErrCodeInvalidWikiSyntax = "INVALID_SYNTAX_ERROR"
const ErrCodeInvalidToken = "TOKEN_INVALID"
const ErrCodeItemLocked = "ITEM_LOCKED"
