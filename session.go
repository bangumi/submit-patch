package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/redis/rueidis"
	"github.com/rs/zerolog/log"
)

type Session struct {
	UserID               int32     `json:"user_id"`
	GroupID              int       `json:"group_id"`
	AccessToken          string    `json:"access_token"`
	RefreshToken         string    `json:"refresh_token"`
	AccessTokenCreatedAt time.Time `json:"access_token_created_at"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	Tz                   int       `json:"tz"`
}

func (s Session) AllowEdit() bool {
	return s.GroupID == 1 || s.GroupID == 2 || s.GroupID == 9 || s.GroupID == 11
}

func (s Session) SuperUser() bool {
	return s.UserID == 287622 || s.UserID == 427613
}

type key int

const sessionKey = key(1)

const cookieName = "bgm-tv-patch-session-id"

func GetSession(ctx context.Context) *Session {
	return ctx.Value(sessionKey).(*Session)
}

const SessionKeyRedisPrefix = "patch:session:"

func (h *handler) SetSession(ctx context.Context, w http.ResponseWriter, session Session) error {
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
			if err != nil {
				log.Err(err).Msg("failed to fetch session from redis")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var session Session
			if err := json.Unmarshal(v, &session); err != nil {
				log.Err(err).Msg("failed to decode session from redis value")
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), sessionKey, &session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
