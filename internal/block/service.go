package block

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	Service interface {
		Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
		Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
		IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error)
		IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error)
		GetBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		GetBlockedUsers(ctx context.Context, blockerID uuid.UUID) ([]repository.BlockedUser, error)
	}

	service struct {
		blockRepo  repository.BlockRepository
		followRepo repository.FollowRepository
		authzSvc   authz.Service
	}
)

func NewService(
	blockRepo repository.BlockRepository,
	followRepo repository.FollowRepository,
	authzSvc authz.Service,
) Service {
	return &service{
		blockRepo:  blockRepo,
		followRepo: followRepo,
		authzSvc:   authzSvc,
	}
}

func (s *service) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return ErrCannotBlockSelf
	}

	r, err := s.authzSvc.GetRole(ctx, blockedID)
	if err != nil {
		return fmt.Errorf("check target role: %w", err)
	}
	if r == authz.RoleSuperAdmin || r == authz.RoleAdmin || r == authz.RoleModerator {
		return ErrCannotBlockStaff
	}

	if err := s.blockRepo.Block(ctx, blockerID, blockedID); err != nil {
		return err
	}

	_ = s.followRepo.Unfollow(ctx, blockerID, blockedID)
	_ = s.followRepo.Unfollow(ctx, blockedID, blockerID)

	return nil
}

func (s *service) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	return s.blockRepo.Unblock(ctx, blockerID, blockedID)
}

func (s *service) IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error) {
	return s.blockRepo.IsBlocked(ctx, blockerID, blockedID)
}

func (s *service) IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error) {
	return s.blockRepo.IsBlockedEither(ctx, userA, userB)
}

func (s *service) GetBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if userID == uuid.Nil {
		return nil, nil
	}
	return s.blockRepo.GetBlockedIDs(ctx, userID)
}

func (s *service) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID) ([]repository.BlockedUser, error) {
	return s.blockRepo.GetBlockedUsers(ctx, blockerID)
}
