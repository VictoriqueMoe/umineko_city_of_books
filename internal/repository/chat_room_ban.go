package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatRoomBanRepository interface {
		dao.ChatRoomBanDAO

		UnbanWithAudit(ctx context.Context, s spec.ChatRoomUnban, tx ...*sql.Tx) error
	}

	chatRoomBanRepository struct {
		db *sql.DB
		dao.ChatRoomBanDAO
		audit AuditLogRepository
	}
)

func NewChatRoomBanRepo(database *sql.DB, bans dao.ChatRoomBanDAO, auditLog AuditLogRepository) ChatRoomBanRepository {
	return &chatRoomBanRepository{db: database, ChatRoomBanDAO: bans, audit: auditLog}
}

func (r *chatRoomBanRepository) UnbanWithAudit(ctx context.Context, s spec.ChatRoomUnban, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ChatRoomBanDAO.Unban(ctx, s.ChatMemberRef, tx); err != nil {
			return err
		}

		entry := audit.NewEntry{
			ActorID:    s.ActorID,
			Action:     audit.ActionChatRoomUnban,
			TargetType: audit.TargetChatRoom,
			TargetID:   s.RoomID.String(),
			SubjectID:  s.UserID,
		}

		return r.audit.Create(ctx, entry, tx)
	})
}
