package dto

import "github.com/google/uuid"

type (
	StreamDefaultMode string

	LiveStreamResponse struct {
		ID                  uuid.UUID         `json:"id"`
		UserID              uuid.UUID         `json:"userId"`
		Title               string            `json:"title"`
		Status              string            `json:"status"`
		ViewerCount         int               `json:"viewerCount"`
		ThumbnailURL        string            `json:"thumbnailUrl,omitempty"`
		StartedAt           string            `json:"startedAt,omitempty"`
		StreamerUsername    string            `json:"streamerUsername"`
		StreamerDisplayName string            `json:"streamerDisplayName"`
		StreamerAvatarURL   string            `json:"streamerAvatarUrl"`
		DefaultMode         StreamDefaultMode `json:"defaultMode"`
		HLSUrl              string            `json:"hlsUrl,omitempty"`
	}

	StartStreamRequest struct {
		Title       string            `json:"title"`
		DefaultMode StreamDefaultMode `json:"defaultMode"`
		Bitrate     int               `json:"bitrate"`
	}

	UpdateStreamTitleRequest struct {
		Title string `json:"title"`
	}

	StreamOwnerResponse struct {
		Stream    LiveStreamResponse `json:"stream"`
		WhipURL   string             `json:"whipUrl"`
		StreamKey string             `json:"streamKey"`
	}

	StreamCredentialsResponse struct {
		WhipURL    string `json:"whipUrl"`
		StreamKey  string `json:"streamKey"`
		HlsEnabled bool   `json:"hlsEnabled"`
	}

	LiveStreamListResponse struct {
		Streams []LiveStreamResponse `json:"streams"`
		Enabled bool                 `json:"enabled"`
	}

	StreamViewerTokenResponse struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}

	StreamViewer struct {
		UserID      uuid.UUID
		DisplayName string
		Username    string
		AvatarURL   string
	}

	StreamViewersEvent struct {
		StreamID    uuid.UUID `json:"streamId"`
		ViewerCount int       `json:"viewerCount"`
	}

	StreamOfflineEvent struct {
		StreamID uuid.UUID `json:"streamId"`
	}

	StreamTitleEvent struct {
		StreamID uuid.UUID `json:"streamId"`
		Title    string    `json:"title"`
	}
)

const (
	StreamDefaultModeWebRTC StreamDefaultMode = "webrtc"
	StreamDefaultModeHLS    StreamDefaultMode = "hls"
)

func NormalizeStreamDefaultMode(m StreamDefaultMode) StreamDefaultMode {
	switch m {
	case StreamDefaultModeWebRTC, StreamDefaultModeHLS:
		return m
	default:
		return StreamDefaultModeWebRTC
	}
}
