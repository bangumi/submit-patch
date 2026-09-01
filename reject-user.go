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

	type rejectedPatch struct {
		numID  int64
		userID int32
	}

	var (
		subjectPatches   []rejectedPatch
		episodePatches   []rejectedPatch
		characterPatches []rejectedPatch
		personPatches    []rejectedPatch
	)

	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)

		subjectRows, err := qx.RejectAllSubjectPatchesByUser(r.Context(), dal.RejectAllSubjectPatchesByUserParams{
			WikiUserID:   s.UserID,
			State:        PatchStateRejected,
			RejectReason: reason,
			FromUserID:   int32(userID),
		})
		if err != nil {
			return err
		}
		for _, p := range subjectRows {
			subjectPatches = append(subjectPatches, rejectedPatch{numID: p.NumID, userID: p.FromUserID})
		}

		episodeRows, err := qx.RejectAllEpisodePatchesByUser(r.Context(), dal.RejectAllEpisodePatchesByUserParams{
			WikiUserID:   s.UserID,
			State:        PatchStateRejected,
			RejectReason: reason,
			FromUserID:   int32(userID),
		})
		if err != nil {
			return err
		}
		for _, p := range episodeRows {
			episodePatches = append(episodePatches, rejectedPatch{numID: p.NumID, userID: p.FromUserID})
		}

		characterRows, err := qx.RejectAllCharacterPatchesByUser(r.Context(), dal.RejectAllCharacterPatchesByUserParams{
			WikiUserID:   s.UserID,
			State:        PatchStateRejected,
			RejectReason: reason,
			FromUserID:   int32(userID),
		})
		if err != nil {
			return err
		}
		for _, p := range characterRows {
			characterPatches = append(characterPatches, rejectedPatch{numID: p.NumID, userID: p.FromUserID})
		}

		personRows, err := qx.RejectAllPersonPatchesByUser(r.Context(), dal.RejectAllPersonPatchesByUserParams{
			WikiUserID:   s.UserID,
			State:        PatchStateRejected,
			RejectReason: reason,
			FromUserID:   int32(userID),
		})
		if err != nil {
			return err
		}
		for _, p := range personRows {
			personPatches = append(personPatches, rejectedPatch{numID: p.NumID, userID: p.FromUserID})
		}

		return nil
	})
	if err != nil {
		return err
	}

	ctx := context.WithoutCancel(r.Context())
	for _, p := range subjectPatches {
		h.sendNotifySubjectPatchRejected(ctx, p.numID, p.userID)
	}
	for _, p := range episodePatches {
		h.sendNotifyEpisodePatchRejected(ctx, p.numID, p.userID)
	}
	for _, p := range characterPatches {
		h.sendNotifyCharacterPatchRejected(ctx, p.numID, p.userID)
	}
	for _, p := range personPatches {
		h.sendNotifyPersonPatchRejected(ctx, p.numID, p.userID)
	}

	total := len(subjectPatches) + len(episodePatches) + len(characterPatches) + len(personPatches)
	log.Info().
		Int32("user_id", int32(userID)).
		Int32("reviewer_id", s.UserID).
		Int("rejected", total).
		Msg("rejected all pending patches of user")

	http.Redirect(w, r, fmt.Sprintf("/contrib/%d", userID), http.StatusSeeOther)
	return nil
}
