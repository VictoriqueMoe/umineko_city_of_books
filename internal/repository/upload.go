package repository

import (
	"database/sql"
)

type UploadRepository interface {
	GetAllReferencedFiles(tx ...*sql.Tx) ([]string, error)
}

type uploadRepository struct {
	dao UploadRepository
}

func NewUploadRepo(dao UploadRepository) UploadRepository {
	return &uploadRepository{dao: dao}
}

func (r *uploadRepository) GetAllReferencedFiles(tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetAllReferencedFiles(tx...)
}
