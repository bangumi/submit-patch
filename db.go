package main

import (
	"context"
	"time"

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
