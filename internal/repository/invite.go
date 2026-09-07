package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	InviteRepository interface {
		dao.InviteDAO
	}

	inviteRepository struct {
		dao.InviteDAO
	}
)

func NewInviteRepo(d dao.InviteDAO) InviteRepository {
	return &inviteRepository{InviteDAO: d}
}
