package main

import (
	"github.com/segmentio/kafka-go"
)

func newKafkaWriter(config Config) *kafka.Writer {
	w := kafka.Writer{
		Addr:                   kafka.TCP(config.KafkaBroker...),
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
	}

	return &w
}
