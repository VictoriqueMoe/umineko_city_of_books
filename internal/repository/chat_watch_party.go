package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	ChatWatchPartySessionRow struct {
		ID                  uuid.UUID
		RoomID              uuid.UUID
		StartedBy           uuid.UUID
		ControllerID        uuid.UUID
		HyperbeamSessionID  string
		HyperbeamAdminToken string
		EmbedURL            string
		VMBaseURL           string
		Title               string
		Type                string
		StartURL            sql.NullString
		Region              sql.NullString
		Status              string
		StartedAt           string
		EndedAt             sql.NullString
		EndedReason         sql.NullString
	}

	ChatWatchPartyParticipantRow struct {
		SessionID           uuid.UUID
		UserID              uuid.UUID
		Username            string
		DisplayName         string
		AvatarURL           string
		HasControl          bool
		HyperbeamIdentifier string
		JoinedAt            string
		LeftAt              sql.NullString
	}

	NewWatchPartySession struct {
		Session        ChatWatchPartySessionRow
		RoomName       string
		RoomSystemKind string
		AuditDetails   string
	}

	ChatWatchPartyDAO interface {
		CreateSession(ctx context.Context, row ChatWatchPartySessionRow, tx ...*sql.Tx) (*ChatWatchPartySessionRow, error)
		GetByID(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (*ChatWatchPartySessionRow, error)
		ListActiveByRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatWatchPartySessionRow, error)
		EndSession(ctx context.Context, sessionID uuid.UUID, reason string, tx ...*sql.Tx) error
		SetControllerID(ctx context.Context, sessionID, controllerID uuid.UUID, tx ...*sql.Tx) error

		UpsertParticipant(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, identifier string, tx ...*sql.Tx) error
		SetParticipantIdentifier(ctx context.Context, sessionID, userID uuid.UUID, identifier string, tx ...*sql.Tx) error
		MarkParticipantLeft(ctx context.Context, sessionID, userID uuid.UUID, tx ...*sql.Tx) error
		MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) error
		GetActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) ([]ChatWatchPartyParticipantRow, error)
		GetParticipant(ctx context.Context, sessionID, userID uuid.UUID, tx ...*sql.Tx) (*ChatWatchPartyParticipantRow, error)
		SetParticipantControl(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, tx ...*sql.Tx) error
		CountActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (int, error)

		ListIdleActiveSessions(ctx context.Context, idleBefore string, tx ...*sql.Tx) ([]ChatWatchPartySessionRow, error)
	}

	ChatWatchPartyRepository interface {
		ChatWatchPartyDAO

		StartSession(ctx context.Context, spec NewWatchPartySession, tx ...*sql.Tx) (*ChatWatchPartySessionRow, error)
		RemoveParticipant(ctx context.Context, sessionID, userID uuid.UUID, tx ...*sql.Tx) error
		TransferControl(ctx context.Context, sessionID uuid.UUID, demoteIDs []uuid.UUID, targetID uuid.UUID, tx ...*sql.Tx) error
	}
)

type chatWatchPartyRepository struct {
	db    *sql.DB
	dao   ChatWatchPartyDAO
	chat  ChatRepository
	audit AuditLogRepository
}

func NewChatWatchPartyRepo(database *sql.DB, dao ChatWatchPartyDAO, chat ChatRepository, audit AuditLogRepository) ChatWatchPartyRepository {
	return &chatWatchPartyRepository{db: database, dao: dao, chat: chat, audit: audit}
}

func (r *chatWatchPartyRepository) StartSession(ctx context.Context, spec NewWatchPartySession, tx ...*sql.Tx) (*ChatWatchPartySessionRow, error) {
	var created *ChatWatchPartySessionRow

	err := db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateSession(ctx, spec.Session, tx)
		if err != nil {
			return err
		}

		host := spec.Session.StartedBy

		if _, err := r.chat.CreateSystemRoomWithHost(ctx, NewChatSystemRoom{
			ID:         created.ID,
			Name:       spec.RoomName,
			SystemKind: spec.RoomSystemKind,
			CreatedBy:  host,
		}, tx); err != nil {
			return fmt.Errorf("create watch party chat room: %w", err)
		}

		if err := r.dao.UpsertParticipant(ctx, created.ID, host, true, "", tx); err != nil {
			return err
		}

		return r.audit.Create(ctx, NewAuditEntry{
			ActorID:    host,
			Action:     "watch_party.start",
			TargetType: "chat_watch_party_session",
			TargetID:   created.ID.String(),
			Details:    spec.AuditDetails,
		}, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatWatchPartyRepository) RemoveParticipant(ctx context.Context, sessionID, userID uuid.UUID, tx ...*sql.Tx) error {
	return db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.MarkParticipantLeft(ctx, sessionID, userID, tx); err != nil {
			return err
		}

		return r.chat.RemoveMember(ctx, sessionID, userID, tx)
	})
}

func (r *chatWatchPartyRepository) TransferControl(ctx context.Context, sessionID uuid.UUID, demoteIDs []uuid.UUID, targetID uuid.UUID, tx ...*sql.Tx) error {
	return db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		for i := range demoteIDs {
			if err := r.dao.SetParticipantControl(ctx, sessionID, demoteIDs[i], false, tx); err != nil {
				return err
			}
		}

		if err := r.dao.SetParticipantControl(ctx, sessionID, targetID, true, tx); err != nil {
			return err
		}

		return r.dao.SetControllerID(ctx, sessionID, targetID, tx)
	})
}

func (r *chatWatchPartyRepository) CreateSession(ctx context.Context, row ChatWatchPartySessionRow, tx ...*sql.Tx) (*ChatWatchPartySessionRow, error) {
	return r.dao.CreateSession(ctx, row, tx...)
}

func (r *chatWatchPartyRepository) GetByID(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (*ChatWatchPartySessionRow, error) {
	return r.dao.GetByID(ctx, sessionID, tx...)
}

func (r *chatWatchPartyRepository) ListActiveByRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatWatchPartySessionRow, error) {
	return r.dao.ListActiveByRoom(ctx, roomID, tx...)
}

func (r *chatWatchPartyRepository) EndSession(ctx context.Context, sessionID uuid.UUID, reason string, tx ...*sql.Tx) error {
	return r.dao.EndSession(ctx, sessionID, reason, tx...)
}

func (r *chatWatchPartyRepository) SetControllerID(ctx context.Context, sessionID, controllerID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetControllerID(ctx, sessionID, controllerID, tx...)
}

func (r *chatWatchPartyRepository) UpsertParticipant(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, identifier string, tx ...*sql.Tx) error {
	return r.dao.UpsertParticipant(ctx, sessionID, userID, hasControl, identifier, tx...)
}

func (r *chatWatchPartyRepository) SetParticipantIdentifier(ctx context.Context, sessionID, userID uuid.UUID, identifier string, tx ...*sql.Tx) error {
	return r.dao.SetParticipantIdentifier(ctx, sessionID, userID, identifier, tx...)
}

func (r *chatWatchPartyRepository) MarkParticipantLeft(ctx context.Context, sessionID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkParticipantLeft(ctx, sessionID, userID, tx...)
}

func (r *chatWatchPartyRepository) MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkAllParticipantsLeft(ctx, sessionID, tx...)
}

func (r *chatWatchPartyRepository) GetActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) ([]ChatWatchPartyParticipantRow, error) {
	return r.dao.GetActiveParticipants(ctx, sessionID, tx...)
}

func (r *chatWatchPartyRepository) GetParticipant(ctx context.Context, sessionID, userID uuid.UUID, tx ...*sql.Tx) (*ChatWatchPartyParticipantRow, error) {
	return r.dao.GetParticipant(ctx, sessionID, userID, tx...)
}

func (r *chatWatchPartyRepository) SetParticipantControl(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, tx ...*sql.Tx) error {
	return r.dao.SetParticipantControl(ctx, sessionID, userID, hasControl, tx...)
}

func (r *chatWatchPartyRepository) CountActiveParticipants(ctx context.Context, sessionID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountActiveParticipants(ctx, sessionID, tx...)
}

func (r *chatWatchPartyRepository) ListIdleActiveSessions(ctx context.Context, idleBefore string, tx ...*sql.Tx) ([]ChatWatchPartySessionRow, error) {
	return r.dao.ListIdleActiveSessions(ctx, idleBefore, tx...)
}
