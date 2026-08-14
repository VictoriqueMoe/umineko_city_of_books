package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type (
	SessionRepository interface {
		Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, tx ...*sql.Tx) error
		GetUserID(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, time.Time, error)
		Delete(ctx context.Context, token string, tx ...*sql.Tx) error
		DeleteAllForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAllForUserExcept(ctx context.Context, userID uuid.UUID, keepToken string, tx ...*sql.Tx) error
		CleanExpired(ctx context.Context, tx ...*sql.Tx) (int, error)
	}
)

type sessionRepository struct {
	dao SessionRepository
}

func NewSessionRepo(dao SessionRepository) SessionRepository {
	return &sessionRepository{dao: dao}
}

func (r *sessionRepository) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, tx ...*sql.Tx) error {
	return r.dao.Create(ctx, token, userID, expiresAt, tx...)
}

func (r *sessionRepository) GetUserID(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, time.Time, error) {
	return r.dao.GetUserID(ctx, token, tx...)
}

func (r *sessionRepository) Delete(ctx context.Context, token string, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, token, tx...)
}

func (r *sessionRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteAllForUser(ctx, userID, tx...)
}

func (r *sessionRepository) DeleteAllForUserExcept(ctx context.Context, userID uuid.UUID, keepToken string, tx ...*sql.Tx) error {
	return r.dao.DeleteAllForUserExcept(ctx, userID, keepToken, tx...)
}

func (r *sessionRepository) CleanExpired(ctx context.Context, tx ...*sql.Tx) (int, error) {
	return r.dao.CleanExpired(ctx, tx...)
}
