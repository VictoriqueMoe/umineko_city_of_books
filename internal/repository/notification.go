package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	NotificationRepository interface {
		dao.NotificationDAO
	}

	notificationRepository struct {
		dao.NotificationDAO
	}
)

func NewNotificationRepo(d dao.NotificationDAO) NotificationRepository {
	return &notificationRepository{NotificationDAO: d}
}
