package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	FollowRepository interface {
		Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error)
		GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error)
		GetFollowingCount(ctx context.Context, userID uuid.UUID) (int, error)
		GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]FollowUser, int, error)
		GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]FollowUser, int, error)
		GetMutualFollowers(ctx context.Context, userID uuid.UUID) ([]FollowUser, error)
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

func (r *followRepository) Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	return r.dao.Follow(ctx, followerID, followingID)
}

func (r *followRepository) Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	return r.dao.Unfollow(ctx, followerID, followingID)
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	return r.dao.IsFollowing(ctx, followerID, followingID)
}

func (r *followRepository) GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.GetFollowerCount(ctx, userID)
}

func (r *followRepository) GetFollowingCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.GetFollowingCount(ctx, userID)
}

func (r *followRepository) GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]FollowUser, int, error) {
	return r.dao.GetFollowers(ctx, userID, limit, offset)
}

func (r *followRepository) GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]FollowUser, int, error) {
	return r.dao.GetFollowing(ctx, userID, limit, offset)
}

func (r *followRepository) GetMutualFollowers(ctx context.Context, userID uuid.UUID) ([]FollowUser, error) {
	return r.dao.GetMutualFollowers(ctx, userID)
}
