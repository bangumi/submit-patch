package main

import (
	"net/http"
	"strconv"

	"app/view"
)

const defaultPageSize = 30

func (h *handler) debug(w http.ResponseWriter, r *http.Request) error {
	//s := GetSession(r.Context())
	rq := r.URL.Query()
	rawPage := rq.Get("page")
	currentPage, err := strconv.ParseInt(rawPage, 10, 64)
	if err != nil {
		currentPage = 1
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	return h.template.Executor.ExecuteTemplate(w, "debug.gohtml", view.Pagination{URL: r.URL, TotalPage: 10, CurrentPage: currentPage})
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) error {
	s := GetSession(r.Context())

	if s.UserID == 0 {
		return h.template.Executor.ExecuteTemplate(w, "login.gohtml", nil)
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
	var stateVals = make([]int32, 0, 5)
	switch state {
	case "", StateFilterPending:
		state = StateFilterPending
		stateVals = append(stateVals, PatchStatePending)
	case StateFilterAll:
		state = StateFilterAll
		stateVals = append(stateVals, PatchStatePending, PatchStateAccepted, PatchStateRejected, PatchStateOutdated)
	case StateFilterAccepted:
		state = StateFilterAccepted
		stateVals = append(stateVals, PatchStateAccepted)
	case StateFilterRejected:
		state = StateFilterRejected
		stateVals = append(stateVals, PatchStateRejected)
	case StateFilterReviewed:
		state = StateFilterReviewed
		stateVals = append(stateVals, PatchStateRejected, PatchStateOutdated, PatchStateAccepted)
	default:
		http.Error(w, "invalid patch state", http.StatusBadRequest)
		return nil
	}

	t := rq.Get("type")
	switch t {
	case "", "subject":
		return h.listSubjectPatches(w, r, state, stateVals, currentPage)
	case "episode":
		return h.listEpisodePatches(w, r, stateVals, currentPage)
	}

	http.Error(w, "invalid patch type", http.StatusBadRequest)
	return nil
}

func (h *handler) listEpisodePatches(
	w http.ResponseWriter,
	r *http.Request,
	stateVals []int32,
	currentPage int64,
) error {
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
