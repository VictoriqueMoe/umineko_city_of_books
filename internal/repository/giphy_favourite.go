package repository

import (
	"context"
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
)

type giphyFavouriteRepository struct {
	dao GiphyFavouriteRepository
}

func NewGiphyFavouriteRepo(dao GiphyFavouriteRepository) GiphyFavouriteRepository {
	return &giphyFavouriteRepository{dao: dao}
}

func (r *giphyFavouriteRepository) Add(ctx context.Context, userID uuid.UUID, fav GiphyFavourite) error {
	return r.dao.Add(ctx, userID, fav)
}

func (r *giphyFavouriteRepository) Remove(ctx context.Context, userID uuid.UUID, giphyID string) error {
	return r.dao.Remove(ctx, userID, giphyID)
}

func (r *giphyFavouriteRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]GiphyFavourite, int, error) {
	return r.dao.List(ctx, userID, limit, offset)
}

func (r *giphyFavouriteRepository) ListIDs(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return r.dao.ListIDs(ctx, userID)
}
