package follow

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
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
		GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.UserResponse, int, error)
		GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.UserResponse, int, error)
		GetMutualFollowers(ctx context.Context, userID uuid.UUID) ([]dto.UserResponse, error)
	}

	service struct {
		followRepo   repository.FollowRepository
		userRepo     repository.UserRepository
		notifService notification.Service
		settingsSvc  settings.Service
	}
)

func NewService(
	followRepo repository.FollowRepository,
	userRepo repository.UserRepository,
	notifService notification.Service,
	settingsSvc settings.Service,
) Service {
	return &service{
		followRepo:   followRepo,
		userRepo:     userRepo,
		notifService: notifService,
		settingsSvc:  settingsSvc,
	}
}

func (s *service) Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}

	if err := s.followRepo.Follow(ctx, followerID, followingID); err != nil {
		return fmt.Errorf("follow: %w", err)
	}

	go func() {
		follower, err := s.userRepo.GetByID(ctx, followerID)
		if err != nil || follower == nil {
			return
		}
		baseURL := s.settingsSvc.Get(ctx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/user/%s", baseURL, follower.Username)
		subject, body := notification.NotifEmail(follower.DisplayName, "started following you", "", linkURL)
		_ = s.notifService.Notify(ctx, dto.NotifyParams{
			RecipientID:   followingID,
			Type:          dto.NotifNewFollower,
			ReferenceID:   followerID,
			ReferenceType: "user",
			ActorID:       followerID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return nil
}

func (s *service) Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error {
	return s.followRepo.Unfollow(ctx, followerID, followingID)
}

func (s *service) IsFollowing(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	return s.followRepo.IsFollowing(ctx, followerID, followingID)
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
		isFollowing, _ = s.followRepo.IsFollowing(ctx, viewerID, userID)
		followsYou, _ = s.followRepo.IsFollowing(ctx, userID, viewerID)
	}

	return &dto.FollowStatsResponse{
		FollowerCount:  followers,
		FollowingCount: following,
		IsFollowing:    isFollowing,
		FollowsYou:     followsYou,
	}, nil
}

func followUsersToDTO(users []repository.FollowUser) []dto.UserResponse {
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

func (s *service) GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.UserResponse, int, error) {
	users, total, err := s.followRepo.GetFollowers(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return followUsersToDTO(users), total, nil
}

func (s *service) GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.UserResponse, int, error) {
	users, total, err := s.followRepo.GetFollowing(ctx, userID, limit, offset)
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
