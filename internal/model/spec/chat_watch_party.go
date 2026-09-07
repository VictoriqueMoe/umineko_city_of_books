package spec

import (
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
)

type (
	NewWatchPartySession struct {
		Session        model.ChatWatchPartySessionRow
		RoomName       string
		RoomSystemKind string
		AuditDetails   string
	}

	WatchPartySessionEnd struct {
		SessionID uuid.UUID
		Reason    string
	}

	WatchPartyControllerUpdate struct {
		SessionID    uuid.UUID
		ControllerID uuid.UUID
	}

	WatchPartyParticipantRef struct {
		SessionID uuid.UUID
		UserID    uuid.UUID
	}

	WatchPartyParticipantUpsert struct {
		SessionID  uuid.UUID
		UserID     uuid.UUID
		HasControl bool
		Identifier string
	}

	WatchPartyIdentifierUpdate struct {
		SessionID  uuid.UUID
		UserID     uuid.UUID
		Identifier string
	}

	WatchPartyControlUpdate struct {
		SessionID  uuid.UUID
		UserID     uuid.UUID
		HasControl bool
	}

	WatchPartyControlTransfer struct {
		SessionID uuid.UUID
		DemoteIDs []uuid.UUID
		TargetID  uuid.UUID
	}
)
