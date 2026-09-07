package spec

import (
	"github.com/google/uuid"
)

type (
	NewStreamCredentials struct {
		UserID    uuid.UUID
		IngressID string
		WhipURL   string
		StreamKey string
		Room      string
	}
)
