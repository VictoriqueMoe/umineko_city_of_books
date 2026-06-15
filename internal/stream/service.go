package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		Enabled() bool
		StartStream(ctx context.Context, userID uuid.UUID, title string, defaultMode dto.StreamDefaultMode, bitrateKbps int) (*dto.StreamOwnerResponse, error)
		StopStream(ctx context.Context, userID, streamID uuid.UUID) error
		MyStream(ctx context.Context, userID uuid.UUID) (*dto.StreamOwnerResponse, error)
		ListLive(ctx context.Context) ([]dto.LiveStreamResponse, error)
		Get(ctx context.Context, streamID uuid.UUID) (*dto.LiveStreamResponse, error)
		MintViewerToken(ctx context.Context, streamID uuid.UUID, viewer *dto.StreamViewer) (token, url string, err error)
		HandleWebhook(ctx context.Context, authHeader string, body []byte) (handled bool, err error)
		ReconcileOnce(ctx context.Context) (int, error)
		JoinChat(ctx context.Context, streamID, userID uuid.UUID) error
		SaveThumbnail(ctx context.Context, streamID uuid.UUID, size int64, reader io.Reader) error
		Credentials(ctx context.Context, userID uuid.UUID, displayName string) (*dto.StreamCredentialsResponse, error)
		ResetCredentials(ctx context.Context, userID uuid.UUID, displayName string) (*dto.StreamCredentialsResponse, error)
		SetChatBinder(chat ChatBinder)
	}

	ChatBinder interface {
		CreateStreamRoom(ctx context.Context, streamID, streamerID uuid.UUID, title string) error
		JoinStreamChat(ctx context.Context, streamID, userID uuid.UUID) error
		DeleteStreamRoom(ctx context.Context, streamID uuid.UUID) error
	}

	service struct {
		repo        repository.LiveStreamRepository
		creds       repository.StreamCredentialsRepository
		livekitSvc  livekit.Service
		settingsSvc settings.Service
		uploadSvc   upload.Service
		hub         *ws.Hub
		chat        ChatBinder
		credsMu     sync.Mutex
		thumbMu     sync.Mutex
		thumbAt     map[uuid.UUID]time.Time
		bitrateMu   sync.Mutex
		bitrates    map[uuid.UUID]int
	}
)

const (
	roomPrefix           = "live_"
	broadcasterPrefix    = "broadcaster_"
	viewerPrefix         = "viewer_"
	hlsRoutePrefix       = "/hls"
	hlsVideoReadAttempts = 6
	hlsVideoReadInterval = 2 * time.Second

	minBitrateKbps = 500
	maxBitrateKbps = 50000

	wsStreamLive    = "stream_live"
	wsStreamOffline = "stream_offline"
	wsStreamViewers = "stream_viewers"

	statusOffline = "offline"
	statusLive    = "live"

	maxTitleLen          = 120
	defaultMaxConcurrent = 3

	startingReapAfter = 3 * time.Minute

	thumbnailSubDir   = "stream-thumbnails"
	maxThumbnailBytes = 3 * 1024 * 1024
	thumbnailThrottle = 20 * time.Second
)

var (
	ErrDisabled       = errors.New("streaming is not configured")
	ErrAtCapacity     = errors.New("maximum concurrent streams reached")
	ErrAlreadyLive    = errors.New("you already have an active stream")
	ErrStreamNotFound = errors.New("stream not found")
	ErrNotOwner       = errors.New("not the stream owner")
	ErrTitleRequired  = errors.New("stream title is required")
	ErrInvalidBitrate = errors.New("stream bitrate must be between 500 and 50000 Kbps")
)

func NewService(repo repository.LiveStreamRepository, creds repository.StreamCredentialsRepository, livekitSvc livekit.Service, settingsSvc settings.Service, uploadSvc upload.Service, hub *ws.Hub) Service {
	return &service{
		repo:        repo,
		creds:       creds,
		livekitSvc:  livekitSvc,
		settingsSvc: settingsSvc,
		uploadSvc:   uploadSvc,
		hub:         hub,
		thumbAt:     make(map[uuid.UUID]time.Time),
		bitrates:    make(map[uuid.UUID]int),
	}
}

func (s *service) SetChatBinder(chat ChatBinder) {
	s.chat = chat
}

func (s *service) Enabled() bool {
	return s.settingsSvc.GetBool(context.Background(), config.SettingStreamingEnabled) && s.livekitSvc.Enabled()
}

func (s *service) createChatRoom(ctx context.Context, streamID, streamerID uuid.UUID, title string) {
	if s.chat == nil {
		return
	}

	if err := s.chat.CreateStreamRoom(ctx, streamID, streamerID, title); err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", streamID.String()).Msg("create stream chat room failed")
	}
}

func (s *service) deleteChatRoom(ctx context.Context, streamID uuid.UUID) {
	if s.chat == nil {
		return
	}

	if err := s.chat.DeleteStreamRoom(ctx, streamID); err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", streamID.String()).Msg("delete stream chat room failed")
	}
}

func (s *service) JoinChat(ctx context.Context, streamID, userID uuid.UUID) error {
	if s.chat == nil {
		return ErrDisabled
	}

	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	if stream == nil || stream.Status != statusLive {
		return ErrStreamNotFound
	}

	return s.chat.JoinStreamChat(ctx, streamID, userID)
}

func (s *service) SaveThumbnail(ctx context.Context, streamID uuid.UUID, size int64, reader io.Reader) error {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	if stream == nil || stream.Status != statusLive {
		return ErrStreamNotFound
	}

	if !s.claimThumbnailSlot(streamID) {
		return nil
	}

	url, err := s.uploadSvc.SaveImage(ctx, thumbnailSubDir, streamID, size, maxThumbnailBytes, reader)
	if err != nil {
		return fmt.Errorf("save thumbnail: %w", err)
	}

	if err := s.repo.SetThumbnail(ctx, streamID, url); err != nil {
		_ = s.uploadSvc.Delete(url)
		return err
	}

	if stream.ThumbnailURL != "" && stream.ThumbnailURL != url {
		_ = s.uploadSvc.Delete(stream.ThumbnailURL)
	}

	return nil
}

func (s *service) claimThumbnailSlot(streamID uuid.UUID) bool {
	s.thumbMu.Lock()
	defer s.thumbMu.Unlock()

	now := time.Now()
	if last, ok := s.thumbAt[streamID]; ok && now.Sub(last) < thumbnailThrottle {
		return false
	}

	s.thumbAt[streamID] = now

	return true
}

func (s *service) clearThumbnailSlot(streamID uuid.UUID) {
	s.thumbMu.Lock()
	delete(s.thumbAt, streamID)
	s.thumbMu.Unlock()
}

func (s *service) maxConcurrent() int {
	n := s.settingsSvc.GetInt(context.Background(), config.SettingStreamMaxConcurrent)
	if n <= 0 {
		return defaultMaxConcurrent
	}

	return n
}

func (s *service) hlsEnabled() bool {
	return s.settingsSvc.GetBool(context.Background(), config.SettingStreamHLSEnabled)
}

func (s *service) setBitrate(streamID uuid.UUID, kbps int) {
	s.bitrateMu.Lock()
	s.bitrates[streamID] = kbps
	s.bitrateMu.Unlock()
}

func (s *service) bitrate(streamID uuid.UUID) int {
	s.bitrateMu.Lock()
	defer s.bitrateMu.Unlock()

	return s.bitrates[streamID]
}

func (s *service) clearBitrate(streamID uuid.UUID) {
	s.bitrateMu.Lock()
	delete(s.bitrates, streamID)
	s.bitrateMu.Unlock()
}

func (s *service) startEgress(streamID uuid.UUID, room, identity, ingressID string) {
	if room == "" || identity == "" || !s.hlsEnabled() {
		return
	}

	go s.runEgress(streamID, room, identity, ingressID)
}

func (s *service) runEgress(streamID uuid.UUID, room, identity, ingressID string) {
	ctx := context.Background()

	width, height, framerate := s.awaitIngressVideo(ctx, ingressID)
	if framerate <= 0 {
		framerate = 60
	}

	bitrate := s.bitrate(streamID)

	outputDir := s.settingsSvc.Get(ctx, config.SettingStreamHLSOutputDir)

	logger.Log.Info().
		Str("stream_id", streamID.String()).
		Int("width", width).
		Int("height", height).
		Int("framerate", framerate).
		Int("bitrate_kbps", bitrate).
		Msg("starting stream HLS egress")

	egressID, err := s.livekitSvc.CreateEgress(ctx, room, identity, outputDir, width, height, framerate, bitrate)
	if err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", streamID.String()).Msg("start stream egress failed")
		return
	}

	if err := s.repo.SetEgress(ctx, streamID, egressID, hlsPlaylistURL(room)); err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", streamID.String()).Msg("save stream egress failed")
		return
	}

	s.broadcastLive(ctx, streamID)
}

func (s *service) awaitIngressVideo(ctx context.Context, ingressID string) (width, height, framerate int) {
	for attempt := 0; attempt < hlsVideoReadAttempts; attempt++ {
		time.Sleep(hlsVideoReadInterval)

		w, h, fps, _, err := s.livekitSvc.IngressVideoState(ctx, ingressID)
		if err != nil {
			continue
		}

		if w > 0 {
			return w, h, fps
		}
	}

	return width, height, framerate
}

func hlsPlaylistURL(room string) string {
	return hlsRoutePrefix + "/" + room + "/live.m3u8"
}

func (s *service) cleanupHLS(room string) {
	if room == "" {
		return
	}

	outputDir := s.settingsSvc.Get(context.Background(), config.SettingStreamHLSOutputDir)
	dir := strings.TrimRight(outputDir, "/") + "/" + room

	if err := os.RemoveAll(dir); err != nil {
		logger.Log.Warn().Err(err).Str("room", room).Msg("cleanup stream hls dir failed")
	}
}

func (s *service) sweepHLS(ctx context.Context) {
	outputDir := s.settingsSvc.Get(ctx, config.SettingStreamHLSOutputDir)
	if outputDir == "" {
		return
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return
	}

	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), roomPrefix) {
			continue
		}

		stream, err := s.repo.GetByRoom(ctx, entry.Name())
		if err == nil && stream != nil && stream.Status == statusLive {
			continue
		}

		s.cleanupHLS(entry.Name())
	}
}

func (s *service) StartStream(ctx context.Context, userID uuid.UUID, title string, defaultMode dto.StreamDefaultMode, bitrateKbps int) (*dto.StreamOwnerResponse, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrTitleRequired
	}

	if bitrateKbps < minBitrateKbps || bitrateKbps > maxBitrateKbps {
		return nil, ErrInvalidBitrate
	}

	titleRunes := []rune(title)
	if len(titleRunes) > maxTitleLen {
		title = string(titleRunes[:maxTitleLen])
	}

	existing, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check active stream: %w", err)
	}
	if existing != nil {
		return nil, ErrAlreadyLive
	}

	active, err := s.repo.CountActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active streams: %w", err)
	}
	if active >= s.maxConcurrent() {
		return nil, ErrAtCapacity
	}

	streamID, err := s.repo.Create(ctx, userID, title, s.maxConcurrent())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLiveStreamActiveExists):
			{
				return nil, ErrAlreadyLive
			}
		case errors.Is(err, repository.ErrLiveStreamCapacity):
			{
				return nil, ErrAtCapacity
			}
		}
		return nil, err
	}

	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil || stream == nil {
		return nil, fmt.Errorf("reload created stream: %w", err)
	}

	room := roomPrefix + streamID.String()
	identity := broadcasterPrefix + userID.String()

	creds, err := s.ensureCredentials(ctx, userID, stream.DisplayName)
	if err != nil {
		_, _ = s.repo.MarkOffline(ctx, streamID)
		return nil, fmt.Errorf("ensure stream credentials: %w", err)
	}

	err = s.livekitSvc.UpdateIngress(ctx, creds.IngressID, room, identity, stream.DisplayName)
	if err != nil && livekit.IsNotFound(err) {
		creds, err = s.reprovisionCredentials(ctx, userID, stream.DisplayName)
		if err == nil {
			err = s.livekitSvc.UpdateIngress(ctx, creds.IngressID, room, identity, stream.DisplayName)
		}
	}
	if err != nil {
		_, _ = s.repo.MarkOffline(ctx, streamID)
		return nil, fmt.Errorf("point ingress at stream room: %w", err)
	}

	if err := s.repo.SetIngress(ctx, streamID, creds.IngressID, room, creds.WhipURL, creds.StreamKey); err != nil {
		_, _ = s.repo.MarkOffline(ctx, streamID)
		return nil, err
	}

	mode := dto.NormalizeStreamDefaultMode(defaultMode)
	if err := s.repo.SetDefaultMode(ctx, streamID, string(mode)); err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", streamID.String()).Msg("set stream default mode failed")
	}

	s.setBitrate(streamID, bitrateKbps)

	s.createChatRoom(ctx, streamID, userID, stream.Title)

	stream.LivekitRoom = room
	stream.IngressID = creds.IngressID
	stream.WhipURL = creds.WhipURL
	stream.StreamKey = creds.StreamKey
	stream.DefaultMode = string(mode)

	return toOwner(stream), nil
}

func userRoom(userID uuid.UUID) string {
	return roomPrefix + "u_" + userID.String()
}

func (s *service) ensureCredentials(ctx context.Context, userID uuid.UUID, displayName string) (*repository.StreamCredentialsRow, error) {
	existing, err := s.creds.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	s.credsMu.Lock()
	defer s.credsMu.Unlock()

	existing, err = s.creds.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	room := userRoom(userID)
	identity := broadcasterPrefix + userID.String()

	ingressID, whipURL, streamKey, err := s.livekitSvc.CreateIngress(ctx, room, identity, displayName)
	if err != nil {
		return nil, fmt.Errorf("create ingress: %w", err)
	}

	if err := s.creds.Upsert(ctx, userID, ingressID, whipURL, streamKey, room); err != nil {
		_ = s.livekitSvc.DeleteIngress(ctx, ingressID)
		return nil, err
	}

	return &repository.StreamCredentialsRow{
		UserID:    userID,
		IngressID: ingressID,
		WhipURL:   whipURL,
		StreamKey: streamKey,
		Room:      room,
	}, nil
}

func (s *service) reprovisionCredentials(ctx context.Context, userID uuid.UUID, displayName string) (*repository.StreamCredentialsRow, error) {
	existing, err := s.creds.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		_ = s.livekitSvc.DeleteIngress(ctx, existing.IngressID)
		if err := s.creds.Delete(ctx, userID); err != nil {
			return nil, err
		}
	}

	return s.ensureCredentials(ctx, userID, displayName)
}

func (s *service) Credentials(ctx context.Context, userID uuid.UUID, displayName string) (*dto.StreamCredentialsResponse, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	creds, err := s.ensureCredentials(ctx, userID, displayName)
	if err != nil {
		return nil, err
	}

	return &dto.StreamCredentialsResponse{WhipURL: creds.WhipURL, StreamKey: creds.StreamKey}, nil
}

func (s *service) ResetCredentials(ctx context.Context, userID uuid.UUID, displayName string) (*dto.StreamCredentialsResponse, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	active, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check active stream: %w", err)
	}
	if active != nil {
		return nil, ErrAlreadyLive
	}

	existing, err := s.creds.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if delErr := s.livekitSvc.DeleteIngress(ctx, existing.IngressID); delErr != nil {
			return nil, fmt.Errorf("delete old ingress on reset: %w", delErr)
		}
		if delErr := s.creds.Delete(ctx, userID); delErr != nil {
			return nil, delErr
		}
	}

	creds, err := s.ensureCredentials(ctx, userID, displayName)
	if err != nil {
		return nil, err
	}

	return &dto.StreamCredentialsResponse{WhipURL: creds.WhipURL, StreamKey: creds.StreamKey}, nil
}

func (s *service) StopStream(ctx context.Context, userID, streamID uuid.UUID) error {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	if stream == nil || stream.Status == statusOffline {
		return ErrStreamNotFound
	}
	if stream.UserID != userID {
		return ErrNotOwner
	}

	s.teardown(ctx, stream)

	return nil
}

func (s *service) MyStream(ctx context.Context, userID uuid.UUID) (*dto.StreamOwnerResponse, error) {
	stream, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, nil
	}

	return toOwner(stream), nil
}

func (s *service) ListLive(ctx context.Context) ([]dto.LiveStreamResponse, error) {
	rows, err := s.repo.ListLive(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]dto.LiveStreamResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		out = append(out, toPublic(&rows[i]))
	}

	return out, nil
}

func (s *service) Get(ctx context.Context, streamID uuid.UUID) (*dto.LiveStreamResponse, error) {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrStreamNotFound
	}

	view := toPublic(stream)

	return &view, nil
}

func (s *service) MintViewerToken(ctx context.Context, streamID uuid.UUID, viewer *dto.StreamViewer) (string, string, error) {
	if !s.Enabled() {
		return "", "", ErrDisabled
	}

	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return "", "", err
	}
	if stream == nil || stream.Status != statusLive {
		return "", "", ErrStreamNotFound
	}

	identity := viewerPrefix + uuid.NewString()
	name, metadata := viewerNameAndMetadata(viewer)

	token, err := s.livekitSvc.MintViewerToken(stream.LivekitRoom, identity, name, metadata)
	if err != nil {
		return "", "", err
	}

	return token, s.livekitSvc.URL(), nil
}

type viewerMeta struct {
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
}

func viewerNameAndMetadata(viewer *dto.StreamViewer) (string, string) {
	if viewer == nil || viewer.UserID == uuid.Nil {
		return "", ""
	}

	name := viewer.DisplayName
	if name == "" {
		name = viewer.Username
	}

	meta, err := json.Marshal(viewerMeta{
		UserID:    viewer.UserID.String(),
		Username:  viewer.Username,
		AvatarURL: viewer.AvatarURL,
	})
	if err != nil {
		return name, ""
	}

	return name, string(meta)
}

func (s *service) HandleWebhook(ctx context.Context, authHeader string, body []byte) (bool, error) {
	event, err := s.livekitSvc.ParseWebhook(authHeader, body)
	if err != nil {
		return false, err
	}

	if !strings.HasPrefix(event.RoomName, roomPrefix) {
		return false, nil
	}

	stream, err := s.repo.GetByRoom(ctx, event.RoomName)
	if err != nil {
		return true, err
	}
	if stream == nil {
		return true, nil
	}

	isBroadcaster := strings.HasPrefix(event.Identity, broadcasterPrefix)

	switch event.Type {
	case livekit.EventParticipantJoined:
		if isBroadcaster {
			if err := s.repo.MarkLive(ctx, stream.ID); err != nil {
				return true, err
			}

			s.broadcastLive(ctx, stream.ID)
		} else {
			s.adjustViewers(ctx, stream.ID, 1)
		}

	case livekit.EventTrackPublished:
		if isBroadcaster && event.TrackKind == "video" && stream.EgressID == "" {
			s.startEgress(stream.ID, stream.LivekitRoom, event.Identity, stream.IngressID)
		}

	case livekit.EventParticipantLeft:
		if isBroadcaster {
			s.teardown(ctx, stream)
		} else {
			s.adjustViewers(ctx, stream.ID, -1)
		}

	case livekit.EventRoomFinished:
		s.teardown(ctx, stream)

	case livekit.EventEgressEnded:
		s.cleanupHLS(event.RoomName)
	}

	return true, nil
}

func (s *service) teardown(ctx context.Context, stream *repository.LiveStreamRow) bool {
	s.clearBitrate(stream.ID)

	transitioned, err := s.repo.MarkOffline(ctx, stream.ID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", stream.ID.String()).Msg("mark stream offline failed")
		return false
	}
	if !transitioned {
		return false
	}

	if stream.EgressID != "" {
		if err := s.livekitSvc.StopEgress(ctx, stream.EgressID); err != nil {
			logger.Log.Warn().Err(err).Str("stream_id", stream.ID.String()).Msg("stop stream egress failed")
		}
	}

	s.deleteChatRoom(ctx, stream.ID)

	if stream.ThumbnailURL != "" {
		if err := s.uploadSvc.Delete(stream.ThumbnailURL); err != nil {
			logger.Log.Warn().Err(err).Str("stream_id", stream.ID.String()).Msg("delete stream thumbnail failed")
		}
	}
	s.clearThumbnailSlot(stream.ID)

	s.hub.BroadcastPublic(ws.Message{
		Type: wsStreamOffline,
		Data: dto.StreamOfflineEvent{StreamID: stream.ID},
	})

	return true
}

func (s *service) adjustViewers(ctx context.Context, id uuid.UUID, delta int) {
	count, ok, err := s.repo.AdjustViewerCount(ctx, id, delta)
	if err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", id.String()).Msg("adjust viewer count failed")
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

func (s *service) ReconcileOnce(ctx context.Context) (int, error) {
	if !s.livekitSvc.Enabled() {
		return 0, nil
	}

	reaped := 0

	cutoff := time.Now().UTC().Add(-startingReapAfter).Format(time.RFC3339Nano)
	stale, err := s.repo.ListStartingBefore(ctx, cutoff)
	if err != nil {
		return reaped, fmt.Errorf("list stale starting streams: %w", err)
	}
	for i := 0; i < len(stale); i++ {
		if s.teardown(ctx, &stale[i]) {
			reaped++
		}
	}

	rooms, err := s.livekitSvc.ActiveRooms(ctx)
	if err != nil {
		return reaped, nil
	}

	live, err := s.repo.ListLive(ctx)
	if err != nil {
		return reaped, nil
	}
	for i := 0; i < len(live); i++ {
		identities, ok := rooms[live[i].LivekitRoom]
		if ok && hasBroadcaster(identities) {
			continue
		}
		if s.teardown(ctx, &live[i]) {
			reaped++
		}
	}

	s.sweepHLS(ctx)

	return reaped, nil
}

func hasBroadcaster(identities []string) bool {
	for i := 0; i < len(identities); i++ {
		if strings.HasPrefix(identities[i], broadcasterPrefix) {
			return true
		}
	}

	return false
}

func (s *service) broadcastLive(ctx context.Context, id uuid.UUID) {
	stream, err := s.repo.GetByID(ctx, id)
	if err != nil || stream == nil {
		return
	}

	s.hub.BroadcastPublic(ws.Message{
		Type: wsStreamLive,
		Data: toPublic(stream),
	})
}

func toPublic(row *repository.LiveStreamRow) dto.LiveStreamResponse {
	started := ""
	if row.StartedAt.Valid {
		started = row.StartedAt.String
	}

	return dto.LiveStreamResponse{
		ID:                  row.ID,
		UserID:              row.UserID,
		Title:               row.Title,
		Status:              row.Status,
		ViewerCount:         row.ViewerCount,
		ThumbnailURL:        row.ThumbnailURL,
		StartedAt:           started,
		StreamerUsername:    row.Username,
		StreamerDisplayName: row.DisplayName,
		StreamerAvatarURL:   row.AvatarURL,
		DefaultMode:         dto.NormalizeStreamDefaultMode(dto.StreamDefaultMode(row.DefaultMode)),
		HLSUrl:              row.HLSPlaylistURL,
	}
}

func toOwner(row *repository.LiveStreamRow) *dto.StreamOwnerResponse {
	return &dto.StreamOwnerResponse{
		Stream:    toPublic(row),
		WhipURL:   row.WhipURL,
		StreamKey: row.StreamKey,
	}
}
