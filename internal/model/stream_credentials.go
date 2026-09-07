package model

import (
	"github.com/google/uuid"
)

type (
	StreamCredentialsRow struct {
		UserID    uuid.UUID
		IngressID string
		WhipURL   string
		StreamKey string
		Room      string
	}
)
