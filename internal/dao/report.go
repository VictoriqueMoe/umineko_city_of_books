package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ReportDAO interface {
		Create(ctx context.Context, s spec.NewReport, tx ...*sql.Tx) (*model.ReportRow, error)
		List(ctx context.Context, q spec.ReportFilter, tx ...*sql.Tx) ([]model.ReportRow, int, error)
		GetByID(ctx context.Context, id int, tx ...*sql.Tx) (*model.ReportRow, error)
		Resolve(ctx context.Context, s spec.ReportResolution, tx ...*sql.Tx) error
	}

	reportDAO struct {
		db *sql.DB
	}

	reportJoinRow = sqlcgen.GetReportByIDRow
)

func toReportRow(row reportJoinRow) model.ReportRow {
	return model.ReportRow{
		ID:             int(row.ID),
		ReporterID:     row.ReporterID,
		ReporterName:   row.DisplayName,
		ReporterAvatar: row.AvatarUrl,
		TargetType:     row.TargetType,
		TargetID:       row.TargetID,
		ContextID:      row.ContextID,
		Reason:         row.Reason,
		Status:         row.Status,
		ResolvedByID:   row.ResolvedBy,
		ResolvedByName: row.ResolvedByName,
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (r *reportDAO) Create(ctx context.Context, s spec.NewReport, tx ...*sql.Tx) (*model.ReportRow, error) {
	created, err := genQueries(r.db, tx).CreateReport(ctx, sqlcgen.CreateReportParams{
		ReporterID: s.ReporterID,
		TargetType: s.TargetType,
		TargetID:   s.TargetID,
		ContextID:  s.ContextID,
		Reason:     s.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}

	return new(toReportRow(reportJoinRow(created))), nil
}

func (r *reportDAO) List(ctx context.Context, q spec.ReportFilter, tx ...*sql.Tx) ([]model.ReportRow, int, error) {
	queries := genQueries(r.db, tx)

	var (
		total  int64
		joined []reportJoinRow
	)

	if q.Status != "" {
		count, err := queries.CountReportsByStatus(ctx, q.Status)
		if err != nil {
			return nil, 0, fmt.Errorf("count reports: %w", err)
		}

		rows, err := queries.ListReportsByStatus(ctx, sqlcgen.ListReportsByStatusParams{
			Status: q.Status,
			Limit:  int32(q.Limit),
			Offset: int32(q.Offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list reports: %w", err)
		}

		total = count
		for _, row := range rows {
			joined = append(joined, reportJoinRow(row))
		}
	} else {
		count, err := queries.CountReports(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("count reports: %w", err)
		}

		rows, err := queries.ListReports(ctx, sqlcgen.ListReportsParams{
			Limit:  int32(q.Limit),
			Offset: int32(q.Offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list reports: %w", err)
		}

		total = count
		for _, row := range rows {
			joined = append(joined, reportJoinRow(row))
		}
	}

	var reports []model.ReportRow
	for _, row := range joined {
		reports = append(reports, toReportRow(row))
	}

	return reports, int(total), nil
}

func (r *reportDAO) GetByID(ctx context.Context, id int, tx ...*sql.Tx) (*model.ReportRow, error) {
	row, err := genQueries(r.db, tx).GetReportByID(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get report by id: %w", err)
	}

	return new(toReportRow(row)), nil
}

func (r *reportDAO) Resolve(ctx context.Context, s spec.ReportResolution, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).ResolveReport(ctx, sqlcgen.ResolveReportParams{
		ResolvedBy:        &s.ResolvedBy,
		ResolutionComment: s.Comment,
		ID:                int64(s.ID),
	})
	if err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}

	return nil
}
