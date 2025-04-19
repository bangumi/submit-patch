package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"app/csrf"
)

const oauthURL = "https://next.bgm.tv/oauth/authorize"

func routers(h *handler, config Config) *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)

	mux.Mount("/static/", http.FileServer(http.FS(staticFiles)))

	r := mux.With(SessionMiddleware(h), csrf.New())

	r.Get("/login", h.loginView)
	r.Get("/callback", logError(h.callback))

	r.Get("/", logError(h.indexView))
	r.Get("/debug", logError(h.debugView))

	r.Get("/subject/{patchID}", logError(h.subjectPatchDetailView))
	r.Get("/episode/{patchID}", logError(h.episodePatchDetailView))

	r.Post("/api/review/{patch_type}/{patch_id}", logError(h.handleReview))

	// subjects
	r.Get("/suggest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/edit/subject/%s", r.URL.Query().Get("subject-id")), http.StatusSeeOther)
	})

	r.Get("/edit/subject/{subject-id}", logError(h.editSubjectView))
	r.Post("/edit/subject/{subject-id}", logError(h.createSubjectEditPatch))

	r.Get("/edit/patch/subject/{patch-id}", logError(h.editSubjectPatchView))
	r.Post("/edit/patch/subject/{patch-id}", logError(h.updateSubjectEditPatch))

	r.Post("/api/delete/patch/subject/{patch-id}", logError(h.deleteSubjectPatch))

	// episodes
	r.Get("/suggest-episode", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/edit/episode/%s", r.URL.Query().Get("episode_id")), http.StatusSeeOther)
	})

	r.Get("/edit/episode/{episode-id}", logError(h.editEpisodeView))
	r.Post("/edit/episode/{episode-id}", logError(h.createEpisodeEditPatch))

	r.Get("/edit/patch/episode/{patch-id}", logError(h.editEpisodePatchView))
	r.Post("/edit/patch/episode/{patch-id}", logError(h.updateSubjectEditPatch))

	r.Post("/api/delete/patch/episode/{patch-id}", logError(h.deleteEpisodePatch))

	return mux
}

func logError(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			if errors.Is(err, ErrLoginRequired) {
				return
			}

			log.Error().Err(err).Msg("error")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
		}
	}
}
