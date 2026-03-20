package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"syscall"
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
	"github.com/trim21/errgo"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	"app/dal"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.MessageFieldName = "msg"
	zerolog.ErrorMarshalFunc = errgo.ZerologErrorMarshaler

	var h *handler
	var m *chi.Mux
	var c Config

	err := fx.New(
		fx.NopLogger,
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

	log.Logger = log.Logger.With().Stack().Caller().Logger()

	if err := runMigration(c); err != nil {
		log.Err(err).Msg("migration failed")
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info().Msgf("start listen http://127.0.0.1:%d/", c.Port)
		srv := &http.Server{Addr: fmt.Sprintf(":%d", c.Port), Handler: m}
		// shut down the server when ctx is cancelled (other component exited)
		go func() {
			<-ctx.Done()
			srv.Shutdown(context.Background()) //nolint:errcheck
		}()
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	c.KafkaTopics = slices.DeleteFunc(c.KafkaTopics, func(s string) bool { return s == "" })
	if c.KafkaBroker != "" && len(c.KafkaTopics) > 0 {
		g.Go(func() error {
			return startCanalConsumer(ctx, c, h)
		})
	}

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("exiting")
		os.Exit(1)
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
