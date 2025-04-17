package main

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
	"github.com/trim21/errgo"

	"app/q"
	"app/templates"
)

const oauthURL = "https://next.bgm.tv/oauth/authorize"

func routers(h *handler, config Config) *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)

	mux.Mount("/static/", http.FileServer(http.FS(staticFiles)))

	r := mux.With(SessionMiddleware(h))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		s := GetSession(r.Context())

		if s.UserID == 0 {
			_ = templates.Layout("登录", templates.Empty(), templates.Login()).Render(r.Context(), w)
			return
		}

		_ = templates.Layout("首页", templates.Empty(), templates.Hello("world")).Render(r.Context(), w)
	})

	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		query := url.Values{}
		query.Add("client_id", config.BangumiAppId)
		query.Add("response_type", "code")
		query.Add("redirect_uri", fmt.Sprintf("%s/callback", config.ExternalHttpAddress))

		state := uuid.Must(uuid.NewV4()).String()

		http.SetCookie(w, &http.Cookie{
			Name:  "bgm-patch-session",
			Value: state,
		})

		http.Redirect(w, r, oauthURL+"?"+query.Encode(), http.StatusFound)
	})

	r.Get("/callback", logError(func(w http.ResponseWriter, r *http.Request) error {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		var oauthResponse OAuthAccessTokenResponse
		resp, err := h.client.R().
			SetFormData(map[string]string{
				"client_id":     config.BangumiAppId,
				"client_secret": config.BangumiAppSecret,
				"grant_type":    "authorization_code",
				"code":          code,
				"redirect_uri":  config.ExternalHttpAddress + "/callback",
			}).
			SetResult(&oauthResponse).
			Post("https://next.bgm.tv/oauth/access_token")

		if err != nil {
			return errgo.Wrap(err, "oauth request")
		}

		if resp.StatusCode() >= 300 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			http.Error(w, "请求错误", http.StatusBadRequest)
			return nil
		}

		now := resp.ReceivedAt()

		userID, err := strconv.ParseInt(oauthResponse.UserID, 10, 32)
		if err != nil {
			return errgo.Wrap(err, "unexpected strconv failure")
		}

		var user UserInfo
		resp, err = h.client.R().
			SetHeader("Authorization", "Bearer "+oauthResponse.AccessToken).
			SetResult(&user).
			Get("https://api.bgm.tv/v0/me")
		if err != nil {
			fmt.Println(err, "failed to get user info")
			return errgo.Wrap(err, "failed to get user info")
		}

		if resp.StatusCode() >= 300 {
			return fmt.Errorf("failed to get user info, unexpected status code %d", resp.StatusCode())
		}

		fmt.Println(user)

		err = h.q.UpsertUser(r.Context(), q.UpsertUserParams{
			UserID:   int32(userID),
			Username: user.Username,
			Nickname: html.UnescapeString(user.Nickname),
		})
		if err != nil {
			return errgo.Wrap(err, "failed to upsert user to database")
		}

		err = h.SetSession(r.Context(), w, Session{
			UserID:               int32(userID),
			GroupID:              user.UserGroup,
			AccessToken:          oauthResponse.AccessToken,
			RefreshToken:         oauthResponse.RefreshToken,
			AccessTokenCreatedAt: now,
			AccessTokenExpiresAt: now.Add(time.Second * time.Duration(oauthResponse.ExpiresIn)),
			Tz:                   oauthResponse.TimeOffset,
		})
		if err != nil {
			return errgo.Wrap(err, "failed to set session")
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	}))

	return mux
}

type UserInfo struct {
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	UserGroup int    `json:"user_group"`
}

type OAuthAccessTokenResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	UserID       string `json:"user_id"`
	TimeOffset   int    `json:"time_offset"`
}

func logError(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			log.Error().Err(err).Msg("error")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
		}
	}
}
