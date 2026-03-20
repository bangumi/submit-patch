package main

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protowire"
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

// encodeNotify encodes a Notify protobuf message matching mq.v1.Notify:
//
//	message Notify {
//	  uint32 mid = 1;
//	  uint32 user_id = 2;
//	  uint32 from_user_id = 3;
//	  int32 type = 4;
//	  string title = 5;
//	}
func encodeNotify(mid uint32, userID uint32, fromUserID uint32, notifyType int32, title string) []byte {
	var buf []byte
	if mid != 0 {
		buf = protowire.AppendTag(buf, 1, protowire.VarintType)
		buf = protowire.AppendVarint(buf, uint64(mid))
	}
	if userID != 0 {
		buf = protowire.AppendTag(buf, 2, protowire.VarintType)
		buf = protowire.AppendVarint(buf, uint64(userID))
	}
	if fromUserID != 0 {
		buf = protowire.AppendTag(buf, 3, protowire.VarintType)
		buf = protowire.AppendVarint(buf, uint64(fromUserID))
	}
	if notifyType != 0 {
		buf = protowire.AppendTag(buf, 4, protowire.VarintType)
		buf = protowire.AppendVarint(buf, uint64(notifyType))
	}
	if title != "" {
		buf = protowire.AppendTag(buf, 5, protowire.BytesType)
		buf = protowire.AppendString(buf, title)
	}
	return buf
}

func (h *handler) sendNotify(ctx context.Context, mid uint32, userID uint32, notifyType int32, title string) {
	if h.k == nil {
		return
	}

	data := encodeNotify(mid, userID, 0, notifyType, title)
	err := h.k.WriteMessages(ctx, kafka.Message{
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
