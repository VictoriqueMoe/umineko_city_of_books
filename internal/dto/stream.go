package dto

import "github.com/google/uuid"

type (
	LiveStreamResponse struct {
		ID                  uuid.UUID `json:"id"`
		UserID              uuid.UUID `json:"userId"`
		Title               string    `json:"title"`
		Status              string    `json:"status"`
		ViewerCount         int       `json:"viewerCount"`
		ThumbnailURL        string    `json:"thumbnailUrl,omitempty"`
		StartedAt           string    `json:"startedAt,omitempty"`
		StreamerUsername    string    `json:"streamerUsername"`
		StreamerDisplayName string    `json:"streamerDisplayName"`
		StreamerAvatarURL   string    `json:"streamerAvatarUrl"`
	}

	StartStreamRequest struct {
		Title string `json:"title"`
	}

	StreamOwnerResponse struct {
		Stream    LiveStreamResponse `json:"stream"`
		WhipURL   string             `json:"whipUrl"`
		StreamKey string             `json:"streamKey"`
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
)
