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
