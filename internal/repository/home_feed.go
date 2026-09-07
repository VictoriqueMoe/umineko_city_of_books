package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	HomeFeedRepository interface {
		dao.HomeFeedDAO
	}

	homeFeedRepository struct {
		dao.HomeFeedDAO
	}
)

func NewHomeFeedRepo(homeFeed dao.HomeFeedDAO) HomeFeedRepository {
	return &homeFeedRepository{HomeFeedDAO: homeFeed}
}
