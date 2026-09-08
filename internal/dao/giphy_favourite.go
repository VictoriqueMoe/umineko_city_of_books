package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
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

	giphyFavouriteRow = sqlcgen.ListGiphyFavouritesRow
)

func toGiphyFavourite(row giphyFavouriteRow) model.GiphyFavourite {
	return model.GiphyFavourite{
		GiphyID:    row.GiphyID,
		URL:        row.Url,
		Title:      row.Title,
		PreviewURL: row.PreviewUrl,
		Width:      int(row.Width),
		Height:     int(row.Height),
		CreatedAt:  row.CreatedAt,
	}
}

func (r *giphyFavouriteDAO) Add(ctx context.Context, s spec.NewGiphyFavourite, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AddGiphyFavourite(ctx, sqlcgen.AddGiphyFavouriteParams{
		UserID:     s.UserID,
		GiphyID:    s.GiphyID,
		Url:        s.URL,
		Title:      s.Title,
		PreviewUrl: s.PreviewURL,
		Width:      int32(s.Width),
		Height:     int32(s.Height),
	})
	if err != nil {
		return fmt.Errorf("add giphy favourite: %w", err)
	}

	return nil
}

func (r *giphyFavouriteDAO) Remove(ctx context.Context, s spec.GiphyFavouriteDeletion, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).RemoveGiphyFavourite(ctx, sqlcgen.RemoveGiphyFavouriteParams{
		UserID:  s.UserID,
		GiphyID: s.GiphyID,
	})
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

	queries := genQueries(r.db, tx)

	total, err := queries.CountGiphyFavourites(ctx, q.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count giphy favourites: %w", err)
	}

	rows, err := queries.ListGiphyFavourites(ctx, sqlcgen.ListGiphyFavouritesParams{
		UserID: q.UserID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list giphy favourites: %w", err)
	}

	var out []model.GiphyFavourite
	for _, row := range rows {
		out = append(out, toGiphyFavourite(row))
	}

	return out, int(total), nil
}

func (r *giphyFavouriteDAO) ListIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	ids, err := genQueries(r.db, tx).ListGiphyFavouriteIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list giphy favourite ids: %w", err)
	}

	return ids, nil
}
