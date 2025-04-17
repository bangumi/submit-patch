package main

import (
	"github.com/rs/zerolog/log"
	"go-simpler.org/env"
)

type Config struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN"`

	BangumiAppId     string `env:"BANGUMI_APP_ID"`
	BangumiAppSecret string `env:"BANGUMI_APP_SECRET"`

	ExternalHttpAddress string `env:"EXTERNAL_HTTP_ADDRESS" default:"http://127.0.0.1:4562"`

	Port uint16 `env:"PORT" default:"4096"`

	RedisDsn string `env:"REDIS_DSN"`

	PgDsn string `env:"PG_DSN"`

	Debug bool `env:"DEBUG" default:"false"`
}

func newConfig() (Config, error) {
	var cfg Config
	log.Info().Msg("load config")
	err := env.Load(&cfg, nil)
	return cfg, err
}
