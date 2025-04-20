package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/trim21/errgo"

	"app/csrf"
	"app/dal"
	"app/dto"
	"app/session"
	"app/templates"
	"app/view"
)

func (h *handler) editEpisodeView(w http.ResponseWriter, r *http.Request) error {
	eid, err := strconv.ParseUint(chi.URLParam(r, "episode-id"), 10, 64)
	if err != nil || eid <= 0 {
		http.Error(w, "episode-id must be a positive integer", http.StatusBadRequest)
		return nil
	}

	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		http.SetCookie(w, &http.Cookie{
			Name:  cookieBackTo,
			Value: fmt.Sprintf("/suggest-episode?episode_id=%d", eid),
		})

		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	var episode dto.WikiEpisode
	resp, err := h.client.R().SetResult(&episode).Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", eid))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	return h.template.EditEpisode.Execute(w, view.EpisodePatchEdit{
		PatchID:          "",
		CsrfToken:        csrf.GetToken(r),
		Reason:           "",
		EpisodeID:        int32(eid),
		Description:      "",
		Data:             episode,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}

func (h *handler) createEpisodeEditPatch(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	episodeID, err := strconv.ParseInt(chi.URLParam(r, "episode-id"), 10, 32)
	if err != nil || episodeID <= 0 {
		http.Error(w, "Invalid or missing episode-id parameter", http.StatusBadRequest)
		return nil
	}

	form := r.PostForm

	reason := form.Get("reason")
	patchDesc := form.Get("patch_desc")
	if err := checkInvalidInputStr(reason, patchDesc); err != nil {
		http.Error(w, fmt.Sprintf("Validation Failed: %v", err), http.StatusBadRequest)
		return nil
	}

	user := session.GetSession(r.Context())
	if user.UserID == 0 {
		http.Error(w, "please login before submit any patch", http.StatusUnauthorized)
		return nil
	}

	cfTurnstileResponse := form.Get("cf_turnstile_response")
	if err := h.validateCaptcha(r.Context(), cfTurnstileResponse); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	// --- Fetch Original Data ---
	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", episodeID)
	var originalWiki dto.WikiEpisode
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fetchURL)
	if err != nil {
		fmt.Printf("Error executing request to fetch original wiki: %v\n", err)
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.IsError() {
		// The request was sent, but the server returned an error status code (>= 400)
		fmt.Printf("Error fetching original wiki (%s): status %d, body: %s\n", fetchURL, resp.StatusCode(), resp.String())
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original subject not found", http.StatusNotFound)
		} else {
			// You might want to return a more specific error message based on apiError if it was populated
			http.Error(w, "Failed to fetch original subject data", http.StatusBadGateway)
		}
		return nil
	}

	name := form.Get("name")
	nameCN := form.Get("name_cn")
	duration := form.Get("duration")
	date := form.Get("date")
	summary := form.Get("summary")

	pk := uuid.Must(uuid.NewV7())
	var param = dal.CreateEpisodePatchParams{
		ID:         pk,
		EpisodeID:  int32(episodeID),
		State:      PatchStatePending,
		FromUserID: user.UserID,
		WikiUserID: 0,
		Reason:     reason,
		PatchDesc:  patchDesc,
	}

	if name != originalWiki.Name {
		param.OriginalName = pgtype.Text{
			String: originalWiki.Name,
			Valid:  true,
		}
		param.Name = pgtype.Text{
			String: name,
			Valid:  true,
		}
	}

	if nameCN != originalWiki.NameCN {
		param.OriginalNameCn = pgtype.Text{
			String: originalWiki.NameCN,
			Valid:  true,
		}
		param.NameCn = pgtype.Text{
			String: nameCN,
			Valid:  true,
		}
	}

	if duration != originalWiki.Duration {
		param.OriginalDuration = pgtype.Text{
			String: originalWiki.Duration,
			Valid:  true,
		}
		param.Duration = pgtype.Text{
			String: duration,
			Valid:  true,
		}
	}

	if date != originalWiki.Date {
		param.OriginalAirdate = pgtype.Text{
			String: originalWiki.Date,
			Valid:  true,
		}
		param.Airdate = pgtype.Text{
			String: date,
			Valid:  true,
		}
	}

	if summary != originalWiki.Summary {
		param.OriginalDescription = pgtype.Text{
			String: originalWiki.Summary,
			Valid:  true,
		}
		param.Description = pgtype.Text{
			String: summary,
			Valid:  true,
		}
	}

	err = h.q.CreateEpisodePatch(r.Context(), param)
	if err != nil {
		fmt.Printf("Error inserting subject patch: %v\n", err)
		http.Error(w, "Failed to save suggestion", http.StatusInternalServerError)
		return nil
	}

	http.Redirect(w, r, fmt.Sprintf("/episode/%s", pk), http.StatusFound)
	return nil
}

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
		data, err := h.q.ListEpisodePatchesByStates(r.Context(), dal.ListEpisodePatchesByStatesParams{
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

	pendingCount, err := h.q.CountPendingPatch(r.Context())
	if err != nil {
		return err
	}

	return templates.EpisodePatchList(r, view.EpisodePatchList{
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		PendingCount: view.PendingPatchCount{
			Subject: pendingCount.SubjectPatchCount,
			Episode: pendingCount.EpisodePatchCount,
		},
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
}

func (h *handler) episodePatchDetailView(
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

	var reviewer *dal.PatchUser
	if patch.WikiUserID != 0 {
		r, err := h.q.GetUserByID(r.Context(), patch.WikiUserID)
		if err != nil {
			return errgo.Wrap(err, "failed to get user")
		}
		reviewer = &r
	}

	comments, err := h.q.GetComments(r.Context(), dal.GetCommentsParams{
		PatchID:   id,
		PatchType: PatchTypeEpisode,
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

func (h *handler) deleteEpisodePatch(w http.ResponseWriter, r *http.Request) error {
	patchID, err := uuid.FromString(chi.URLParam(r, "patch-id"))
	if err != nil {
		http.Error(w, "invalid patch id, must be uuid", http.StatusBadRequest)
		return nil
	}

	user := session.GetSession(r.Context())
	if user.UserID == 0 {
		http.Error(w, "please login before submit any patch", http.StatusUnauthorized)
		return nil
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "must be a valid form", http.StatusBadRequest)
		return nil
	}

	if !csrf.Verify(r, r.PostForm.Get(csrf.FormName)) {
		http.Error(w, "csrf failed", http.StatusBadRequest)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second*5)
	defer cancel()

	return h.tx(ctx, func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)
		p, err := qx.GetEpisodePatchByIDForUpdate(ctx, patchID)
		if err != nil {
			return err
		}
		if p.State != PatchStatePending {
			http.Error(w, "patch is not pending", http.StatusBadRequest)
			return nil
		}

		if p.FromUserID != user.UserID {
			http.Error(w, "this it not your patch", http.StatusUnauthorized)
			return nil
		}

		err = qx.DeleteEpisodePatch(ctx, patchID)
		if err != nil {
			return errgo.Wrap(err, "failed to delete subject patch")
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	})
}

func (h *handler) editEpisodePatchView(w http.ResponseWriter, r *http.Request) error {
	patchID, err := uuid.FromString(chi.URLParam(r, "patch-id"))
	if err != nil {
		http.Error(w, "patch-id must be a valid uuid", http.StatusBadRequest)
		return nil
	}

	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		needLogin(w, r, r.URL.Path)
		return nil
	}

	patch, err := h.q.GetEpisodePatchByID(r.Context(), patchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return nil
		}

		return err
	}

	if patch.FromUserID != s.UserID {
		http.Error(w, "this is not your patch", http.StatusUnauthorized)
		return nil
	}

	if patch.State != PatchStatePending {
		http.Error(w, "patch must be a pending state", http.StatusBadRequest)
		return nil
	}

	var episode dto.WikiEpisode
	resp, err := h.client.R().
		SetResult(&episode).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", patch.EpisodeID))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	if patch.Name.Valid {
		episode.Name = patch.Name.String
	}

	if patch.NameCn.Valid {
		episode.NameCN = patch.NameCn.String
	}

	if patch.Duration.Valid {
		episode.Duration = patch.Duration.String
	}

	if patch.Airdate.Valid {
		episode.Date = patch.Airdate.String
	}

	return h.template.EditEpisode.Execute(w, view.EpisodePatchEdit{
		PatchID:          patch.ID.String(),
		EpisodeID:        patch.EpisodeID,
		CsrfToken:        csrf.GetToken(r),
		Reason:           patch.Reason,
		Description:      patch.PatchDesc,
		Data:             episode,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}

func (h *handler) updateEpisodeEditPatch(w http.ResponseWriter, r *http.Request) error {
	patchID, err := uuid.FromString(chi.URLParam(r, "patch-id"))
	if err != nil {
		http.Error(w, "Invalid patch-id", http.StatusBadRequest)
		return nil
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	form := r.PostForm

	reason := form.Get("reason")
	patchDesc := form.Get("patch_desc")
	if err := checkInvalidInputStr(reason, patchDesc); err != nil {
		http.Error(w, fmt.Sprintf("Validation Failed: %v", err), http.StatusBadRequest)
		return nil
	}

	patch, err := h.q.GetEpisodePatchByID(r.Context(), patchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Patch not found", http.StatusNotFound)
			return nil
		}
		return err
	}

	user := session.GetSession(r.Context())
	if user.UserID == 0 {
		http.Error(w, "please login before submit any patch", http.StatusUnauthorized)
		return nil
	}

	if err := h.validateCaptcha(r.Context(), form.Get("cf_turnstile_response")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", patch.EpisodeID)
	var originalWiki dto.WikiEpisode
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fetchURL)
	if err != nil {
		fmt.Printf("Error executing request to fetch original wiki: %v\n", err)
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.IsError() {
		fmt.Printf("Error fetching original wiki (%s): status %d, body: %s\n",
			fetchURL, resp.StatusCode(), resp.String())
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original subject not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch original subject data", http.StatusBadGateway)
		}
		return nil
	}

	var changed bool
	var param = dal.UpdateEpisodePatchParams{
		ID:                  patchID,
		Reason:              reason,
		PatchDesc:           patchDesc,
		OriginalName:        pgtype.Text{},
		Name:                pgtype.Text{},
		OriginalNameCn:      pgtype.Text{},
		NameCn:              pgtype.Text{},
		OriginalDuration:    pgtype.Text{},
		Duration:            pgtype.Text{},
		OriginalAirdate:     pgtype.Text{},
		Airdate:             pgtype.Text{},
		OriginalDescription: pgtype.Text{},
		Description:         pgtype.Text{},
	}

	name := form.Get("name")
	if name != originalWiki.Name {
		param.OriginalName = pgtype.Text{
			String: originalWiki.Name,
			Valid:  true,
		}
		param.Name = pgtype.Text{
			String: name,
			Valid:  true,
		}
		changed = true
	}

	nameCN := form.Get("name_cn")
	if nameCN != originalWiki.NameCN {
		param.OriginalNameCn = pgtype.Text{
			String: originalWiki.NameCN,
			Valid:  true,
		}
		param.NameCn = pgtype.Text{
			String: nameCN,
			Valid:  true,
		}
		changed = true
	}

	duration := form.Get("duration")
	if duration != originalWiki.Duration {
		param.OriginalDuration = pgtype.Text{
			String: originalWiki.Duration,
			Valid:  true,
		}
		param.Duration = pgtype.Text{
			String: duration,
			Valid:  true,
		}
		changed = true
	}

	date := form.Get("date")
	if date != originalWiki.Date {
		param.OriginalAirdate = pgtype.Text{
			String: originalWiki.Date,
			Valid:  true,
		}
		param.Airdate = pgtype.Text{
			String: date,
			Valid:  true,
		}
		changed = true
	}

	summary := form.Get("summary")
	if summary != originalWiki.Summary {
		param.OriginalDescription = pgtype.Text{
			String: originalWiki.Summary,
			Valid:  true,
		}
		param.Description = pgtype.Text{
			String: summary,
			Valid:  true,
		}
		changed = true
	}

	if reason != patch.Reason {
		changed = true
	}

	if patchDesc != patch.PatchDesc {
		changed = true
	}

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second*10)
	defer cancel()

	err = h.tx(ctx, func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)
		p, err := qx.GetEpisodePatchByIDForUpdate(ctx, patchID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "patch not found", http.StatusNotFound)
				return nil
			}
			return err
		}
		if p.State != PatchStatePending {
			http.Error(w, "patch is not pending", http.StatusBadRequest)
			return nil
		}

		if p.FromUserID != user.UserID {
			http.Error(w, "this it not your patch", http.StatusUnauthorized)
			return nil
		}

		err = qx.UpdateEpisodePatch(ctx, param)
		if err != nil {
			return errgo.Wrap(err, "failed to update subject patch")
		}

		return nil
	})

	if err != nil {
		return err
	}

	http.Redirect(w, r, fmt.Sprintf("/episode/%s", patch.ID), http.StatusSeeOther)
	return nil
}
