package stream

import (
	"bytes"
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type streamMocks struct {
	repo       *repository.MockLiveStreamRepository
	creds      *repository.MockStreamCredentialsRepository
	followRepo *repository.MockFollowRepository
	lk         *livekit.MockService
	settings   *settings.MockService
	upload     *upload.MockService
	notif      *notification.MockService
}

func newTestStreamService(t *testing.T) (Service, *streamMocks) {
	repo := repository.NewMockLiveStreamRepository(t)
	creds := repository.NewMockStreamCredentialsRepository(t)
	followRepo := repository.NewMockFollowRepository(t)
	lk := livekit.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	notifSvc := notification.NewMockService(t)

	svc := NewService(repo, creds, followRepo, lk, settingsSvc, uploadSvc, notifSvc, ws.NewHub())

	settingsSvc.EXPECT().Get(mock.Anything, config.SettingStreamHLSOutputDir).Return("").Maybe()
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	notifSvc.EXPECT().HasRecentFromActor(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false).Maybe()
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()

	return svc, &streamMocks{repo: repo, creds: creds, followRepo: followRepo, lk: lk, settings: settingsSvc, upload: uploadSvc, notif: notifSvc}
}

func credentialsSpec(userID uuid.UUID, ingressID, whipURL, streamKey string) any {
	return mock.MatchedBy(func(spec repository.NewStreamCredentials) bool {
		return spec.UserID == userID &&
			spec.IngressID == ingressID &&
			spec.WhipURL == whipURL &&
			spec.StreamKey == streamKey
	})
}

func activationSpec(streamID uuid.UUID, ingressID, whipURL, streamKey, defaultMode string) any {
	return mock.MatchedBy(func(spec repository.LiveStreamActivation) bool {
		return spec.Ingress.ID == streamID &&
			spec.Ingress.IngressID == ingressID &&
			spec.Ingress.WhipURL == whipURL &&
			spec.Ingress.StreamKey == streamKey &&
			spec.DefaultMode == defaultMode
	})
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
	_, err := svc.StartStream(context.Background(), uuid.New(), "title", dto.StreamDefaultModeWebRTC, 6000)

	// then
	require.ErrorIs(t, err, ErrDisabled)
}

func TestStartStream_TitleRequired(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)

	// when
	_, err := svc.StartStream(context.Background(), uuid.New(), "   ", dto.StreamDefaultModeWebRTC, 6000)

	// then
	require.ErrorIs(t, err, ErrTitleRequired)
}

func TestStartStream_InvalidBitrate(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(true)

	// when
	_, err := svc.StartStream(context.Background(), uuid.New(), "title", dto.StreamDefaultModeWebRTC, 3)

	// then
	require.ErrorIs(t, err, ErrInvalidBitrate)
}

func TestStartStream_AlreadyLive(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)
	userID := uuid.New()

	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(&repository.LiveStreamRow{ID: uuid.New()}, nil)

	// when
	_, err := svc.StartStream(context.Background(), userID, "title", dto.StreamDefaultModeWebRTC, 6000)

	// then
	require.ErrorIs(t, err, ErrAlreadyLive)
}

func TestStartStream_AtCapacity(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	expectMaxConcurrent(m, 3)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)
	userID := uuid.New()

	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
	m.repo.EXPECT().CountActive(mock.Anything).Return(3, nil)

	// when
	_, err := svc.StartStream(context.Background(), userID, "title", dto.StreamDefaultModeWebRTC, 6000)

	// then
	require.ErrorIs(t, err, ErrAtCapacity)
}

func TestStartStream_HappyPath(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	expectMaxConcurrent(m, 3)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)
	userID := uuid.New()
	streamID := uuid.New()

	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
	m.repo.EXPECT().CountActive(mock.Anything).Return(0, nil)
	m.repo.EXPECT().Create(mock.Anything, userID, "My Stream", 3).Return(&repository.LiveStreamRow{
		ID:          streamID,
		UserID:      userID,
		Title:       "My Stream",
		Status:      "starting",
		DisplayName: "Beatrice",
	}, nil)
	m.creds.EXPECT().Get(mock.Anything, userID).Return(nil, nil)
	m.lk.EXPECT().CreateIngress(mock.Anything, mock.Anything, mock.Anything, "Beatrice").
		Return("ing_1", "https://whip.example/w", "key_123", nil)
	m.creds.EXPECT().Upsert(mock.Anything, credentialsSpec(userID, "ing_1", "https://whip.example/w", "key_123")).Return(nil)
	m.lk.EXPECT().UpdateIngress(mock.Anything, "ing_1", mock.Anything, mock.Anything, "Beatrice").Return(nil)
	m.repo.EXPECT().Activate(mock.Anything, activationSpec(streamID, "ing_1", "https://whip.example/w", "key_123", "webrtc")).Return(nil)

	// when
	resp, err := svc.StartStream(context.Background(), userID, "My Stream", dto.StreamDefaultModeWebRTC, 6000)

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
			m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)
			userID := uuid.New()

			m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
			m.repo.EXPECT().CountActive(mock.Anything).Return(0, nil)
			m.repo.EXPECT().Create(mock.Anything, userID, "title", 3).Return(nil, tc.repoErr)

			// when
			_, err := svc.StartStream(context.Background(), userID, "title", dto.StreamDefaultModeWebRTC, 6000)

			// then
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCredentials_Disabled(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, false)

	// when
	_, err := svc.Credentials(context.Background(), uuid.New(), "Beato")

	// then
	require.ErrorIs(t, err, ErrDisabled)
}

func TestCredentials_ReturnsExistingWithoutCreatingIngress(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	userID := uuid.New()
	m.creds.EXPECT().Get(mock.Anything, userID).Return(&repository.StreamCredentialsRow{
		UserID: userID, IngressID: "ing", WhipURL: "https://whip/w", StreamKey: "key", Room: userRoom(userID),
	}, nil)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)

	// when
	resp, err := svc.Credentials(context.Background(), userID, "Beato")

	// then
	require.NoError(t, err)
	assert.Equal(t, "https://whip/w", resp.WhipURL)
	assert.Equal(t, "key", resp.StreamKey)
}

func TestCredentials_CreatesIngressWhenMissing(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	userID := uuid.New()
	m.creds.EXPECT().Get(mock.Anything, userID).Return(nil, nil)
	m.lk.EXPECT().CreateIngress(mock.Anything, mock.Anything, mock.Anything, "Beato").Return("ing_new", "https://whip/w", "key_new", nil)
	m.creds.EXPECT().Upsert(mock.Anything, credentialsSpec(userID, "ing_new", "https://whip/w", "key_new")).Return(nil)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)

	// when
	resp, err := svc.Credentials(context.Background(), userID, "Beato")

	// then
	require.NoError(t, err)
	assert.Equal(t, "key_new", resp.StreamKey)
}

func TestResetCredentials_BlockedWhileLive(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	userID := uuid.New()
	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(&repository.LiveStreamRow{ID: uuid.New()}, nil)

	// when
	_, err := svc.ResetCredentials(context.Background(), userID, "Beato")

	// then
	require.ErrorIs(t, err, ErrAlreadyLive)
}

func TestResetCredentials_DeletesOldIngressThenRecreates(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	userID := uuid.New()
	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
	m.creds.EXPECT().Get(mock.Anything, userID).Return(&repository.StreamCredentialsRow{
		UserID: userID, IngressID: "old_ing", Room: userRoom(userID),
	}, nil).Once()
	m.lk.EXPECT().DeleteIngress(mock.Anything, "old_ing").Return(nil)
	m.creds.EXPECT().Delete(mock.Anything, userID).Return(nil)
	m.creds.EXPECT().Get(mock.Anything, userID).Return(nil, nil)
	m.lk.EXPECT().CreateIngress(mock.Anything, mock.Anything, mock.Anything, "Beato").Return("new_ing", "https://whip/w", "new_key", nil)
	m.creds.EXPECT().Upsert(mock.Anything, credentialsSpec(userID, "new_ing", "https://whip/w", "new_key")).Return(nil)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)

	// when
	resp, err := svc.ResetCredentials(context.Background(), userID, "Beato")

	// then
	require.NoError(t, err)
	assert.Equal(t, "new_key", resp.StreamKey)
}

func TestResetCredentials_DeleteIngressFailureKeepsCreds(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	userID := uuid.New()
	m.repo.EXPECT().GetActiveByUser(mock.Anything, userID).Return(nil, nil)
	m.creds.EXPECT().Get(mock.Anything, userID).Return(&repository.StreamCredentialsRow{
		UserID: userID, IngressID: "old_ing", Room: userRoom(userID),
	}, nil)
	m.lk.EXPECT().DeleteIngress(mock.Anything, "old_ing").Return(assert.AnError)

	// when
	_, err := svc.ResetCredentials(context.Background(), userID, "Beato")

	// then
	require.Error(t, err)
}

func TestMintViewerToken_NotLive(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	streamID := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{ID: streamID, Status: "starting"}, nil)

	// when
	_, _, err := svc.MintViewerToken(context.Background(), streamID, nil)

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
	m.lk.EXPECT().MintViewerToken("live_room", mock.Anything, "", "").Return("tok", nil)
	m.lk.EXPECT().URL().Return("ws://lk")

	// when
	token, url, err := svc.MintViewerToken(context.Background(), streamID, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, "tok", token)
	assert.Equal(t, "ws://lk", url)
}

func TestMintViewerToken_LoggedInCarriesNameAndMetadata(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	expectStreamingEnabled(m, true)
	streamID := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, Status: "live", LivekitRoom: "live_room",
	}, nil)
	expectedMeta := `{"userId":"` + userID.String() + `","username":"beato","avatarUrl":"/a.png"}`
	m.lk.EXPECT().MintViewerToken("live_room", mock.Anything, "Beatrice", expectedMeta).Return("tok", nil)
	m.lk.EXPECT().URL().Return("ws://lk")

	// when
	_, _, err := svc.MintViewerToken(context.Background(), streamID, &dto.StreamViewer{
		UserID: userID, DisplayName: "Beatrice", Username: "beato", AvatarURL: "/a.png",
	})

	// then
	require.NoError(t, err)
}

func TestSaveThumbnail_RejectsOfflineStream(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	ownerID := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{ID: streamID, UserID: ownerID, Status: "starting"}, nil)

	// when
	err := svc.SaveThumbnail(context.Background(), ownerID, streamID, 100, bytes.NewReader([]byte("x")))

	// then
	require.ErrorIs(t, err, ErrStreamNotFound)
}

func TestSaveThumbnail_RejectsNonOwner(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: ownerID, Status: "live", ThumbnailURL: "/uploads/old.webp",
	}, nil)

	// when
	err := svc.SaveThumbnail(context.Background(), attackerID, streamID, 100, bytes.NewReader([]byte("x")))

	// then
	require.ErrorIs(t, err, ErrNotOwner)
}

func TestSaveThumbnail_StoresAndDeletesOldThumbnail(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	ownerID := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: ownerID, Status: "live", ThumbnailURL: "/uploads/old.webp",
	}, nil)
	m.upload.EXPECT().SaveImage(mock.Anything, "stream-thumbnails", streamID, int64(100), mock.Anything, mock.Anything).Return("/uploads/new.webp", nil)
	m.repo.EXPECT().SetThumbnail(mock.Anything, streamID, "/uploads/new.webp").Return(nil)
	m.upload.EXPECT().Delete([]string{"/uploads/old.webp"}).Return()

	// when
	err := svc.SaveThumbnail(context.Background(), ownerID, streamID, 100, bytes.NewReader([]byte("x")))

	// then
	require.NoError(t, err)
}

func TestSaveThumbnail_ThrottlesRapidUploads(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	ownerID := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{ID: streamID, UserID: ownerID, Status: "live"}, nil).Twice()
	m.upload.EXPECT().SaveImage(mock.Anything, "stream-thumbnails", streamID, mock.Anything, mock.Anything, mock.Anything).Return("/uploads/new.webp", nil).Once()
	m.repo.EXPECT().SetThumbnail(mock.Anything, streamID, "/uploads/new.webp").Return(nil).Once()

	// when
	err1 := svc.SaveThumbnail(context.Background(), ownerID, streamID, 100, bytes.NewReader([]byte("x")))
	err2 := svc.SaveThumbnail(context.Background(), ownerID, streamID, 100, bytes.NewReader([]byte("y")))

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
}

func TestJoinChat_NilBinderReturnsDisabled(t *testing.T) {
	// given
	svc, _ := newTestStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()

	// when
	err := svc.JoinChat(context.Background(), streamID, userID)

	// then
	require.ErrorIs(t, err, ErrDisabled)
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

	// when
	err := svc.StopStream(context.Background(), owner, streamID)

	// then
	require.NoError(t, err)
}

func TestUpdateTitle_TitleRequired(t *testing.T) {
	// given
	svc, _ := newTestStreamService(t)

	// when
	_, err := svc.UpdateTitle(context.Background(), uuid.New(), uuid.New(), "   ")

	// then
	require.ErrorIs(t, err, ErrTitleRequired)
}

func TestUpdateTitle_OfflineStreamNotFound(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: uuid.New(), Status: "offline",
	}, nil)

	// when
	_, err := svc.UpdateTitle(context.Background(), uuid.New(), streamID, "New title")

	// then
	require.ErrorIs(t, err, ErrStreamNotFound)
}

func TestUpdateTitle_NotOwner(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: uuid.New(), Status: "live",
	}, nil)

	// when
	_, err := svc.UpdateTitle(context.Background(), uuid.New(), streamID, "New title")

	// then
	require.ErrorIs(t, err, ErrNotOwner)
}

func TestUpdateTitle_HappyPath(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	owner := uuid.New()

	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(&repository.LiveStreamRow{
		ID: streamID, UserID: owner, Status: "live", Title: "Old title",
	}, nil)
	m.repo.EXPECT().SetTitle(mock.Anything, streamID, "New title").Return(nil)

	// when
	resp, err := svc.UpdateTitle(context.Background(), owner, streamID, "  New title  ")

	// then
	require.NoError(t, err)
	assert.Equal(t, streamID, resp.ID)
	assert.Equal(t, "New title", resp.Title)
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

func newFanoutStreamService(t *testing.T) (Service, *streamMocks) {
	repo := repository.NewMockLiveStreamRepository(t)
	creds := repository.NewMockStreamCredentialsRepository(t)
	followRepo := repository.NewMockFollowRepository(t)
	lk := livekit.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	notifSvc := notification.NewMockService(t)

	svc := NewService(repo, creds, followRepo, lk, settingsSvc, uploadSvc, notifSvc, ws.NewHub())

	settingsSvc.EXPECT().Get(mock.Anything, config.SettingStreamHLSOutputDir).Return("").Maybe()

	return svc, &streamMocks{repo: repo, creds: creds, followRepo: followRepo, lk: lk, settings: settingsSvc, upload: uploadSvc, notif: notifSvc}
}

func expectBroadcasterJoined(m *streamMocks, streamID uuid.UUID, userID uuid.UUID, room string) *repository.LiveStreamRow {
	row := &repository.LiveStreamRow{ID: streamID, UserID: userID, Status: "starting", LivekitRoom: room, Title: "reading EP4"}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantJoined,
		RoomName: room,
		Identity: "broadcaster_" + userID.String(),
	}, nil)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)
	m.repo.EXPECT().MarkLive(mock.Anything, streamID).Return(nil)
	m.repo.EXPECT().GetByID(mock.Anything, streamID).Return(row, nil).Maybe()

	return row
}

func TestHandleWebhook_BroadcasterJoinedNotifiesFollowers(t *testing.T) {
	// given
	svc, m := newFanoutStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()
	follower := uuid.New()
	expectBroadcasterJoined(m, streamID, userID, "live_"+streamID.String())

	m.notif.EXPECT().HasRecentFromActor(mock.Anything, dto.NotifStreamLive, userID, liveNotifyCooldown).Return(false)
	m.followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, userID).Return([]uuid.UUID{follower}, nil)

	sent := make(chan []dto.NotifyParams, 1)
	m.notif.EXPECT().NotifyMany(mock.Anything, mock.Anything).Run(func(_ context.Context, params []dto.NotifyParams) {
		sent <- params
	}).Return()

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.True(t, handled)

	select {
	case params := <-sent:
		require.Len(t, params, 1)
		assert.Equal(t, follower, params[0].RecipientID)
		assert.Equal(t, dto.NotifStreamLive, params[0].Type)
		assert.Equal(t, streamID, params[0].ReferenceID)
		assert.Equal(t, "stream", params[0].ReferenceType)
		assert.Equal(t, "went live: reading EP4", params[0].Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the follower fan-out")
	}
}

func TestHandleWebhook_BroadcasterRejoinWithinCooldownDoesNotNotify(t *testing.T) {
	// given
	svc, m := newFanoutStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()
	expectBroadcasterJoined(m, streamID, userID, "live_"+streamID.String())

	checked := make(chan struct{}, 1)
	m.notif.EXPECT().HasRecentFromActor(mock.Anything, dto.NotifStreamLive, userID, liveNotifyCooldown).Run(func(_ context.Context, _ dto.NotificationType, _ uuid.UUID, _ time.Duration) {
		checked <- struct{}{}
	}).Return(true)

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.True(t, handled)

	select {
	case <-checked:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the cooldown check")
	}

	m.followRepo.AssertNumberOfCalls(t, "GetFollowerIDsToNotify", 0)
	m.notif.AssertNumberOfCalls(t, "NotifyMany", 0)
}

func TestHandleWebhook_ViewerJoinedNeverNotifiesFollowers(t *testing.T) {
	// given
	svc, m := newFanoutStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()
	room := "live_" + streamID.String()
	row := &repository.LiveStreamRow{ID: streamID, UserID: userID, Status: "live", LivekitRoom: room}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:     livekit.EventParticipantJoined,
		RoomName: room,
		Identity: "viewer_" + uuid.New().String(),
	}, nil)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)
	m.repo.EXPECT().AdjustViewerCount(mock.Anything, streamID, 1).Return(0, false, nil).Maybe()

	// when
	handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

	// then
	require.NoError(t, err)
	assert.True(t, handled)
	m.notif.AssertNumberOfCalls(t, "HasRecentFromActor", 0)
	m.followRepo.AssertNumberOfCalls(t, "GetFollowerIDsToNotify", 0)
}

func TestHandleWebhook_BroadcasterVideoPublishedStartsEgress(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	streamID := uuid.New()
	userID := uuid.New()
	room := "live_" + streamID.String()
	row := &repository.LiveStreamRow{ID: streamID, UserID: userID, Status: "live", LivekitRoom: room}

	m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
		Type:      livekit.EventTrackPublished,
		RoomName:  room,
		Identity:  "broadcaster_" + userID.String(),
		TrackKind: "video",
	}, nil)
	m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)
	m.settings.EXPECT().GetBool(mock.Anything, config.SettingStreamHLSEnabled).Return(false)

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

func TestHandleWebhook_MonitorJoinLeaveDoesNotAdjustCount(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
	}{
		{name: "joined", eventType: livekit.EventParticipantJoined},
		{name: "left", eventType: livekit.EventParticipantLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestStreamService(t)
			streamID := uuid.New()
			room := "live_" + streamID.String()
			row := &repository.LiveStreamRow{ID: streamID, Status: "live", LivekitRoom: room}

			m.lk.EXPECT().ParseWebhook("auth", []byte("body")).Return(&livekit.Event{
				Type:     tt.eventType,
				RoomName: room,
				Identity: "monitor_" + uuid.New().String(),
			}, nil)
			m.repo.EXPECT().GetByRoom(mock.Anything, room).Return(row, nil)

			// when
			handled, err := svc.HandleWebhook(context.Background(), "auth", []byte("body"))

			// then
			require.NoError(t, err)
			assert.True(t, handled)
			m.repo.AssertNotCalled(t, "AdjustViewerCount", mock.Anything, mock.Anything, mock.Anything)
		})
	}
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
	m.lk.EXPECT().ActiveRooms(mock.Anything).Return(map[string][]string{}, nil)
	m.repo.EXPECT().ListLive(mock.Anything).Return(nil, nil)

	// when
	n, err := svc.ReconcileOnce(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestReconcileOnce_ReapsLiveRoomWithNoBroadcaster(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	liveID := uuid.New()
	room := "live_" + liveID.String()

	m.lk.EXPECT().Enabled().Return(true)
	m.repo.EXPECT().ListStartingBefore(mock.Anything, mock.Anything).Return(nil, nil)
	m.lk.EXPECT().ActiveRooms(mock.Anything).Return(map[string][]string{
		room: {"viewer_" + uuid.NewString()},
	}, nil)
	m.repo.EXPECT().ListLive(mock.Anything).Return([]repository.LiveStreamRow{
		{ID: liveID, Status: "live", LivekitRoom: room},
	}, nil)
	m.repo.EXPECT().MarkOffline(mock.Anything, liveID).Return(true, nil)

	// when
	n, err := svc.ReconcileOnce(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestReconcileOnce_KeepsLiveRoomWithBroadcaster(t *testing.T) {
	// given
	svc, m := newTestStreamService(t)
	liveID := uuid.New()
	userID := uuid.New()
	room := "live_" + liveID.String()

	m.lk.EXPECT().Enabled().Return(true)
	m.repo.EXPECT().ListStartingBefore(mock.Anything, mock.Anything).Return(nil, nil)
	m.lk.EXPECT().ActiveRooms(mock.Anything).Return(map[string][]string{
		room: {"broadcaster_" + userID.String()},
	}, nil)
	m.repo.EXPECT().ListLive(mock.Anything).Return([]repository.LiveStreamRow{
		{ID: liveID, Status: "live", LivekitRoom: room},
	}, nil)

	// when
	n, err := svc.ReconcileOnce(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}
