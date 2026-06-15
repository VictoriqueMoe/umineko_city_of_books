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
	"github.com/twitchtv/twirp"
)

type (
	Service interface {
		Enabled() bool
		URL() string
		MintToken(roomName, identity, displayName string, canMic, canScreen bool) (string, error)
		MintViewerToken(roomName, identity, name, metadata string) (string, error)
		SetCanPublish(ctx context.Context, roomName, identity string, canMic, canScreen bool) error
		ParseWebhook(authHeader string, body []byte) (*Event, error)
		ActiveRooms(ctx context.Context) (map[string][]string, error)
		CreateIngress(ctx context.Context, roomName, identity, displayName string) (ingressID, url, streamKey string, err error)
		UpdateIngress(ctx context.Context, ingressID, roomName, identity, displayName string) error
		DeleteIngress(ctx context.Context, ingressID string) error
		CreateEgress(ctx context.Context, roomName, identity, outputDir string, width, height, framerate, bitrateKbps int) (egressID string, err error)
		StopEgress(ctx context.Context, egressID string) error
		IngressVideoState(ctx context.Context, ingressID string) (width, height, framerate, bitrateKbps int, err error)
	}

	Event struct {
		Type      string
		RoomName  string
		Identity  string
		TrackKind string
	}

	service struct {
		settingsSvc settings.Service
	}
)

const (
	tokenTTL = 24 * time.Hour

	EventParticipantJoined = "participant_joined"
	EventParticipantLeft   = "participant_left"
	EventRoomFinished      = "room_finished"
	EventTrackPublished    = "track_published"
	EventEgressEnded       = "egress_ended"
)

var (
	ErrDisabled = errors.New("livekit is not configured")
)

func NewService(settingsSvc settings.Service) Service {
	return &service{settingsSvc: settingsSvc}
}

func IsNotFound(err error) bool {
	var twerr twirp.Error
	if errors.As(err, &twerr) {
		return twerr.Code() == twirp.NotFound
	}

	return false
}

func (s *service) creds() (url, key, secret string) {
	ctx := context.Background()

	url = strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingLiveKitURL))
	key = strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingLiveKitAPIKey))
	secret = strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingLiveKitAPISecret))

	return
}

func (s *service) Enabled() bool {
	url, key, secret := s.creds()

	return url != "" && key != "" && secret != ""
}

func (s *service) URL() string {
	return strings.TrimSpace(s.settingsSvc.Get(context.Background(), config.SettingLiveKitURL))
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

	sources := publishSources(canMic, canScreen)
	grant.SetCanPublish(len(sources) > 0)
	grant.SetCanSubscribe(true)
	grant.SetCanPublishSources(sources)

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

func (s *service) MintViewerToken(roomName, identity, name, metadata string) (string, error) {
	_, key, secret := s.creds()

	if key == "" || secret == "" {
		return "", ErrDisabled
	}

	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	grant.SetCanPublish(false)
	grant.SetCanSubscribe(true)

	at := auth.NewAccessToken(key, secret).
		SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(name).
		SetValidFor(tokenTTL)

	if metadata != "" {
		at = at.SetMetadata(metadata)
	}

	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("mint viewer token: %w", err)
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

	ev := &Event{
		Type:     raw.GetEvent(),
		RoomName: raw.GetRoom().GetName(),
		Identity: raw.GetParticipant().GetIdentity(),
	}

	if track := raw.GetTrack(); track != nil {
		ev.TrackKind = strings.ToLower(track.GetType().String())
	}

	if info := raw.GetEgressInfo(); info != nil {
		ev.RoomName = info.GetRoomName()
	}

	return ev, nil
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

func (s *service) CreateIngress(ctx context.Context, roomName, identity, displayName string) (string, string, string, error) {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return "", "", "", ErrDisabled
	}

	client := lksdk.NewIngressClient(toHTTPURL(url), key, secret)

	info, err := client.CreateIngress(ctx, &livekit.CreateIngressRequest{
		InputType:           livekit.IngressInput_WHIP_INPUT,
		Name:                roomName,
		RoomName:            roomName,
		ParticipantIdentity: identity,
		ParticipantName:     displayName,
		EnableTranscoding:   new(false),
		Video:               &livekit.IngressVideoOptions{Source: livekit.TrackSource_CAMERA},
		Audio:               &livekit.IngressAudioOptions{Source: livekit.TrackSource_MICROPHONE},
	})
	if err != nil {
		return "", "", "", fmt.Errorf("create livekit ingress: %w", err)
	}

	return info.GetIngressId(), info.GetUrl(), info.GetStreamKey(), nil
}

func (s *service) UpdateIngress(ctx context.Context, ingressID, roomName, identity, displayName string) error {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return ErrDisabled
	}

	client := lksdk.NewIngressClient(toHTTPURL(url), key, secret)

	if _, err := client.UpdateIngress(ctx, &livekit.UpdateIngressRequest{
		IngressId:           ingressID,
		RoomName:            roomName,
		ParticipantIdentity: identity,
		ParticipantName:     displayName,
	}); err != nil {
		return fmt.Errorf("update livekit ingress: %w", err)
	}

	return nil
}

func (s *service) DeleteIngress(ctx context.Context, ingressID string) error {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return ErrDisabled
	}

	client := lksdk.NewIngressClient(toHTTPURL(url), key, secret)

	if _, err := client.DeleteIngress(ctx, &livekit.DeleteIngressRequest{IngressId: ingressID}); err != nil {
		return fmt.Errorf("delete livekit ingress: %w", err)
	}

	return nil
}

func (s *service) CreateEgress(ctx context.Context, roomName, identity, outputDir string, width, height, framerate, bitrateKbps int) (string, error) {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return "", ErrDisabled
	}

	client := lksdk.NewEgressClient(toHTTPURL(url), key, secret)

	prefix := strings.TrimRight(outputDir, "/") + "/" + roomName

	advanced := &livekit.EncodingOptions{
		VideoCodec:       livekit.VideoCodec_H264_BASELINE,
		AudioCodec:       livekit.AudioCodec_AAC,
		AudioBitrate:     320,
		AudioFrequency:   48000,
		KeyFrameInterval: 2,
	}
	if width > 0 {
		advanced.Width = int32(width)
	}
	if height > 0 {
		advanced.Height = int32(height)
	}
	if framerate > 0 {
		advanced.Framerate = int32(framerate)
	}
	if bitrateKbps > 0 {
		advanced.VideoBitrate = int32(bitrateKbps)
	}

	req := &livekit.ParticipantEgressRequest{
		RoomName: roomName,
		Identity: identity,
		SegmentOutputs: []*livekit.SegmentedFileOutput{
			{
				FilenamePrefix:   prefix + "/segment",
				PlaylistName:     prefix + "/index.m3u8",
				LivePlaylistName: prefix + "/live.m3u8",
				SegmentDuration:  2,
			},
		},
		Options: &livekit.ParticipantEgressRequest_Advanced{Advanced: advanced},
	}

	info, err := client.StartParticipantEgress(ctx, req)
	if err != nil {
		return "", fmt.Errorf("start livekit egress: %w", err)
	}

	return info.GetEgressId(), nil
}

func (s *service) StopEgress(ctx context.Context, egressID string) error {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return ErrDisabled
	}

	client := lksdk.NewEgressClient(toHTTPURL(url), key, secret)

	if _, err := client.StopEgress(ctx, &livekit.StopEgressRequest{EgressId: egressID}); err != nil {
		return fmt.Errorf("stop livekit egress: %w", err)
	}

	return nil
}

func (s *service) IngressVideoState(ctx context.Context, ingressID string) (int, int, int, int, error) {
	url, key, secret := s.creds()

	if url == "" || key == "" || secret == "" {
		return 0, 0, 0, 0, ErrDisabled
	}

	client := lksdk.NewIngressClient(toHTTPURL(url), key, secret)

	resp, err := client.ListIngress(ctx, &livekit.ListIngressRequest{IngressId: ingressID})
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("list livekit ingress: %w", err)
	}

	items := resp.GetItems()
	for i := 0; i < len(items); i++ {
		video := items[i].GetState().GetVideo()
		if video == nil {
			continue
		}

		return int(video.GetWidth()), int(video.GetHeight()), int(video.GetFramerate()), int(video.GetAverageBitrate()) / 1000, nil
	}

	return 0, 0, 0, 0, nil
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
