package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	SessionRepository interface {
		dao.SessionDAO
	}

	sessionRepository struct {
		dao.SessionDAO
	}
)

func NewSessionRepo(d dao.SessionDAO) SessionRepository {
	return &sessionRepository{SessionDAO: d}
}
