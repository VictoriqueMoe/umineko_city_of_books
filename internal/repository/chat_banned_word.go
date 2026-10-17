package repository

import (
	"context"
	"database/sql"
	"slices"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
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

		cache *cache.Manager
	}
)

func NewChatBannedWordRepo(database *sql.DB, bannedWords dao.ChatBannedWordDAO, auditLog AuditLogRepository, c *cache.Manager) ChatBannedWordRepository {
	return &chatBannedWordRepository{db: database, ChatBannedWordDAO: bannedWords, audit: auditLog, cache: c}
}

func (r *chatBannedWordRepository) ListApplicable(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatBannedWordRow, error) {
	loadGlobal := func(ctx context.Context) ([]model.ChatBannedWordRow, error) {
		return r.ChatBannedWordDAO.ListGlobal(ctx, tx...)
	}

	global, err := r.cache.Load(ctx, cache.ChatBannedWordsGlobal.Key(), loadGlobal)
	if err != nil {
		return nil, err
	}

	loadRoom := func(ctx context.Context) ([]model.ChatBannedWordRow, error) {
		return r.ChatBannedWordDAO.ListForRoom(ctx, roomID, tx...)
	}

	room, err := r.cache.Load(ctx, cache.ChatBannedWordsRoom.Key(roomID.String()), loadRoom)
	if err != nil {
		return nil, err
	}

	return slices.Concat(global, room), nil
}

func (r *chatBannedWordRepository) Create(ctx context.Context, s spec.ChatBannedWordSpec, tx ...*sql.Tx) (*model.ChatBannedWordRow, error) {
	created, err := r.ChatBannedWordDAO.Create(ctx, s, tx...)
	if err != nil {
		return nil, err
	}

	r.forgetApplicable(ctx, created.RoomID)

	return created, nil
}

func (r *chatBannedWordRepository) Update(ctx context.Context, s spec.ChatBannedWordUpdate, tx ...*sql.Tx) error {
	existing, err := r.ChatBannedWordDAO.GetByID(ctx, s.ID, tx...)
	if err != nil {
		return err
	}

	if err := r.ChatBannedWordDAO.Update(ctx, s, tx...); err != nil {
		return err
	}

	if existing != nil {
		r.forgetApplicable(ctx, existing.RoomID)
	}

	return nil
}

func (r *chatBannedWordRepository) Delete(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	existing, err := r.ChatBannedWordDAO.GetByID(ctx, id, tx...)
	if err != nil {
		return err
	}

	if err := r.ChatBannedWordDAO.Delete(ctx, id, tx...); err != nil {
		return err
	}

	if existing != nil {
		r.forgetApplicable(ctx, existing.RoomID)
	}

	return nil
}

func (r *chatBannedWordRepository) forgetApplicable(ctx context.Context, roomID *uuid.UUID) {
	key := cache.ChatBannedWordsGlobal.Key()
	if roomID != nil {
		key = cache.ChatBannedWordsRoom.Key(roomID.String())
	}

	if err := r.cache.Del(ctx, key); err != nil {
		logger.Ctx(ctx).Error().Err(err).Stringer("key", key).Msg("failed to invalidate banned word cache after write")
	}
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

	r.forgetApplicable(ctx, created.RoomID)

	return created, nil
}

func (r *chatBannedWordRepository) DeleteWithAudit(ctx context.Context, s spec.ChatBannedWordDeletion, tx ...*sql.Tx) error {
	existing, err := r.ChatBannedWordDAO.GetByID(ctx, s.ID, tx...)
	if err != nil {
		return err
	}

	err = db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ChatBannedWordDAO.Delete(ctx, s.ID, tx); err != nil {
			return err
		}

		return r.audit.Create(ctx, s.Audit, tx)
	})
	if err != nil {
		return err
	}

	if existing != nil {
		r.forgetApplicable(ctx, existing.RoomID)
	}

	return nil
}
