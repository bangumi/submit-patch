package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/redis/rueidis"
	"github.com/rs/zerolog/log"

	"app/session"
)

type key int

const sessionKey = key(1)

const cookieName = "bgm-tv-patch-session-id"

func GetSession(ctx context.Context) *session.Session {
	s := ctx.Value(sessionKey)
	if s == nil {
		return &session.Session{}
	}

	return s.(*session.Session)
}

const SessionKeyRedisPrefix = "patch:session:"

func (h *handler) SetSession(ctx context.Context, w http.ResponseWriter, session session.Session) error {
	state := uuid.Must(uuid.NewV4()).String()

	err := h.r.Do(ctx, h.r.B().Set().Key(SessionKeyRedisPrefix+state).Value(rueidis.JSON(session)).Ex(time.Hour*24*30).Build()).Error()
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    state,
		HttpOnly: true,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	})

	return nil
}

func SessionMiddleware(h *handler) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
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

			ctx := context.WithValue(r.Context(), sessionKey, &s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
