package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"

	"app/csrf"
)

const oauthURL = "https://next.bgm.tv/oauth/authorize"

func routers(h *handler) *chi.Mux {
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

	r.Post("/api/review/{patch-type}/{patch-id}", handleError(h.handleReview))

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

	// other json APIS
	r.Get("/api/subject/pending", handleError(func(w http.ResponseWriter, r *http.Request) error {
		rows, err := h.q.ListPendingSubjectPatches(r.Context())
		if err != nil {
			return err
		}

		type Res struct {
			ID        uuid.UUID `json:"id"`
			SubjectID int32     `json:"subject_id"`
			FromUser  int32     `json:"from_user"`
			CreatedAt int64     `json:"created_at"`
			UpdatedAt int64     `json:"updated_at"`
		}

		var res = make([]Res, 0, len(rows))

		for _, row := range rows {
			res = append(res, Res{
				ID:        row.ID,
				SubjectID: row.SubjectID,
				FromUser:  row.FromUserID,
				CreatedAt: row.CreatedAt.Time.Unix(),
				UpdatedAt: row.UpdatedAt.Time.Unix(),
			})
		}

		w.Header().Set("content-type", contentTypeApplicationJSON)
		w.WriteHeader(http.StatusOK)
		return json.NewEncoder(w).Encode(map[string]any{
			"data": res,
		})
	}))

	r.Get("/api/episode/pending", handleError(func(w http.ResponseWriter, r *http.Request) error {
		rows, err := h.q.ListPendingEpisodePatches(r.Context())
		if err != nil {
			return err
		}

		type Res struct {
			ID        uuid.UUID `json:"id"`
			EpisodeID int32     `json:"episode_id"`
			FromUser  int32     `json:"from_user"`
			CreatedAt int64     `json:"created_at"`
			UpdatedAt int64     `json:"updated_at"`
		}

		var res = make([]Res, 0, len(rows))

		for _, row := range rows {
			res = append(res, Res{
				ID:        row.ID,
				EpisodeID: row.EpisodeID,
				FromUser:  row.FromUserID,
				CreatedAt: row.CreatedAt.Time.Unix(),
				UpdatedAt: row.UpdatedAt.Time.Unix(),
			})
		}

		w.Header().Set("content-type", contentTypeApplicationJSON)
		w.WriteHeader(http.StatusOK)
		return json.NewEncoder(w).Encode(map[string]any{
			"data": res,
		})
	}))

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
