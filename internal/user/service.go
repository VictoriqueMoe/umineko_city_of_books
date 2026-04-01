package user

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	Service interface {
		Create(ctx context.Context, username, password, displayName string) (*dto.UserResponse, error)
		GetByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error)
		ValidateCredentials(ctx context.Context, username, password string) (*dto.UserResponse, error)
		CheckUsernameAvailable(ctx context.Context, username string) error
	}

	service struct {
		repo     repository.UserRepository
		roleRepo repository.RoleRepository
		authz    authz.Service
	}
)

func NewService(repo repository.UserRepository, roleRepo repository.RoleRepository, authzService authz.Service) Service {
	return &service{repo: repo, roleRepo: roleRepo, authz: authzService}
}

func (s *service) Create(ctx context.Context, username, password, displayName string) (*dto.UserResponse, error) {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	user, err := s.repo.Create(ctx, username, password, displayName)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if count == 0 {
		if err := s.roleRepo.SetRole(ctx, user.ID, authz.RoleSuperAdmin); err != nil {
			logger.Log.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to assign super admin role to first user")
		} else {
			logger.Log.Info().Str("user_id", user.ID.String()).Str("username", username).Msg("first user created, assigned super admin role")
		}
	}

	resp := user.ToResponse()
	resp.Role, _ = s.authz.GetRole(ctx, user.ID)
	return resp, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	resp := user.ToResponse()
	resp.Role, _ = s.authz.GetRole(ctx, user.ID)
	return resp, nil
}

func (s *service) ValidateCredentials(ctx context.Context, username, password string) (*dto.UserResponse, error) {
	user, err := s.repo.ValidatePassword(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("validate credentials: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	resp := user.ToResponse()
	resp.Role, _ = s.authz.GetRole(ctx, user.ID)
	return resp, nil
}

func (s *service) CheckUsernameAvailable(ctx context.Context, username string) error {
	exists, err := s.repo.ExistsByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if exists {
		return ErrUsernameTaken
	}
	return nil
}
