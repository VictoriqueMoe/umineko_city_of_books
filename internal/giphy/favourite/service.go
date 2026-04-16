package favourite

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	Service interface {
		Add(ctx context.Context, userID uuid.UUID, fav Favourite) error
		Remove(ctx context.Context, userID uuid.UUID, giphyID string) error
		List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Favourite, int, error)
		ListIDs(ctx context.Context, userID uuid.UUID) ([]string, error)
	}

	Favourite struct {
		GiphyID    string `json:"giphy_id"`
		URL        string `json:"url"`
		Title      string `json:"title"`
		PreviewURL string `json:"preview_url"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
	}

	service struct {
		repo repository.GiphyFavouriteRepository
	}
)

var (
	ErrGiphyIDRequired = errors.New("giphy_id required")
	ErrURLRequired     = errors.New("url required")
)

func NewService(repo repository.GiphyFavouriteRepository) Service {
	return &service{repo: repo}
}

func (s *service) Add(ctx context.Context, userID uuid.UUID, fav Favourite) error {
	if fav.GiphyID == "" {
		return ErrGiphyIDRequired
	}
	if fav.URL == "" {
		return ErrURLRequired
	}
	return s.repo.Add(ctx, userID, repository.GiphyFavourite{
		GiphyID:    fav.GiphyID,
		URL:        fav.URL,
		Title:      fav.Title,
		PreviewURL: fav.PreviewURL,
		Width:      fav.Width,
		Height:     fav.Height,
	})
}

func (s *service) Remove(ctx context.Context, userID uuid.UUID, giphyID string) error {
	return s.repo.Remove(ctx, userID, giphyID)
}

func (s *service) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Favourite, int, error) {
	rows, total, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Favourite, len(rows))
	for i, r := range rows {
		out[i] = Favourite{
			GiphyID:    r.GiphyID,
			URL:        r.URL,
			Title:      r.Title,
			PreviewURL: r.PreviewURL,
			Width:      r.Width,
			Height:     r.Height,
		}
	}
	return out, total, nil
}

func (s *service) ListIDs(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.repo.ListIDs(ctx, userID)
}
