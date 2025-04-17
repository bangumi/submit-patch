package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/rueidis"

	"app/q"
)

func newHandler(db *pgxpool.Pool, r rueidis.Client, q *q.Queries, config Config) *handler {
	return &handler{
		db:     db,
		r:      r,
		config: config,
		q:      q,
		client: resty.New().SetHeader("User-Agent", "trim21/submit-patch"),
		//tmpl:   tmpl,
	}
}

type handler struct {
	q      *q.Queries
	config Config
	db     *pgxpool.Pool
	r      rueidis.Client
	client *resty.Client

	//tmpl *template.Template
}
