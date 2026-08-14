package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type (
	GiphyFavouriteRepository interface {
		Add(ctx context.Context, userID uuid.UUID, fav GiphyFavourite, tx ...*sql.Tx) error
		Remove(ctx context.Context, userID uuid.UUID, giphyID string, tx ...*sql.Tx) error
		List(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]GiphyFavourite, int, error)
		ListIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error)
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
)

type giphyFavouriteRepository struct {
	dao GiphyFavouriteRepository
}

func NewGiphyFavouriteRepo(dao GiphyFavouriteRepository) GiphyFavouriteRepository {
	return &giphyFavouriteRepository{dao: dao}
}

func (r *giphyFavouriteRepository) Add(ctx context.Context, userID uuid.UUID, fav GiphyFavourite, tx ...*sql.Tx) error {
	return r.dao.Add(ctx, userID, fav, tx...)
}

func (r *giphyFavouriteRepository) Remove(ctx context.Context, userID uuid.UUID, giphyID string, tx ...*sql.Tx) error {
	return r.dao.Remove(ctx, userID, giphyID, tx...)
}

func (r *giphyFavouriteRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]GiphyFavourite, int, error) {
	return r.dao.List(ctx, userID, limit, offset, tx...)
}

func (r *giphyFavouriteRepository) ListIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.ListIDs(ctx, userID, tx...)
}
