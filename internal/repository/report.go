package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	ReportRepository interface {
		dao.ReportDAO
	}

	reportRepository struct {
		dao.ReportDAO
	}
)

func NewReportRepo(d dao.ReportDAO) ReportRepository {
	return &reportRepository{ReportDAO: d}
}
