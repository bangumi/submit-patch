package main

import (
	"net/http"
	"strconv"

	"app/session"
	"app/templates"
)

const defaultPageSize = 30

func (h *handler) indexView(w http.ResponseWriter, r *http.Request) error {
	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		return templates.Login().Render(r.Context(), w)
	}

	rq := r.URL.Query()

	rawPage := rq.Get("page")
	currentPage, err := strconv.ParseInt(rawPage, 10, 64)
	if err != nil {
		currentPage = 1
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	state := rq.Get("state")
	stateVals, state, err := readableStateToDBValues(state, StateFilterPending)
	if err != nil {
		return err
	}

	t := rq.Get("type")
	switch t {
	case "", "subject":
		return h.listSubjectPatches(w, r, state, stateVals, currentPage)
	case "episode":
		return h.listEpisodePatches(w, r, state, stateVals, currentPage)
	}

	http.Error(w, "invalid patch type", http.StatusBadRequest)
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
