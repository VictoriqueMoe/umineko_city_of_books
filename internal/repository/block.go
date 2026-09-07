package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	BlockRepository interface {
		dao.BlockDAO
	}

	blockRepository struct {
		dao.BlockDAO
	}
)

func NewBlockRepo(d dao.BlockDAO) BlockRepository {
	return &blockRepository{BlockDAO: d}
}
