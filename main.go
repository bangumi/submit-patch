package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/rueidis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"go.uber.org/fx"

	"app/dal"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.MessageFieldName = "msg"

	var h *handler
	var m *chi.Mux
	var c Config

	err := fx.New(
		fx.Provide(
			newConfig,
			newDB,
			func(p *pgxpool.Pool) *dal.Queries {
				return dal.New(p)
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

	if c.Debug {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	if err := runMigration(c); err != nil {
		log.Err(err).Msg("migration failed")
		panic(err)
	}

	log.Info().Msgf("start listen http://127.0.0.1:%d/", c.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), m)
	if err != nil {
		panic(err)
	}
}

//go:embed db/migrations
var migrations embed.FS

func runMigration(config Config) error {
	log.Info().Msg("start migration")
	db, err := sql.Open("pgx", config.PgDsn)
	if err != nil {
		return err
	}

	driver, err := pgx.WithInstance(db, &pgx.Config{
		MigrationsTable: "patch_tables_migrations",
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"embed", lo.Must(iofs.New(migrations, "db/migrations")),
		"pgx", driver,
	)
	if err != nil {
		return err
	}

	v, d, err := m.Version()
	if err != nil {
		if !errors.Is(err, migrate.ErrNilVersion) {
			return err
		}
		log.Info().Msgf("before migration: version=%d (dirty=%v)", 1, false)
	} else {
		log.Info().Msgf("before migration: version=%d (dirty=%v)", v, d)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	log.Info().Msg("migration done")

	v, d, err = m.Version()
	if err != nil {
		return err
	}

	log.Info().Msgf("after migration: version=%d (dirty=%v)", v, d)
	return nil
}
