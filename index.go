package main

import (
	"errors"
	"net/http"
	"strconv"

	"app/q"
	"app/templates"
	"app/view"
)

const defaultPageSize = 30

func (h *handler) debug(w http.ResponseWriter, r *http.Request) error {
	//s := GetSession(r.Context())
	return h.template.DebugPage.ExecuteTemplate(w, "debug.gohtml", nil)
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) error {
	s := GetSession(r.Context())

	if s.UserID == 0 {
		return h.template.LoginPage.ExecuteTemplate(w, "login.gohtml", nil)
	}

	return h.template.LoginPage.ExecuteTemplate(w, "index.gohtml", nil)

	rq := r.URL.Query()

	t := rq.Get("type")
	if t == "" {
		t = string(q.PatchTypeSubject)
	}

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
		stateVals = append(stateVals, PatchStatePending)
	case StateFilterAll:
		stateVals = append(stateVals, PatchStatePending, PatchStateAccepted, PatchStateRejected, PatchStateOutdated)
	case StateFilterAccepted:
		stateVals = append(stateVals, PatchStateAccepted)
	case StateFilterRejected:
		stateVals = append(stateVals, PatchStateRejected)
	case StateFilterReviewed:
		stateVals = append(stateVals, PatchStateRejected, PatchStateOutdated, PatchStateAccepted)
	default:
		return errors.New("invalid patch state")
	}

	c, err := h.q.CountSubjectPatchesByStates(r.Context(), stateVals)
	if err != nil {
		return err
	}

	var patches []q.SubjectPatch
	if c != 0 {
		patches, err = h.q.ListSubjectPatchesByStates(r.Context(), q.ListSubjectPatchesByStatesParams{
			State: stateVals,
			Size:  defaultPageSize,
			Skip:  (currentPage - 1) * defaultPageSize,
		})
		if err != nil {
			return err
		}
	}

	totalPage := (c + defaultPageSize - 1) / defaultPageSize

	_ = templates.SubjectPatchList(r, view.SubjectPatchList{
		Session: GetSession(r.Context()),
		Patches: patches,
		Pagination: view.Pagination{
			URL:         r.URL,
			TotalPage:   totalPage,
			CurrentPage: currentPage,
		},
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
