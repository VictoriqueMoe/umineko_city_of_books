package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	DeviceTokenRepository interface {
		dao.DeviceTokenDAO
	}

	deviceTokenRepository struct {
		dao.DeviceTokenDAO
	}
)

func NewDeviceTokenRepo(d dao.DeviceTokenDAO) DeviceTokenRepository {
	return &deviceTokenRepository{DeviceTokenDAO: d}
}
