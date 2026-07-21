package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	AuditLogEntry struct {
		ID         int
		ActorID    uuid.UUID
		ActorName  string
		Action     string
		TargetType string
		TargetID   string
		Details    string
		CreatedAt  string
	}

	AuditLogRepository interface {
		Create(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) error
		CreateSystem(ctx context.Context, action, targetType, targetID, details string) error
		List(ctx context.Context, action string, limit, offset int) ([]AuditLogEntry, int, error)
	}
)
