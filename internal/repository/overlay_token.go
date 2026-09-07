package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	OverlayTokenRepository interface {
		dao.OverlayTokenDAO
	}

	overlayTokenRepository struct {
		dao.OverlayTokenDAO
	}
)

func NewOverlayTokenRepo(d dao.OverlayTokenDAO) OverlayTokenRepository {
	return &overlayTokenRepository{OverlayTokenDAO: d}
}
