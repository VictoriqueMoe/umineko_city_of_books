package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	SidebarLastVisitedRepository interface {
		dao.SidebarLastVisitedDAO
	}

	sidebarLastVisitedRepository struct {
		dao.SidebarLastVisitedDAO
	}
)

func NewSidebarLastVisitedRepo(visits dao.SidebarLastVisitedDAO) SidebarLastVisitedRepository {
	return &sidebarLastVisitedRepository{SidebarLastVisitedDAO: visits}
}
