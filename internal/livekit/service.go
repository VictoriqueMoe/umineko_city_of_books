package livekit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	"github.com/livekit/server-sdk-go/v2"
)

type (
	Service interface {
		Enabled() bool
		URL() string
		MintToken(roomName, identity, displayName string, canMic, canScreen bool) (string, error)
		SetCanPublish(ctx context.Context, roomName, identity string, canMic, canScreen bool) error
		ParseWebhook(authHeader string, body []byte) (*Event, error)
		ActiveRooms(ctx context.Context) (map[string][]string, error)
	}

	Event struct {
		Type     string
		RoomName string
		Identity string
	}

	service struct {
		settingsSvc settings.Service
	}
)

const (
	tokenTTL = time.Hour

	EventParticipantJoined = "participant_joined"
	EventParticipantLeft   = "participant_left"
	EventRoomFinished      = "room_finished"
)

var (
	ErrDisabled = errors.New("livekit is not configured")
)

func NewService(settingsSvc settings.Service) Service {
	return &service{settingsSvc: settingsSvc}
}

func (s *service) creds() (url, key, secret string) {
	ctx := context.Background()

	url = s.settingsSvc.Get(ctx, config.SettingLiveKitURL)
	key = s.settingsSvc.Get(ctx, config.SettingLiveKitAPIKey)
	secret = s.settingsSvc.Get(ctx, config.SettingLiveKitAPISecret)

	return
}

func (s *service) Enabled() bool {
	url, key, secret := s.creds()

	return url != "" && key != "" && secret != ""
}

func (s *service) URL() string {
	return s.settingsSvc.Get(context.Background(), config.SettingLiveKitURL)
}

func (s *service) MintToken(roomName, identity, displayName string, canMic, canScreen bool) (string, error) {
	_, key, secret := s.creds()

	if key == "" || secret == "" {
		return "", ErrDisabled
	}

	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	grant.SetCanPublishSources(publishSources(canMic, canScreen))

	at := auth.NewAccessToken(key, secret).
		SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(displayName).
		SetValidFor(tokenTTL)

	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("mint livekit token: %w", err)
	}

	return token, nil
}

func publishSources(canMic, canScreen bool) []livekit.TrackSource {
	sources := make([]livekit.TrackSource, 0, 3)
	if canMic {
		sources = append(sources, livekit.TrackSource_MICROPHONE)
	}
	if canScreen {
		sources = append(sources, livekit.TrackSource_SCREEN_SHARE, livekit.TrackSource_SCREEN_SHARE_AUDIO)
	}

	return sources
}

func (s *service) SetCanPublish(ctx context.Context, roomName, identity string, canMic, canScreen bool) error {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return ErrDisabled
	}

	client := lksdk.NewRoomServiceClient(toHTTPURL(url), key, secret)

	sources := publishSources(canMic, canScreen)

	if _, err := client.UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
		Room:     roomName,
		Identity: identity,
		Permission: &livekit.ParticipantPermission{
			CanSubscribe:      true,
			CanPublish:        len(sources) > 0,
			CanPublishData:    true,
			CanPublishSources: sources,
		},
	}); err != nil {
		return fmt.Errorf("update livekit participant: %w", err)
	}

	return nil
}

func (s *service) ParseWebhook(authHeader string, body []byte) (*Event, error) {
	_, key, secret := s.creds()

	if key == "" || secret == "" {
		return nil, ErrDisabled
	}

	provider := auth.NewSimpleKeyProvider(key, secret)

	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build webhook request: %w", err)
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/webhook+json")
	req.ContentLength = int64(len(body))

	raw, err := webhook.ReceiveWebhookEvent(req, provider)
	if err != nil {
		return nil, fmt.Errorf("verify livekit webhook: %w", err)
	}

	return &Event{
		Type:     raw.GetEvent(),
		RoomName: raw.GetRoom().GetName(),
		Identity: raw.GetParticipant().GetIdentity(),
	}, nil
}

func (s *service) ActiveRooms(ctx context.Context) (map[string][]string, error) {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return nil, ErrDisabled
	}

	client := lksdk.NewRoomServiceClient(toHTTPURL(url), key, secret)

	rooms, err := client.ListRooms(ctx, &livekit.ListRoomsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list livekit rooms: %w", err)
	}

	result := make(map[string][]string, len(rooms.GetRooms()))
	for i := 0; i < len(rooms.GetRooms()); i++ {
		name := rooms.GetRooms()[i].GetName()

		parts, err := client.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: name})
		if err != nil {
			continue
		}

		identities := make([]string, 0, len(parts.GetParticipants()))
		for j := 0; j < len(parts.GetParticipants()); j++ {
			identities = append(identities, parts.GetParticipants()[j].GetIdentity())
		}

		result[name] = identities
	}

	return result, nil
}

func toHTTPURL(u string) string {
	if strings.HasPrefix(u, "wss://") {
		return "https://" + strings.TrimPrefix(u, "wss://")
	}

	if strings.HasPrefix(u, "ws://") {
		return "http://" + strings.TrimPrefix(u, "ws://")
	}

	return u
}
