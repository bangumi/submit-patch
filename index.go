package main

import (
	"errors"
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

	var stateVals = make([]int32, 0, 5)

	switch state {
	case "", StateFilterPending:
		stateVals = append(stateVals, PatchStatePending)
	case StateFilterAll:
		stateVals = append(stateVals, PatchStatePending, PatchStateAccepted, PatchStateRejected, PatchStateOutdated)
	case StateFilterAccepted:
		stateVals = append(stateVals, PatchStateAccepted)
	case StateFilterRejected:
		stateVals = append(stateVals, PatchStateRejected)
	default:
		return errors.New("invalid patch state")
	}

	c, err := h.q.CountSubjectPatchesByStates(r.Context(), stateVals)
	if err != nil {
		return err
	}

	_ = templates.Index(r, data.IndexPage{
		Session: GetSession(r.Context()),
	}).Render(r.Context(), w)
	return nil
}

const PatchStatePending = 0
const PatchStateAccepted = 1
const PatchStateRejected = 2
const PatchStateOutdated = 3

const StateFilterPending = "pending"
const StateFilterReviewed = "reviewed"
const StateFilterAll = "all"
const StateFilterRejected = "rejected"
const StateFilterAccepted = "accepted"
