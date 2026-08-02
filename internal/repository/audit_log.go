package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	AuditLogEntry struct {
		ID              int
		ActorID         uuid.UUID
		ActorName       string
		Action          string
		TargetType      string
		TargetID        string
		Details         string
		CreatedAt       string
		SubjectID       *uuid.UUID
		SubjectName     string
		SubjectUsername string
	}

	AuditLogRepository interface {
		Create(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) error
		CreateSystem(ctx context.Context, action, targetType, targetID, details string) error
		CreateForSubject(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string, subjectID uuid.UUID) error
		CreateSystemForSubject(ctx context.Context, action, targetType, targetID, details string, subjectID uuid.UUID) error
		List(ctx context.Context, action string, limit, offset int) ([]AuditLogEntry, int, error)
		ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]AuditLogEntry, int, error)
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

func (r *auditLogRepository) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]AuditLogEntry, int, error) {
	return r.dao.ListForUser(ctx, userID, limit, offset)
}

func (r *auditLogRepository) CreateForSubject(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string, subjectID uuid.UUID) error {
	return r.dao.CreateForSubject(ctx, actorID, action, targetType, targetID, details, subjectID)
}

func (r *auditLogRepository) CreateSystemForSubject(ctx context.Context, action, targetType, targetID, details string, subjectID uuid.UUID) error {
	return r.dao.CreateSystemForSubject(ctx, action, targetType, targetID, details, subjectID)
}
