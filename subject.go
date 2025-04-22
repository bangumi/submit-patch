package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	wiki "github.com/bangumi/wiki-parser-go"
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

func (h *handler) editSubjectView(w http.ResponseWriter, r *http.Request) error {
	sid, err := strconv.ParseInt(chi.URLParam(r, "subject-id"), 10, 32)
	if err != nil || sid <= 0 {
		http.Error(w, "subject-id must be a positive integer", http.StatusBadRequest)
		return nil
	}

	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		http.SetCookie(w, &http.Cookie{
			Name:  cookieBackTo,
			Value: fmt.Sprintf("/suggest-subject?subject_id=%d", sid),
		})

		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	var subject dto.WikiSubject
	resp, err := h.client.R().
		SetResult(&subject).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", sid))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	return h.template.EditSubject.Execute(w, view.SubjectPatchEdit{
		PatchID:          "",
		SubjectID:        int32(sid),
		CsrfToken:        csrf.GetToken(r),
		Reason:           "",
		Description:      "",
		Data:             subject,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}

func (h *handler) editSubjectPatchView(w http.ResponseWriter, r *http.Request) error {
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

	patch, err := h.q.GetSubjectPatchByID(r.Context(), patchID)
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

	var subject dto.WikiSubject
	resp, err := h.client.R().
		SetResult(&subject).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", patch.SubjectID))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	if patch.Name.Valid {
		subject.Name = patch.Name.String
	}

	if patch.Infobox.Valid {
		subject.Infobox = patch.Infobox.String
	}

	if patch.Summary.Valid {
		subject.Summary = patch.Summary.String
	}

	if patch.Nsfw.Valid {
		subject.Nsfw = patch.Nsfw.Bool
	}

	return h.template.EditSubject.Execute(w, view.SubjectPatchEdit{
		PatchID:          patch.ID.String(),
		SubjectID:        patch.SubjectID,
		CsrfToken:        csrf.GetToken(r),
		Reason:           patch.Reason,
		Description:      patch.PatchDesc,
		Data:             subject,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}

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
		data, err := h.q.ListSubjectPatchesByStates(r.Context(), dal.ListSubjectPatchesByStatesParams{
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

	pendingCount, err := h.q.CountPendingPatch(r.Context())
	if err != nil {
		return err
	}

	_ = templates.SubjectPatchList(r, view.SubjectPatchList{
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
	return nil
}

func (h *handler) subjectPatchDetailView(
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
		PatchType: PatchTypeSubject,
	})
	if err != nil {
		return errgo.Wrap(err, "GetComments")
	}

	var changes = make([]view.Change, 0, 4)

	if patch.Name.Valid && patch.OriginalName != patch.Name.String {
		changes = append(changes, view.Change{
			Name: "条目名",
			Diff: Diff("name", EscapeInvisible(patch.OriginalName),
				EscapeInvisible(patch.Name.String)),
		})
	}

	if patch.OriginalInfobox.Valid && patch.Infobox.Valid {
		changes = append(changes, view.Change{
			Name: "wiki",
			Diff: Diff("wiki", EscapeInvisible(patch.OriginalInfobox.String),
				EscapeInvisible(patch.Infobox.String)),
		})
	}

	if patch.OriginalSummary.Valid && patch.Summary.Valid {
		changes = append(changes, view.Change{
			Name: "简介",
			Diff: Diff("summary", EscapeInvisible(patch.OriginalSummary.String),
				EscapeInvisible(patch.Summary.String)),
		})
	}

	return templates.SubjectPatchPage(
		csrf.GetToken(r),
		s,
		patch,
		author,
		reviewer,
		comments,
		changes,
	).Render(r.Context(), w)
}

func (h *handler) createSubjectEditPatch(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	subjectID, err := strconv.ParseInt(chi.URLParam(r, "subject-id"), 10, 32)
	if err != nil || subjectID <= 0 {
		http.Error(w, "Invalid or missing subject_id query parameter", http.StatusBadRequest)
		return nil
	}

	form := r.PostForm
	data := CreateSubjectPatch{
		Name:                form.Get("name"),
		Infobox:             form.Get("infobox"),
		Summary:             form.Get("summary"),
		Reason:              strings.TrimSpace(form.Get("reason")),
		PatchDesc:           strings.TrimSpace(form.Get("patch_desc")),
		CfTurnstileResponse: form.Get("cf_turnstile_response"),
		Nsfw:                form.Get("nsfw"), // Will be "on" if checked, "" otherwise
	}

	if data.Reason == "" {
		http.Error(w, "Validation Failed: missing suggestion description (reason)", http.StatusBadRequest)
		return nil
	}

	if err := checkInvalidInputStr(data.Reason, data.PatchDesc); err != nil {
		http.Error(w, fmt.Sprintf("Validation Failed: %v", err), http.StatusBadRequest)
		return nil
	}

	user := session.GetSession(r.Context())
	if user.UserID == 0 {
		http.Error(w, "please login before submit any patch", http.StatusUnauthorized)
		return nil
	}

	if err := h.validateCaptcha(r.Context(), data.CfTurnstileResponse); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	// --- Fetch Original Data ---
	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", subjectID)
	var originalWiki dto.WikiSubject
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki). // Tell resty to unmarshal into originalWiki on success
		Get(fetchURL)
	if err != nil {
		// This usually indicates a network error or DNS issue before the request was even sent
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

	var changed bool
	pk := uuid.Must(uuid.NewV7())
	var param = dal.CreateSubjectEditPatchParams{
		ID:           pk,
		SubjectID:    int32(subjectID),
		FromUserID:   user.UserID,
		Reason:       data.Reason,
		OriginalName: originalWiki.Name,
		SubjectType:  originalWiki.TypeID,
		PatchDesc:    data.PatchDesc,
	}

	if data.Name != originalWiki.Name {
		changed = true
		param.Name = pgtype.Text{
			String: data.Name,
			Valid:  true,
		}
	}

	if data.Infobox != originalWiki.Infobox {
		changed = true
		param.OriginalInfobox = pgtype.Text{
			String: originalWiki.Infobox,
			Valid:  true,
		}

		param.Infobox = pgtype.Text{
			String: data.Infobox,
			Valid:  true,
		}
	}

	if data.Summary != originalWiki.Summary {
		changed = true
		param.OriginalSummary = pgtype.Text{
			String: originalWiki.Summary,
			Valid:  true,
		}

		param.Summary = pgtype.Text{
			String: data.Summary,
			Valid:  true,
		}
	}

	// Compare NSFW status
	// Form sends "on" if checked, "" if not. Presence means true.
	nsfwInput := data.Nsfw != "" // True if checkbox was checked
	if nsfwInput != originalWiki.Nsfw {
		changed = true
		param.Nsfw = pgtype.Bool{
			Bool:  nsfwInput,
			Valid: true,
		}
	}

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	err = h.q.CreateSubjectEditPatch(r.Context(), param)
	if err != nil {
		fmt.Printf("Error inserting subject patch: %v\n", err)
		http.Error(w, "Failed to save suggestion", http.StatusInternalServerError)
		return nil
	}

	if param.Infobox.Valid {
		if _, err := wiki.Parse(param.Infobox.String); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = h.q.CreateComment(ctx, dal.CreateCommentParams{
				ID:        uuid.Must(uuid.NewV7()),
				PatchID:   param.ID,
				PatchType: PatchTypeSubject,
				Text:      fmt.Sprintf("包含语法错误，请仔细检查\n\n%s", err.Error()),
				FromUser:  wikiBotUserID,
			})
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/subject/%s", pk), http.StatusFound)
	return nil
}

func (h *handler) updateSubjectEditPatch(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	patchID, err := uuid.FromString(chi.URLParam(r, "patch-id"))
	if err != nil {
		http.Error(w, "Invalid patch-id", http.StatusBadRequest)
		return nil
	}

	patch, err := h.q.GetSubjectPatchByID(r.Context(), patchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Patch not found", http.StatusNotFound)
			return nil
		}
		return err
	}

	subjectID := patch.SubjectID

	form := r.PostForm
	data := CreateSubjectPatch{
		Name:                form.Get("name"),
		Infobox:             form.Get("infobox"),
		Summary:             form.Get("summary"),
		Reason:              strings.TrimSpace(form.Get("reason")),
		PatchDesc:           strings.TrimSpace(form.Get("patch_desc")),
		CfTurnstileResponse: form.Get("cf_turnstile_response"),
		Nsfw:                form.Get("nsfw"),
	}

	if data.Reason == "" {
		http.Error(w, "Validation Failed: missing suggestion description (reason)",
			http.StatusBadRequest)
		return nil
	}

	if err := checkInvalidInputStr(data.Reason, data.PatchDesc); err != nil {
		http.Error(w, fmt.Sprintf("Validation Failed: %v", err), http.StatusBadRequest)
		return nil
	}

	user := session.GetSession(r.Context())
	if user.UserID == 0 {
		http.Error(w, "please login before submit any patch", http.StatusUnauthorized)
		return nil
	}

	if err := h.validateCaptcha(r.Context(), data.CfTurnstileResponse); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", subjectID)
	var originalWiki dto.WikiSubject
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
	var param = dal.UpdateSubjectPatchParams{
		ID:           patchID,
		Reason:       data.Reason,
		PatchDesc:    data.PatchDesc,
		OriginalName: originalWiki.Name,
	}

	if data.Name != originalWiki.Name {
		changed = true
		param.Name = pgtype.Text{String: data.Name, Valid: true}
	}

	if data.Infobox != originalWiki.Infobox {
		changed = true

		param.OriginalInfobox = pgtype.Text{String: originalWiki.Infobox, Valid: true}
		param.Infobox = pgtype.Text{String: data.Infobox, Valid: true}
	}

	if data.Summary != originalWiki.Summary {
		changed = true

		param.OriginalSummary = pgtype.Text{String: originalWiki.Summary, Valid: true}
		param.Summary = pgtype.Text{String: data.Summary, Valid: true}
	}

	nsfwInput := data.Nsfw != ""
	if nsfwInput != originalWiki.Nsfw {
		changed = true
		param.Nsfw = pgtype.Bool{
			Bool:  nsfwInput,
			Valid: true,
		}
	}

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second*10)
	defer cancel()

	err = h.tx(ctx, func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)
		p, err := qx.GetSubjectPatchByIDForUpdate(ctx, patchID)
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

		err = qx.UpdateSubjectPatch(ctx, param)
		if err != nil {
			return errgo.Wrap(err, "failed to update subject patch")
		}

		_ = h.q.CreateComment(ctx, dal.CreateCommentParams{
			ID:        uuid.Must(uuid.NewV7()),
			PatchID:   param.ID,
			PatchType: PatchTypeSubject,
			Text:      "作者进行了修改",
			FromUser:  0,
		})

		if param.Infobox.Valid {
			if _, err := wiki.Parse(param.Infobox.String); err != nil {
				_ = h.q.CreateComment(ctx, dal.CreateCommentParams{
					ID:        uuid.Must(uuid.NewV7()),
					PatchID:   param.ID,
					PatchType: PatchTypeSubject,
					Text:      fmt.Sprintf("包含语法错误，请仔细检查\n\n%s", err.Error()),
					FromUser:  wikiBotUserID,
				})
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	http.Redirect(w, r, fmt.Sprintf("/subject/%s", patch.ID), http.StatusSeeOther)
	return nil
}

type CreateSubjectPatch struct {
	Name                string
	Infobox             string
	Summary             string
	Reason              string
	PatchDesc           string
	CfTurnstileResponse string
	Nsfw                string
}

func (h *handler) deleteSubjectPatch(w http.ResponseWriter, r *http.Request) error {
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
		p, err := qx.GetSubjectPatchByIDForUpdate(ctx, patchID)
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

		err = qx.DeleteSubjectPatch(ctx, patchID)
		if err != nil {
			return errgo.Wrap(err, "failed to delete subject patch")
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	})
}

const contentTypeApplicationJSON = "application/json"

type RequestToUpdateSubject struct {
	Name    null.String `json:"name"`
	Infobox null.String `json:"infobox"`
	Summary null.String `json:"summary"`
	Nsfw    null.Bool   `json:"nsfw"`

	Reason    string `json:"reason"`
	PatchDesc string `json:"patch_desc"`

	CfTurnstileResponse string `json:"cf_turnstile_response"`
}

func (h *handler) createSubjectEditPatchAPI(w http.ResponseWriter, r *http.Request) error {
	subjectID, err := strconv.ParseInt(chi.URLParam(r, "subject-id"), 10, 32)
	if err != nil || subjectID <= 0 {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    "Invalid or missing subject_id query parameter",
		}
	}

	if r.Header.Get("Content-Type") != contentTypeApplicationJSON {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("content-type must be %q to use this API", contentTypeApplicationJSON),
		}
	}

	var req RequestToUpdateSubject
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

	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", subjectID)
	var originalWiki dto.WikiSubject
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

	if resp.IsError() {
		// The request was sent, but the server returned an error status code (>= 400)
		fmt.Printf("Error fetching original wiki (%s): status %d, body: %s\n", fetchURL, resp.StatusCode(), resp.String())
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original subject not found", http.StatusNotFound)
		} else {
			// You might want to return a more specific error message based on apiError if it was populated
			http.Error(w, "Failed to fetch original subject req", http.StatusBadGateway)
		}
		return nil
	}

	var changed bool
	var param = dal.CreateSubjectEditPatchParams{
		SubjectID:    int32(subjectID),
		FromUserID:   user.UserID,
		Reason:       req.Reason,
		OriginalName: originalWiki.Name,
		SubjectType:  originalWiki.TypeID,
		PatchDesc:    req.PatchDesc,
	}

	if req.Name.Set && req.Name.Value != originalWiki.Name {
		changed = true
		param.Name = pgtype.Text{
			String: req.Name.Value,
			Valid:  true,
		}
	}

	if req.Infobox.Set && req.Infobox.Value != originalWiki.Infobox {
		changed = true
		param.OriginalInfobox = pgtype.Text{
			String: originalWiki.Infobox,
			Valid:  true,
		}

		param.Infobox = pgtype.Text{
			String: req.Infobox.Value,
			Valid:  true,
		}
	}

	if req.Summary.Set && req.Summary.Value != originalWiki.Summary {
		changed = true
		param.OriginalSummary = pgtype.Text{
			String: originalWiki.Summary,
			Valid:  true,
		}

		param.Summary = pgtype.Text{
			String: req.Summary.Value,
			Valid:  true,
		}
	}

	if req.Nsfw.Set && req.Nsfw.Value != originalWiki.Nsfw {
		changed = true
		param.Nsfw = pgtype.Bool{
			Bool:  req.Nsfw.Value,
			Valid: true,
		}
	}

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	param.ID = uuid.Must(uuid.NewV7())

	err = h.q.CreateSubjectEditPatch(r.Context(), param)
	if err != nil {
		fmt.Printf("Error inserting subject patch: %v\n", err)
		http.Error(w, "Failed to save suggestion", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("content-type", contentTypeApplicationJSON)
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(map[string]string{
		"id": param.ID.String(),
	})
}
