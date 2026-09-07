package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ChatWatchPartyRepository interface {
		dao.ChatWatchPartyDAO

		StartSession(ctx context.Context, party spec.NewWatchPartySession, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error)
		RemoveParticipant(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) error
		TransferControl(ctx context.Context, s spec.WatchPartyControlTransfer, tx ...*sql.Tx) error
	}
)

type chatWatchPartyRepository struct {
	db *sql.DB
	dao.ChatWatchPartyDAO
	chat  ChatRepository
	audit AuditLogRepository
}

func NewChatWatchPartyRepo(database *sql.DB, partyDAO dao.ChatWatchPartyDAO, chat ChatRepository, auditLog AuditLogRepository) ChatWatchPartyRepository {
	return &chatWatchPartyRepository{db: database, ChatWatchPartyDAO: partyDAO, chat: chat, audit: auditLog}
}

func (r *chatWatchPartyRepository) StartSession(ctx context.Context, party spec.NewWatchPartySession, tx ...*sql.Tx) (*model.ChatWatchPartySessionRow, error) {
	var created *model.ChatWatchPartySessionRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.ChatWatchPartyDAO.CreateSession(ctx, party.Session, tx)
		if err != nil {
			return err
		}

		host := party.Session.StartedBy

		if _, err := r.chat.CreateSystemRoomWithHost(ctx, spec.NewChatSystemRoom{
			ID:         created.ID,
			Name:       party.RoomName,
			SystemKind: party.RoomSystemKind,
			CreatedBy:  host,
		}, tx); err != nil {
			return fmt.Errorf("create watch party chat room: %w", err)
		}

		if err := r.ChatWatchPartyDAO.UpsertParticipant(ctx, spec.WatchPartyParticipantUpsert{
			SessionID:  created.ID,
			UserID:     host,
			HasControl: true,
		}, tx); err != nil {
			return err
		}

		return r.audit.Create(ctx, audit.NewEntry{
			ActorID:    host,
			Action:     audit.ActionWatchPartyStart,
			TargetType: audit.TargetChatWatchPartySession,
			TargetID:   created.ID.String(),
			Details:    party.AuditDetails,
		}, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatWatchPartyRepository) RemoveParticipant(ctx context.Context, s spec.WatchPartyParticipantRef, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ChatWatchPartyDAO.MarkParticipantLeft(ctx, s, tx); err != nil {
			return err
		}

		return r.chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: s.SessionID, UserID: s.UserID}, tx)
	})
}

func (r *chatWatchPartyRepository) TransferControl(ctx context.Context, s spec.WatchPartyControlTransfer, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		for _, demoteID := range s.DemoteIDs {
			if err := r.ChatWatchPartyDAO.SetParticipantControl(ctx, spec.WatchPartyControlUpdate{
				SessionID:  s.SessionID,
				UserID:     demoteID,
				HasControl: false,
			}, tx); err != nil {
				return err
			}
		}

		if err := r.ChatWatchPartyDAO.SetParticipantControl(ctx, spec.WatchPartyControlUpdate{
			SessionID:  s.SessionID,
			UserID:     s.TargetID,
			HasControl: true,
		}, tx); err != nil {
			return err
		}

		return r.ChatWatchPartyDAO.SetControllerID(ctx, spec.WatchPartyControllerUpdate{
			SessionID:    s.SessionID,
			ControllerID: s.TargetID,
		}, tx)
	})
}
