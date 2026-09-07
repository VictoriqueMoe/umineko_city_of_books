package model

import (
	"github.com/google/uuid"
)

type (
	Invite struct {
		Code      string
		CreatedBy uuid.UUID
		UsedBy    *uuid.UUID
		UsedAt    *string
		CreatedAt string
	}
)
