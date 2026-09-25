package follow

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

type (
	Service interface {
		Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		GetFollowStats(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID) (*dto.FollowStatsResponse, error)
		IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error)
		GetFollowers(ctx context.Context, userID uuid.UUID, page bounds.Page) ([]dto.UserResponse, int, error)
		GetFollowing(ctx context.Context, userID uuid.UUID, page bounds.Page) ([]dto.UserResponse, int, error)
		GetMutualFollowers(ctx context.Context, userID uuid.UUID) ([]dto.UserResponse, error)
	}

	service struct {
		followRepo   repository.FollowRepository
		userRepo     repository.UserRepository
		blockSvc     block.Service
		notifService notification.Service
		settingsSvc  settings.Service
	}
)

func NewService(
	followRepo repository.FollowRepository,
	userRepo repository.UserRepository,
	blockSvc block.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
) Service {
	return &service{
		followRepo:   followRepo,
		userRepo:     userRepo,
		blockSvc:     blockSvc,
		notifService: notifService,
		settingsSvc:  settingsSvc,
	}
}

func (s *service) Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}

	blocked, err := s.blockSvc.IsBlockedEither(ctx, followerID, followingID)
	if err != nil {
		return fmt.Errorf("block check: %w", err)
	}
	if blocked {
		return block.ErrUserBlocked
	}

	if err := s.followRepo.Follow(ctx, spec.FollowSpec{FollowerID: followerID, FollowingID: followingID}); err != nil {
		return fmt.Errorf("follow: %w", err)
	}

	go func() {
		follower, err := s.userRepo.GetByID(ctx, followerID)
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("user_id", followerID.String()).Msg("new follower notification skipped, follower lookup failed")

			return
		}
		if follower == nil {
			return
		}
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   followingID,
			Type:          dto.NotifNewFollower,
			ReferenceID:   followerID,
			ReferenceType: "user",
			ActorID:       followerID,
			EmailActor:    follower.DisplayName,
			EmailAction:   "started following you",
			EmailLink:     fmt.Sprintf("/user/%s", follower.Username),
		})
	}()

	return nil
}

func (s *service) Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	return s.followRepo.Unfollow(ctx, spec.FollowSpec{FollowerID: followerID, FollowingID: followingID})
}

func (s *service) IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	return s.followRepo.IsFollowing(ctx, spec.FollowSpec{FollowerID: followerID, FollowingID: followingID})
}

func (s *service) GetFollowStats(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID) (*dto.FollowStatsResponse, error) {
	followers, err := s.followRepo.GetFollowerCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	following, err := s.followRepo.GetFollowingCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	isFollowing := false
	followsYou := false
	if viewerID != uuid.Nil && viewerID != userID {
		isFollowing, err = s.followRepo.IsFollowing(ctx, spec.FollowSpec{FollowerID: viewerID, FollowingID: userID})
		if err != nil {
			return nil, fmt.Errorf("viewer follows user: %w", err)
		}

		followsYou, err = s.followRepo.IsFollowing(ctx, spec.FollowSpec{FollowerID: userID, FollowingID: viewerID})
		if err != nil {
			return nil, fmt.Errorf("user follows viewer: %w", err)
		}
	}

	return &dto.FollowStatsResponse{
		FollowerCount:  followers,
		FollowingCount: following,
		IsFollowing:    isFollowing,
		FollowsYou:     followsYou,
	}, nil
}

func followUsersToDTO(users []model.FollowUser) []dto.UserResponse {
	result := make([]dto.UserResponse, len(users))
	for i, u := range users {
		result[i] = dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        role.Role(u.Role),
		}
	}
	return result
}

func (s *service) GetFollowers(ctx context.Context, userID uuid.UUID, page bounds.Page) ([]dto.UserResponse, int, error) {
	users, total, err := s.followRepo.GetFollowers(ctx, spec.FollowListSpec{UserID: userID, Limit: page.Limit(), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	return followUsersToDTO(users), total, nil
}

func (s *service) GetFollowing(ctx context.Context, userID uuid.UUID, page bounds.Page) ([]dto.UserResponse, int, error) {
	users, total, err := s.followRepo.GetFollowing(ctx, spec.FollowListSpec{UserID: userID, Limit: page.Limit(), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	return followUsersToDTO(users), total, nil
}

func (s *service) GetMutualFollowers(ctx context.Context, userID uuid.UUID) ([]dto.UserResponse, error) {
	users, err := s.followRepo.GetMutualFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}
	return followUsersToDTO(users), nil
}
