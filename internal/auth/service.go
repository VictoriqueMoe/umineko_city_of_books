package auth

import (
	"context"
	"fmt"
	"regexp"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
)

type (
	Service interface {
		Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, string, error)
		Login(ctx context.Context, req dto.LoginRequest) (*dto.UserResponse, string, error)
		Logout(ctx context.Context, token string) error
		GetMe(ctx context.Context, userID uuid.UUID) (*dto.UserResponse, error)
	}

	service struct {
		userService user.Service
		session     *session.Manager
		settingsSvc settings.Service
		inviteRepo  repository.InviteRepository
	}
)

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)


func NewService(userService user.Service, sessionMgr *session.Manager, settingsSvc settings.Service, inviteRepo repository.InviteRepository) Service {
	return &service{
		userService: userService,
		session:     sessionMgr,
		settingsSvc: settingsSvc,
		inviteRepo:  inviteRepo,
	}
}

func (s *service) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, string, error) {
	regType := s.settingsSvc.Get(ctx, config.SettingRegistrationType)

	switch regType {
	case "closed":
		return nil, "", ErrRegistrationDisabled
	case "invite":
		if req.InviteCode == "" {
			return nil, "", ErrInviteRequired
		}
		invite, err := s.inviteRepo.GetByCode(ctx, req.InviteCode)
		if err != nil {
			return nil, "", fmt.Errorf("check invite: %w", err)
		}
		if invite == nil || invite.UsedBy != nil {
			return nil, "", ErrInvalidInvite
		}
	}

	if !isValidUsername(req.Username) {
		return nil, "", ErrInvalidUsername
	}

	minLen := s.settingsSvc.GetInt(ctx, config.SettingMinPasswordLength)
	if minLen > 0 && len(req.Password) < minLen {
		return nil, "", ErrPasswordTooShort
	}

	logger.Log.Debug().Str("username", req.Username).Msg("registering user")
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	if err := s.userService.CheckUsernameAvailable(ctx, req.Username); err != nil {
		return nil, "", err
	}

	userResp, err := s.userService.Create(ctx, req.Username, req.Password, req.DisplayName)
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	if regType == "invite" {
		if err := s.inviteRepo.MarkUsed(ctx, req.InviteCode, userResp.ID); err != nil {
			logger.Log.Error().Err(err).Str("code", req.InviteCode).Msg("failed to mark invite as used")
		}
	}

	token, err := s.session.Create(ctx, userResp.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return userResp, token, nil
}

func (s *service) Login(ctx context.Context, req dto.LoginRequest) (*dto.UserResponse, string, error) {
	logger.Log.Debug().Str("username", req.Username).Msg("login attempt")
	userResp, err := s.userService.ValidateCredentials(ctx, req.Username, req.Password)
	if err != nil {
		return nil, "", err
	}

	token, err := s.session.Create(ctx, userResp.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return userResp, token, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	if token != "" {
		return s.session.Delete(ctx, token)
	}
	return nil
}

func (s *service) GetMe(ctx context.Context, userID uuid.UUID) (*dto.UserResponse, error) {
	return s.userService.GetByID(ctx, userID)
}

func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 30 {
		return false
	}
	return validUsername.MatchString(username)
}
