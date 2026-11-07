package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	PendingUploadRepository interface {
		dao.PendingUploadDAO
	}

	pendingUploadRepository struct {
		dao.PendingUploadDAO
	}
)

func NewPendingUploadRepo(d dao.PendingUploadDAO) PendingUploadRepository {
	return &pendingUploadRepository{PendingUploadDAO: d}
}
