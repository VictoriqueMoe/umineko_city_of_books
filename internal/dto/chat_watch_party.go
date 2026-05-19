package dto

import "github.com/google/uuid"

type (
	WatchPartySession struct {
		ID           uuid.UUID                `json:"id"`
		RoomID       uuid.UUID                `json:"room_id"`
		StartedBy    uuid.UUID                `json:"started_by"`
		ControllerID uuid.UUID                `json:"controller_id"`
		Title        string                   `json:"title"`
		StartURL     string                   `json:"start_url,omitempty"`
		Region       string                   `json:"region,omitempty"`
		Status       string                   `json:"status"`
		StartedAt    string                   `json:"started_at"`
		EndedAt      string                   `json:"ended_at,omitempty"`
		Participants []WatchPartyParticipant  `json:"participants"`
		Viewer       *WatchPartyViewerContext `json:"viewer,omitempty"`
	}

	WatchPartyParticipant struct {
		User       UserResponse `json:"user"`
		HasControl bool         `json:"has_control"`
		JoinedAt   string       `json:"joined_at"`
	}

	WatchPartyViewerContext struct {
		IsParticipant bool   `json:"is_participant"`
		HasControl    bool   `json:"has_control"`
		EmbedURL      string `json:"embed_url,omitempty"`
	}

	WatchPartyMessage struct {
		ID        uuid.UUID    `json:"id"`
		SessionID uuid.UUID    `json:"session_id"`
		Sender    UserResponse `json:"sender"`
		Body      string       `json:"body"`
		CreatedAt string       `json:"created_at"`
	}

	StartWatchPartyRequest struct {
		StartURL string `json:"start_url,omitempty"`
		Region   string `json:"region,omitempty"`
		Title    string `json:"title,omitempty"`
	}

	StartWatchPartyResponse struct {
		Session  WatchPartySession `json:"session"`
		EmbedURL string            `json:"embed_url"`
	}

	JoinWatchPartyResponse struct {
		Session  WatchPartySession `json:"session"`
		EmbedURL string            `json:"embed_url"`
	}

	GrantWatchPartyControlRequest struct{}

	IdentifyWatchPartyParticipantRequest struct {
		Identifier string `json:"identifier"`
	}

	SendWatchPartyMessageRequest struct {
		Body string `json:"body"`
	}

	WatchPartyListResponse struct {
		Sessions []WatchPartySession `json:"sessions"`
		Enabled  bool                `json:"enabled"`
	}

	WatchPartyMessagesResponse struct {
		Messages []WatchPartyMessage `json:"messages"`
	}

	WatchPartyStartedEvent struct {
		Session WatchPartySession `json:"session"`
	}

	WatchPartyEndedEvent struct {
		SessionID uuid.UUID `json:"session_id"`
		RoomID    uuid.UUID `json:"room_id"`
		Reason    string    `json:"reason"`
	}

	WatchPartyParticipantEvent struct {
		SessionID   uuid.UUID             `json:"session_id"`
		RoomID      uuid.UUID             `json:"room_id"`
		Participant WatchPartyParticipant `json:"participant"`
	}

	WatchPartyParticipantLeftEvent struct {
		SessionID uuid.UUID `json:"session_id"`
		RoomID    uuid.UUID `json:"room_id"`
		UserID    uuid.UUID `json:"user_id"`
	}

	WatchPartyControlChangedEvent struct {
		SessionID  uuid.UUID `json:"session_id"`
		RoomID     uuid.UUID `json:"room_id"`
		UserID     uuid.UUID `json:"user_id"`
		HasControl bool      `json:"has_control"`
	}

	WatchPartyMessageEvent struct {
		SessionID uuid.UUID         `json:"session_id"`
		RoomID    uuid.UUID         `json:"room_id"`
		Message   WatchPartyMessage `json:"message"`
	}
)
