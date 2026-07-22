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

type reportRepository struct {
	dao ReportRepository
}

func NewReportRepo(dao ReportRepository) ReportRepository {
	return &reportRepository{dao: dao}
}

func (r *reportRepository) Create(ctx context.Context, reporterID uuid.UUID, targetType, targetID, contextID, reason string) (int64, error) {
	return r.dao.Create(ctx, reporterID, targetType, targetID, contextID, reason)
}

func (r *reportRepository) List(ctx context.Context, status string, limit, offset int) ([]ReportRow, int, error) {
	return r.dao.List(ctx, status, limit, offset)
}

func (r *reportRepository) GetByID(ctx context.Context, id int) (*ReportRow, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *reportRepository) Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string) error {
	return r.dao.Resolve(ctx, id, resolvedBy, comment)
}
