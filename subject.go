package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5"

	"app/q"
	"app/session"
	"app/templates"
	"app/view"

	"github.com/trim21/errgo"
)

func (h *handler) listSubjectPatches(
	w http.ResponseWriter,
	r *http.Request,
	patchStateFilter string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountSubjectPatchesByStates(r.Context(), stateVals)
	if err != nil {
		return err
	}

	var patches = make([]view.SubjectPatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListSubjectPatchesByStates(r.Context(), q.ListSubjectPatchesByStatesParams{
			State: stateVals,
			Size:  defaultPageSize,
			Skip:  (currentPage - 1) * defaultPageSize,
		})
		if err != nil {
			return errgo.Wrap(err, "failed to query data")
		}

		for _, v := range data {
			var reviewer *view.User
			if v.ReviewerNickname.Valid && v.ReviewerUserID.Valid {
				reviewer = &view.User{
					ID:       v.ReviewerUserID.Int32,
					Username: v.ReviewerNickname.String,
					Nickname: v.ReviewerNickname.String,
				}
			}

			patches = append(patches, view.SubjectPatchListItem{
				ID:            v.ID.String(),
				UpdatedAt:     v.UpdatedAt.Time,
				CreatedAt:     v.CreatedAt.Time,
				State:         v.State,
				Action:        v.Action.Int32,
				Name:          v.OriginalName,
				CommentsCount: v.CommentsCount,
				Reason:        v.Reason,
				SubjectType:   v.SubjectType,
				Author: view.User{
					ID:       v.AuthorUserID,
					Username: v.AuthorUsername,
					Nickname: v.AuthorNickname,
				},
				Reviewer: reviewer,
			})
		}
	}

	totalPage := (c + defaultPageSize - 1) / defaultPageSize

	_ = templates.SubjectPatchList(r, view.SubjectPatchList{
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
	return nil
}

func (h *handler) subjectPatchDetail(
	w http.ResponseWriter,
	r *http.Request,
) error {
	s := session.GetSession(r.Context())

	patchID := chi.URLParam(r, "patchID")
	if patchID == "" {
		http.Error(w, "missing patch id", http.StatusBadRequest)
		return nil
	}

	id, err := uuid.FromString(patchID)
	if err != nil {
		http.Error(w, "invalid patch id, must be uuid", http.StatusBadRequest)
		return nil
	}

	patch, err := h.q.GetSubjectPatchByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "patch not found", http.StatusNotFound)
			return nil
		}
		return errgo.Wrap(err, "GetSubjectPatchByID")
	}

	author, err := h.q.GetUserByID(r.Context(), patch.FromUserID)
	if err != nil {
		return errgo.Wrap(err, "failed to get user")
	}

	var reviewer *q.PatchUser
	if patch.WikiUserID != 0 {
		r, err := h.q.GetUserByID(r.Context(), patch.WikiUserID)
		if err != nil {
			return errgo.Wrap(err, "failed to get user")
		}
		reviewer = &r
	}

	comments, err := h.q.GetComments(r.Context(), q.GetCommentsParams{
		PatchID:   id,
		PatchType: q.PatchTypeSubject,
	})
	if err != nil {
		return errgo.Wrap(err, "GetComments")
	}

	var changes = make([]view.Change, 0, 4)

	if patch.Name.Valid && patch.OriginalName != patch.Name.String {
		changes = append(changes, view.Change{
			Name: "条目名",
			Diff: Diff("name", EscapeInvisible(patch.OriginalName), EscapeInvisible(patch.Name.String)),
		})
	}

	if patch.OriginalInfobox.Valid && patch.Infobox.Valid {
		changes = append(changes, view.Change{
			Name: "wiki",
			Diff: Diff("wiki", EscapeInvisible(patch.OriginalInfobox.String), EscapeInvisible(patch.Infobox.String)),
		})
	}

	if patch.OriginalSummary.Valid && patch.Summary.Valid {
		changes = append(changes, view.Change{
			Name: "简介",
			Diff: Diff("summary", EscapeInvisible(patch.OriginalSummary.String), EscapeInvisible(patch.Summary.String)),
		})
	}

	return templates.SubjectPatchPage(
		csrf.Token(r),
		s,
		patch,
		author,
		reviewer,
		comments,
		changes,
	).Render(r.Context(), w)
}
