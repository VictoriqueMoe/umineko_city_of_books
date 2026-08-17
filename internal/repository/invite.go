package repository

import (
	"context"
	"database/sql"

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
		Create(ctx context.Context, code string, createdBy uuid.UUID, tx ...*sql.Tx) error
		GetByCode(ctx context.Context, code string, tx ...*sql.Tx) (*Invite, error)
		MarkUsed(ctx context.Context, code string, usedBy uuid.UUID, tx ...*sql.Tx) error
		List(ctx context.Context, limit, offset int, tx ...*sql.Tx) ([]Invite, int, error)
		Delete(ctx context.Context, code string, tx ...*sql.Tx) error
	}
)

type inviteRepository struct {
	dao InviteRepository
}

func NewInviteRepo(dao InviteRepository) InviteRepository {
	return &inviteRepository{dao: dao}
}

func (r *inviteRepository) Create(ctx context.Context, code string, createdBy uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Create(ctx, code, createdBy, tx...)
}

func (r *inviteRepository) GetByCode(ctx context.Context, code string, tx ...*sql.Tx) (*Invite, error) {
	return r.dao.GetByCode(ctx, code, tx...)
}

func (r *inviteRepository) MarkUsed(ctx context.Context, code string, usedBy uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkUsed(ctx, code, usedBy, tx...)
}

func (r *inviteRepository) List(ctx context.Context, limit, offset int, tx ...*sql.Tx) ([]Invite, int, error) {
	return r.dao.List(ctx, limit, offset, tx...)
}

func (r *inviteRepository) Delete(ctx context.Context, code string, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, code, tx...)
}
