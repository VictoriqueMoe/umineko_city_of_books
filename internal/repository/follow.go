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
