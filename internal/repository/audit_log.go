package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	AuditLogEntry struct {
		ID         int
		ActorID    uuid.UUID
		ActorName  string
		Action     string
		TargetType string
		TargetID   string
		Details    string
		CreatedAt  string
	}

	AuditLogRepository interface {
		Create(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) error
		CreateSystem(ctx context.Context, action, targetType, targetID, details string) error
		List(ctx context.Context, action string, limit, offset int) ([]AuditLogEntry, int, error)
	}

	auditLogRepository struct {
		db *sql.DB
	}
)

func (r *auditLogRepository) Create(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, details) VALUES (?, ?, ?, ?, ?)`,
		actorID, action, targetType, targetID, details,
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *auditLogRepository) CreateSystem(ctx context.Context, action, targetType, targetID, details string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, details) VALUES (NULL, ?, ?, ?, ?)`,
		action, targetType, targetID, details,
	)
	if err != nil {
		return fmt.Errorf("create system audit log: %w", err)
	}
	return nil
}

func (r *auditLogRepository) List(ctx context.Context, action string, limit, offset int) ([]AuditLogEntry, int, error) {
	where := ""
	var args []interface{}
	if action != "" {
		where = " WHERE a.action = ?"
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

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.actor_id, COALESCE(u.display_name, ''), a.action, a.target_type, a.target_id, a.details, a.created_at
		 FROM audit_log a
		 LEFT JOIN users u ON a.actor_id = u.id`+where+`
		 ORDER BY a.created_at DESC
		 LIMIT ? OFFSET ?`, args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var actorID sql.NullString
		if err := rows.Scan(&e.ID, &actorID, &e.ActorName, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		if actorID.Valid {
			if id, err := uuid.Parse(actorID.String); err == nil {
				e.ActorID = id
			}
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}
