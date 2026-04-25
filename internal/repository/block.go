package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type (
	BlockRepository interface {
		Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
		Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
		IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error)
		IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error)
		GetBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		GetBlockedUsers(ctx context.Context, blockerID uuid.UUID) ([]BlockedUser, error)
	}

	BlockedUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		BlockedAt   string
	}

	blockRepository struct {
		db *sql.DB
	}
)

func (r *blockRepository) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		blockerID, blockedID,
	)
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	return nil
}

func (r *blockRepository) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID,
	)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}
	return nil
}

func (r *blockRepository) IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check block: %w", err)
	}
	return count > 0, nil
}

func (r *blockRepository) IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $3 AND blocked_id = $4)`,
		userA, userB, userB, userA,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check block either: %w", err)
	}
	return count > 0, nil
}

func (r *blockRepository) GetBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT blocked_id FROM blocks WHERE blocker_id = $1
		UNION
		SELECT blocker_id FROM blocks WHERE blocked_id = $2`,
		userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get blocked ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan blocked id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *blockRepository) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID) ([]BlockedUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, b.created_at
		FROM blocks b
		JOIN users u ON b.blocked_id = u.id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC`,
		blockerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get blocked users: %w", err)
	}
	defer rows.Close()

	var users []BlockedUser
	for rows.Next() {
		var (
			u         BlockedUser
			blockedAt time.Time
		)
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &blockedAt); err != nil {
			return nil, fmt.Errorf("scan blocked user: %w", err)
		}
		u.BlockedAt = blockedAt.UTC().Format(time.RFC3339)
		users = append(users, u)
	}
	return users, rows.Err()
}
