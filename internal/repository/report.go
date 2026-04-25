package repository

import (
	"context"
	"database/sql"
	"fmt"

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

	reportRepository struct {
		db *sql.DB
	}
)

func (r *reportRepository) Create(ctx context.Context, reporterID uuid.UUID, targetType, targetID, contextID, reason string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO reports (reporter_id, target_type, target_id, context_id, reason) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		reporterID, targetType, targetID, contextID, reason,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create report: %w", err)
	}
	return id, nil
}

func (r *reportRepository) List(ctx context.Context, status string, limit, offset int) ([]ReportRow, int, error) {
	where := ""
	var args []interface{}
	if status != "" {
		where = " WHERE r.status = $1"
		args = append(args, status)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM reports r"+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	query := fmt.Sprintf(
		`SELECT r.id, r.reporter_id, u.display_name, u.avatar_url,
		        r.target_type, r.target_id, COALESCE(r.context_id, ''), r.reason, r.status,
		        r.resolved_by, COALESCE(ru.display_name, ''), r.created_at
		 FROM reports r
		 JOIN users u ON r.reporter_id = u.id
		 LEFT JOIN users ru ON r.resolved_by = ru.id
		 %s ORDER BY r.created_at DESC LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx,
	)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []ReportRow
	for rows.Next() {
		var row ReportRow
		if err := rows.Scan(
			&row.ID, &row.ReporterID, &row.ReporterName, &row.ReporterAvatar,
			&row.TargetType, &row.TargetID, &row.ContextID, &row.Reason, &row.Status,
			&row.ResolvedByID, &row.ResolvedByName, &row.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, row)
	}
	return reports, total, rows.Err()
}

func (r *reportRepository) GetByID(ctx context.Context, id int) (*ReportRow, error) {
	var row ReportRow
	err := r.db.QueryRowContext(ctx,
		`SELECT r.id, r.reporter_id, u.display_name, u.avatar_url,
		        r.target_type, r.target_id, COALESCE(r.context_id, ''), r.reason, r.status,
		        r.resolved_by, COALESCE(ru.display_name, ''), r.created_at
		 FROM reports r
		 JOIN users u ON r.reporter_id = u.id
		 LEFT JOIN users ru ON r.resolved_by = ru.id
		 WHERE r.id = $1`, id,
	).Scan(
		&row.ID, &row.ReporterID, &row.ReporterName, &row.ReporterAvatar,
		&row.TargetType, &row.TargetID, &row.ContextID, &row.Reason, &row.Status,
		&row.ResolvedByID, &row.ResolvedByName, &row.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get report by id: %w", err)
	}
	return &row, nil
}

func (r *reportRepository) Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reports SET status = 'resolved', resolved_by = $1, resolution_comment = $2 WHERE id = $3`,
		resolvedBy, comment, id,
	)
	if err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}
	return nil
}
