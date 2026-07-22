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

type auditLogRepository struct {
	dao AuditLogRepository
}

func NewAuditLogRepo(dao AuditLogRepository) AuditLogRepository {
	return &auditLogRepository{dao: dao}
}

func (r *auditLogRepository) Create(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) error {
	return r.dao.Create(ctx, actorID, action, targetType, targetID, details)
}

func (r *auditLogRepository) CreateSystem(ctx context.Context, action, targetType, targetID, details string) error {
	return r.dao.CreateSystem(ctx, action, targetType, targetID, details)
}

func (r *auditLogRepository) List(ctx context.Context, action string, limit, offset int) ([]AuditLogEntry, int, error) {
	return r.dao.List(ctx, action, limit, offset)
}
