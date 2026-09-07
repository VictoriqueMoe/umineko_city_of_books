package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	PasswordResetRepository interface {
		dao.PasswordResetDAO

		Issue(ctx context.Context, s spec.NewPasswordReset, tx ...*sql.Tx) error
	}

	passwordResetRepository struct {
		db *sql.DB
		dao.PasswordResetDAO
	}
)

func NewPasswordResetRepo(database *sql.DB, d dao.PasswordResetDAO) PasswordResetRepository {
	return &passwordResetRepository{db: database, PasswordResetDAO: d}
}

func (r *passwordResetRepository) Issue(ctx context.Context, s spec.NewPasswordReset, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.PasswordResetDAO.DeleteUnusedForUser(ctx, s.UserID, tx); err != nil {
			return err
		}

		return r.PasswordResetDAO.Create(ctx, s, tx)
	})
}
