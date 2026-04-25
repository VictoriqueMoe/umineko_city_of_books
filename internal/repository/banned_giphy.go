package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type (
	BannedGiphyRepository interface {
		List(ctx context.Context) ([]BannedGiphyRow, error)
		Add(ctx context.Context, kind, value, reason string, createdBy *string) error
		Remove(ctx context.Context, kind, value string) error
	}

	bannedGiphyRepository struct {
		db *sql.DB
	}

	BannedGiphyRow struct {
		Kind      string
		Value     string
		CreatedAt time.Time
		CreatedBy *string
		Reason    string
	}
)

func (r *bannedGiphyRepository) List(ctx context.Context) ([]BannedGiphyRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT kind, value, created_at, created_by, reason FROM banned_giphy ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list banned giphy: %w", err)
	}
	defer rows.Close()

	var result []BannedGiphyRow
	for rows.Next() {
		var row BannedGiphyRow
		var reason sql.NullString
		if err := rows.Scan(&row.Kind, &row.Value, &row.CreatedAt, &row.CreatedBy, &reason); err != nil {
			return nil, fmt.Errorf("scan banned giphy: %w", err)
		}
		if reason.Valid {
			row.Reason = reason.String
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *bannedGiphyRepository) Add(ctx context.Context, kind, value, reason string, createdBy *string) error {
	var reasonVal any
	if reason == "" {
		reasonVal = nil
	} else {
		reasonVal = reason
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO banned_giphy (kind, value, created_by, reason) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
		kind, value, createdBy, reasonVal,
	)
	if err != nil {
		return fmt.Errorf("add banned giphy: %w", err)
	}
	return nil
}

func (r *bannedGiphyRepository) Remove(ctx context.Context, kind, value string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM banned_giphy WHERE kind = $1 AND value = $2`,
		kind, value,
	)
	if err != nil {
		return fmt.Errorf("remove banned giphy: %w", err)
	}
	return nil
}
