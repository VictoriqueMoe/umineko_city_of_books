package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	FollowDAO interface {
		Follow(ctx context.Context, s spec.FollowSpec, tx ...*sql.Tx) error
		Unfollow(ctx context.Context, s spec.FollowSpec, tx ...*sql.Tx) error
		IsFollowing(ctx context.Context, s spec.FollowSpec, tx ...*sql.Tx) (bool, error)
		GetFollowerCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetFollowingCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetFollowers(ctx context.Context, s spec.FollowListSpec, tx ...*sql.Tx) ([]model.FollowUser, int, error)
		GetFollowing(ctx context.Context, s spec.FollowListSpec, tx ...*sql.Tx) ([]model.FollowUser, int, error)
		GetMutualFollowers(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.FollowUser, error)
		GetFollowerIDsToNotify(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	followDAO struct {
		db *sql.DB
	}

	followUserRow = sqlcgen.ListFollowersRow
)

func toFollowUser(row followUserRow) model.FollowUser {
	return model.FollowUser{
		ID:          row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl,
		Role:        row.Role,
	}
}

func (r *followDAO) Follow(ctx context.Context, s spec.FollowSpec, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreateFollow(ctx, sqlcgen.CreateFollowParams{
		FollowerID:  s.FollowerID,
		FollowingID: s.FollowingID,
	})
	if err != nil {
		return fmt.Errorf("follow: %w", err)
	}

	return nil
}

func (r *followDAO) Unfollow(ctx context.Context, s spec.FollowSpec, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).DeleteFollow(ctx, sqlcgen.DeleteFollowParams{
		FollowerID:  s.FollowerID,
		FollowingID: s.FollowingID,
	})
	if err != nil {
		return fmt.Errorf("unfollow: %w", err)
	}

	return nil
}

func (r *followDAO) IsFollowing(ctx context.Context, s spec.FollowSpec, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountFollowRelation(ctx, sqlcgen.CountFollowRelationParams{
		FollowerID:  s.FollowerID,
		FollowingID: s.FollowingID,
	})
	if err != nil {
		return false, fmt.Errorf("check following: %w", err)
	}

	return count > 0, nil
}

func (r *followDAO) GetFollowerCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountFollowers(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("follower count: %w", err)
	}

	return int(count), nil
}

func (r *followDAO) GetFollowingCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountFollowing(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("following count: %w", err)
	}

	return int(count), nil
}

func (r *followDAO) GetFollowers(ctx context.Context, s spec.FollowListSpec, tx ...*sql.Tx) ([]model.FollowUser, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountFollowers(ctx, s.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count followers: %w", err)
	}

	rows, err := queries.ListFollowers(ctx, sqlcgen.ListFollowersParams{
		FollowingID: s.UserID,
		Limit:       int32(s.Limit),
		Offset:      int32(s.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get followers: %w", err)
	}

	var users []model.FollowUser
	for _, row := range rows {
		users = append(users, toFollowUser(row))
	}

	return users, int(total), nil
}

func (r *followDAO) GetFollowing(ctx context.Context, s spec.FollowListSpec, tx ...*sql.Tx) ([]model.FollowUser, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountFollowing(ctx, s.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count following: %w", err)
	}

	rows, err := queries.ListFollowing(ctx, sqlcgen.ListFollowingParams{
		FollowerID: s.UserID,
		Limit:      int32(s.Limit),
		Offset:     int32(s.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get following: %w", err)
	}

	var users []model.FollowUser
	for _, row := range rows {
		users = append(users, toFollowUser(followUserRow(row)))
	}

	return users, int(total), nil
}

func (r *followDAO) GetFollowerIDsToNotify(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).ListFollowerIDsToNotify(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get follower ids to notify: %w", err)
	}

	return ids, nil
}

func (r *followDAO) GetMutualFollowers(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.FollowUser, error) {
	rows, err := genQueries(r.db, tx).ListMutualFollowers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get mutual followers: %w", err)
	}

	var users []model.FollowUser
	for _, row := range rows {
		users = append(users, toFollowUser(followUserRow(row)))
	}

	return users, nil
}
