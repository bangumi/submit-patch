package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newDB(config Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pool, err := pgxpool.New(ctx, config.PgDsn)
	if err != nil {
		return nil, err
	}

	_, err = pool.Exec(ctx, "select 1;")
	return pool, err
}

func (h *handler) tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		rec := recover()
		if rec != nil {
			tx.Rollback(ctx)
			panic(rec)
		}
	}()

	err = fn(tx)
	if err != nil {
		return errors.Join(err, tx.Rollback(ctx))
	}

	return tx.Commit(ctx)
}
