package dto

import "github.com/google/uuid"

type (
	PendingUploadResponse struct {
		ID           uuid.UUID `json:"id"`
		MediaType    string    `json:"media_type"`
		Filename     string    `json:"filename"`
		Status       string    `json:"status"`
		MediaURL     string    `json:"media_url,omitempty"`
		ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	}

	UploadRef struct {
		ID        uuid.UUID `json:"upload_id"`
		IsSpoiler bool      `json:"is_spoiler,omitempty"`
	}

	UploadStatusRequest struct {
		IDs []uuid.UUID `json:"ids"`
	}
)
