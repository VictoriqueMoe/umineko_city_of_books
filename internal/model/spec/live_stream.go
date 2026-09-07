package spec

import (
	"github.com/google/uuid"
)

type (
	NewLiveStream struct {
		UserID        uuid.UUID
		Title         string
		MaxConcurrent int
	}

	LiveStreamIngressUpdate struct {
		ID        uuid.UUID
		IngressID string
		Room      string
		WhipURL   string
		StreamKey string
	}

	LiveStreamActivation struct {
		Ingress     LiveStreamIngressUpdate
		DefaultMode string
	}

	LiveStreamViewerAdjustment struct {
		ID    uuid.UUID
		Delta int
	}

	LiveStreamThumbnailUpdate struct {
		ID  uuid.UUID
		URL string
	}

	LiveStreamEgressUpdate struct {
		ID       uuid.UUID
		EgressID string
		HLSURL   string
	}

	LiveStreamDefaultModeUpdate struct {
		ID          uuid.UUID
		DefaultMode string
	}

	LiveStreamTitleUpdate struct {
		ID    uuid.UUID
		Title string
	}
)
