package spec

import (
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
)

type (
	NewChatRoom struct {
		Name        string
		Description string
		Type        string
		IsPublic    bool
		IsRP        bool
		CreatedBy   uuid.UUID
	}

	NewChatSystemRoom struct {
		ID          uuid.UUID
		Name        string
		Description string
		SystemKind  string
		CreatedBy   uuid.UUID
	}

	NewChatGroupRoom struct {
		Name        string
		Description string
		IsPublic    bool
		IsRP        bool
		CreatedBy   uuid.UUID
		Tags        []string
		MemberIDs   []uuid.UUID
	}

	UpdateChatRoom struct {
		RoomID      uuid.UUID
		Name        string
		Description string
		Tags        []string
		IsPublic    bool
		IsRP        bool
	}

	NewChatRoomMember struct {
		RoomID uuid.UUID
		UserID uuid.UUID
		Role   string
		Ghost  bool
	}

	NewChatMessage struct {
		RoomID    uuid.UUID
		SenderID  uuid.UUID
		Body      string
		ReplyToID *uuid.UUID
		IsSystem  bool
	}

	NewChatMessageMedia struct {
		NewMedia
		Width  int
		Height int
	}

	SystemRoomMembership struct {
		RoomID         uuid.UUID
		UserID         uuid.UUID
		ShouldBeMember bool
		DesiredRole    string
	}

	ChatMemberRef struct {
		RoomID uuid.UUID
		UserID uuid.UUID
	}

	ChatRoomViewer struct {
		RoomID   uuid.UUID
		ViewerID uuid.UUID
	}

	ChatDMPair struct {
		UserA uuid.UUID
		UserB uuid.UUID
	}

	ChatDMMembers struct {
		RoomID uuid.UUID
		UserA  uuid.UUID
		UserB  uuid.UUID
	}

	ChatMemberRoleUpdate struct {
		RoomID uuid.UUID
		UserID uuid.UUID
		Role   string
	}

	ChatMemberMuteUpdate struct {
		RoomID uuid.UUID
		UserID uuid.UUID
		Muted  bool
	}

	ChatVoiceForceMuteUpdate struct {
		RoomID  uuid.UUID
		UserID  uuid.UUID
		MutedBy uuid.UUID
		Muted   bool
	}

	ChatMemberNicknameUpdate struct {
		RoomID   uuid.UUID
		UserID   uuid.UUID
		Nickname string
		Locked   bool
	}

	ChatMemberAvatarUpdate struct {
		RoomID    uuid.UUID
		UserID    uuid.UUID
		AvatarURL string
	}

	ChatMemberTimeout struct {
		RoomID  uuid.UUID
		UserID  uuid.UUID
		Until   string
		ByStaff bool
	}

	ChatRoomTags struct {
		RoomID uuid.UUID
		Tags   []string
	}

	ChatRoomActivityTouch struct {
		RoomID   uuid.UUID
		IsSystem bool
	}

	ChatRoomFilter struct {
		Search          string
		IsRPOnly        bool
		Tag             string
		IncludeArchived bool
		Limit           int
		Offset          int
	}

	ChatUserRoomFilter struct {
		ChatRoomFilter
		UserID uuid.UUID
		Role   string
	}

	ChatPublicRoomFilter struct {
		ChatRoomFilter
		ViewerID       uuid.UUID
		ExcludeUserIDs []uuid.UUID
	}

	ChatMessageUpdate struct {
		MessageID uuid.UUID
		Body      string
	}

	ChatMessagePage struct {
		RoomID   uuid.UUID
		ViewerID uuid.UUID
		Limit    int
		Offset   int
	}

	ChatMessageCursorPage struct {
		RoomID   uuid.UUID
		ViewerID uuid.UUID
		Before   string
		Limit    int
	}

	ChatMessageSearch struct {
		ViewerID uuid.UUID
		RoomID   uuid.UUID
		Query    string
		Limit    int
		Offset   int
	}

	ChatRoomAttachmentQuery struct {
		ChatMessageCursorPage
		Kind model.AttachmentKind
	}

	ChatMessagePin struct {
		MessageID uuid.UUID
		PinnedBy  uuid.UUID
	}

	ChatMessageReaction struct {
		MessageID uuid.UUID
		UserID    uuid.UUID
		Emoji     string
	}

	ChatReactionCount struct {
		MessageID uuid.UUID
		Emoji     string
	}

	ChatReactionsQuery struct {
		MessageIDs []uuid.UUID
		ViewerID   uuid.UUID
	}

	ChatMemberJoinAnnouncement struct {
		Member  NewChatRoomMember
		Message NewChatMessage
	}
)
