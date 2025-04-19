package view

import (
	"net/url"
	"time"

	"app/api"
	"app/q"
	"app/session"
)

type Change struct {
	Diff string
	Name string
}

type IndexPage struct {
	Session *session.Session
	Patches []q.SubjectPatch
}

type SubjectPatchList struct {
	Session            *session.Session
	CurrentStateFilter string
	Patches            []SubjectPatchListItem
	Pagination         Pagination
}

type EpisodePatchList struct {
	Session            *session.Session
	CurrentStateFilter string
	Patches            []EpisodePatchListItem
	Pagination         Pagination
}

type CurrentUser = session.Session

type Pagination struct {
	URL         *url.URL
	TotalPage   int64
	CurrentPage int64
}

type User struct {
	ID       int32
	Username string
	Nickname string
}

type SubjectPatchListItem struct {
	ID            string
	Reason        string
	UpdatedAt     time.Time
	CreatedAt     time.Time
	CommentsCount int32

	State  int32
	Action int32

	Author   User
	Reviewer *User

	Name        string
	SubjectType int64
}

type EpisodePatchListItem struct {
	ID            string
	Reason        string
	UpdatedAt     time.Time
	CreatedAt     time.Time
	CommentsCount int32

	State  int32
	Action int32

	Author   User
	Reviewer *User

	Name string
}

type SubjectPatchEdit struct {
	PatchID   string
	SubjectID uint64
	CsrfToken string

	Reason      string
	Description string

	Data api.WikiSubject

	TurnstileSiteKey string
}
