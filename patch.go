package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aymanbagabas/go-udiff"

	"app/internal/myers"
)

const wikiBotUserID = 427613

func Diff(name, before, after string) string {
	edits := myers.ComputeEdits(before, after)
	unified, err := udiff.ToUnified(name, name, before, edits, 3)
	if err != nil {
		// Can't happen: edits are consistent.
		log.Fatalf("internal error in diff.Unified: %v", err)
	}

	return unified
}

const PatchTypeSubject string = "subject"
const PatchTypeEpisode string = "episode"

const OrderByCreatedAt string = "created_at"
const OrderByUpdatedAt string = "updated_at"

func readableStateToDBValues(state string, defaultState string) ([]int32, string, string, error) {
	if state == "" {
		state = defaultState
	}

	var order string

	var stateVals = make([]int32, 0, 5)
	switch state {
	case StateFilterPending:
		order = OrderByCreatedAt
		state = StateFilterPending
		stateVals = append(stateVals, PatchStatePending)
	case StateFilterAll:
		order = OrderByCreatedAt
		state = StateFilterAll
		stateVals = append(stateVals, PatchStatePending, PatchStateAccepted, PatchStateRejected, PatchStateOutdated)
	case StateFilterAccepted:
		order = OrderByUpdatedAt
		state = StateFilterAccepted
		stateVals = append(stateVals, PatchStateAccepted)
	case StateFilterRejected:
		order = OrderByUpdatedAt
		state = StateFilterRejected
		stateVals = append(stateVals, PatchStateRejected)
	case StateFilterReviewed:
		order = OrderByUpdatedAt
		state = StateFilterReviewed
		stateVals = append(stateVals, PatchStateRejected, PatchStateOutdated, PatchStateAccepted)
	default:
		return nil, "", "", &HttpError{
			StatusCode: http.StatusBadGateway,
			Message:    "invalid patch state",
		}
	}

	return stateVals, state, order, nil
}

type HttpError struct {
	StatusCode int
	Message    string
}

func (h *HttpError) Error() string {
	return fmt.Sprintf("%d: %s", h.StatusCode, h.Message)
}
