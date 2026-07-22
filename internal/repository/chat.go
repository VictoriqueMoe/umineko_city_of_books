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

type chatRepository struct {
	dao ChatRepository
}

func NewChatRepo(dao ChatRepository) ChatRepository {
	return &chatRepository{dao: dao}
}

func (r *chatRepository) CreateRoom(ctx context.Context, id uuid.UUID, name, description, roomType string, isPublic, isRP bool, createdBy uuid.UUID) error {
	return r.dao.CreateRoom(ctx, id, name, description, roomType, isPublic, isRP, createdBy)
}

func (r *chatRepository) CreateSystemRoom(ctx context.Context, id uuid.UUID, name, description, systemKind string, createdBy uuid.UUID) error {
	return r.dao.CreateSystemRoom(ctx, id, name, description, systemKind, createdBy)
}

func (r *chatRepository) GetSystemRoomID(ctx context.Context, systemKind string) (uuid.UUID, error) {
	return r.dao.GetSystemRoomID(ctx, systemKind)
}

func (r *chatRepository) CreateDMRoomAtomic(ctx context.Context, id, userA, userB uuid.UUID) (uuid.UUID, error) {
	return r.dao.CreateDMRoomAtomic(ctx, id, userA, userB)
}

func (r *chatRepository) AddMember(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.AddMember(ctx, roomID, userID)
}

func (r *chatRepository) AddMemberWithRole(ctx context.Context, roomID, userID uuid.UUID, role string, ghost bool) error {
	return r.dao.AddMemberWithRole(ctx, roomID, userID, role, ghost)
}

func (r *chatRepository) IsGhostMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.IsGhostMember(ctx, roomID, userID)
}

func (r *chatRepository) HasGhostMembers(ctx context.Context, roomID uuid.UUID) (bool, error) {
	return r.dao.HasGhostMembers(ctx, roomID)
}

func (r *chatRepository) SetMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string) error {
	return r.dao.SetMemberRole(ctx, roomID, userID, role)
}

func (r *chatRepository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.RemoveMember(ctx, roomID, userID)
}

func (r *chatRepository) CountRoomMembers(ctx context.Context, roomID uuid.UUID) (int, error) {
	return r.dao.CountRoomMembers(ctx, roomID)
}

func (r *chatRepository) DeleteRoom(ctx context.Context, roomID uuid.UUID) error {
	return r.dao.DeleteRoom(ctx, roomID)
}

func (r *chatRepository) ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID) ([]string, error) {
	return r.dao.ListRoomMediaURLs(ctx, roomID)
}

func (r *chatRepository) GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]ChatRoomRow, error) {
	return r.dao.GetRoomsByUser(ctx, userID)
}

func (r *chatRepository) ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, includeArchived bool, limit, offset int) ([]ChatRoomRow, int, error) {
	return r.dao.ListUserGroupRooms(ctx, userID, search, isRPOnly, tag, role, includeArchived, limit, offset)
}

func (r *chatRepository) GetRoomByID(ctx context.Context, roomID, viewerID uuid.UUID) (*ChatRoomRow, error) {
	return r.dao.GetRoomByID(ctx, roomID, viewerID)
}

func (r *chatRepository) GetRoomSendContext(ctx context.Context, roomID uuid.UUID) (*ChatRoomSendContext, error) {
	return r.dao.GetRoomSendContext(ctx, roomID)
}

func (r *chatRepository) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.GetRoomMembers(ctx, roomID)
}

func (r *chatRepository) GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID) ([]ChatRoomMemberRow, error) {
	return r.dao.GetRoomMembersDetailed(ctx, roomID)
}

func (r *chatRepository) GetMemberRole(ctx context.Context, roomID, userID uuid.UUID) (string, error) {
	return r.dao.GetMemberRole(ctx, roomID, userID)
}

func (r *chatRepository) GetMemberNickname(ctx context.Context, roomID, userID uuid.UUID) (string, error) {
	return r.dao.GetMemberNickname(ctx, roomID, userID)
}

func (r *chatRepository) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.IsMember(ctx, roomID, userID)
}

func (r *chatRepository) SetMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error {
	return r.dao.SetMuted(ctx, roomID, userID, muted)
}

func (r *chatRepository) IsMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.IsMuted(ctx, roomID, userID)
}

func (r *chatRepository) GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.GetRoomMembersUnmuted(ctx, roomID)
}

func (r *chatRepository) ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, includeArchived bool, limit, offset int) ([]ChatRoomRow, int, error) {
	return r.dao.ListPublicRooms(ctx, search, isRPOnly, tag, viewerID, excludeUserIDs, includeArchived, limit, offset)
}

func (r *chatRepository) FindDMRoom(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, error) {
	return r.dao.FindDMRoom(ctx, userA, userB)
}

func (r *chatRepository) AddRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error {
	return r.dao.AddRoomTags(ctx, roomID, tags)
}

func (r *chatRepository) ReplaceRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error {
	return r.dao.ReplaceRoomTags(ctx, roomID, tags)
}

func (r *chatRepository) GetRoomTags(ctx context.Context, roomID uuid.UUID) ([]string, error) {
	return r.dao.GetRoomTags(ctx, roomID)
}

func (r *chatRepository) GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	return r.dao.GetRoomTagsBatch(ctx, roomIDs)
}

func (r *chatRepository) InsertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string, replyToID *uuid.UUID) error {
	return r.dao.InsertMessage(ctx, id, roomID, senderID, body, replyToID)
}

func (r *chatRepository) InsertSystemMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string) error {
	return r.dao.InsertSystemMessage(ctx, id, roomID, senderID, body)
}

func (r *chatRepository) EditMessage(ctx context.Context, messageID uuid.UUID, body string) error {
	return r.dao.EditMessage(ctx, messageID, body)
}

func (r *chatRepository) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]ChatMessageRow, int, error) {
	return r.dao.GetMessages(ctx, roomID, limit, offset)
}

func (r *chatRepository) SearchMessagesForViewer(ctx context.Context, viewerID, roomID uuid.UUID, query string, limit, offset int) ([]SearchResult, int, error) {
	return r.dao.SearchMessagesForViewer(ctx, viewerID, roomID, query, limit, offset)
}

func (r *chatRepository) GetMessagesBefore(ctx context.Context, roomID uuid.UUID, before string, limit int) ([]ChatMessageRow, error) {
	return r.dao.GetMessagesBefore(ctx, roomID, before, limit)
}

func (r *chatRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessageRow, error) {
	return r.dao.GetMessageByID(ctx, messageID)
}

func (r *chatRepository) DeleteMessages(ctx context.Context, roomID uuid.UUID) error {
	return r.dao.DeleteMessages(ctx, roomID)
}

func (r *chatRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	return r.dao.DeleteMessage(ctx, messageID)
}

func (r *chatRepository) GetMessageSenderID(ctx context.Context, messageID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetMessageSenderID(ctx, messageID)
}

func (r *chatRepository) GetMessageRoomID(ctx context.Context, messageID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetMessageRoomID(ctx, messageID)
}

func (r *chatRepository) AddMessageMedia(ctx context.Context, messageID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddMessageMedia(ctx, messageID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *chatRepository) UpdateMessageMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateMessageMediaURL(ctx, id, mediaURL)
}

func (r *chatRepository) UpdateMessageMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateMessageMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *chatRepository) GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]dto.PostMediaResponse, error) {
	return r.dao.GetMessageMediaBatch(ctx, messageIDs)
}

func (r *chatRepository) TouchRoomActivity(ctx context.Context, roomID uuid.UUID) error {
	return r.dao.TouchRoomActivity(ctx, roomID)
}

func (r *chatRepository) ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	return r.dao.ArchiveStaleGroupRooms(ctx, cutoff)
}

func (r *chatRepository) MarkRoomRead(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.MarkRoomRead(ctx, roomID, userID)
}

func (r *chatRepository) CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.CountUnreadRoomsForUser(ctx, userID)
}

func (r *chatRepository) SetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) error {
	return r.dao.SetMemberNickname(ctx, roomID, userID, nickname)
}

func (r *chatRepository) SetMemberNicknameWithLock(ctx context.Context, roomID, userID uuid.UUID, nickname string, locked bool) error {
	return r.dao.SetMemberNicknameWithLock(ctx, roomID, userID, nickname, locked)
}

func (r *chatRepository) IsMemberNicknameLocked(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.IsMemberNicknameLocked(ctx, roomID, userID)
}

func (r *chatRepository) SetMemberAvatar(ctx context.Context, roomID, userID uuid.UUID, avatarURL string) error {
	return r.dao.SetMemberAvatar(ctx, roomID, userID, avatarURL)
}

func (r *chatRepository) SetMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, until string, byStaff bool) error {
	return r.dao.SetMemberTimeout(ctx, roomID, userID, until, byStaff)
}

func (r *chatRepository) ClearMemberTimeout(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.ClearMemberTimeout(ctx, roomID, userID)
}

func (r *chatRepository) GetMemberTimeoutState(ctx context.Context, roomID, userID uuid.UUID) (bool, string, bool, error) {
	return r.dao.GetMemberTimeoutState(ctx, roomID, userID)
}

func (r *chatRepository) HasActiveMemberTimeout(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.HasActiveMemberTimeout(ctx, roomID, userID)
}

func (r *chatRepository) PinMessage(ctx context.Context, messageID, pinnedBy uuid.UUID) error {
	return r.dao.PinMessage(ctx, messageID, pinnedBy)
}

func (r *chatRepository) UnpinMessage(ctx context.Context, messageID uuid.UUID) error {
	return r.dao.UnpinMessage(ctx, messageID)
}

func (r *chatRepository) ListPinnedMessages(ctx context.Context, roomID uuid.UUID) ([]ChatMessageRow, error) {
	return r.dao.ListPinnedMessages(ctx, roomID)
}

func (r *chatRepository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	return r.dao.AddReaction(ctx, messageID, userID, emoji)
}

func (r *chatRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	return r.dao.RemoveReaction(ctx, messageID, userID, emoji)
}

func (r *chatRepository) CountReactions(ctx context.Context, messageID uuid.UUID, emoji string) (int, error) {
	return r.dao.CountReactions(ctx, messageID, emoji)
}

func (r *chatRepository) GetReactionsBatch(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]ReactionGroup, error) {
	return r.dao.GetReactionsBatch(ctx, messageIDs, viewerID)
}
