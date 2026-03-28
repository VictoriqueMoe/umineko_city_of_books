package user

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
)

type (
	Service interface {
		Create(ctx context.Context, username, password, displayName string) (*dto.UserResponse, error)
		GetByID(ctx context.Context, id int) (*dto.UserResponse, error)
		ValidateCredentials(ctx context.Context, username, password string) (*dto.UserResponse, error)
		CheckUsernameAvailable(ctx context.Context, username string) error
	}

	service struct {
		repo repository.UserRepository
	}
)

func NewService(repo repository.UserRepository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, username, password, displayName string) (*dto.UserResponse, error) {
	user, err := s.repo.Create(ctx, username, password, displayName)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user.ToResponse(), nil
}

func (s *service) GetByID(ctx context.Context, id int) (*dto.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user.ToResponse(), nil
}

func (s *service) ValidateCredentials(ctx context.Context, username, password string) (*dto.UserResponse, error) {
	user, err := s.repo.ValidatePassword(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("validate credentials: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	return user.ToResponse(), nil
}

func (s *service) CheckUsernameAvailable(ctx context.Context, username string) error {
	existing, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if existing != nil {
		return ErrUsernameTaken
	}
	return nil
}
