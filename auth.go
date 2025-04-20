package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
	"github.com/trim21/errgo"

	"app/dal"
	"app/dto"
	"app/session"
)

func (h *handler) loginView(w http.ResponseWriter, r *http.Request) {
	query := url.Values{}
	query.Add("client_id", h.config.BangumiAppId)
	query.Add("response_type", "code")
	query.Add("redirect_uri", h.callbackURL)

	state := uuid.Must(uuid.NewV4()).String()

	http.SetCookie(w, &http.Cookie{
		Name:  "bgm-patch-session",
		Value: state,
	})

	http.Redirect(w, r, oauthURL+"?"+query.Encode(), http.StatusFound)
}

type UserInfo struct {
	ID         int32  `json:"id"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	UserGroup  int    `json:"user_group"`
	TimeOffset int    `json:"time_offset"`
}

func (h *handler) callback(w http.ResponseWriter, r *http.Request) error {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	var oauthResponse dto.OAuthAccessTokenResponse
	resp, err := h.client.R().
		SetFormData(map[string]string{
			"client_id":     h.config.BangumiAppId,
			"client_secret": h.config.BangumiAppSecret,
			"grant_type":    "authorization_code",
			"code":          code,
			"redirect_uri":  h.config.ExternalHttpAddress + "/callback",
		}).
		SetResult(&oauthResponse).
		Post("https://next.bgm.tv/oauth/access_token")

	if err != nil {
		return errgo.Wrap(err, "oauth request")
	}

	if resp.StatusCode() >= 300 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "请求错误", http.StatusBadRequest)
		log.Warn().Str("body", resp.String()).Msg("oauth request failed")
		return nil
	}

	now := resp.ReceivedAt()

	var user UserInfo
	resp, err = h.client.R().
		SetHeader("Authorization", "Bearer "+oauthResponse.AccessToken).
		SetResult(&user).
		Get("https://api.bgm.tv/v0/me")
	if err != nil {
		log.Error().Err(err).Msg("failed to get user info")
		return errgo.Wrap(err, "failed to get user info")
	}
	if resp.StatusCode() >= 300 {
		return fmt.Errorf("failed to get user info, unexpected status code %d", resp.StatusCode())
	}

	err = h.q.UpsertUser(r.Context(), dal.UpsertUserParams{
		UserID:   user.ID,
		Username: user.Username,
		Nickname: html.UnescapeString(user.Nickname),
	})
	if err != nil {
		return errgo.Wrap(err, "failed to upsert user to database")
	}

	err = h.NewSession(r.Context(), w, session.Session{
		UserID:               user.ID,
		GroupID:              user.UserGroup,
		AccessToken:          oauthResponse.AccessToken,
		RefreshToken:         oauthResponse.RefreshToken,
		AccessTokenCreatedAt: now,
		AccessTokenExpiresAt: now.Add(time.Second * time.Duration(oauthResponse.ExpiresIn)),
		Tz:                   user.TimeOffset,
		Key:                  uuid.Must(uuid.NewV4()).String(),
	})
	if err != nil {
		return errgo.Wrap(err, "failed to set session")
	}

	cookie, err := r.Cookie(cookieBackTo)
	if err == nil && cookie.Value != "" {
		if strings.HasPrefix(cookie.Value, "/") {
			http.Redirect(w, r, cookie.Value, http.StatusSeeOther)
			return nil
		}
	}

	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

var ErrLoginRequired = errors.New("need user to login again")

func needLogin(w http.ResponseWriter, r *http.Request, backTo string) {
	http.SetCookie(w, &http.Cookie{Name: cookieBackTo, Value: backTo})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *handler) GetFreshSession(w http.ResponseWriter, r *http.Request, backTo string) (*session.Session, error) {
	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		needLogin(w, r, backTo)
		return nil, ErrLoginRequired
	}
	if s.TokenFresh() {
		return s, nil
	}

	log.Debug().Int32("user_id", s.UserID).Msg("refresh token")

	now := time.Now()
	var body dto.OAuthAccessTokenResponse
	res, err := h.client.R().
		SetResult(&body).
		SetFormData(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": s.RefreshToken,
			"client_id":     h.config.BangumiAppId,
			"redirect_uri":  h.callbackURL,
			"client_secret": h.config.BangumiAppSecret,
		}).Post("https://next.bgm.tv/oauth/access_token")
	if err != nil {
		return nil, errgo.Wrap(err, "failed to refresh access token")
	}
	if res.StatusCode() >= 300 {
		var e dto.ErrorResponse
		if err = json.Unmarshal(res.Body(), &e); err != nil {
			return nil, fmt.Errorf("failed to get user info, unexpected status code %d", res.StatusCode())
		}

		// refresh token expired, redirect to login page
		if e.Code == "INVALID_REFRESH_TOKEN" {
			needLogin(w, r, backTo)
			return nil, ErrLoginRequired
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "请求错误", http.StatusBadRequest)
		log.Warn().Str("body", res.String()).Msg("oauth request failed")
		return nil, fmt.Errorf("failed to refresh access token, unexpected status code %d", res.StatusCode())
	}

	var user UserInfo
	res, err = h.client.R().
		SetHeader("Authorization", "Bearer "+body.AccessToken).
		SetResult(&user).
		Get("https://api.bgm.tv/v0/me")
	if err != nil {
		fmt.Println(err, "failed to get user info")
		return nil, errgo.Wrap(err, "failed to get user info")
	}
	if res.StatusCode() >= 300 {
		return nil, fmt.Errorf("failed to get user info, unexpected status code %d", res.StatusCode())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = h.q.UpsertUser(ctx, dal.UpsertUserParams{
		UserID:   user.ID,
		Username: user.Username,
		Nickname: html.UnescapeString(user.Nickname),
	})
	if err != nil {
		return nil, errgo.Wrap(err, "failed to upsert user to database")
	}

	n := session.Session{
		UserID:               user.ID,
		GroupID:              user.UserGroup,
		AccessToken:          body.AccessToken,
		RefreshToken:         body.RefreshToken,
		AccessTokenCreatedAt: now,
		AccessTokenExpiresAt: now.Add(time.Second * time.Duration(body.ExpiresIn)),
		Tz:                   user.TimeOffset,
		Key:                  s.Key,
	}

	err = h.NewSession(ctx, w, n)
	if err != nil {
		return nil, errgo.Wrap(err, "failed to set session")
	}

	return &n, nil
}
