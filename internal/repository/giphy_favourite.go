package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	GiphyFavouriteRepository interface {
		dao.GiphyFavouriteDAO
	}

	giphyFavouriteRepository struct {
		dao.GiphyFavouriteDAO
	}
)

func NewGiphyFavouriteRepo(d dao.GiphyFavouriteDAO) GiphyFavouriteRepository {
	return &giphyFavouriteRepository{GiphyFavouriteDAO: d}
}
