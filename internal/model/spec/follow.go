package spec

import (
	"github.com/google/uuid"
)

type (
	FollowSpec struct {
		FollowerID  uuid.UUID
		FollowingID uuid.UUID
	}

	FollowListSpec struct {
		UserID uuid.UUID
		Limit  int
		Offset int
	}
)
