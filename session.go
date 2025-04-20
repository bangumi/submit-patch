package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/redis/rueidis"
	"github.com/rs/zerolog/log"

	"app/session"
)

const SessionKeyRedisPrefix = "patch:session:"

func (h *handler) NewSession(ctx context.Context, w http.ResponseWriter, s session.Session) error {
	return h.setSession(ctx, w, &s)
}

func (h *handler) setSession(ctx context.Context, w http.ResponseWriter, s *session.Session) error {
	err := h.r.Do(ctx, h.r.B().Set().Key(SessionKeyRedisPrefix+s.Key).Value(rueidis.JSON(s)).Ex(time.Hour*24*30).Build()).Error()
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    s.Key,
		HttpOnly: true,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	})

	return nil
}

func SessionMiddleware(h *handler) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(session.CookieName)
			if err != nil { // no cookie, do nothing
				next.ServeHTTP(w, r)
				return
			}

			v, err := h.r.Do(r.Context(), h.r.B().Get().Key(SessionKeyRedisPrefix+c.Value).Build()).AsBytes()
			if rueidis.IsRedisNil(err) {
				next.ServeHTTP(w, r)
				return
			}

			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				log.Err(err).Msg("failed to fetch session from redis")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var s session.Session
			if err := json.Unmarshal(v, &s); err != nil {
				log.Err(err).Msg("failed to decode session from redis value")
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(session.SetSession(r.Context(), &s)))
		})
	}
}
