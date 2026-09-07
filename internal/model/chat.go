package model

import (
	"database/sql"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

const (
	AttachmentKindMedia AttachmentKind = "media"
	AttachmentKindLinks AttachmentKind = "links"
)

type (
	AttachmentKind string

	ChatRoomRow struct {
		ID            uuid.UUID
		Name          string
		Description   string
		Type          dto.RoomType
		IsPublic      bool
		IsRP          bool
		IsSystem      bool
		SystemKind    string
		CreatedBy     uuid.UUID
		CreatedAt     string
		LastMessageAt sql.NullString
		LastReadAt    sql.NullString
		ArchivedAt    sql.NullString
		MemberCount   int
		HotScore      int
		ViewerRole    string
		ViewerMuted   bool
		ViewerGhost   bool
		IsMember      bool
		Tags          []string
	}

	ChatRoomSendContext struct {
		ID            uuid.UUID
		Name          string
		Type          dto.RoomType
		IsPublic      bool
		IsSystem      bool
		SystemKind    string
		CreatedBy     uuid.UUID
		LastMessageAt sql.NullString
	}

	ChatRoomMemberRow struct {
		UserID          uuid.UUID
		Username        string
		DisplayName     string
		AvatarURL       string
		Role            string
		AuthorRole      string
		AuthorRoleTyped role.Role
		JoinedAt        string
		Nickname        string
		NicknameLocked  bool
		MemberAvatarURL string
		TimeoutUntil    string
		TimeoutByStaff  bool
		Ghost           bool
	}

	ChatMessageRow struct {
		ID                 uuid.UUID
		RoomID             uuid.UUID
		SenderID           uuid.UUID
		SenderUsername     string
		SenderDisplayName  string
		SenderAvatarURL    string
		SenderRole         string
		SenderRoleTyped    role.Role
		Body               string
		IsSystem           bool
		CreatedAt          string
		ReplyToID          *uuid.UUID
		ReplyToSenderID    *uuid.UUID
		ReplyToSenderName  *string
		ReplyToBody        *string
		PinnedAt           *string
		PinnedBy           *uuid.UUID
		EditedAt           *string
		SenderNickname     string
		SenderMemberAvatar string
	}

	ReactionGroup struct {
		Emoji         string
		Count         int
		ViewerReacted bool
		DisplayNames  []string
	}

	SystemRoomMembershipChange struct {
		RoomID uuid.UUID
		Joined bool
		Left   bool
	}
)

func (r *ChatRoomRow) PubliclyVisible() bool {
	if r == nil {
		return false
	}

	return dto.PubliclyVisibleRoom(r.Type, r.IsPublic, r.IsSystem)
}

func (c *ChatRoomSendContext) PubliclyVisible() bool {
	if c == nil {
		return false
	}

	return dto.PubliclyVisibleRoom(c.Type, c.IsPublic, c.IsSystem)
}
