package repository

import (
	"umineko_city_of_books/internal/dao"
)

type (
	AuditLogRepository interface {
		dao.AuditLogDAO
	}

	auditLogRepository struct {
		dao.AuditLogDAO
	}
)

func NewAuditLogRepo(d dao.AuditLogDAO) AuditLogRepository {
	return &auditLogRepository{AuditLogDAO: d}
}
