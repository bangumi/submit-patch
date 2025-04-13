package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"app/templates"
)

func main() {
	fmt.Println("hello world")

	mux := chi.NewRouter()

	mux.Mount("/static/", http.FileServer(http.FS(staticFiles)))

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = templates.Index(templates.Empty(), templates.Hello("world")).Render(r.Context(), w)
	})

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = templates.Index(templates.Empty(), templates.Hello("world")).Render(r.Context(), w)
	})

	_ = http.ListenAndServe(":4096", mux)
}
