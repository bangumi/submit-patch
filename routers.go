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

	r := mux.With(SessionMiddleware(h), csrf.New())

	r.Get("/login", h.loginView)
	r.Get("/callback", handleError(h.callback))

	r.Get("/badge.svg", h.badge)

	r.Get("/", handleError(h.indexView))

	r.Get("/subject/{patchID}", handleError(h.subjectPatchDetailView))
	r.Get("/episode/{patchID}", handleError(h.episodePatchDetailView))
	r.Get("/contrib/{user-id}", handleError(h.userContributionView))
	r.Get("/review/{user-id}", handleError(h.userReviewView))

	r.Post("/api/review/{patch_type}/{patch_id}", handleError(h.handleReview))

	// subjects
	r.Get("/suggest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/edit/subject/%s", r.URL.Query().Get("subject_id")), http.StatusSeeOther)
	})

	r.Get("/edit/subject/{subject-id}", handleError(h.editSubjectView))
	r.Post("/edit/subject/{subject-id}", handleError(h.createSubjectEditPatch))

	// json API to create patch from partial subjects
	r.Patch("/edit/subject/{subject-id}", handleError(h.createSubjectEditPatchAPI))

	r.Get("/edit/patch/subject/{patch-id}", handleError(h.editSubjectPatchView))
	r.Post("/edit/patch/subject/{patch-id}", handleError(h.updateSubjectEditPatch))

	r.Post("/api/delete/patch/subject/{patch-id}", handleError(h.deleteSubjectPatch))

	// episodes
	r.Get("/suggest-episode", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/edit/episode/%s", r.URL.Query().Get("episode_id")), http.StatusSeeOther)
	})

	r.Get("/edit/episode/{episode-id}", handleError(h.editEpisodeView))
	r.Post("/edit/episode/{episode-id}", handleError(h.createEpisodeEditPatch))

	r.Get("/edit/patch/episode/{patch-id}", handleError(h.editEpisodePatchView))
	r.Post("/edit/patch/episode/{patch-id}", handleError(h.updateEpisodeEditPatch))

	// json API to create patch from partial episode
	r.Patch("/edit/episode/{episode-id}", handleError(h.createEpisodeEditPatchAPI))

	r.Post("/api/delete/patch/episode/{patch-id}", handleError(h.deleteEpisodePatch))

	return mux
}

func handleError(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			if errors.Is(err, ErrLoginRequired) {
				return
			}

			var he *HttpError
			if errors.As(err, &he) {
				http.Error(w, he.Message, he.StatusCode)
				return
			}

			log.Error().Err(err).Msg("error")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
		}
	}
}
