package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	SidebarLastVisitedRepository interface {
		Upsert(ctx context.Context, userID uuid.UUID, key string) error
		ListForUser(ctx context.Context, userID uuid.UUID) (map[string]string, error)
	}

	sidebarLastVisitedRepository struct {
		db *sql.DB
	}
)

func (r *sidebarLastVisitedRepository) Upsert(ctx context.Context, userID uuid.UUID, key string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO sidebar_last_visited (user_id, key, visited_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)`,
		userID, key,
	)
	if err != nil {
		return fmt.Errorf("upsert sidebar last visited: %w", err)
	}
	return nil
}

func (r *sidebarLastVisitedRepository) ListForUser(ctx context.Context, userID uuid.UUID) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, visited_at FROM sidebar_last_visited WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sidebar last visited: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var key, visitedAt string
		if err := rows.Scan(&key, &visitedAt); err != nil {
			return nil, fmt.Errorf("scan sidebar last visited: %w", err)
		}
		out[key] = visitedAt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sidebar last visited: %w", err)
	}
	return out, nil
}
