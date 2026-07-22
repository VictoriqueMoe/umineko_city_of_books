package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	SessionRepository interface {
		Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error
		GetUserID(ctx context.Context, token string) (uuid.UUID, time.Time, error)
		Delete(ctx context.Context, token string) error
		DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
		CleanExpired(ctx context.Context) error
	}
)

type sessionRepository struct {
	dao SessionRepository
}

func NewSessionRepo(dao SessionRepository) SessionRepository {
	return &sessionRepository{dao: dao}
}

func (r *sessionRepository) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error {
	return r.dao.Create(ctx, token, userID, expiresAt)
}

func (r *sessionRepository) GetUserID(ctx context.Context, token string) (uuid.UUID, time.Time, error) {
	return r.dao.GetUserID(ctx, token)
}

func (r *sessionRepository) Delete(ctx context.Context, token string) error {
	return r.dao.Delete(ctx, token)
}

func (r *sessionRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.dao.DeleteAllForUser(ctx, userID)
}

func (r *sessionRepository) CleanExpired(ctx context.Context) error {
	return r.dao.CleanExpired(ctx)
}
