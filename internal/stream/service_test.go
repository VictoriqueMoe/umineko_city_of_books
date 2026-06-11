package stream

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type streamMocks struct {
	repo     *repository.MockLiveStreamRepository
	lk       *livekit.MockService
	settings *settings.MockService
}

func newTestStreamService(t *testing.T) (Service, *streamMocks) {
	repo := repository.NewMockLiveStreamRepository(t)
	lk := livekit.NewMockService(t)
	settingsSvc := settings.NewMockService(t)

	svc := NewService(repo, lk, settingsSvc, ws.NewHub())

	return svc, &streamMocks{repo: repo, lk: lk, settings: settingsSvc}
}

func expectStreamingEnabled(m *streamMocks, enabled bool) {
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamingEnabled).Return(enabled).Maybe()
	m.lk.EXPECT().Enabled().Return(true).Maybe()
}

func expectMaxConcurrent(m *streamMocks, n int) {
	m.settings.EXPECT().GetInt(mock.Anything, config.SettingStreamMaxConcurrent).Return(n).Maybe()
}

func TestStartStream_Disabled(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, false)

	// when
	_, err := svc.StartStream(context.Background(), uuid.New(), "title")

	// then
	require.ErrorIs(t, err, ErrDisabled)
}

func TestStartStream_TitleRequired(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)

	// when
	_, err := svc.StartStream(context.Background(), uuid.New(), "   ")

	// then
	require.ErrorIs(t, err, ErrTitleRequired)
}

func TestStartStream_AlreadyLive(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	userID := uuid.New()

	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(&repository.LiveStreamRow{ID: uuid.New()}, nil)

	// when
	_, err := svc.StartStream(context.Background(), userID, "title")

	// then
	require.ErrorIs(t, err, ErrAlreadyLive)
}

func TestStartStream_AtCapacity(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	expectMaxConcurrent(m, 3)
	userID := uuid.New()

	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
	m.repo.EXPECT().CountActive(mock.Anything).Return(3, nil)

	// when
	_, err := svc.StartStream(context.Background(), userID, "title")

	// then
	require.ErrorIs(t, err, ErrAtCapacity)
}

func TestStartStream_HappyPath(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	expectMaxConcurrent(m, 3)
	userID := uuid.New()
	streamID := uuid.New()

	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
	m.repo.EXPECT().CountActive(mock.Anything).Return(0, nil)
	m.repo.EXPECT().Create(mock.Anything, userID, "My Stream", 3).Return(streamID, nil)
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID:          streamID,
		UserID:      userID,
		Title:       "My Stream",
		Status:      "starting",
		DisplayName: "Beatrice",
	}, nil)
	m.lk.EXPECT().CreateIngress(mock.Anything, mock.Anything, mock.Anything, "Beatrice").
		Return("ing_1", "https://whip.example/w", "key_123", nil)
	m.repo.EXPECT().SetIngress(mock.Anything, streamID, "ing_1", mock.Anything, "https://whip.example/w", "key_123").Return(nil)

	// when
	resp, err := svc.StartStream(context.Background(), userID, "My Stream")

	// then
	require.NoError(t, err)
	assert.Equal(t, streamID, resp.Stream.ID)
	assert.Equal(t, "https://whip.example/w", resp.WhipURL)
	assert.Equal(t, "key_123", resp.StreamKey)
}

func TestStartStream_CreateRaceMapsErrors(t *testing.T) {
	// given
	cases := []struct {
		name    string
		repoErr error
		want    error
	}{
		{"capacity", repository.ErrLiveStreamCapacity, ErrAtCapacity},
		{"duplicate", repository.ErrLiveStreamActiveExists, ErrAlreadyLive},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestStreamService(t)
			expectStreamingEnabled(m, true)
			expectMaxConcurrent(m, 3)
			userID := uuid.New()

			m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
			m.repo.EXPECT().CountActive(mock.Anything).Return(0, nil)
			m.repo.EXPECT().Create(mock.Anything, userID, "title", 3).Return(uuid.Nil, tc.repoErr)

			// when
			_, err := svc.StartStream(context.Background(), userID, "title")

			// then
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMintViewerToken_NotLive(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	streamID := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{ID: streamID, Status: "starting"}, nil)

	// when
	_, _, err := svc.MintViewerToken(context.Background(), streamID)

	// then
	require.ErrorIs(t, err, ErrStreamNotFound)
}

func TestMintViewerToken_Live_IsSubscribeOnly(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	streamID := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, Status: "live", LivekitRoom: "live_room",
	}, nil)
	m.lk.EXPECT().MintToken("live_room", mock.Anything, "", false, false).Return("tok", nil)
	m.lk.EXPECT().URL().Return("ws://lk")

	// when
	token, url, err := svc.MintViewerToken(context.Background(), streamID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "tok", token)
	assert.Equal(t, "ws://lk", url)
}

func TestStopStream_NotOwner(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: uuid.New(), Status: "live",
	}, nil)

	// when
	err := svc.StopStream(context.Background(), uuid.New(), streamID)

	// then
	require.ErrorIs(t, err, ErrNotOwner)
}

func TestStopStream_HappyPath(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	owner := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: owner, Status: "live", IngressID: "ing",
	}, nil)
	m.repo.EXPECT().MarkOffline(mock.Anything, streamID).Return(true, nil)
	m.lk.EXPECT().DeleteIngress(mock.Anything, "ing").Return(nil)

	// when
	err := svc.StopStream(context.Background(), owner, streamID)

	// then
	require.NoError(t, err)
}

func TestHandleWebhook_NonLiveRoomFallsThrough(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantJoined,
		RoomName: uuid.New().String(),
		Identity: uuid.New().String(),
	}, nil)

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.False(t, handled)
}

func TestHandleWebhook_BroadcasterJoinedMarksLive(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()
	room := "live_" + streamID.String()
	row := &repository.LiveStreamRow{ID: streamID, UserID: userID, Status: "starting", LivekitRoom: room}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantJoined,
		RoomName: room,
		Identity: "broadcaster_" + userID.String(),
	}, nil)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)
	m.repo.EXPECT().MarkLive(mock.Anything, streamID).Return(nil)
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(row, nil)

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.True(t, handled)
}

func TestHandleWebhook_BroadcasterLeftTearsDown(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()
	room := "live_" + streamID.String()
	row := &repository.LiveStreamRow{ID: streamID, UserID: userID, Status: "live", LivekitRoom: room, IngressID: "ing"}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantLeft,
		RoomName: room,
		Identity: "broadcaster_" + userID.String(),
	}, nil)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)
	m.repo.EXPECT().MarkOffline(mock.Anything, streamID).Return(true, nil)
	m.lk.EXPECT().DeleteIngress(mock.Anything, "ing").Return(nil)

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.True(t, handled)
}

func TestHandleWebhook_ViewerJoinedAdjustsCount(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	room := "live_" + streamID.String()
	row := &repository.LiveStreamRow{ID: streamID, Status: "live", LivekitRoom: room}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantJoined,
		RoomName: room,
		Identity: "viewer_" + uuid.New().String(),
	}, nil)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)
	m.repo.EXPECT().AdjustViewerCount(mock.Anything, streamID, 1).Return(1, true, nil)

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.True(t, handled)
}

func TestReconcileOnce_ReapsStaleStarting(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	staleID := uuid.New()

	m.lk.EXPECT().Enabled().Return(true)
	m.repo.EXPECT().ListStartingBefore(mock.Anything, mock.Anything).Return([]repository.LiveStreamRow{
		{ID: staleID, Status: "starting", IngressID: "ing", LivekitRoom: "live_" + staleID.String()},
	}, nil)
	m.repo.EXPECT().MarkOffline(mock.Anything, staleID).Return(true, nil)
	m.lk.EXPECT().DeleteIngress(mock.Anything, "ing").Return(nil)
	m.lk.EXPECT().ActiveRooms(mock.Anything).Return(map[string][]string{}, nil)
	m.repo.EXPECT().ListLive(mock.Anything).Return(nil, nil)

	// when
	n, err := svc.ReconcileOnce(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}
