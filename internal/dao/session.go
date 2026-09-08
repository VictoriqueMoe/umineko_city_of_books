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

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model/spec"
)

type (
	SessionDAO interface {
		Create(ctx context.Context, s spec.NewSession, tx ...*sql.Tx) error
		GetUserID(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, time.Time, error)
		Delete(ctx context.Context, token string, tx ...*sql.Tx) error
		DeleteAllForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		DeleteAllForUserExcept(ctx context.Context, s spec.SessionDeletionExcept, tx ...*sql.Tx) error
		CleanExpired(ctx context.Context, tx ...*sql.Tx) (int, error)
	}

	sessionDAO struct {
		db *sql.DB
	}
)

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

func (r *sessionDAO) Create(ctx context.Context, s spec.NewSession, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreateSession(ctx, sqlcgen.CreateSessionParams{
		Token:     hashSessionToken(s.Token),
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

func (r *sessionDAO) GetUserID(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, time.Time, error) {
	row, err := genQueries(r.db, tx).GetSessionUserID(ctx, hashSessionToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, time.Time{}, fmt.Errorf("session not found")
	}
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("query session: %w", err)
	}

	return row.UserID, row.ExpiresAt, nil
}

func (r *sessionDAO) Delete(ctx context.Context, token string, tx ...*sql.Tx) error {
	return genQueries(r.db, tx).DeleteSession(ctx, hashSessionToken(token))
}

func (r *sessionDAO) DeleteAllForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return genQueries(r.db, tx).DeleteSessionsForUser(ctx, userID)
}

func (r *sessionDAO) DeleteAllForUserExcept(ctx context.Context, s spec.SessionDeletionExcept, tx ...*sql.Tx) error {
	return genQueries(r.db, tx).DeleteSessionsForUserExcept(ctx, sqlcgen.DeleteSessionsForUserExceptParams{
		UserID: s.UserID,
		Token:  hashSessionToken(s.KeepToken),
	})
}

func (r *sessionDAO) CleanExpired(ctx context.Context, tx ...*sql.Tx) (int, error) {
	n, err := genQueries(r.db, tx).DeleteExpiredSessions(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("clean expired sessions: %w", err)
	}

	return int(n), nil
}
