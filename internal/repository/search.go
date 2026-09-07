package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	SearchRepository interface {
		dao.SearchDAO
	}

	searchRepository struct {
		dao.SearchDAO
	}
)

func NewSearchRepo(d dao.SearchDAO) SearchRepository {
	return &searchRepository{SearchDAO: d}
}
