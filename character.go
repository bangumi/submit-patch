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

	"github.com/bangumi/wiki-parser-go"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/trim21/errgo"
	"github.com/trim21/pkg/null"

	"app/csrf"
	"app/dal"
	"app/dto"
	"app/internal/diff"
	"app/session"
	"app/templates"
	"app/view"
)

func (h *handler) editCharacterView(w http.ResponseWriter, r *http.Request) error {
	cid, err := strconv.ParseInt(chi.URLParam(r, "character-id"), 10, 32)
	if err != nil || cid <= 0 {
		http.Error(w, "character-id must be a positive integer", http.StatusBadRequest)
		return nil
	}

	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		http.SetCookie(w, &http.Cookie{
			Name:  cookieBackTo,
			Value: fmt.Sprintf("/suggest-character?character_id=%d", cid),
		})

		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	var character dto.WikiCharacter
	resp, err := h.client.R().
		SetResult(&character).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/characters/%d", cid))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	return h.template.EditCharacter.Execute(w, view.CharacterPatchEdit{
		PatchID:          "",
		CharacterID:      int32(cid),
		CsrfToken:        csrf.GetToken(r),
		Reason:           "",
		Description:      "",
		Data:             character,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}

func (h *handler) editCharacterPatchView(w http.ResponseWriter, r *http.Request) error {
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

	patch, err := h.q.GetCharacterPatchByID(r.Context(), patchID)
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

	var character dto.WikiCharacter
	resp, err := h.client.R().
		SetResult(&character).
		Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/characters/%d", patch.CharacterID))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	if patch.Name.Valid {
		character.Name = patch.Name.String
	}

	if patch.Infobox.Valid {
		character.Infobox = patch.Infobox.String
	}

	if patch.Summary.Valid {
		character.Summary = patch.Summary.String
	}

	return h.template.EditCharacter.Execute(w, view.CharacterPatchEdit{
		PatchID:          patch.ID.String(),
		CharacterID:      patch.CharacterID,
		CsrfToken:        csrf.GetToken(r),
		Reason:           patch.Reason,
		Description:      patch.PatchDesc,
		Data:             character,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}

func (h *handler) listCharacterPatches(
	w http.ResponseWriter,
	r *http.Request,
	patchStateFilter string,
	order string,
	stateVals []int32,
	currentPage int64,
) error {
	c, err := h.q.CountCharacterPatches(r.Context(), dal.CountCharacterPatchesParams{
		State:      stateVals,
		FromUserID: 0,
		WikiUserID: 0,
	})
	if err != nil {
		return err
	}

	var patches = make([]view.CharacterPatchListItem, 0, defaultPageSize)
	if c != 0 {
		data, err := h.q.ListCharacterPatches(r.Context(), dal.ListCharacterPatchesParams{
			State:   stateVals,
			OrderBy: order,
			Size:    defaultPageSize,
			Skip:    (currentPage - 1) * defaultPageSize,
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

			patches = append(patches, view.CharacterPatchListItem{
				ID:            v.ID.String(),
				UpdatedAt:     v.UpdatedAt.Time,
				CreatedAt:     v.CreatedAt.Time,
				State:         v.State,
				Action:        v.Action.Int32,
				Name:          v.OriginalName,
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

	pendingCount, err := CountPendingPatch(r.Context(), h.q)
	if err != nil {
		return err
	}

	return templates.CharacterPatchList(r, view.CharacterPatchList{
		Session:            session.GetSession(r.Context()),
		Patches:            patches,
		CurrentStateFilter: patchStateFilter,
		PendingCount: view.PendingPatchCount{
			Subject:   pendingCount.SubjectPatchCount,
			Episode:   pendingCount.EpisodePatchCount,
			Character: pendingCount.CharacterPatchCount,
			Person:    pendingCount.PersonPatchCount,
		},
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
	}).Render(r.Context(), w)
}

func (h *handler) characterPatchShortLink(
	w http.ResponseWriter,
	r *http.Request,
) error {
	patchID := chi.URLParam(r, "patchID")
	if patchID == "" {
		http.Error(w, "missing patch id", http.StatusBadRequest)
		return nil
	}

	numID, err := strconv.ParseUint(patchID, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	var id uuid.UUID
	row := h.db.QueryRow(r.Context(), `select id from character_patch where num_id = $1 limit 1`, numID)
	err = row.Scan(&id)
	if err != nil {
		return errgo.Wrap(err, "failed to query character_path")
	}

	http.Redirect(w, r, fmt.Sprintf("/character/%s", id), http.StatusSeeOther)

	return nil
}

func (h *handler) characterPatchDetailView(
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

	patch, err := h.q.GetCharacterPatchByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "patch not found", http.StatusNotFound)
			return nil
		}
		return errgo.Wrap(err, "GetCharacterPatchByID")
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
		PatchType: PatchTypeCharacter,
	})
	if err != nil {
		return errgo.Wrap(err, "GetComments")
	}

	var changes = make([]view.Change, 0, 3)

	if patch.Name.Valid && patch.OriginalName != patch.Name.String {
		changes = append(changes, view.Change{
			Name: "角色名",
			Diff: diff.Diff("name", patch.OriginalName, patch.Name.String),
		})
	}

	if patch.OriginalInfobox.Valid && patch.Infobox.Valid {
		changes = append(changes, view.Change{
			Name: "wiki",
			Diff: diff.Diff("wiki", patch.OriginalInfobox.String, patch.Infobox.String),
		})
	}

	if patch.OriginalSummary.Valid && patch.Summary.Valid {
		changes = append(changes, view.Change{
			Name: "简介",
			Diff: diff.Diff("summary", patch.OriginalSummary.String, patch.Summary.String),
		})
	}

	return templates.CharacterPatchPage(
		csrf.GetToken(r),
		s,
		patch,
		author,
		reviewer,
		comments,
		changes,
	).Render(r.Context(), w)
}

func (h *handler) createCharacterEditPatch(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	characterID, err := strconv.ParseInt(chi.URLParam(r, "character-id"), 10, 32)
	if err != nil || characterID <= 0 {
		http.Error(w, "Invalid or missing character_id query parameter", http.StatusBadRequest)
		return nil
	}

	form := r.PostForm
	data := CreateCharacterPatch{
		Name:                form.Get("name"),
		Infobox:             form.Get("infobox"),
		Summary:             form.Get("summary"),
		Reason:              strings.TrimSpace(form.Get("reason")),
		PatchDesc:           strings.TrimSpace(form.Get("patch_desc")),
		CfTurnstileResponse: form.Get("cf_turnstile_response"),
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
	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/characters/%d", characterID)
	var originalWiki dto.WikiCharacter
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fetchURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch original wiki")
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.IsError() {
		log.Warn().Str("url", fetchURL).Int("status", resp.StatusCode()).Msg("error fetching original wiki")
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original character not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch original character data", http.StatusBadGateway)
		}
		return nil
	}

	var changed bool
	pk := uuid.Must(uuid.NewV7())
	var param = dal.CreateCharacterEditPatchParams{
		ID:           pk,
		CharacterID:  int32(characterID),
		FromUserID:   user.UserID,
		Reason:       data.Reason,
		OriginalName: originalWiki.Name,
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

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.q.CreateCharacterEditPatch(ctx, param)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert character patch")
		http.Error(w, "Failed to save suggestion", http.StatusInternalServerError)
		return nil
	}

	if param.Infobox.Valid {
		if _, err := wiki.Parse(param.Infobox.String); err != nil {
			_ = h.q.CreateComment(ctx, dal.CreateCommentParams{
				ID:        uuid.Must(uuid.NewV7()),
				PatchID:   param.ID,
				PatchType: PatchTypeCharacter,
				Text:      fmt.Sprintf("包含语法错误，请仔细检查\n\n%s", err.Error()),
				FromUser:  wikiBotUserID,
			})
			_ = h.q.UpdateCharacterPatchCommentCount(ctx, param.ID)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/character/%s", pk), http.StatusFound)
	return nil
}

func (h *handler) updateCharacterEditPatch(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	patchID, err := uuid.FromString(chi.URLParam(r, "patch-id"))
	if err != nil {
		http.Error(w, "Invalid patch-id", http.StatusBadRequest)
		return nil
	}

	patch, err := h.q.GetCharacterPatchByID(r.Context(), patchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Patch not found", http.StatusNotFound)
			return nil
		}
		return err
	}

	characterID := patch.CharacterID

	form := r.PostForm
	data := CreateCharacterPatch{
		Name:                form.Get("name"),
		Infobox:             form.Get("infobox"),
		Summary:             form.Get("summary"),
		Reason:              strings.TrimSpace(form.Get("reason")),
		PatchDesc:           strings.TrimSpace(form.Get("patch_desc")),
		CfTurnstileResponse: form.Get("cf_turnstile_response"),
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

	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/characters/%d", characterID)
	var originalWiki dto.WikiCharacter
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fetchURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch original wiki")
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.IsError() {
		log.Warn().Str("url", fetchURL).Int("status", resp.StatusCode()).Msg("error fetching original wiki")
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original character not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch original character data", http.StatusBadGateway)
		}
		return nil
	}

	var changed bool
	var param = dal.UpdateCharacterPatchParams{
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

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second*10)
	defer cancel()

	err = h.tx(ctx, func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)
		p, err := qx.GetCharacterPatchByIDForUpdate(ctx, patchID)
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

		err = qx.UpdateCharacterPatch(ctx, param)
		if err != nil {
			return errgo.Wrap(err, "failed to update character patch")
		}

		_ = qx.CreateComment(ctx, dal.CreateCommentParams{
			ID:        uuid.Must(uuid.NewV7()),
			PatchID:   param.ID,
			PatchType: PatchTypeCharacter,
			Text:      "作者进行了修改",
			FromUser:  0,
		})

		if param.Infobox.Valid {
			if _, err := wiki.Parse(param.Infobox.String); err != nil {
				_ = qx.CreateComment(ctx, dal.CreateCommentParams{
					ID:        uuid.Must(uuid.NewV7()),
					PatchID:   param.ID,
					PatchType: PatchTypeCharacter,
					Text:      fmt.Sprintf("包含语法错误，请仔细检查\n\n%s", err.Error()),
					FromUser:  wikiBotUserID,
				})
				_ = qx.UpdateCharacterPatchCommentCount(ctx, patchID)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	http.Redirect(w, r, fmt.Sprintf("/character/%s", patch.ID), http.StatusSeeOther)
	return nil
}

type CreateCharacterPatch struct {
	Name                string
	Infobox             string
	Summary             string
	Reason              string
	PatchDesc           string
	CfTurnstileResponse string
}

func (h *handler) deleteCharacterPatch(w http.ResponseWriter, r *http.Request) error {
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
		csrf.Clear(w, r)
		http.Error(w, "csrf failed, please go-back and retry", http.StatusBadRequest)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second*5)
	defer cancel()

	return h.tx(ctx, func(tx pgx.Tx) error {
		qx := h.q.WithTx(tx)
		p, err := qx.GetCharacterPatchByIDForUpdate(ctx, patchID)
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

		err = qx.DeleteCharacterPatch(ctx, patchID)
		if err != nil {
			return errgo.Wrap(err, "failed to delete character patch")
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	})
}

type RequestToUpdateCharacter struct {
	Name    null.String `json:"name"`
	Infobox null.String `json:"infobox"`
	Summary null.String `json:"summary"`

	Reason    string `json:"reason"`
	PatchDesc string `json:"patch_desc"`

	CfTurnstileResponse string `json:"cf_turnstile_response"`
}

func (h *handler) createCharacterEditPatchAPI(w http.ResponseWriter, r *http.Request) error {
	characterID, err := strconv.ParseInt(chi.URLParam(r, "character-id"), 10, 32)
	if err != nil || characterID <= 0 {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    "Invalid or missing character-id query parameter",
		}
	}

	if r.Header.Get("Content-Type") != contentTypeApplicationJSON {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("content-type must be %q to use this API", contentTypeApplicationJSON),
		}
	}

	var req RequestToUpdateCharacter
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("failed to parse request body as json: %s", err.Error()),
		}
	}

	if req.Reason == "" {
		return &HttpError{
			StatusCode: http.StatusBadRequest,
			Message:    "at least give a reason to change the character",
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

	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/characters/%d", characterID)
	var originalWiki dto.WikiCharacter
	resp, err := h.client.R().
		SetContext(r.Context()).
		SetResult(&originalWiki).
		Get(fetchURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch original wiki")
		http.Error(w, "Failed to communicate with wiki service", http.StatusBadGateway)
		return nil
	}

	if resp.IsError() {
		log.Warn().Str("url", fetchURL).Int("status", resp.StatusCode()).Msg("error fetching original wiki")
		if resp.StatusCode() == http.StatusNotFound {
			http.Error(w, "Original character not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch original character req", http.StatusBadGateway)
		}
		return nil
	}

	var changed bool
	var param = dal.CreateCharacterEditPatchParams{
		CharacterID:  int32(characterID),
		FromUserID:   user.UserID,
		Reason:       req.Reason,
		OriginalName: originalWiki.Name,
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

	if !changed {
		http.Error(w, "No changes found", http.StatusBadRequest)
		return nil
	}

	param.ID = uuid.Must(uuid.NewV7())

	err = h.q.CreateCharacterEditPatch(r.Context(), param)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert character patch")
		http.Error(w, "Failed to save suggestion", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("content-type", contentTypeApplicationJSON)
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(map[string]string{
		"id": param.ID.String(),
	})
}
