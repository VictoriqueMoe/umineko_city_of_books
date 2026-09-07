package spec

import (
	"github.com/google/uuid"
)

type (
	BlockSpec struct {
		BlockerID uuid.UUID
		BlockedID uuid.UUID
	}

	BlockPairSpec struct {
		UserA uuid.UUID
		UserB uuid.UUID
	}
)
