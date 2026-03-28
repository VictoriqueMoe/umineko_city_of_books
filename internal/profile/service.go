package profile

import (
	"context"
	"fmt"
	"io"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/upload"
)

type (
	Service interface {
		GetProfile(ctx context.Context, username string) (*dto.UserProfileResponse, error)
		UpdateProfile(ctx context.Context, userID int, req dto.UpdateProfileRequest) error
		UploadAvatar(ctx context.Context, userID int, contentType string, fileSize int64, reader io.Reader) (string, error)
	}

	service struct {
		userRepo      repository.UserRepository
		uploadService upload.Service
	}
)

func NewService(userRepo repository.UserRepository, uploadService upload.Service) Service {
	return &service{
		userRepo:      userRepo,
		uploadService: uploadService,
	}
}

func (s *service) GetProfile(ctx context.Context, username string) (*dto.UserProfileResponse, error) {
	user, stats, err := s.userRepo.GetProfileByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user.ToProfileResponse(stats), nil
}

func (s *service) UpdateProfile(ctx context.Context, userID int, req dto.UpdateProfileRequest) error {
	return s.userRepo.UpdateProfile(ctx, userID, req)
}

func (s *service) UploadAvatar(ctx context.Context, userID int, contentType string, fileSize int64, reader io.Reader) (string, error) {
	avatarURL, err := s.uploadService.SaveImage("avatars", userID, contentType, fileSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.userRepo.UpdateAvatarURL(ctx, userID, avatarURL); err != nil {
		return "", fmt.Errorf("update avatar url: %w", err)
	}

	return avatarURL, nil
}
