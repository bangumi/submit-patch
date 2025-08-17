package main

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

func getKafkaWriter(config Config) {
	w := kafka.Writer{Addr: kafka.TCP(config.KafkaAddr)}

	err := w.WriteMessages(context.Background(), kafka.Message{
		Topic: "",
		Key:   nil,
		Value: []byte("Hello World"),
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to write messages")
	}
}
