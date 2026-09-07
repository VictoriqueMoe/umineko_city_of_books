package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	GiphyFavouriteDAO interface {
		Add(ctx context.Context, s spec.NewGiphyFavourite, tx ...*sql.Tx) error
		Remove(ctx context.Context, s spec.GiphyFavouriteDeletion, tx ...*sql.Tx) error
		List(ctx context.Context, q spec.GiphyFavouritePage, tx ...*sql.Tx) ([]model.GiphyFavourite, int, error)
		ListIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	giphyFavouriteDAO struct {
		db *sql.DB
	}
)

func (r *giphyFavouriteDAO) Add(ctx context.Context, s spec.NewGiphyFavourite, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO giphy_favourites (user_id, giphy_id, url, title, preview_url, width, height, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (user_id, giphy_id) DO UPDATE SET
		     url = EXCLUDED.url,
		     title = EXCLUDED.title,
		     preview_url = EXCLUDED.preview_url,
		     width = EXCLUDED.width,
		     height = EXCLUDED.height,
		     created_at = EXCLUDED.created_at`,
		s.UserID, s.GiphyID, s.URL, s.Title, s.PreviewURL, s.Width, s.Height,
	)
	if err != nil {
		return fmt.Errorf("add giphy favourite: %w", err)
	}

	return nil
}

func (r *giphyFavouriteDAO) Remove(ctx context.Context, s spec.GiphyFavouriteDeletion, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM giphy_favourites WHERE user_id = $1 AND giphy_id = $2`,
		s.UserID, s.GiphyID,
	)
	if err != nil {
		return fmt.Errorf("remove giphy favourite: %w", err)
	}

	return nil
}

func (r *giphyFavouriteDAO) List(ctx context.Context, q spec.GiphyFavouritePage, tx ...*sql.Tx) ([]model.GiphyFavourite, int, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	var total int

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM giphy_favourites WHERE user_id = $1`,
		q.UserID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count giphy favourites: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT giphy_id, url, title, preview_url, width, height, created_at
		 FROM giphy_favourites WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		q.UserID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list giphy favourites: %w", err)
	}
	defer rows.Close()

	var out []model.GiphyFavourite

	for rows.Next() {
		var f model.GiphyFavourite
		if err := rows.Scan(&f.GiphyID, &f.URL, &f.Title, &f.PreviewURL, &f.Width, &f.Height, &f.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan giphy favourite: %w", err)
		}

		out = append(out, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate giphy favourites: %w", err)
	}

	return out, total, nil
}

func (r *giphyFavouriteDAO) ListIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT giphy_id FROM giphy_favourites WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list giphy favourite ids: %w", err)
	}
	defer rows.Close()

	var out []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan giphy favourite id: %w", err)
		}

		out = append(out, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate giphy favourite ids: %w", err)
	}

	return out, nil
}
