package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type (
	GiphyFavouriteRepository interface {
		Add(ctx context.Context, userID uuid.UUID, fav GiphyFavourite) error
		Remove(ctx context.Context, userID uuid.UUID, giphyID string) error
		List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]GiphyFavourite, int, error)
		ListIDs(ctx context.Context, userID uuid.UUID) ([]string, error)
	}

	GiphyFavourite struct {
		GiphyID    string
		URL        string
		Title      string
		PreviewURL string
		Width      int
		Height     int
		CreatedAt  time.Time
	}

	giphyFavouriteRepository struct {
		db *sql.DB
	}
)

func (r *giphyFavouriteRepository) Add(ctx context.Context, userID uuid.UUID, fav GiphyFavourite) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO giphy_favourites (user_id, giphy_id, url, title, preview_url, width, height, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (user_id, giphy_id) DO UPDATE SET
		     url = EXCLUDED.url,
		     title = EXCLUDED.title,
		     preview_url = EXCLUDED.preview_url,
		     width = EXCLUDED.width,
		     height = EXCLUDED.height,
		     created_at = EXCLUDED.created_at`,
		userID, fav.GiphyID, fav.URL, fav.Title, fav.PreviewURL, fav.Width, fav.Height,
	)
	if err != nil {
		return fmt.Errorf("add giphy favourite: %w", err)
	}
	return nil
}

func (r *giphyFavouriteRepository) Remove(ctx context.Context, userID uuid.UUID, giphyID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM giphy_favourites WHERE user_id = $1 AND giphy_id = $2`,
		userID, giphyID,
	)
	if err != nil {
		return fmt.Errorf("remove giphy favourite: %w", err)
	}
	return nil
}

func (r *giphyFavouriteRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]GiphyFavourite, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM giphy_favourites WHERE user_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count giphy favourites: %w", err)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT giphy_id, url, title, preview_url, width, height, created_at
		 FROM giphy_favourites WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list giphy favourites: %w", err)
	}
	defer rows.Close()
	var out []GiphyFavourite
	for rows.Next() {
		var f GiphyFavourite
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

func (r *giphyFavouriteRepository) ListIDs(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
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
