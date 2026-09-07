package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	StreamCredentialsRepository interface {
		dao.StreamCredentialsDAO
	}

	streamCredentialsRepository struct {
		dao.StreamCredentialsDAO
	}
)

func NewStreamCredentialsRepo(d dao.StreamCredentialsDAO) StreamCredentialsRepository {
	return &streamCredentialsRepository{StreamCredentialsDAO: d}
}
