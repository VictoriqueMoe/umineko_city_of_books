package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type (
	StreamCredentialsRow struct {
		UserID    uuid.UUID
		IngressID string
		WhipURL   string
		StreamKey string
		Room      string
	}

	StreamCredentialsRepository interface {
		Get(ctx context.Context, userID uuid.UUID) (*StreamCredentialsRow, error)
		Upsert(ctx context.Context, userID uuid.UUID, ingressID, whipURL, streamKey, room string) error
		Delete(ctx context.Context, userID uuid.UUID) error
	}

	streamCredentialsRepository struct {
		db *sql.DB
	}
)

func (r *streamCredentialsRepository) Get(ctx context.Context, userID uuid.UUID) (*StreamCredentialsRow, error) {
	var row StreamCredentialsRow
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, ingress_id, whip_url, stream_key, room
		   FROM stream_credentials
		  WHERE user_id = $1`,
		userID,
	).Scan(&row.UserID, &row.IngressID, &row.WhipURL, &row.StreamKey, &row.Room)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get stream credentials: %w", err)
	}

	return &row, nil
}

func (r *streamCredentialsRepository) Upsert(ctx context.Context, userID uuid.UUID, ingressID, whipURL, streamKey, room string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO stream_credentials (user_id, ingress_id, whip_url, stream_key, room)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE
		    SET ingress_id = excluded.ingress_id,
		        whip_url = excluded.whip_url,
		        stream_key = excluded.stream_key,
		        room = excluded.room,
		        updated_at = NOW()`,
		userID, ingressID, whipURL, streamKey, room,
	)
	if err != nil {
		return fmt.Errorf("upsert stream credentials: %w", err)
	}

	return nil
}

func (r *streamCredentialsRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM stream_credentials WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete stream credentials: %w", err)
	}

	return nil
}
