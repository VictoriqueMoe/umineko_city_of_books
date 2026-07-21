package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository"
)

type (
	bannedGiphyDAO struct {
		db *sql.DB
	}

	bannedGiphyRepository struct {
		repository.BannedGiphyRepository
	}
)

func (r *bannedGiphyDAO) List(ctx context.Context) ([]repository.BannedGiphyRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT kind, value, created_at, created_by, reason FROM banned_giphy ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list banned giphy: %w", err)
	}
	defer rows.Close()

	var result []repository.BannedGiphyRow
	for rows.Next() {
		var row repository.BannedGiphyRow
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

func (r *bannedGiphyDAO) Add(ctx context.Context, kind, value, reason string, createdBy *string) error {
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

func (r *bannedGiphyDAO) Remove(ctx context.Context, kind, value string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM banned_giphy WHERE kind = $1 AND value = $2`,
		kind, value,
	)
	if err != nil {
		return fmt.Errorf("remove banned giphy: %w", err)
	}
	return nil
}
