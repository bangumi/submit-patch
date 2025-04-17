package data

import "app/session"

type IndexPage struct {
	Session *session.Session
}

type CurrentUser = session.Session
