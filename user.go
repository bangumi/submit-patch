package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/trim21/errgo"

	"app/q"
	"app/session"
	"app/templates"
	"app/view"
)

func (h *handler) userReviewView(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.ParseInt(r.PathValue("user-id"), 10, 32)
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return nil
	}

	rq := r.URL.Query()

	rawPage := rq.Get("page")
	currentPage, err := strconv.ParseInt(rawPage, 10, 64)
	if err != nil {
		currentPage = 1
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	stateVals, state, err := readableStateToDBValues(rq.Get("state"), StateFilterAll)
	if err != nil {
		return err
	}

	t := rq.Get("type")
	switch t {
	case "", "subject":
		return h.listSubjectPatchesReviewedUser(w, r, int32(userID), state, stateVals, currentPage)
	case "episode":
		return h.listEpisodePatchesReviewedUser(w, r, int32(userID), state, stateVals, currentPage)
	}

	http.Error(w, "invalid patch type", http.StatusBadRequest)
	return nil
}

func (h *handler) listSubjectPatchesReviewedUser(
	w http.ResponseWriter,
	r *http.Request,
	userID int32,
	patchStateFilter string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountSubjectPatchesByStatesReviewedByUser(r.Context(), q.CountSubjectPatchesByStatesReviewedByUserParams{
		UserID: userID,
		State:  stateVals,
	})
	if err != nil {
		return err
	}

	var patches = make([]view.SubjectPatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListSubjectPatchesByStatesReviewedByUser(r.Context(), q.ListSubjectPatchesByStatesReviewedByUserParams{
			WikiUserID: userID,
			State:      stateVals,
			Skip:       (currentPage - 1) * defaultPageSize,
			Size:       defaultPageSize,
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

	return templates.SubjectPatchList(r, view.SubjectPatchList{
		Title:              fmt.Sprintf("%d reviewed subject patches", userID),
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
}

func (h *handler) listEpisodePatchesReviewedUser(
	w http.ResponseWriter,
	r *http.Request,
	userID int32,
	patchStateFilter string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountEpisodePatchesByStatesReviewedByUser(r.Context(), q.CountEpisodePatchesByStatesReviewedByUserParams{
		UserID: userID,
		State:  stateVals,
	})
	if err != nil {
		return err
	}

	var patches = make([]view.EpisodePatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListEpisodePatchesByStatesReviewedByUser(r.Context(), q.ListEpisodePatchesByStatesReviewedByUserParams{
			UserID: userID,
			State:  stateVals,
			Skip:   (currentPage - 1) * defaultPageSize,
			Size:   defaultPageSize,
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

	return templates.EpisodePatchList(r, view.EpisodePatchList{
		Title:              fmt.Sprintf("%d reviewed episode patches", userID),
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
}

func (h *handler) userContributionView(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.ParseInt(r.PathValue("user-id"), 10, 32)
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return nil
	}

	rq := r.URL.Query()

	rawPage := rq.Get("page")
	currentPage, err := strconv.ParseInt(rawPage, 10, 64)
	if err != nil {
		currentPage = 1
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	state := rq.Get("state")
	stateVals, state, err := readableStateToDBValues(state, StateFilterAll)
	if err != nil {
		return err
	}

	t := rq.Get("type")
	switch t {
	case "", "subject":
		return h.listSubjectPatchesFromUser(w, r, int32(userID), state, stateVals, currentPage)
	case "episode":
		return h.listEpisodePatchesFromUser(w, r, int32(userID), state, stateVals, currentPage)
	}

	http.Error(w, "invalid patch type", http.StatusBadRequest)
	return nil
}

func (h *handler) listSubjectPatchesFromUser(
	w http.ResponseWriter,
	r *http.Request,
	userID int32,
	patchStateFilter string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountSubjectPatchesByStatesFromUser(r.Context(), q.CountSubjectPatchesByStatesFromUserParams{
		UserID: userID,
		State:  stateVals,
	})
	if err != nil {
		return err
	}

	var patches = make([]view.SubjectPatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListSubjectPatchesByStatesFromUser(r.Context(), q.ListSubjectPatchesByStatesFromUserParams{
			FromUserID: userID,
			State:      stateVals,
			Skip:       (currentPage - 1) * defaultPageSize,
			Size:       defaultPageSize,
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

	return templates.SubjectPatchList(r, view.SubjectPatchList{
		Title:              fmt.Sprintf("%d subject patches", userID),
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
}

func (h *handler) listEpisodePatchesFromUser(
	w http.ResponseWriter,
	r *http.Request,
	userID int32,
	patchStateFilter string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountEpisodePatchesByStatesFromUser(r.Context(), q.CountEpisodePatchesByStatesFromUserParams{
		UserID: userID,
		State:  stateVals,
	})
	if err != nil {
		return err
	}

	var patches = make([]view.EpisodePatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListEpisodePatchesByStatesFromUser(r.Context(), q.ListEpisodePatchesByStatesFromUserParams{
			UserID: userID,
			State:  stateVals,
			Skip:   (currentPage - 1) * defaultPageSize,
			Size:   defaultPageSize,
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

	return templates.EpisodePatchList(r, view.EpisodePatchList{
		Title:              fmt.Sprintf("%d episode patches", userID),
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
}
