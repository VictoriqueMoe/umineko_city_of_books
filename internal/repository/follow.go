package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	FollowRepository interface {
		Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID, tx ...*sql.Tx) error
		Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID, tx ...*sql.Tx) error
		IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetFollowerCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetFollowingCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]FollowUser, int, error)
		GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]FollowUser, int, error)
		GetMutualFollowers(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]FollowUser, error)
		GetFollowerIDsToNotify(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	FollowUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
	}
)

type followRepository struct {
	dao FollowRepository
}

func NewFollowRepo(dao FollowRepository) FollowRepository {
	return &followRepository{dao: dao}
}

func (r *followRepository) Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Follow(ctx, followerID, followingID, tx...)
}

func (r *followRepository) Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unfollow(ctx, followerID, followingID, tx...)
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsFollowing(ctx, followerID, followingID, tx...)
}

func (r *followRepository) GetFollowerCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetFollowerCount(ctx, userID, tx...)
}

func (r *followRepository) GetFollowingCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetFollowingCount(ctx, userID, tx...)
}

func (r *followRepository) GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]FollowUser, int, error) {
	return r.dao.GetFollowers(ctx, userID, limit, offset, tx...)
}

func (r *followRepository) GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]FollowUser, int, error) {
	return r.dao.GetFollowing(ctx, userID, limit, offset, tx...)
}

func (r *followRepository) GetMutualFollowers(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]FollowUser, error) {
	return r.dao.GetMutualFollowers(ctx, userID, tx...)
}

func (r *followRepository) GetFollowerIDsToNotify(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetFollowerIDsToNotify(ctx, userID, tx...)
}
