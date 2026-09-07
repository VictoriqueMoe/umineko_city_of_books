package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	SitemapRepository interface {
		dao.SitemapDAO
	}

	sitemapRepository struct {
		dao.SitemapDAO
	}
)

func NewSitemapRepo(d dao.SitemapDAO) SitemapRepository {
	return &sitemapRepository{SitemapDAO: d}
}
