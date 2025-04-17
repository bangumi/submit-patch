package main

import (
	"net/http"

	"app/data"
	"app/q"
	"app/templates"
)

func (h *handler) index(w http.ResponseWriter, r *http.Request) error {
	s := GetSession(r.Context())

	if s.UserID == 0 {
		_ = templates.Login().Render(r.Context(), w)
		return nil
	}

	rq := r.URL.Query()

	t := rq.Get("type")
	if t == "" {
		t = string(q.PatchTypeSubject)
	}

	state := rq.Get("state")
	if state == "" {
		state = "pending"
	}

	_ = templates.Index(r, data.IndexPage{
		Session: GetSession(r.Context()),
	}).Render(r.Context(), w)
	return nil
}

const PatchStatePending = 0
const PatchStateAccept = 1
const PatchStateRejected = 2
const PatchStateOutdated = 3
