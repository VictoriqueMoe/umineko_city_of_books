package stream

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"umineko_city_of_books/internal/coalesce"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleWebhook_ViewerBurstWritesOneSummedUpdate(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	impl := svc.(*service)
	impl.viewers = coalesce.NewKeyed(500*time.Millisecond, coalesce.Sum[int], impl.writeViewers)
	streamID := uuid.New()
	room := "live_" + streamID.String()
	row := &model.LiveStreamRow{ID: streamID, Status: "live", LivekitRoom: room}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantJoined,
		RoomName: room,
		Identity: "viewer_" + uuid.New().String(),
	}, nil).Times(4)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil).Times(4)
	m.repo.EXPECT().AdjustViewerCount(mock.Anything, spec.LiveStreamViewerAdjustment{ID: streamID, Delta: 1}).Return(1, true, nil).Once()

	var trailing atomic.Int32
	m.repo.EXPECT().AdjustViewerCount(mock.Anything, spec.LiveStreamViewerAdjustment{ID: streamID, Delta: 3}).
		RunAndReturn(func(context.Context, spec.LiveStreamViewerAdjustment, ...*sql.Tx) (int, bool, error) {
			trailing.Add(1)

			return 4, true, nil
		}).Once()

	// when
	for range 4 {
		handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))
		require.NoError(t, err)
		assert.True(t, handled)
	}

	// then
	assert.Eventually(t, func() bool { return trailing.Load() == 1 }, 3*time.Second, 10*time.Millisecond)
}

func TestWriteViewers_ANetZeroChangeWritesNothing(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)

	// when
	svc.(*service).writeViewers(context.Background(), uuid.New(), 0)

	// then
	m.repo.AssertNumberOfCalls(t, "AdjustViewerCount", 0)
}
