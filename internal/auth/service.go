package auth

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/user"
)

type (
	Service interface {
		Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, string, error)
		Login(ctx context.Context, req dto.LoginRequest) (*dto.UserResponse, string, error)
		Logout(ctx context.Context, token string) error
		GetMe(ctx context.Context, userID int) (*dto.UserResponse, error)
	}

	service struct {
		userService user.Service
		session     *session.Manager
	}
)

func NewService(userService user.Service, session *session.Manager) Service {
	return &service{
		userService: userService,
		session:     session,
	}
}

func (s *service) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, string, error) {
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

	token, err := s.session.Create(ctx, userResp.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return userResp, token, nil
}

func (s *service) Login(ctx context.Context, req dto.LoginRequest) (*dto.UserResponse, string, error) {
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

func (s *service) GetMe(ctx context.Context, userID int) (*dto.UserResponse, error) {
	return s.userService.GetByID(ctx, userID)
}
