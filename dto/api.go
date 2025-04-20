package dto

import (
	"time"
)

type WikiSubject struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	TypeID   int64    `json:"typeID"`
	Infobox  string   `json:"infobox"`
	Platform int      `json:"platform"`
	MetaTags []string `json:"metaTags"`
	Summary  string   `json:"summary"`
	Nsfw     bool     `json:"nsfw"`
}

type WikiEpisode struct {
	ID        int    `json:"id"`
	SubjectID int    `json:"subjectID"`
	Name      string `json:"name"`
	NameCN    string `json:"nameCN"`
	Type      int    `json:"type"`
	Ep        int    `json:"ep"`
	Duration  string `json:"duration"`
	Summary   string `json:"summary"`
	Disc      int    `json:"disc"`
	Date      string `json:"date"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type OAuthAccessTokenResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	UserID       string `json:"user_id"`
}

type TurnstileResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Action      string    `json:"action"`
	CData       string    `json:"cdata"`
}
