package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"app/dal"
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
	log.Debug().
		Uint32("mid", mid).
		Uint32("user_id", userID).
		Int32("notify_type", notifyType).
		Str("title", title).
		Msg("sending notification")

	if h.k == nil {
		return
	}

	data, err := proto.Marshal(&notifypb.Notify{
		Mid:        mid,
		UserId:     userID,
		FromUserId: 427613,
		Type:       notifyType,
		Title:      title,
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

func (h *handler) sendNotifySubjectPatchAccepted(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifySubjectPatchAccepted")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), NotifyTypeSubjectPatchAccepted, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifySubjectPatchRejected(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifySubjectPatchRejected")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), NotifyTypeSubjectPatchRejected, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifySubjectPatchExpired(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifySubjectPatchExpired")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), NotifyTypeSubjectPatchExpired, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyEpisodePatchAccepted(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyEpisodePatchAccepted")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), NotifyTypeEpisodePatchAccepted, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyEpisodePatchRejected(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyEpisodePatchRejected")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), NotifyTypeEpisodePatchRejected, fmt.Sprintf("#%d", numID))
}

func (h *handler) sendNotifyEpisodePatchExpired(ctx context.Context, numID int64, fromUserID int32) {
	log.Debug().Int64("num_id", numID).Int32("from_user_id", fromUserID).Msg("sendNotifyEpisodePatchExpired")
	h.sendNotify(ctx, uint32(numID), uint32(fromUserID), NotifyTypeEpisodePatchExpired, fmt.Sprintf("#%d", numID))
}

// TODO: implement character patch expired notification
func (h *handler) sendNotifyCharacterPatchExpired(_ context.Context, p dal.GetPendingCharacterPatchesByCharacterIDRow) {
	log.Debug().Stringer("patch_id", p.ID).Msg("sendNotifyCharacterPatchExpired (not implemented)")
}

// TODO: implement person patch expired notification
func (h *handler) sendNotifyPersonPatchExpired(_ context.Context, p dal.GetPendingPersonPatchesByPersonIDRow) {
	log.Debug().Stringer("patch_id", p.ID).Msg("sendNotifyPersonPatchExpired (not implemented)")
}
