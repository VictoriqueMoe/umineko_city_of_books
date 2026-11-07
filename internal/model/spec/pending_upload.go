package spec

import (
	"time"

	"github.com/google/uuid"
)

type (
	NewPendingUpload struct {
		ID        uuid.UUID
		UserID    uuid.UUID
		MediaType string
		Filename  string
		FileSize  int64
	}

	ProcessedUpload struct {
		ID           uuid.UUID
		MediaURL     string
		ThumbnailURL string
		Width        int
		Height       int
	}

	UserUploads struct {
		UserID uuid.UUID
		IDs    []uuid.UUID
	}

	UserUpload struct {
		ID     uuid.UUID
		UserID uuid.UUID
	}

	UploadClaim struct {
		UserID       uuid.UUID
		IDs          []uuid.UUID
		MediaTypes   []string
		CreatedAfter time.Time
	}

	UploadRef struct {
		ID        uuid.UUID
		IsSpoiler bool
	}

	ClaimedUploads struct {
		UserID     uuid.UUID
		Uploads    []UploadRef
		MediaTypes []string
		MaxAge     time.Duration
	}

	MediaAttachment struct {
		TargetID  uuid.UUID
		SortOrder int
		Uploads   ClaimedUploads
	}

	CommentMediaAttachment struct {
		Kind CommentKind
		MediaAttachment
	}
)
