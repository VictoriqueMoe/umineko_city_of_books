package stream

import (
	"context"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	defaultViewerFlushInterval = 2 * time.Second
)

func (s *service) writeViewers(ctx context.Context, id uuid.UUID, delta int) {
	if delta == 0 {
		return
	}

	count, ok, err := s.repo.AdjustViewerCount(ctx, spec.LiveStreamViewerAdjustment{ID: id, Delta: delta})
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("stream_id", id.String()).Msg("adjust viewer count failed")
		return
	}
	if !ok {
		return
	}

	s.hub.BroadcastPublic(ws.Message{
		Type: wsStreamViewers,
		Data: dto.StreamViewersEvent{StreamID: id, ViewerCount: count},
	})
}
