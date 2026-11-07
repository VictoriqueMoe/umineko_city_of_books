package model

import (
	"time"

	"github.com/google/uuid"
)

type (
	PendingUploadStatus string

	PendingUploadRow struct {
		ID           uuid.UUID
		UserID       uuid.UUID
		MediaType    string
		Filename     string
		FileSize     int64
		Status       PendingUploadStatus
		MediaURL     string
		ThumbnailURL string
		Width        int
		Height       int
		CreatedAt    time.Time
	}
)

const (
	PendingUploadProcessing PendingUploadStatus = "processing"
	PendingUploadReady      PendingUploadStatus = "ready"
	PendingUploadFailed     PendingUploadStatus = "failed"
)
