package session

import (
	"time"

	"github.com/samber/lo"
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

func (s *Session) AllowEdit() bool {
	return s.GroupID == 1 || s.GroupID == 2 || s.GroupID == 9 || s.GroupID == 11
}

func (s *Session) SuperUser() bool {
	return s.UserID == 287622 || s.UserID == 427613
}

var defaultTZ = lo.Must(time.LoadLocation("Asia/Shanghai"))

func (s *Session) LocalTime(t time.Time) string {
	if s.UserID == 0 {
		return t.In(defaultTZ).Format("2006-01-02 15:04:05")
	}

	return t.In(time.FixedZone("", s.Tz*3600)).Format("2006-01-02 15:04:05")
}
