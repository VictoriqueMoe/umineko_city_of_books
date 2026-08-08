package repository

import (
	"context"
	"database/sql"

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

	ChatWatchPartyRepository interface {
		CreateSession(ctx context.Context, row ChatWatchPartySessionRow) (uuid.UUID, error)
		GetByID(ctx context.Context, sessionID uuid.UUID) (*ChatWatchPartySessionRow, error)
		ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]ChatWatchPartySessionRow, error)
		EndSession(ctx context.Context, sessionID uuid.UUID, reason string) error
		SetControllerID(ctx context.Context, sessionID, controllerID uuid.UUID) error

		UpsertParticipant(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, identifier string) error
		SetParticipantIdentifier(ctx context.Context, sessionID, userID uuid.UUID, identifier string) error
		MarkParticipantLeft(ctx context.Context, sessionID, userID uuid.UUID) error
		MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID) error
		GetActiveParticipants(ctx context.Context, sessionID uuid.UUID) ([]ChatWatchPartyParticipantRow, error)
		GetParticipant(ctx context.Context, sessionID, userID uuid.UUID) (*ChatWatchPartyParticipantRow, error)
		SetParticipantControl(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool) error
		CountActiveParticipants(ctx context.Context, sessionID uuid.UUID) (int, error)

		ListIdleActiveSessions(ctx context.Context, idleBefore string) ([]ChatWatchPartySessionRow, error)
	}
)

type chatWatchPartyRepository struct {
	dao ChatWatchPartyRepository
}

func NewChatWatchPartyRepo(dao ChatWatchPartyRepository) ChatWatchPartyRepository {
	return &chatWatchPartyRepository{dao: dao}
}

func (r *chatWatchPartyRepository) CreateSession(ctx context.Context, row ChatWatchPartySessionRow) (uuid.UUID, error) {
	return r.dao.CreateSession(ctx, row)
}

func (r *chatWatchPartyRepository) GetByID(ctx context.Context, sessionID uuid.UUID) (*ChatWatchPartySessionRow, error) {
	return r.dao.GetByID(ctx, sessionID)
}

func (r *chatWatchPartyRepository) ListActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]ChatWatchPartySessionRow, error) {
	return r.dao.ListActiveByRoom(ctx, roomID)
}

func (r *chatWatchPartyRepository) EndSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	return r.dao.EndSession(ctx, sessionID, reason)
}

func (r *chatWatchPartyRepository) SetControllerID(ctx context.Context, sessionID, controllerID uuid.UUID) error {
	return r.dao.SetControllerID(ctx, sessionID, controllerID)
}

func (r *chatWatchPartyRepository) UpsertParticipant(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool, identifier string) error {
	return r.dao.UpsertParticipant(ctx, sessionID, userID, hasControl, identifier)
}

func (r *chatWatchPartyRepository) SetParticipantIdentifier(ctx context.Context, sessionID, userID uuid.UUID, identifier string) error {
	return r.dao.SetParticipantIdentifier(ctx, sessionID, userID, identifier)
}

func (r *chatWatchPartyRepository) MarkParticipantLeft(ctx context.Context, sessionID, userID uuid.UUID) error {
	return r.dao.MarkParticipantLeft(ctx, sessionID, userID)
}

func (r *chatWatchPartyRepository) MarkAllParticipantsLeft(ctx context.Context, sessionID uuid.UUID) error {
	return r.dao.MarkAllParticipantsLeft(ctx, sessionID)
}

func (r *chatWatchPartyRepository) GetActiveParticipants(ctx context.Context, sessionID uuid.UUID) ([]ChatWatchPartyParticipantRow, error) {
	return r.dao.GetActiveParticipants(ctx, sessionID)
}

func (r *chatWatchPartyRepository) GetParticipant(ctx context.Context, sessionID, userID uuid.UUID) (*ChatWatchPartyParticipantRow, error) {
	return r.dao.GetParticipant(ctx, sessionID, userID)
}

func (r *chatWatchPartyRepository) SetParticipantControl(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool) error {
	return r.dao.SetParticipantControl(ctx, sessionID, userID, hasControl)
}

func (r *chatWatchPartyRepository) CountActiveParticipants(ctx context.Context, sessionID uuid.UUID) (int, error) {
	return r.dao.CountActiveParticipants(ctx, sessionID)
}

func (r *chatWatchPartyRepository) ListIdleActiveSessions(ctx context.Context, idleBefore string) ([]ChatWatchPartySessionRow, error) {
	return r.dao.ListIdleActiveSessions(ctx, idleBefore)
}
