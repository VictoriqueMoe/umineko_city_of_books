package model

import (
	"database/sql"

	"github.com/google/uuid"
)

type (
	LiveStreamRow struct {
		ID             uuid.UUID
		UserID         uuid.UUID
		Title          string
		Status         string
		LivekitRoom    string
		IngressID      string
		WhipURL        string
		StreamKey      string
		ViewerCount    int
		StartedAt      sql.NullString
		EndedAt        sql.NullString
		CreatedAt      string
		ThumbnailURL   string
		EgressID       string
		HLSPlaylistURL string
		DefaultMode    string
		Username       string
		DisplayName    string
		AvatarURL      string
	}
)
