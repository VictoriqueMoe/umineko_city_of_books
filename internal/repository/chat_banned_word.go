package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatBannedWordRepository interface {
		dao.ChatBannedWordDAO

		CreateWithAudit(ctx context.Context, s spec.ChatBannedWordCreation, tx ...*sql.Tx) (*model.ChatBannedWordRow, error)
		DeleteWithAudit(ctx context.Context, s spec.ChatBannedWordDeletion, tx ...*sql.Tx) error
	}

	chatBannedWordRepository struct {
		db *sql.DB
		dao.ChatBannedWordDAO
		audit AuditLogRepository
	}
)

func NewChatBannedWordRepo(database *sql.DB, bannedWords dao.ChatBannedWordDAO, auditLog AuditLogRepository) ChatBannedWordRepository {
	return &chatBannedWordRepository{db: database, ChatBannedWordDAO: bannedWords, audit: auditLog}
}

func (r *chatBannedWordRepository) CreateWithAudit(ctx context.Context, s spec.ChatBannedWordCreation, tx ...*sql.Tx) (*model.ChatBannedWordRow, error) {
	var created *model.ChatBannedWordRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.ChatBannedWordDAO.Create(ctx, s.ChatBannedWordSpec, tx)
		if err != nil {
			return err
		}

		entry := s.Audit
		if entry.TargetID == "" {
			entry.TargetID = created.ID.String()
		}

		return r.audit.Create(ctx, entry, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatBannedWordRepository) DeleteWithAudit(ctx context.Context, s spec.ChatBannedWordDeletion, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ChatBannedWordDAO.Delete(ctx, s.ID, tx); err != nil {
			return err
		}

		return r.audit.Create(ctx, s.Audit, tx)
	})
}
