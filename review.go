package main

import (
	"fmt"
	"net/http"
)

func (h *handler) handleReview(w http.ResponseWriter, r *http.Request) error {
	fmt.Println(r.Form)

	return nil
}
