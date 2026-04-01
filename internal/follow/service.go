package follow

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

type (
	Service interface {
		Follow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		Unfollow(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) error
		GetFollowStats(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID) (*dto.FollowStatsResponse, error)
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
	if viewerID != uuid.Nil && viewerID != userID {
		isFollowing, _ = s.followRepo.IsFollowing(ctx, viewerID, userID)
	}

	return &dto.FollowStatsResponse{
		FollowerCount:  followers,
		FollowingCount: following,
		IsFollowing:    isFollowing,
	}, nil
}
