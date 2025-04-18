package view

import (
	"net/url"

	"app/q"
	"app/session"
)

type IndexPage struct {
	Session *session.Session
	Patches []q.SubjectPatch
}

type SubjectPatchList struct {
	Session    *session.Session
	Patches    []q.SubjectPatch
	Pagination Pagination
}

type CurrentUser = session.Session

type Pagination struct {
	URL         *url.URL
	TotalPage   int64
	CurrentPage int64
}
