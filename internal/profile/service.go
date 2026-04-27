package profile

import (
	"context"
	"fmt"
	"io"
	"time"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
)

type (
	Service interface {
		GetProfile(ctx context.Context, username string, viewerID uuid.UUID) (*dto.UserProfileResponse, error)
		UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error
		UploadAvatar(ctx context.Context, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error)
		UploadBanner(ctx context.Context, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error)
		ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error
		DeleteAccount(ctx context.Context, userID uuid.UUID, req dto.DeleteAccountRequest) error
		GetActivity(ctx context.Context, username string, limit, offset int) (*dto.ActivityListResponse, error)
		ListPublicUsers(ctx context.Context) ([]dto.UserResponse, error)
		SearchUsers(ctx context.Context, query string, limit int) ([]dto.UserResponse, error)
	}

	service struct {
		userRepo       repository.UserRepository
		userSecretRepo repository.UserSecretRepository
		theoryRepo     repository.TheoryRepository
		authz          authz.Service
		uploadSvc      upload.Service
		settingsSvc    settings.Service
		contentFilter  *contentfilter.Manager
	}
)

func NewService(
	userRepo repository.UserRepository,
	userSecretRepo repository.UserSecretRepository,
	theoryRepo repository.TheoryRepository,
	authzService authz.Service,
	uploadSvc upload.Service,
	settingsSvc settings.Service,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		userRepo:       userRepo,
		userSecretRepo: userSecretRepo,
		theoryRepo:     theoryRepo,
		authz:          authzService,
		uploadSvc:      uploadSvc,
		settingsSvc:    settingsSvc,
		contentFilter:  contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func (s *service) GetProfile(ctx context.Context, username string, viewerID uuid.UUID) (*dto.UserProfileResponse, error) {
	user, stats, err := s.userRepo.GetProfileByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	secrets, _ := s.userSecretRepo.ListForUser(ctx, user.ID)
	resp := user.ToProfileResponse(stats, user.ID == viewerID)
	resp.Secrets = secrets
	return resp, nil
}

const maxPronounLength = 10

func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error {
	if err := validateDOB(req.DOB); err != nil {
		return err
	}
	if err := s.filterTexts(ctx, req.DisplayName, req.Bio, req.Website, req.FavouriteCharacter); err != nil {
		return err
	}
	req.PronounSubject = capLen(req.PronounSubject, maxPronounLength)
	req.PronounPossessive = capLen(req.PronounPossessive, maxPronounLength)
	return s.userRepo.UpdateProfile(ctx, userID, req)
}

func capLen(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func validateDOB(dob string) error {
	if dob == "" {
		return nil
	}

	parsed, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return ErrInvalidDOB
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dobDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)

	if dobDate.After(today) {
		return ErrFutureDOB
	}

	return nil
}

func (s *service) UploadAvatar(ctx context.Context, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error) {
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	avatarURL, err := s.uploadSvc.SaveImage(ctx, "avatars", userID, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.userRepo.UpdateAvatarURL(ctx, userID, avatarURL); err != nil {
		return "", fmt.Errorf("update avatar url: %w", err)
	}

	return avatarURL, nil
}

func (s *service) UploadBanner(ctx context.Context, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error) {
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	bannerURL, err := s.uploadSvc.SaveImage(ctx, "banners", userID, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.userRepo.UpdateBannerURL(ctx, userID, bannerURL); err != nil {
		return "", fmt.Errorf("update banner url: %w", err)
	}

	return bannerURL, nil
}

func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	minLen := s.settingsSvc.GetInt(ctx, config.SettingMinPasswordLength)
	if minLen > 0 && len(req.NewPassword) < minLen {
		return ErrPasswordTooShort
	}
	return s.userRepo.ChangePassword(ctx, userID, req.OldPassword, req.NewPassword)
}

func (s *service) DeleteAccount(ctx context.Context, userID uuid.UUID, req dto.DeleteAccountRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user for cleanup: %w", err)
	}

	if err := s.userRepo.DeleteAccount(ctx, userID, req.Password); err != nil {
		return err
	}

	if user != nil {
		_ = s.uploadSvc.Delete(user.AvatarURL)
		_ = s.uploadSvc.Delete(user.BannerURL)
	}

	return nil
}

func (s *service) GetActivity(ctx context.Context, username string, limit, offset int) (*dto.ActivityListResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	items, total, err := s.theoryRepo.GetRecentActivityByUser(ctx, user.ID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get activity: %w", err)
	}

	return &dto.ActivityListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) ListPublicUsers(ctx context.Context) ([]dto.UserResponse, error) {
	users, err := s.userRepo.ListPublic(ctx)
	if err != nil {
		return nil, err
	}

	return s.usersToResponses(ctx, users), nil
}

func (s *service) SearchUsers(ctx context.Context, query string, limit int) ([]dto.UserResponse, error) {
	users, err := s.userRepo.SearchByName(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	return s.usersToResponses(ctx, users), nil
}

func (s *service) usersToResponses(ctx context.Context, users []model.User) []dto.UserResponse {
	result := make([]dto.UserResponse, len(users))
	if len(users) == 0 {
		return result
	}
	ids := make([]uuid.UUID, len(users))
	for i := 0; i < len(users); i++ {
		ids[i] = users[i].ID
	}
	roles, _ := s.authz.GetRoles(ctx, ids)
	for i := 0; i < len(users); i++ {
		u := users[i]
		result[i] = dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        roles[u.ID],
		}
	}
	return result
}
