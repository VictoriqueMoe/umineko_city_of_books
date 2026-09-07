package spec

import (
	"github.com/google/uuid"
)

type (
	OverlayTokenUpsert struct {
		UserID uuid.UUID
		Token  string
	}
)
