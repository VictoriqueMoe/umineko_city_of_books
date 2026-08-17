package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	reportDAO struct {
		db *sql.DB
	}
)

func (r *reportDAO) Create(ctx context.Context, spec repository.NewReport, tx ...*sql.Tx) (*repository.ReportRow, error) {
	var row repository.ReportRow
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH rep AS (
		     INSERT INTO reports (reporter_id, target_type, target_id, context_id, reason)
		     VALUES ($1, $2, $3, $4, $5)
		     RETURNING id, reporter_id, target_type, target_id, context_id, reason, status, resolved_by, created_at
		 )
		 SELECT rep.id, rep.reporter_id, u.display_name, u.avatar_url,
		        rep.target_type, rep.target_id, COALESCE(rep.context_id, ''), rep.reason, rep.status,
		        rep.resolved_by, ''::text, rep.created_at
		 FROM rep
		 JOIN users u ON rep.reporter_id = u.id`,
		spec.ReporterID, spec.TargetType, spec.TargetID, spec.ContextID, spec.Reason,
	).Scan(
		&row.ID, &row.ReporterID, &row.ReporterName, &row.ReporterAvatar,
		&row.TargetType, &row.TargetID, &row.ContextID, &row.Reason, &row.Status,
		&row.ResolvedByID, &row.ResolvedByName, &row.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}

	return &row, nil
}

func (r *reportDAO) List(ctx context.Context, status string, limit, offset int, tx ...*sql.Tx) ([]repository.ReportRow, int, error) {
	where := ""
	var args []any
	if status != "" {
		where = " WHERE r.status = $1"
		args = append(args, status)
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []repository.ReportRow
	for rows.Next() {
		var row repository.ReportRow
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

func (r *reportDAO) GetByID(ctx context.Context, id int, tx ...*sql.Tx) (*repository.ReportRow, error) {
	var row repository.ReportRow
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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

func (r *reportDAO) Resolve(ctx context.Context, id int, resolvedBy uuid.UUID, comment string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE reports SET status = 'resolved', resolved_by = $1, resolution_comment = $2 WHERE id = $3`,
		resolvedBy, comment, id,
	)
	if err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}
	return nil
}
