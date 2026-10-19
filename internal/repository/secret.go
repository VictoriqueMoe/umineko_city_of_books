package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	SecretRepository interface {
		dao.SecretDAO
	}

	secretRepository struct {
		dao.SecretDAO
	}
)

func NewSecretRepo(d dao.SecretDAO) SecretRepository {
	return &secretRepository{SecretDAO: d}
}
