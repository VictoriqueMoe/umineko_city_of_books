package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	StoredFileRepository interface {
		dao.StoredFileDAO
	}

	storedFileRepository struct {
		dao.StoredFileDAO
	}
)

func NewStoredFileRepo(d dao.StoredFileDAO) StoredFileRepository {
	return &storedFileRepository{StoredFileDAO: d}
}
