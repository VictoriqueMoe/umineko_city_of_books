package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	ChatBannedWordDAO interface {
		Create(ctx context.Context, spec ChatBannedWordSpec, tx ...*sql.Tx) (*ChatBannedWordRow, error)
		Update(ctx context.Context, id uuid.UUID, spec ChatBannedWordUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*ChatBannedWordRow, error)
		ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]ChatBannedWordRow, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatBannedWordRow, error)
		ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatBannedWordRow, error)
	}

	ChatBannedWordRepository interface {
		ChatBannedWordDAO

		CreateWithAudit(ctx context.Context, spec ChatBannedWordSpec, audit NewAuditEntry, tx ...*sql.Tx) (*ChatBannedWordRow, error)
		DeleteWithAudit(ctx context.Context, id uuid.UUID, audit NewAuditEntry, tx ...*sql.Tx) error
	}

	ChatBannedWordSpec struct {
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
	}

	ChatBannedWordUpdate struct {
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
	}

	ChatBannedWordRow struct {
		ID            uuid.UUID
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
		CreatedByName string
		CreatedAt     string
	}
)

type chatBannedWordRepository struct {
	db    *sql.DB
	dao   ChatBannedWordDAO
	audit AuditLogRepository
}

func NewChatBannedWordRepo(database *sql.DB, dao ChatBannedWordDAO, audit AuditLogRepository) ChatBannedWordRepository {
	return &chatBannedWordRepository{db: database, dao: dao, audit: audit}
}

func (r *chatBannedWordRepository) CreateWithAudit(ctx context.Context, spec ChatBannedWordSpec, audit NewAuditEntry, tx ...*sql.Tx) (*ChatBannedWordRow, error) {
	var created *ChatBannedWordRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.Create(ctx, spec, tx)
		if err != nil {
			return err
		}

		if audit.TargetID == "" {
			audit.TargetID = created.ID.String()
		}

		return r.audit.Create(ctx, audit, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatBannedWordRepository) DeleteWithAudit(ctx context.Context, id uuid.UUID, audit NewAuditEntry, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.Delete(ctx, id, tx); err != nil {
			return err
		}

		return r.audit.Create(ctx, audit, tx)
	})
}

func (r *chatBannedWordRepository) Create(ctx context.Context, spec ChatBannedWordSpec, tx ...*sql.Tx) (*ChatBannedWordRow, error) {
	return r.dao.Create(ctx, spec, tx...)
}

func (r *chatBannedWordRepository) Update(ctx context.Context, id uuid.UUID, spec ChatBannedWordUpdate, tx ...*sql.Tx) error {
	return r.dao.Update(ctx, id, spec, tx...)
}

func (r *chatBannedWordRepository) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, id, tx...)
}

func (r *chatBannedWordRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*ChatBannedWordRow, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *chatBannedWordRepository) ListGlobal(ctx context.Context, tx ...*sql.Tx) ([]ChatBannedWordRow, error) {
	return r.dao.ListGlobal(ctx, tx...)
}

func (r *chatBannedWordRepository) ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatBannedWordRow, error) {
	return r.dao.ListForRoom(ctx, roomID, tx...)
}

func (r *chatBannedWordRepository) ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatBannedWordRow, error) {
	return r.dao.ListApplicable(ctx, roomID, tx...)
}
