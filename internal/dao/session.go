package dao

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type (
	sessionDAO struct {
		db *sql.DB
	}
)

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

func (r *sessionDAO) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashSessionToken(token), userID, expiresAt,
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
		`SELECT user_id, expires_at FROM sessions WHERE token = $1`, hashSessionToken(token),
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
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, hashSessionToken(token))
	return err
}

func (r *sessionDAO) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (r *sessionDAO) DeleteAllForUserExcept(ctx context.Context, userID uuid.UUID, keepToken string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1 AND token <> $2`, userID, hashSessionToken(keepToken))
	return err
}

func (r *sessionDAO) CleanExpired(ctx context.Context) (int, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < $1`, time.Now())
	if err != nil {
		return 0, fmt.Errorf("clean expired sessions: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("clean expired sessions rows affected: %w", err)
	}

	return int(n), nil
}
