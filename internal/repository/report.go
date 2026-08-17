package repository

import (
	"context"
	"database/sql"

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

	NewReport struct {
		ReporterID uuid.UUID
		TargetType string
		TargetID   string
		ContextID  string
		Reason     string
	}

	ReportRepository interface {
		Create(ctx context.Context, spec NewReport, tx ...*sql.Tx) (*ReportRow, error)
		List(ctx context.Context, status string, limit, offset int, tx ...*sql.Tx) ([]ReportRow, int, error)
		GetByID(ctx context.Context, id int, tx ...*sql.Tx) (*ReportRow, error)
		Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string, tx ...*sql.Tx) error
	}
)

type reportRepository struct {
	dao ReportRepository
}

func NewReportRepo(dao ReportRepository) ReportRepository {
	return &reportRepository{dao: dao}
}

func (r *reportRepository) Create(ctx context.Context, spec NewReport, tx ...*sql.Tx) (*ReportRow, error) {
	return r.dao.Create(ctx, spec, tx...)
}

func (r *reportRepository) List(ctx context.Context, status string, limit, offset int, tx ...*sql.Tx) ([]ReportRow, int, error) {
	return r.dao.List(ctx, status, limit, offset, tx...)
}

func (r *reportRepository) GetByID(ctx context.Context, id int, tx ...*sql.Tx) (*ReportRow, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *reportRepository) Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string, tx ...*sql.Tx) error {
	return r.dao.Resolve(ctx, id, resolvedBy, comment, tx...)
}
