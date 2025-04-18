package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"app/api"
	"app/csrf"
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
		csrf.GetToken(r),
		s,
		patch,
		author,
		reviewer,
		comments,
		changes,
	).Render(r.Context(), w)
}

func (h *handler) newSubjectEditPatch(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
		return nil
	}

	subjectID, err := strconv.ParseInt(r.URL.Query().Get("subject_id"), 10, 32)
	if err != nil || subjectID <= 0 {
		http.Error(w, "Invalid or missing subject_id query parameter", http.StatusBadRequest)
		return nil
	}

	form := r.PostForm

	var data CreateSubjectPatch
	data.Name = form.Get("name")
	data.Infobox = form.Get("infobox")
	data.Summary = form.Get("summary")
	data.Reason = form.Get("reason")
	data.PatchDesc = form.Get("patch_desc") // Defaults to "" if missing
	data.CfTurnstileResponse = form.Get("cf_turnstile_response")
	data.Nsfw = form.Get("nsfw") // Will be "on" if checked, "" otherwise

	data.Reason = strings.TrimSpace(data.Reason)
	data.PatchDesc = strings.TrimSpace(data.PatchDesc)

	if data.Reason == "" {
		http.Error(w, "Validation Failed: missing suggestion description (reason)", http.StatusBadRequest)
		return nil
	}

	if err := checkInvalidInputStr(data.Reason, data.PatchDesc); err != nil {
		http.Error(w, fmt.Sprintf("Validation Failed: %v", err), http.StatusBadRequest)
		return nil
	}

	user := session.GetSession(r.Context())
	// --- Captcha Validation (if not superuser) ---
	if !user.SuperUser() {
		if err := h.validateCaptcha(r.Context(), data.CfTurnstileResponse); err != nil {
			// Use StatusBadRequest as per Python's BadRequestException for captcha failure
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
	}

	// --- Fetch Original Data ---
	fetchURL := fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", subjectID)
	var originalWiki api.WikiSubject
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
	var param = q.CreateSubjectEditPatchParams{
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

	http.Redirect(w, r, fmt.Sprintf("/subject/%s", pk), http.StatusFound)
	return nil
}

// Placeholder: Replace with your actual validation logic
func checkInvalidInputStr(inputs ...string) error {
	for _, s := range inputs {
		// Add your validation logic here (e.g., check for forbidden characters)
		if strings.Contains(s, "<script>") {
			return fmt.Errorf("invalid characters found in input: %s", s)
		}
	}
	return nil
}

// Placeholder: Implement Captcha Validation
type turnstileResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Action      string    `json:"action"`
	CData       string    `json:"cdata"`
}
type CreateSubjectPatch struct {
	Name                string
	Infobox             string
	Summary             string
	Reason              string
	PatchDesc           string
	CfTurnstileResponse string
	// NSFW comes from form as string ("on" or empty), presence indicates true
	Nsfw string
}

// Corresponds to Python's PartialCreateSubjectPatch
type PartialCreateSubjectPatch struct {
	Name      *string `json:"name"`
	Infobox   *string `json:"infobox"`
	Summary   *string `json:"summary"`
	Nsfw      *bool   `json:"nsfw"`
	Reason    string  `json:"reason"`
	PatchDesc string  `json:"patch_desc"`
}

func (h *handler) validateCaptcha(ctx context.Context, turnstileResponseToken string) error {
	formData := url.Values{}
	formData.Set("secret", h.config.TurnstileSecretKey)
	formData.Set("response", turnstileResponseToken)
	// You might want to include remote IP: formData.Set("remoteip", r.RemoteAddr)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(formData.Encode()))
	if err != nil {
		// Log internal error
		fmt.Printf("Error creating Turnstile request: %v\n", err)
		return errors.New("failed to create captcha verification request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.client.GetClient().Do(req)
	if err != nil {
		// Log internal error
		fmt.Printf("Error calling Turnstile API: %v\n", err)
		return errors.New("failed to contact captcha verification service")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		// Log error details
		fmt.Printf("Turnstile API error: status %d, body: %s\n", resp.StatusCode, string(bodyBytes))
		return errors.New("captcha verification failed (API error)")
	}

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Log internal error
		fmt.Printf("Error decoding Turnstile response: %v\n", err)
		return errors.New("failed to parse captcha verification response")
	}

	if !result.Success {
		// Log error codes if needed: fmt.Printf("Turnstile verification failed: %v\n", result.ErrorCodes)
		return errors.New("验证码无效") // "Invalid CAPTCHA"
	}

	return nil
}
