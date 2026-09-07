package spec

import (
	"github.com/google/uuid"
)

type (
	NewSidebarVisit struct {
		UserID uuid.UUID
		Key    string
	}
)
