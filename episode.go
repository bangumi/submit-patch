package main

import (
	"context"
	"encoding/json"
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
	"github.com/trim21/pkg/null"

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
	resp, err := h.client.R().SetResult(&episode).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", eid))
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

	var originalWiki dto.WikiEpisode
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", episodeID))
	if err != nil {
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.StatusCode() >= 300 {
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original subject not found", http.StatusNotFound)
		} else {
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
		ID:                  pk,
		EpisodeID:           int32(episodeID),
		State:               PatchStatePending,
		FromUserID:          user.UserID,
		WikiUserID:          0,
		Reason:              reason,
		OriginalName:        pgtype.Text{String: originalWiki.Name, Valid: true},
		OriginalNameCn:      pgtype.Text{String: originalWiki.NameCN, Valid: true},
		OriginalDuration:    pgtype.Text{String: originalWiki.Duration, Valid: true},
		OriginalAirdate:     pgtype.Text{String: originalWiki.Date, Valid: true},
		OriginalDescription: pgtype.Text{String: originalWiki.Summary, Valid: true},
		Name:                pgtype.Text{String: name, Valid: true},
		NameCn:              pgtype.Text{String: nameCN, Valid: true},
		Duration:            pgtype.Text{String: duration, Valid: true},
		Airdate:             pgtype.Text{String: date, Valid: true},
		Description:         pgtype.Text{String: summary, Valid: true},
		PatchDesc:           patchDesc,
	}

	var changed bool
	if name != originalWiki.Name {
		changed = true
	}

	if nameCN != originalWiki.NameCN {
		changed = true
	}

	if duration != originalWiki.Duration {
		changed = true
	}

	if date != originalWiki.Date {
		changed = true
	}

	if summary != originalWiki.Summary {
		changed = true
	}

	if !changed {
		return &HttpError{http.StatusBadRequest, "No changes detected"}
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
	order string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountEpisodePatches(r.Context(), dal.CountEpisodePatchesParams{State: stateVals})
	if err != nil {
		return err
	}

	var patches = make([]view.EpisodePatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListEpisodePatches(r.Context(), dal.ListEpisodePatchesParams{
			State:   stateVals,
			OrderBy: order,
			Skip:    (currentPage - 1) * defaultPageSize,
			Size:    defaultPageSize,
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

	return templates.EpisodePatchPage(
		csrf.GetToken(r),
		s,
		patch,
		author,
		reviewer,
		comments,
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
		csrf.Clear(w)
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
	name := form.Get("name")
	if name != originalWiki.Name {
		changed = true
	}

	nameCN := form.Get("name_cn")
	if nameCN != originalWiki.NameCN {
		changed = true
	}

	duration := form.Get("duration")
	if duration != originalWiki.Duration {
		changed = true
	}

	date := form.Get("date")
	if date != originalWiki.Date {
		changed = true
	}

	summary := form.Get("summary")
	if summary != originalWiki.Summary {
		changed = true
	}

	if reason != patch.Reason {
		changed = true
	}

	if patchDesc != patch.PatchDesc {
		changed = true
	}

	var param = dal.UpdateEpisodePatchParams{
		ID:                  patchID,
		Reason:              reason,
		PatchDesc:           patchDesc,
		OriginalName:        pgtype.Text{String: originalWiki.Name, Valid: true},
		OriginalNameCn:      pgtype.Text{String: originalWiki.NameCN, Valid: true},
		OriginalDuration:    pgtype.Text{String: originalWiki.Duration, Valid: true},
		OriginalAirdate:     pgtype.Text{String: originalWiki.Date, Valid: true},
		OriginalDescription: pgtype.Text{String: originalWiki.Summary, Valid: true},
		Name:                pgtype.Text{String: name, Valid: true},
		NameCn:              pgtype.Text{String: nameCN, Valid: true},
		Duration:            pgtype.Text{String: duration, Valid: true},
		Airdate:             pgtype.Text{String: date, Valid: true},
		Description:         pgtype.Text{String: summary, Valid: true},
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

		_ = qx.CreateComment(ctx, dal.CreateCommentParams{
			ID:        uuid.Must(uuid.NewV7()),
			PatchID:   param.ID,
			PatchType: PatchTypeEpisode,
			Text:      "作者进行了修改",
			FromUser:  0,
		})

		return nil
	})

	if err != nil {
		return err
	}

	http.Redirect(w, r, fmt.Sprintf("/episode/%s", patch.ID), http.StatusSeeOther)
	return nil
}

type RequestToUpdateEpisode struct {
	Name     null.String `json:"name"`
	NameCN   null.String `json:"name_cn"`
	Date     null.String `json:"date"`
	Duration null.String `json:"duration"`
	Summary  null.String `json:"summary"`

	Reason    string `json:"reason"`
	PatchDesc string `json:"patch_desc"`

	CfTurnstileResponse string `json:"cf_turnstile_response"`
}

func (h *handler) createEpisodeEditPatchAPI(w http.ResponseWriter, r *http.Request) error {
	episodeID, err := strconv.ParseInt(chi.URLParam(r, "episode-id"), 10, 32)
	if err != nil || episodeID <= 0 {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    "Invalid or missing episode-id query parameter",
		}
	}

	if r.Header.Get("Content-Type") != contentTypeApplicationJSON {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("content-type must be %q to use this API", contentTypeApplicationJSON),
		}
	}

	var req RequestToUpdateEpisode
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("failed to parse request body as json: %s", err.Error()),
		}
	}

	if req.Reason == "" {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    "at least give a reason to change the subject",
		}
	}

	user := session.GetSession(r.Context())
	if user.UserID == 0 {
		http.Error(w, "please login before submit any patch", http.StatusUnauthorized)
		return nil
	}

	if !user.SuperUser() {
		if err := h.validateCaptcha(r.Context(), req.CfTurnstileResponse); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
	}

	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/ep/%d", episodeID)
	var originalWiki dto.WikiEpisode
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fetchURL)
	if err != nil {
		// This usually indicates a network error or DNS issue before the request was even sent
		fmt.Printf("Error executing request to fetch original wiki: %v\n", err)
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.StatusCode() >= 300 {
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original episode not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch original subject req", http.StatusBadGateway)
		}
		return nil
	}

	var changed bool
	var param = dal.CreateEpisodePatchParams{
		EpisodeID:           int32(episodeID),
		State:               0,
		FromUserID:          user.UserID,
		WikiUserID:          0,
		Reason:              req.Reason,
		PatchDesc:           req.PatchDesc,
		OriginalName:        pgtype.Text{String: originalWiki.Name, Valid: true},
		OriginalNameCn:      pgtype.Text{String: originalWiki.NameCN, Valid: true},
		OriginalDuration:    pgtype.Text{String: originalWiki.Duration, Valid: true},
		OriginalAirdate:     pgtype.Text{String: originalWiki.Date, Valid: true},
		OriginalDescription: pgtype.Text{String: originalWiki.Summary, Valid: true},
		Name:                pgtype.Text{String: originalWiki.Name, Valid: true},
		NameCn:              pgtype.Text{String: originalWiki.NameCN, Valid: true},
		Duration:            pgtype.Text{String: originalWiki.Duration, Valid: true},
		Airdate:             pgtype.Text{String: originalWiki.Date, Valid: true},
		Description:         pgtype.Text{String: originalWiki.Summary, Valid: true},
	}

	if req.Name.Set && req.Name.Value != originalWiki.Name {
		changed = true
		param.Name = pgtype.Text{
			String: req.Name.Value,
			Valid:  true,
		}
	}

	if req.NameCN.Set && req.NameCN.Value != originalWiki.NameCN {
		changed = true
		param.NameCn = pgtype.Text{
			String: req.NameCN.Value,
			Valid:  true,
		}
	}

	if req.Duration.Set && req.Duration.Value != originalWiki.Duration {
		changed = true
		param.Duration = pgtype.Text{
			String: req.Duration.Value,
			Valid:  true,
		}
	}

	if req.Date.Set && req.Date.Value != originalWiki.Date {
		changed = true
		param.Airdate = pgtype.Text{
			String: req.Date.Value,
			Valid:  true,
		}
	}

	if req.Summary.Set && req.Summary.Value != originalWiki.Summary {
		changed = true
		param.Description = pgtype.Text{
			String: req.Summary.Value,
			Valid:  true,
		}
	}

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	param.ID = uuid.Must(uuid.NewV7())

	err = h.q.CreateEpisodePatch(r.Context(), param)
	if err != nil {
		http.Error(w, "Error inserting episode patch", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("content-type", contentTypeApplicationJSON)
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(map[string]string{
		"id": param.ID.String(),
	})
}
