package main

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	notifypb "app/gen/proto/mq/v1"
)

const notifyTopic = "notify.v1"

// Notify types matching old-server and server-private definitions.
const (
	NotifyTypeSubjectPatchAccepted = 35
	NotifyTypeEpisodePatchAccepted = 36
	NotifyTypeSubjectPatchRejected = 37
	NotifyTypeEpisodePatchRejected = 38
	NotifyTypeSubjectPatchExpired  = 39
	NotifyTypeEpisodePatchExpired  = 40
)

func (h *handler) sendNotify(ctx context.Context, mid uint32, userID uint32, notifyType int32, title string) {
	if h.k == nil {
		return
	}

	data, err := proto.Marshal(&notifypb.Notify{
		Mid:    mid,
		UserId: userID,
		Type:   notifyType,
		Title:  title,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal notification")
		return
	}

	err = h.k.WriteMessages(ctx, kafka.Message{
		Topic: notifyTopic,
		Value: data,
	})
	if err != nil {
		log.Error().Err(err).
			Uint32("user_id", userID).
			Int32("notify_type", notifyType).
			Msg("failed to send notification")
	}
}
