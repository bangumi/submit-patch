package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/rueidis"
	"github.com/segmentio/kafka-go"

	"app/dal"
	"app/dto"
)

func newHandler(db *pgxpool.Pool, r rueidis.Client, q *dal.Queries, config Config, template Template, k *kafka.Writer) *handler {
	return &handler{
		db:          db,
		r:           r,
		config:      config,
		callbackURL: fmt.Sprintf("%s/callback", strings.TrimSuffix(config.ExternalHttpAddress, "/")),
		q:           q,
		client:      resty.New().SetHeader("User-Agent", "trim21/submit-patch").SetJSONEscapeHTML(false),
		template:    template,
		k:           k,
	}
}

type handler struct {
	q           *dal.Queries
	config      Config
	callbackURL string
	db          *pgxpool.Pool
	r           rueidis.Client
	client      *resty.Client
	k           *kafka.Writer

	template Template
}

func (h *handler) validateCaptcha(ctx context.Context, turnstileResponseToken string) error {
	var result dto.TurnstileResponse
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
		return &HttpError{
			StatusCode: http.StatusBadGateway,
			Message:    "captcha verification failed",
		}
	}

	return nil
}
