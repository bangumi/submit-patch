package main

import (
	"fmt"
	"net/http"

	"app/csrf"
)

func (h *handler) handleReview(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "must be a valid form", http.StatusBadRequest)
		return nil
	}

	fmt.Println(r.PostForm)

	ok := csrf.Verify(r, r.PostForm.Get(csrf.FormName))
	fmt.Println(ok)

	return nil
}
