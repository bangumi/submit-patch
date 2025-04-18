package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"app/csrf"
)

const CsrfFieldName = "x-csrf-token"

const oauthURL = "https://next.bgm.tv/oauth/authorize"

func routers(h *handler, config Config) *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)

	mux.Mount("/static/", http.FileServer(http.FS(staticFiles)))

	r := mux.With(SessionMiddleware(h), csrf.New())

	r.Get("/login", h.login)
	r.Get("/callback", logError(h.callback))

	r.Get("/", logError(h.index))
	r.Get("/debug", logError(h.debug))

	r.Get("/subject/{patchID}", logError(h.subjectPatchDetail))

	r.Post("/api/review/subject/{patchID}", logError(h.handleReview))

	return mux
}

func logError(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			log.Error().Err(err).Msg("error")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
		}
	}
}
