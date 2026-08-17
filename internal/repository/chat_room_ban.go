package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	ChatRoomBanDAO interface {
		Ban(ctx context.Context, roomID, userID uuid.UUID, bannedBy *uuid.UUID, reason string, tx ...*sql.Tx) error
		Unban(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		IsBanned(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatRoomBanRow, error)
		BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	ChatRoomBanRepository interface {
		ChatRoomBanDAO

		UnbanWithAudit(ctx context.Context, roomID, userID, actorID uuid.UUID, tx ...*sql.Tx) error
	}

	ChatRoomBanRow struct {
		RoomID            uuid.UUID
		UserID            uuid.UUID
		Username          string
		DisplayName       string
		AvatarURL         string
		Role              string
		BannedByID        *uuid.UUID
		BannedByUsername  string
		BannedByDisplay   string
		BannedByAvatarURL string
		Reason            string
		CreatedAt         string
	}
)

type chatRoomBanRepository struct {
	db    *sql.DB
	dao   ChatRoomBanDAO
	audit AuditLogRepository
}

func NewChatRoomBanRepo(database *sql.DB, dao ChatRoomBanDAO, audit AuditLogRepository) ChatRoomBanRepository {
	return &chatRoomBanRepository{db: database, dao: dao, audit: audit}
}

func (r *chatRoomBanRepository) UnbanWithAudit(ctx context.Context, roomID, userID, actorID uuid.UUID, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.Unban(ctx, roomID, userID, tx); err != nil {
			return err
		}

		return r.audit.Create(ctx, NewAuditEntry{
			ActorID:    actorID,
			Action:     AuditActionChatRoomUnban,
			TargetType: AuditTargetChatRoom,
			TargetID:   roomID.String(),
			SubjectID:  userID,
		}, tx)
	})
}

func (r *chatRoomBanRepository) Ban(ctx context.Context, roomID, userID uuid.UUID, bannedBy *uuid.UUID, reason string, tx ...*sql.Tx) error {
	return r.dao.Ban(ctx, roomID, userID, bannedBy, reason, tx...)
}

func (r *chatRoomBanRepository) Unban(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unban(ctx, roomID, userID, tx...)
}

func (r *chatRoomBanRepository) IsBanned(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsBanned(ctx, roomID, userID, tx...)
}

func (r *chatRoomBanRepository) ListForRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatRoomBanRow, error) {
	return r.dao.ListForRoom(ctx, roomID, tx...)
}

func (r *chatRoomBanRepository) BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.BannedRoomIDsForUser(ctx, userID, tx...)
}
