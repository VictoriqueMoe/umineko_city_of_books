package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	FollowRepository interface {
		Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error)
		GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error)
		GetFollowingCount(ctx context.Context, userID uuid.UUID) (int, error)
	}

	followRepository struct {
		db *sql.DB
	}
)

func (r *followRepository) Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO follows (follower_id, following_id) VALUES (?, ?)`,
		followerID, followingID,
	)
	if err != nil {
		return fmt.Errorf("follow: %w", err)
	}
	return nil
}

func (r *followRepository) Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM follows WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	)
	if err != nil {
		return fmt.Errorf("unfollow: %w", err)
	}
	return nil
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check following: %w", err)
	}
	return count > 0, nil
}

func (r *followRepository) GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE following_id = ?`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("follower count: %w", err)
	}
	return count, nil
}

func (r *followRepository) GetFollowingCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = ?`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("following count: %w", err)
	}
	return count, nil
}
