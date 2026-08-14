package repository

import (
	"context"
	"database/sql"

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

	NewAuditEntry struct {
		ActorID    uuid.UUID
		Action     string
		TargetType string
		TargetID   string
		Details    string
	}

	NewAuditSubjectEntry struct {
		ActorID    uuid.UUID
		Action     string
		TargetType string
		TargetID   string
		Details    string
		SubjectID  uuid.UUID
	}

	AuditLogRepository interface {
		Create(ctx context.Context, spec NewAuditEntry, tx ...*sql.Tx) error
		CreateSystem(ctx context.Context, action, targetType, targetID, details string, tx ...*sql.Tx) error
		CreateForSubject(ctx context.Context, spec NewAuditSubjectEntry, tx ...*sql.Tx) error
		CreateSystemForSubject(ctx context.Context, spec NewAuditSubjectEntry, tx ...*sql.Tx) error
		List(ctx context.Context, action string, limit, offset int, tx ...*sql.Tx) ([]AuditLogEntry, int, error)
		ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]AuditLogEntry, int, error)
	}
)

type auditLogRepository struct {
	dao AuditLogRepository
}

func NewAuditLogRepo(dao AuditLogRepository) AuditLogRepository {
	return &auditLogRepository{dao: dao}
}

func (r *auditLogRepository) Create(ctx context.Context, spec NewAuditEntry, tx ...*sql.Tx) error {
	return r.dao.Create(ctx, spec, tx...)
}

func (r *auditLogRepository) CreateSystem(ctx context.Context, action, targetType, targetID, details string, tx ...*sql.Tx) error {
	return r.dao.CreateSystem(ctx, action, targetType, targetID, details, tx...)
}

func (r *auditLogRepository) List(ctx context.Context, action string, limit, offset int, tx ...*sql.Tx) ([]AuditLogEntry, int, error) {
	return r.dao.List(ctx, action, limit, offset, tx...)
}

func (r *auditLogRepository) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]AuditLogEntry, int, error) {
	return r.dao.ListForUser(ctx, userID, limit, offset, tx...)
}

func (r *auditLogRepository) CreateForSubject(ctx context.Context, spec NewAuditSubjectEntry, tx ...*sql.Tx) error {
	return r.dao.CreateForSubject(ctx, spec, tx...)
}

func (r *auditLogRepository) CreateSystemForSubject(ctx context.Context, spec NewAuditSubjectEntry, tx ...*sql.Tx) error {
	return r.dao.CreateSystemForSubject(ctx, spec, tx...)
}
