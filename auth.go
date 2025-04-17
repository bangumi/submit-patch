package main

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/trim21/errgo"

	"app/q"
	"app/session"
)

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	query := url.Values{}
	query.Add("client_id", h.config.BangumiAppId)
	query.Add("response_type", "code")
	query.Add("redirect_uri", fmt.Sprintf("%s/callback", h.config.ExternalHttpAddress))

	state := uuid.Must(uuid.NewV4()).String()

	http.SetCookie(w, &http.Cookie{
		Name:  "bgm-patch-session",
		Value: state,
	})

	http.Redirect(w, r, oauthURL+"?"+query.Encode(), http.StatusFound)
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

func (h *handler) callback(w http.ResponseWriter, r *http.Request) error {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	var oauthResponse OAuthAccessTokenResponse
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

	err = h.q.UpsertUser(r.Context(), q.UpsertUserParams{
		UserID:   int32(userID),
		Username: user.Username,
		Nickname: html.UnescapeString(user.Nickname),
	})
	if err != nil {
		return errgo.Wrap(err, "failed to upsert user to database")
	}

	err = h.SetSession(r.Context(), w, session.Session{
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
}
