package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/rueidis"

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
		SetResult(&result). // Decode into result on success (2xx)
		Post("https://challenges.cloudflare.com/turnstile/v0/siteverify")
	if err != nil {
		fmt.Printf("Error executing Turnstile request: %v\n", err)
		return errors.New("failed to contact captcha verification service")
	}

	if resp.IsError() {
		// Turnstile API returned an error status code (>= 400)
		fmt.Printf("Turnstile API error: status %d, body: %s\n", resp.StatusCode(), resp.String())
		// You could potentially check apiError["error-codes"] here if needed
		return errors.New("captcha verification failed (API error)")
	}

	// We got a 2xx response, now check the 'success' field in the JSON
	if !result.Success {
		// Log error codes if needed: fmt.Printf("Turnstile verification failed: %v\n", result.ErrorCodes)
		return errors.New("验证码无效") // "Invalid CAPTCHA"
	}

	return nil
}
