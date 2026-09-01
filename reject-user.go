package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"app/csrf"
	"app/dal"
)

func (h *handler) handleRejectUserPatches(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "must be a valid form", http.StatusBadRequest)
		return nil
	}

	if !csrf.Verify(r, r.PostForm.Get(csrf.FormName)) {
		csrf.Clear(w, r)
		http.Error(w, "csrf failed, please go-back and retry", http.StatusBadRequest)
		return nil
	}

	userID, err := strconv.ParseInt(r.PathValue("user-id"), 10, 32)
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return nil
	}

	s, err := h.GetFreshSession(w, r, fmt.Sprintf("/contrib/%d", userID))
	if err != nil {
		return err
	}

	if !s.SuperUser() {
		http.Error(w, "permission denied", http.StatusForbidden)
		return nil
	}

	reason := r.PostForm.Get("reason")
	patchType := r.PostForm.Get("type")

	ctx := context.WithoutCancel(r.Context())

	var rejected int
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)

		switch patchType {
		case PatchTypeSubject:
			rows, err := qx.RejectAllSubjectPatchesByUser(ctx, dal.RejectAllSubjectPatchesByUserParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				RejectReason: reason,
				FromUserID:   int32(userID),
			})
			if err != nil {
				return err
			}
			rejected = len(rows)
			for _, p := range rows {
				h.sendNotifySubjectPatchRejected(ctx, p.NumID, p.FromUserID)
			}
		case PatchTypeEpisode:
			rows, err := qx.RejectAllEpisodePatchesByUser(ctx, dal.RejectAllEpisodePatchesByUserParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				RejectReason: reason,
				FromUserID:   int32(userID),
			})
			if err != nil {
				return err
			}
			rejected = len(rows)
			for _, p := range rows {
				h.sendNotifyEpisodePatchRejected(ctx, p.NumID, p.FromUserID)
			}
		case PatchTypeCharacter:
			rows, err := qx.RejectAllCharacterPatchesByUser(ctx, dal.RejectAllCharacterPatchesByUserParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				RejectReason: reason,
				FromUserID:   int32(userID),
			})
			if err != nil {
				return err
			}
			rejected = len(rows)
			for _, p := range rows {
				h.sendNotifyCharacterPatchRejected(ctx, p.NumID, p.FromUserID)
			}
		case PatchTypePerson:
			rows, err := qx.RejectAllPersonPatchesByUser(ctx, dal.RejectAllPersonPatchesByUserParams{
				WikiUserID:   s.UserID,
				State:        PatchStateRejected,
				RejectReason: reason,
				FromUserID:   int32(userID),
			})
			if err != nil {
				return err
			}
			rejected = len(rows)
			for _, p := range rows {
				h.sendNotifyPersonPatchRejected(ctx, p.NumID, p.FromUserID)
			}
		default:
			return &HttpError{
				StatusCode: http.StatusBadRequest,
				Message:    "invalid patch type",
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Info().
		Int32("user_id", int32(userID)).
		Int32("reviewer_id", s.UserID).
		Str("patch_type", patchType).
		Int("rejected", rejected).
		Msg("rejected pending patches of user")

	http.Redirect(w, r, fmt.Sprintf("/contrib/%d", userID), http.StatusSeeOther)
	return nil
}
