package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	StatsRepository interface {
		dao.StatsDAO
	}

	statsRepository struct {
		dao.StatsDAO
	}
)

func NewStatsRepo(d dao.StatsDAO) StatsRepository {
	return &statsRepository{StatsDAO: d}
}
