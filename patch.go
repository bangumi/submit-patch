package main

import (
	"fmt"
	"net/http"

	"github.com/aymanbagabas/go-udiff"
)

const wikiBotUserID = 427613

func Diff(name, before, after string) string {
	return udiff.Unified(name, name, before, after)
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
