package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	ReportRow struct {
		ID             int
		ReporterID     uuid.UUID
		ReporterName   string
		ReporterAvatar string
		TargetType     string
		TargetID       string
		ContextID      string
		Reason         string
		Status         string
		ResolvedByID   *uuid.UUID
		ResolvedByName string
		CreatedAt      string
	}

	ReportRepository interface {
		Create(ctx context.Context, reporterID uuid.UUID, targetType, targetID, contextID, reason string) (int64, error)
		List(ctx context.Context, status string, limit, offset int) ([]ReportRow, int, error)
		GetByID(ctx context.Context, id int) (*ReportRow, error)
		Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string) error
	}
)
