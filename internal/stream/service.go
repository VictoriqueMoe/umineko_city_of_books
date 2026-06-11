package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		Enabled() bool
		StartStream(ctx context.Context, userID uuid.UUID, title string) (*dto.StreamOwnerResponse, error)
		StopStream(ctx context.Context, userID, streamID uuid.UUID) error
		MyStream(ctx context.Context, userID uuid.UUID) (*dto.StreamOwnerResponse, error)
		ListLive(ctx context.Context) ([]dto.LiveStreamResponse, error)
		Get(ctx context.Context, streamID uuid.UUID) (*dto.LiveStreamResponse, error)
		MintViewerToken(ctx context.Context, streamID uuid.UUID) (token, url string, err error)
		HandleWebhook(ctx context.Context, authHeader string, body []byte) (handled bool, err error)
		ReconcileOnce(ctx context.Context) (int, error)
	}

	service struct {
		repo        repository.LiveStreamRepository
		livekitSvc  livekit.Service
		settingsSvc settings.Service
		hub         *ws.Hub
	}
)

const (
	roomPrefix        = "live_"
	broadcasterPrefix = "broadcaster_"
	viewerPrefix      = "viewer_"

	wsStreamLive    = "stream_live"
	wsStreamOffline = "stream_offline"
	wsStreamViewers = "stream_viewers"

	statusOffline = "offline"
	statusLive    = "live"

	maxTitleLen          = 120
	defaultMaxConcurrent = 3

	startingReapAfter = 3 * time.Minute
)

var (
	ErrDisabled       = errors.New("streaming is not configured")
	ErrAtCapacity     = errors.New("maximum concurrent streams reached")
	ErrAlreadyLive    = errors.New("you already have an active stream")
	ErrStreamNotFound = errors.New("stream not found")
	ErrNotOwner       = errors.New("not the stream owner")
	ErrTitleRequired  = errors.New("stream title is required")
)

func NewService(repo repository.LiveStreamRepository, livekitSvc livekit.Service, settingsSvc settings.Service, hub *ws.Hub) Service {
	return &service{
		repo:        repo,
		livekitSvc:  livekitSvc,
		settingsSvc: settingsSvc,
		hub:         hub,
	}
}

func (s *service) Enabled() bool {
	return s.settingsSvc.GetBool(context.Background(), config.SettingStreamingEnabled) && s.livekitSvc.Enabled()
}

func (s *service) maxConcurrent() int {
	n := s.settingsSvc.GetInt(context.Background(), config.SettingStreamMaxConcurrent)
	if n <= 0 {
		return defaultMaxConcurrent
	}

	return n
}

func (s *service) StartStream(ctx context.Context, userID uuid.UUID, title string) (*dto.StreamOwnerResponse, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrTitleRequired
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

	ingressID, whipURL, streamKey, err := s.livekitSvc.CreateIngress(ctx, room, identity, stream.DisplayName)
	if err != nil {
		_, _ = s.repo.MarkOffline(ctx, streamID)
		return nil, fmt.Errorf("create ingress: %w", err)
	}

	if err := s.repo.SetIngress(ctx, streamID, ingressID, room, whipURL, streamKey); err != nil {
		_ = s.livekitSvc.DeleteIngress(ctx, ingressID)
		_, _ = s.repo.MarkOffline(ctx, streamID)
		return nil, err
	}

	stream.LivekitRoom = room
	stream.IngressID = ingressID
	stream.WhipURL = whipURL
	stream.StreamKey = streamKey

	return toOwner(stream), nil
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

func (s *service) MintViewerToken(ctx context.Context, streamID uuid.UUID) (string, string, error) {
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

	token, err := s.livekitSvc.MintToken(stream.LivekitRoom, identity, "", false, false)
	if err != nil {
		return "", "", err
	}

	return token, s.livekitSvc.URL(), nil
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

	case livekit.EventParticipantLeft:
		if isBroadcaster {
			s.teardown(ctx, stream)
		} else {
			s.adjustViewers(ctx, stream.ID, -1)
		}

	case livekit.EventRoomFinished:
		s.teardown(ctx, stream)
	}

	return true, nil
}

func (s *service) teardown(ctx context.Context, stream *repository.LiveStreamRow) bool {
	transitioned, err := s.repo.MarkOffline(ctx, stream.ID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("stream_id", stream.ID.String()).Msg("mark stream offline failed")
		return false
	}
	if !transitioned {
		return false
	}

	if stream.IngressID != "" {
		if err := s.livekitSvc.DeleteIngress(ctx, stream.IngressID); err != nil {
			logger.Log.Warn().Err(err).Str("ingress_id", stream.IngressID).Msg("delete ingress failed")
		}
	}

	s.hub.Broadcast(ws.Message{
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

	s.hub.Broadcast(ws.Message{
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
		if _, ok := rooms[live[i].LivekitRoom]; ok {
			continue
		}
		if s.teardown(ctx, &live[i]) {
			reaped++
		}
	}

	return reaped, nil
}

func (s *service) broadcastLive(ctx context.Context, id uuid.UUID) {
	stream, err := s.repo.GetByID(ctx, id)
	if err != nil || stream == nil {
		return
	}

	s.hub.Broadcast(ws.Message{
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
		StartedAt:           started,
		StreamerUsername:    row.Username,
		StreamerDisplayName: row.DisplayName,
		StreamerAvatarURL:   row.AvatarURL,
	}
}

func toOwner(row *repository.LiveStreamRow) *dto.StreamOwnerResponse {
	return &dto.StreamOwnerResponse{
		Stream:    toPublic(row),
		WhipURL:   row.WhipURL,
		StreamKey: row.StreamKey,
	}
}
