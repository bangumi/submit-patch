package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/rueidis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"go.uber.org/fx"

	"app/q"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.MessageFieldName = "msg"

	var h *handler
	var m *chi.Mux
	var c Config

	err := fx.New(
		fx.Provide(
			newConfig,
			newDB,
			func(p *pgxpool.Pool) *q.Queries {
				return q.New(p)
			},
			newHandler,
			loadTemplates,
			routers,
			func(config Config) (rueidis.Client, error) {
				redisDSN := lo.Must(url.Parse(config.RedisDsn))
				redisPassword, _ := redisDSN.User.Password()
				return rueidis.NewClient(rueidis.ClientOption{
					Password:    redisPassword,
					InitAddress: []string{redisDSN.Host}},
				)
			},
		),
		fx.Populate(&h, &m, &c),
	).Err()

	if err != nil {
		panic(err)
	}

	log.Info().Msgf("start listen http://127.0.0.1:%d/", c.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), m)
	if err != nil {
		panic(err)
	}
}
