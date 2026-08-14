package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	streamCredentialsDAO struct {
		db *sql.DB
	}
)

func (r *streamCredentialsDAO) Get(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*repository.StreamCredentialsRow, error) {
	var row repository.StreamCredentialsRow
	err := getDb(r.db, tx).QueryRowContext(ctx,
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

func (r *streamCredentialsDAO) Upsert(ctx context.Context, spec repository.NewStreamCredentials, tx ...*sql.Tx) error {
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`INSERT INTO stream_credentials (user_id, ingress_id, whip_url, stream_key, room)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE
		    SET ingress_id = excluded.ingress_id,
		        whip_url = excluded.whip_url,
		        stream_key = excluded.stream_key,
		        room = excluded.room,
		        updated_at = NOW()`,
		spec.UserID, spec.IngressID, spec.WhipURL, spec.StreamKey, spec.Room,
	)
	if err != nil {
		return fmt.Errorf("upsert stream credentials: %w", err)
	}

	return nil
}

func (r *streamCredentialsDAO) Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	_, err := getDb(r.db, tx).ExecContext(ctx,
		`DELETE FROM stream_credentials WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete stream credentials: %w", err)
	}

	return nil
}
