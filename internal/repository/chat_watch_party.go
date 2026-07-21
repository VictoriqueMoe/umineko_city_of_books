package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/watchparty"

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

	ChatWatchPartyMessageRow struct {
		ID                uuid.UUID
		SessionID         uuid.UUID
		Kind              watchparty.MessageKind
		SenderID          uuid.NullUUID
		SenderUsername    sql.NullString
		SenderDisplayName sql.NullString
		SenderAvatarURL   sql.NullString
		Body              string
		CreatedAt         string
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

		InsertMessage(ctx context.Context, id, sessionID, senderID uuid.UUID, body string) error
		DeleteMessagesForSession(ctx context.Context, sessionID uuid.UUID) error
		InsertSystemMessage(ctx context.Context, id, sessionID uuid.UUID, body string) error
		ListMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]ChatWatchPartyMessageRow, error)
		GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatWatchPartyMessageRow, error)

		ListIdleActiveSessions(ctx context.Context, idleBefore string) ([]ChatWatchPartySessionRow, error)
	}
)
