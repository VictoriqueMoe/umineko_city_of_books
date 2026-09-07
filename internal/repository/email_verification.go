package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	EmailVerificationRepository interface {
		dao.EmailVerificationDAO

		Issue(ctx context.Context, s spec.NewEmailVerification, tx ...*sql.Tx) error
	}

	emailVerificationRepository struct {
		db *sql.DB
		dao.EmailVerificationDAO
	}
)

func NewEmailVerificationRepo(database *sql.DB, d dao.EmailVerificationDAO) EmailVerificationRepository {
	return &emailVerificationRepository{db: database, EmailVerificationDAO: d}
}

func (r *emailVerificationRepository) Issue(ctx context.Context, s spec.NewEmailVerification, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.EmailVerificationDAO.DeleteUnusedForUser(ctx, s.UserID, tx); err != nil {
			return err
		}

		return r.EmailVerificationDAO.Create(ctx, s, tx)
	})
}
