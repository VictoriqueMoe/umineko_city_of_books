package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	followDAO struct {
		db *sql.DB
	}

	followRepository struct {
		repository.FollowRepository
	}
)

func (r *followDAO) Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		followerID, followingID,
	)
	if err != nil {
		return fmt.Errorf("follow: %w", err)
	}
	return nil
}

func (r *followDAO) Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	)
	if err != nil {
		return fmt.Errorf("unfollow: %w", err)
	}
	return nil
}

func (r *followDAO) IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check following: %w", err)
	}
	return count > 0, nil
}

func (r *followDAO) GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE following_id = $1`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("follower count: %w", err)
	}
	return count, nil
}

func (r *followDAO) GetFollowingCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = $1`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("following count: %w", err)
	}
	return count, nil
}

func (r *followDAO) GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.FollowUser, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM follows WHERE following_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count followers: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM follows f
		JOIN users u ON f.follower_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE f.following_id = $1
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get followers: %w", err)
	}
	defer rows.Close()

	var users []repository.FollowUser
	for rows.Next() {
		var u repository.FollowUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role); err != nil {
			return nil, 0, fmt.Errorf("scan follower: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *followDAO) GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.FollowUser, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM follows WHERE follower_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count following: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM follows f
		JOIN users u ON f.following_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE f.follower_id = $1
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get following: %w", err)
	}
	defer rows.Close()

	var users []repository.FollowUser
	for rows.Next() {
		var u repository.FollowUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role); err != nil {
			return nil, 0, fmt.Errorf("scan following: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *followDAO) GetMutualFollowers(ctx context.Context, userID uuid.UUID) ([]repository.FollowUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM follows f1
		JOIN follows f2 ON f1.following_id = f2.follower_id AND f2.following_id = f1.follower_id
		JOIN users u ON f1.following_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE f1.follower_id = $1
		ORDER BY LOWER(u.display_name)`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get mutual followers: %w", err)
	}
	defer rows.Close()

	var users []repository.FollowUser
	for rows.Next() {
		var u repository.FollowUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role); err != nil {
			return nil, fmt.Errorf("scan mutual: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
