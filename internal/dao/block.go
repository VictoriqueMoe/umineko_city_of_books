package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/repository"
)

type (
	blockDAO struct {
		db *sql.DB
	}
)

func (r *blockDAO) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		blockerID, blockedID,
	)
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	return nil
}

func (r *blockDAO) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID,
	)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}
	return nil
}

func (r *blockDAO) IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check block: %w", err)
	}
	return count > 0, nil
}

func (r *blockDAO) IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $3 AND blocked_id = $4)`,
		userA, userB, userB, userA,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check block either: %w", err)
	}
	return count > 0, nil
}

func (r *blockDAO) GetBlockedIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT blocked_id FROM blocks WHERE blocker_id = $1
		UNION
		SELECT blocker_id FROM blocks WHERE blocked_id = $2`,
		userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get blocked ids: %w", err)
	}
	return utils.ScanIDs(rows, "blocked id")
}

func (r *blockDAO) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, tx ...*sql.Tx) ([]repository.BlockedUser, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
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

	var users []repository.BlockedUser
	for rows.Next() {
		var (
			u         repository.BlockedUser
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
