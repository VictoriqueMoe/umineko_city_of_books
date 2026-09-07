package spec

import (
	"github.com/google/uuid"
)

type (
	NewDeviceToken struct {
		UserID   uuid.UUID
		Token    string
		Platform string
	}

	DeviceTokenDeletion struct {
		UserID uuid.UUID
		Token  string
	}

	DeviceTokenBulkDeletion struct {
		UserID uuid.UUID
		Tokens []string
	}
)
