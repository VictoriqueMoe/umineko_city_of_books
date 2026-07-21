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

	auditLogRepository struct {
		repository.AuditLogRepository
	}
)

func (r *auditLogDAO) Create(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, details) VALUES ($1, $2, $3, $4, $5)`,
		actorID, action, targetType, targetID, details,
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *auditLogDAO) CreateSystem(ctx context.Context, action, targetType, targetID, details string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, details) VALUES (NULL, $1, $2, $3, $4)`,
		action, targetType, targetID, details,
	)
	if err != nil {
		return fmt.Errorf("create system audit log: %w", err)
	}
	return nil
}

func (r *auditLogDAO) List(ctx context.Context, action string, limit, offset int) ([]repository.AuditLogEntry, int, error) {
	where := ""
	var args []interface{}
	if action != "" {
		where = " WHERE a.action = $1"
		args = append(args, action)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM audit_log a"+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.actor_id, COALESCE(u.display_name, ''), a.action, a.target_type, a.target_id, a.details, a.created_at
		 FROM audit_log a
		 LEFT JOIN users u ON a.actor_id = u.id`+where+`
		 ORDER BY a.created_at DESC
		 LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []repository.AuditLogEntry
	for rows.Next() {
		var e repository.AuditLogEntry
		var actorID *uuid.UUID
		if err := rows.Scan(&e.ID, &actorID, &e.ActorName, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		if actorID != nil {
			e.ActorID = *actorID
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}
