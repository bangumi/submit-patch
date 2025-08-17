package main

import (
	"github.com/rs/zerolog/log"
	"go-simpler.org/env"
)

type Config struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN"`

	AdminToken string `env:"ADMIN_TOKEN"`

	KafkaAddr string `env:"KAFKA_ADDR"`

	BangumiAppId     string `env:"BGM_TV_APP_ID"`
	BangumiAppSecret string `env:"BGM_TV_APP_SECRET"`

	ExternalHttpAddress string `env:"EXTERNAL_HTTP_ADDRESS" default:"http://127.0.0.1:4562"`

	Port uint16 `env:"PORT" default:"4562"`

	RedisDsn string `env:"REDIS_DSN"`

	PgDsn string `env:"PG_DSN"`

	Debug bool `env:"DEBUG" default:"false"`

	TurnstileSiteKey   string `env:"TURNSTILE_SITE_KEY"`
	TurnstileSecretKey string `env:"TURNSTILE_SECRET_KEY"`
}

func newConfig() (Config, error) {
	var cfg Config
	log.Info().Msg("load config")
	err := env.Load(&cfg, nil)
	return cfg, err
}
