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

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/rueidis"
	"github.com/trim21/errgo"

	"app/q"
)

func newHandler(db *pgxpool.Pool, r rueidis.Client, q *q.Queries, config Config, template Template) *handler {
	return &handler{
		db:          db,
		r:           r,
		config:      config,
		callbackURL: fmt.Sprintf("%s/callback", strings.TrimSuffix(config.ExternalHttpAddress, "/")),
		q:           q,
		client:      resty.New().SetHeader("User-Agent", "trim21/submit-patch").SetJSONEscapeHTML(false),
		template:    template,
		//tmpl:   tmpl,
	}
}

type handler struct {
	q           *q.Queries
	config      Config
	callbackURL string
	db          *pgxpool.Pool
	r           rueidis.Client
	client      *resty.Client

	template Template
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

func (h *handler) validateCaptcha(ctx context.Context, turnstileResponseToken string) error {
	var result turnstileResponse
	resp, err := h.client.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"secret":   h.config.TurnstileSecretKey,
			"response": turnstileResponseToken,
		}).
		SetResult(&result).
		Post("https://challenges.cloudflare.com/turnstile/v0/siteverify")
	if err != nil {
		return errors.New("failed to contact captcha verification service")
	}

	if resp.StatusCode() >= 300 {
		return errors.New("captcha verification failed (API error)")
	}

	if !result.Success {
		return errors.New("验证码无效")
	}

	return nil
}

func (h *handler) userContributionView(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.ParseInt(r.PathValue("user-id"), 10, 32)
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return nil
	}

	count, err := h.q.CountPendingPatchesForUser(r.Context(), int32(userID))
	if err != nil {
		return errgo.Wrap(err, "failed to count patches")
	}

	return json.NewEncoder(w).Encode(count)
}

func (h *handler) userReviewView(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.ParseInt(r.PathValue("user-id"), 10, 32)
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return nil
	}

	count, err := h.q.CountPendingPatchesForUser(r.Context(), int32(userID))
	if err != nil {
		return errgo.Wrap(err, "failed to count patches")
	}

	return json.NewEncoder(w).Encode(count)
}
