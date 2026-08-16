package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	auditLogDAO struct {
		db *sql.DB
	}
)

func (r *auditLogDAO) Create(ctx context.Context, spec repository.NewAuditEntry, tx ...*sql.Tx) error {
	if err := r.insert(ctx, spec.ActorID, spec, tx); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *auditLogDAO) CreateSystem(ctx context.Context, spec repository.NewAuditEntry, tx ...*sql.Tx) error {
	if err := r.insert(ctx, nil, spec, tx); err != nil {
		return fmt.Errorf("create system audit log: %w", err)
	}
	return nil
}

func (r *auditLogDAO) insert(ctx context.Context, actorID any, spec repository.NewAuditEntry, tx []*sql.Tx) error {
	var subjectID any
	if spec.SubjectID != uuid.Nil {
		subjectID = spec.SubjectID
	}

	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, details, subject_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		actorID, spec.Action, spec.TargetType, spec.TargetID, spec.Details, subjectID,
	)

	return err
}

func (r *auditLogDAO) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]repository.AuditLogEntry, int, error) {
	const scope = `((a.target_type = '` + string(repository.AuditTargetUser) + `' AND a.target_id = $1) OR a.subject_id = $1::uuid)`

	id := userID.String()

	var total int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_log a WHERE `+scope, id,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit log for user: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT a.id, a.actor_id, COALESCE(u.display_name, ''), a.action, a.target_type, a.target_id, a.details, a.created_at, a.subject_id, COALESCE(s.display_name, ''), COALESCE(s.username, '')
		 FROM audit_log a
		 LEFT JOIN users u ON a.actor_id = u.id
		 LEFT JOIN users s ON a.subject_id = s.id
		 WHERE `+scope+`
		 ORDER BY a.created_at DESC
		 LIMIT $2 OFFSET $3`,
		id, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log for user: %w", err)
	}
	defer rows.Close()

	entries, err := scanAuditLogRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, rows.Err()
}

func scanAuditLogRows(rows *sql.Rows) ([]repository.AuditLogEntry, error) {
	var entries []repository.AuditLogEntry
	for rows.Next() {
		var e repository.AuditLogEntry
		var actorID *uuid.UUID
		if err := rows.Scan(&e.ID, &actorID, &e.ActorName, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.CreatedAt, &e.SubjectID, &e.SubjectName, &e.SubjectUsername); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if actorID != nil {
			e.ActorID = *actorID
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *auditLogDAO) List(ctx context.Context, action repository.AuditAction, limit, offset int, tx ...*sql.Tx) ([]repository.AuditLogEntry, int, error) {
	where := ""
	var args []any
	if action != "" {
		where = " WHERE a.action = $1"
		args = append(args, action)
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM audit_log a"+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	args = append(args, limit, offset)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT a.id, a.actor_id, COALESCE(u.display_name, ''), a.action, a.target_type, a.target_id, a.details, a.created_at, a.subject_id, COALESCE(s.display_name, ''), COALESCE(s.username, '')
		 FROM audit_log a
		 LEFT JOIN users u ON a.actor_id = u.id
		 LEFT JOIN users s ON a.subject_id = s.id`+where+`
		 ORDER BY a.created_at DESC
		 LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	entries, err := scanAuditLogRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, rows.Err()
}
