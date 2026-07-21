package repository

import (
	"context"
	"database/sql"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	ChatRoomRow struct {
		ID            uuid.UUID
		Name          string
		Description   string
		Type          string
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
		ID         uuid.UUID
		Name       string
		Type       string
		IsSystem   bool
		SystemKind string
		CreatedBy  uuid.UUID
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

	ChatRepository interface {
		CreateRoom(ctx context.Context, id uuid.UUID, name, description, roomType string, isPublic, isRP bool, createdBy uuid.UUID) error
		CreateSystemRoom(ctx context.Context, id uuid.UUID, name, description, systemKind string, createdBy uuid.UUID) error
		GetSystemRoomID(ctx context.Context, systemKind string) (uuid.UUID, error)
		CreateDMRoomAtomic(ctx context.Context, id, userA, userB uuid.UUID) (uuid.UUID, error)
		AddMember(ctx context.Context, roomID, userID uuid.UUID) error
		AddMemberWithRole(ctx context.Context, roomID, userID uuid.UUID, role string, ghost bool) error
		IsGhostMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		HasGhostMembers(ctx context.Context, roomID uuid.UUID) (bool, error)
		SetMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string) error
		RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error
		CountRoomMembers(ctx context.Context, roomID uuid.UUID) (int, error)
		DeleteRoom(ctx context.Context, roomID uuid.UUID) error
		ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID) ([]string, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]ChatRoomRow, error)
		ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, includeArchived bool, limit, offset int) ([]ChatRoomRow, int, error)
		GetRoomByID(ctx context.Context, roomID, viewerID uuid.UUID) (*ChatRoomRow, error)
		GetRoomSendContext(ctx context.Context, roomID uuid.UUID) (*ChatRoomSendContext, error)
		GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
		GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID) ([]ChatRoomMemberRow, error)
		GetMemberRole(ctx context.Context, roomID, userID uuid.UUID) (string, error)
		GetMemberNickname(ctx context.Context, roomID, userID uuid.UUID) (string, error)
		IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		SetMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error
		IsMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
		ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, includeArchived bool, limit, offset int) ([]ChatRoomRow, int, error)
		FindDMRoom(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, error)
		AddRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error
		ReplaceRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error
		GetRoomTags(ctx context.Context, roomID uuid.UUID) ([]string, error)
		GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]string, error)

		InsertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string, replyToID *uuid.UUID) error
		InsertSystemMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string) error
		EditMessage(ctx context.Context, messageID uuid.UUID, body string) error
		GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]ChatMessageRow, int, error)
		SearchMessagesForViewer(ctx context.Context, viewerID, roomID uuid.UUID, query string, limit, offset int) ([]SearchResult, int, error)
		GetMessagesBefore(ctx context.Context, roomID uuid.UUID, before string, limit int) ([]ChatMessageRow, error)
		GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessageRow, error)
		DeleteMessages(ctx context.Context, roomID uuid.UUID) error
		DeleteMessage(ctx context.Context, messageID uuid.UUID) error
		GetMessageSenderID(ctx context.Context, messageID uuid.UUID) (uuid.UUID, error)
		GetMessageRoomID(ctx context.Context, messageID uuid.UUID) (uuid.UUID, error)
		AddMessageMedia(ctx context.Context, messageID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error)
		UpdateMessageMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateMessageMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]dto.PostMediaResponse, error)

		TouchRoomActivity(ctx context.Context, roomID uuid.UUID) error
		ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error)
		MarkRoomRead(ctx context.Context, roomID, userID uuid.UUID) error
		CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID) (int, error)

		SetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) error
		SetMemberNicknameWithLock(ctx context.Context, roomID, userID uuid.UUID, nickname string, locked bool) error
		IsMemberNicknameLocked(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		SetMemberAvatar(ctx context.Context, roomID, userID uuid.UUID, avatarURL string) error
		SetMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, until string, byStaff bool) error
		ClearMemberTimeout(ctx context.Context, roomID, userID uuid.UUID) error
		GetMemberTimeoutState(ctx context.Context, roomID, userID uuid.UUID) (bool, string, bool, error)
		HasActiveMemberTimeout(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		PinMessage(ctx context.Context, messageID, pinnedBy uuid.UUID) error
		UnpinMessage(ctx context.Context, messageID uuid.UUID) error
		ListPinnedMessages(ctx context.Context, roomID uuid.UUID) ([]ChatMessageRow, error)
		AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error)
		RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error)
		CountReactions(ctx context.Context, messageID uuid.UUID, emoji string) (int, error)
		GetReactionsBatch(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]ReactionGroup, error)
	}
)
