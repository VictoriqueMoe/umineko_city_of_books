package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	BannedGiphyRepository interface {
		dao.BannedGiphyDAO
	}

	bannedGiphyRepository struct {
		dao.BannedGiphyDAO
	}
)

func NewBannedGiphyRepo(d dao.BannedGiphyDAO) BannedGiphyRepository {
	return &bannedGiphyRepository{BannedGiphyDAO: d}
}
