package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	Invite struct {
		Code      string
		CreatedBy uuid.UUID
		UsedBy    *uuid.UUID
		UsedAt    *string
		CreatedAt string
	}

	InviteRepository interface {
		Create(ctx context.Context, code string, createdBy uuid.UUID) error
		GetByCode(ctx context.Context, code string) (*Invite, error)
		MarkUsed(ctx context.Context, code string, usedBy uuid.UUID) error
		List(ctx context.Context, limit, offset int) ([]Invite, int, error)
		Delete(ctx context.Context, code string) error
	}
)

type inviteRepository struct {
	dao InviteRepository
}

func NewInviteRepo(dao InviteRepository) InviteRepository {
	return &inviteRepository{dao: dao}
}

func (r *inviteRepository) Create(ctx context.Context, code string, createdBy uuid.UUID) error {
	return r.dao.Create(ctx, code, createdBy)
}

func (r *inviteRepository) GetByCode(ctx context.Context, code string) (*Invite, error) {
	return r.dao.GetByCode(ctx, code)
}

func (r *inviteRepository) MarkUsed(ctx context.Context, code string, usedBy uuid.UUID) error {
	return r.dao.MarkUsed(ctx, code, usedBy)
}

func (r *inviteRepository) List(ctx context.Context, limit, offset int) ([]Invite, int, error) {
	return r.dao.List(ctx, limit, offset)
}

func (r *inviteRepository) Delete(ctx context.Context, code string) error {
	return r.dao.Delete(ctx, code)
}
