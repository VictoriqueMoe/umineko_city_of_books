package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	UploadRepository interface {
		dao.UploadDAO
	}

	uploadRepository struct {
		dao.UploadDAO
	}
)

func NewUploadRepo(d dao.UploadDAO) UploadRepository {
	return &uploadRepository{UploadDAO: d}
}
