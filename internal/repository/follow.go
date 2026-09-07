package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	FollowRepository interface {
		dao.FollowDAO
	}

	followRepository struct {
		dao.FollowDAO
	}
)

func NewFollowRepo(d dao.FollowDAO) FollowRepository {
	return &followRepository{FollowDAO: d}
}
