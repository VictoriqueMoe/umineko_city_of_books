package spec

import (
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"

	"github.com/google/uuid"
)

type (
	AuditLogListing struct {
		Action audit.Action
		Page   bounds.Page
	}

	AuditLogUserListing struct {
		UserID uuid.UUID
		Page   bounds.Page
	}
)
