package main

import (
	"fmt"
	"net/http"

	"github.com/aymanbagabas/go-udiff"
)

func Diff(name, before, after string) string {
	return udiff.Unified(name, name, before, after)
}

const PatchTypeSubject string = "subject"
const PatchTypeEpisode string = "episode"

func readableStateToDBValues(state string, defaultState string) ([]int32, string, error) {
	if state == "" {
		state = defaultState
	}

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
		return nil, "", &HttpError{
			StatusCode: http.StatusBadGateway,
			Message:    "invalid patch state",
		}
	}

	return stateVals, state, nil
}

type HttpError struct {
	StatusCode int
	Message    string
}

func (h HttpError) Error() string {
	return fmt.Sprintf("%d: %s", h.StatusCode, h.Message)
}
