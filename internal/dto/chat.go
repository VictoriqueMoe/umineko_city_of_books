package dto

import "github.com/google/uuid"

type (
	CreateGroupRoomRequest struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		IsPublic    bool        `json:"is_public"`
		IsRP        bool        `json:"is_rp"`
		Tags        []string    `json:"tags"`
		MemberIDs   []uuid.UUID `json:"member_ids"`
	}

	InviteMembersRequest struct {
		UserIDs []uuid.UUID `json:"user_ids"`
	}

	JoinRoomRequest struct {
		Ghost bool `json:"ghost,omitempty"`
	}

	InviteMembersResponse struct {
		InvitedCount int `json:"invited_count"`
		SkippedCount int `json:"skipped_count"`
	}

	SendMessageRequest struct {
		Body      string     `json:"body"`
		ReplyToID *uuid.UUID `json:"reply_to_id,omitempty"`
	}

	ChatMessageReplyPreview struct {
		ID          uuid.UUID `json:"id"`
		SenderID    uuid.UUID `json:"sender_id"`
		SenderName  string    `json:"sender_name"`
		BodyPreview string    `json:"body_preview"`
	}

	ResolveDMResponse struct {
		Room      *ChatRoomResponse `json:"room"`
		Recipient UserResponse      `json:"recipient"`
	}

	SendDMResponse struct {
		Room    ChatRoomResponse    `json:"room"`
		Message ChatMessageResponse `json:"message"`
	}

	ChatRoomResponse struct {
		ID            uuid.UUID      `json:"id"`
		Name          string         `json:"name"`
		Description   string         `json:"description"`
		Type          string         `json:"type"`
		IsPublic      bool           `json:"is_public"`
		IsRP          bool           `json:"is_rp"`
		IsSystem      bool           `json:"is_system"`
		SystemKind    string         `json:"system_kind,omitempty"`
		Tags          []string       `json:"tags"`
		ViewerRole    string         `json:"viewer_role,omitempty"`
		ViewerMuted   bool           `json:"viewer_muted"`
		ViewerGhost   bool           `json:"viewer_ghost"`
		IsMember      bool           `json:"is_member"`
		MemberCount   int            `json:"member_count"`
		Members       []UserResponse `json:"members"`
		CreatedAt     string         `json:"created_at"`
		LastMessageAt string         `json:"last_message_at,omitempty"`
		Unread        bool           `json:"unread"`
	}

	ChatRoomMemberResponse struct {
		User            UserResponse `json:"user"`
		Role            string       `json:"role"`
		JoinedAt        string       `json:"joined_at"`
		Nickname        string       `json:"nickname"`
		NicknameLocked  bool         `json:"nickname_locked"`
		MemberAvatarURL string       `json:"member_avatar_url"`
		TimeoutUntil    string       `json:"timeout_until,omitempty"`
		TimeoutByStaff  bool         `json:"timeout_set_by_staff"`
		Presence        string       `json:"presence,omitempty"`
		Ghost           bool         `json:"ghost"`
	}

	ChatMessageResponse struct {
		ID                    uuid.UUID                `json:"id"`
		RoomID                uuid.UUID                `json:"room_id"`
		Sender                UserResponse             `json:"sender"`
		SenderNickname        string                   `json:"sender_nickname,omitempty"`
		SenderMemberAvatarURL string                   `json:"sender_member_avatar_url,omitempty"`
		Body                  string                   `json:"body"`
		IsSystem              bool                     `json:"is_system"`
		CreatedAt             string                   `json:"created_at"`
		Media                 []PostMediaResponse      `json:"media,omitempty"`
		ReplyTo               *ChatMessageReplyPreview `json:"reply_to,omitempty"`
		Pinned                bool                     `json:"pinned"`
		PinnedAt              *string                  `json:"pinned_at,omitempty"`
		PinnedBy              *uuid.UUID               `json:"pinned_by,omitempty"`
		EditedAt              *string                  `json:"edited_at,omitempty"`
		Reactions             []ReactionGroup          `json:"reactions"`
	}

	EditMessageRequest struct {
		Body string `json:"body"`
	}

	ReactionGroup struct {
		Emoji         string   `json:"emoji"`
		Count         int      `json:"count"`
		ViewerReacted bool     `json:"viewer_reacted"`
		DisplayNames  []string `json:"display_names"`
	}

	UpdateMemberProfileRequest struct {
		Nickname string `json:"nickname"`
	}

	SetMemberTimeoutRequest struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	}

	AddReactionRequest struct {
		Emoji string `json:"emoji"`
	}

	ChatRoomListResponse struct {
		Rooms []ChatRoomResponse `json:"rooms"`
		Total int                `json:"total"`
	}

	ChatMessageListResponse struct {
		Messages []ChatMessageResponse `json:"messages"`
		Total    int                   `json:"total"`
		Limit    int                   `json:"limit"`
		Offset   int                   `json:"offset"`
	}
)
