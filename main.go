package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"app/templates"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.MessageFieldName = "msg"

	var h *handler
	var m *chi.Mux

	err := fx.New(
		fx.Provide(
			newDB,
			newHandler,
			routers,
		),
		fx.Populate(&h, &m),
	).Err()

	if err != nil {
		panic(err)
	}

	log.Info().Msg("start listen")
	err = http.ListenAndServe(":4096", m)
	if err != nil {
		panic(err)
	}
}

func newDB() (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pool, err := pgxpool.New(ctx, os.Getenv("PG_DSN"))
	if err != nil {
		return nil, err
	}

	_, err = pool.Exec(ctx, "select 1;")
	return pool, err
}

func newHandler(db *pgxpool.Pool) *handler {
	return &handler{
		db: db,
	}
}

type handler struct {
	db *pgxpool.Pool
}

func routers(h *handler) *chi.Mux {
	mux := chi.NewRouter()

	mux.Mount("/static/", http.FileServer(http.FS(staticFiles)))

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = templates.Index(templates.Empty(), templates.Hello("world")).Render(r.Context(), w)
	})

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = templates.Index(templates.Empty(), templates.Hello("world")).Render(r.Context(), w)
	})

	return mux
}
