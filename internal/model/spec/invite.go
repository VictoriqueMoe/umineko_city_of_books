package spec

import (
	"github.com/google/uuid"
)

type (
	NewInvite struct {
		Code      string
		CreatedBy uuid.UUID
	}

	InviteRedemption struct {
		Code   string
		UsedBy uuid.UUID
	}

	InviteListQuery struct {
		Limit  int
		Offset int
	}
)
