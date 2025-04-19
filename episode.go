package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/trim21/errgo"

	"app/csrf"
	"app/q"
	"app/session"
	"app/templates"
	"app/view"
)

func (h *handler) listEpisodePatches(
	w http.ResponseWriter,
	r *http.Request,
	patchStateFilter string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountEpisodePatchesByStates(r.Context(), stateVals)
	if err != nil {
		return err
	}

	var patches = make([]view.EpisodePatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListEpisodePatchesByStates(r.Context(), q.ListEpisodePatchesByStatesParams{
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

			patches = append(patches, view.EpisodePatchListItem{
				ID:            v.ID.String(),
				UpdatedAt:     v.UpdatedAt.Time,
				CreatedAt:     v.CreatedAt.Time,
				State:         v.State,
				Name:          v.OriginalName.String,
				CommentsCount: v.CommentsCount,
				Reason:        v.Reason,
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

	_ = templates.EpisodePatchList(r, view.EpisodePatchList{
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

func (h *handler) episodePatchDetail(
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

	patch, err := h.q.GetEpisodePatchByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "patch not found", http.StatusNotFound)
			return nil
		}
		return errgo.Wrap(err, "GetEpisodePatchByID")
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
		PatchType: q.PatchTypeEpisode,
	})
	if err != nil {
		return errgo.Wrap(err, "GetComments")
	}

	var changes = make([]view.Change, 0, 5)

	type Change struct {
		name     string
		original string
		current  string
	}

	for _, c := range []Change{
		{
			name:     "原名",
			original: patch.OriginalName.String,
			current:  patch.Name.String,
		},
		{
			name:     "中文名",
			original: patch.OriginalNameCn.String,
			current:  patch.NameCn.String,
		},
		{
			name:     "时长",
			original: patch.OriginalDuration.String,
			current:  patch.Duration.String,
		},
		{
			name:     "播出时间",
			original: patch.OriginalAirdate.String,
			current:  patch.Airdate.String,
		},
		{
			name:     "简介",
			original: patch.OriginalDescription.String,
			current:  patch.Description.String,
		},
	} {
		if c.original != c.current {
			changes = append(changes, view.Change{
				Name: c.name,
				Diff: Diff(c.name, EscapeInvisible(c.original), EscapeInvisible(c.current)),
			})
		}
	}

	return templates.EpisodePatchPage(
		csrf.GetToken(r),
		s,
		patch,
		author,
		reviewer,
		comments,
		changes,
	).Render(r.Context(), w)
}
