package model

import "github.com/google/uuid"

type (
	Journal struct {
		ID                   uuid.UUID
		UserID               uuid.UUID
		Title                string
		Body                 string
		Work                 string
		CreatedAt            string
		UpdatedAt            *string
		LastAuthorActivityAt string
		ArchivedAt           *string
	}
)
