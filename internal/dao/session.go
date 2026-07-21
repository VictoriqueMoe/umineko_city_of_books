package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	sessionDAO struct {
		db *sql.DB
	}

	sessionRepository struct {
		repository.SessionRepository
	}
)

func (r *sessionDAO) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *sessionDAO) GetUserID(ctx context.Context, token string) (uuid.UUID, time.Time, error) {
	var userID uuid.UUID
	var expiresAt time.Time

	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM sessions WHERE token = $1`, token,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, time.Time{}, fmt.Errorf("session not found")
	}
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("query session: %w", err)
	}

	return userID, expiresAt, nil
}

func (r *sessionDAO) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (r *sessionDAO) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (r *sessionDAO) CleanExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < $1`, time.Now())
	return err
}
