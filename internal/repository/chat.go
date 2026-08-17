package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
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
		ID         uuid.UUID
		Name       string
		Type       dto.RoomType
		IsPublic   bool
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
		MessageID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
		Width        int
		Height       int
	}

	SystemRoomMembership struct {
		RoomID         uuid.UUID
		UserID         uuid.UUID
		ShouldBeMember bool
		DesiredRole    string
	}

	SystemRoomMembershipChange struct {
		RoomID uuid.UUID
		Joined bool
		Left   bool
	}

	ChatDAO interface {
		CreateRoom(ctx context.Context, spec NewChatRoom, tx ...*sql.Tx) (*ChatRoomRow, error)
		CreateSystemRoom(ctx context.Context, spec NewChatSystemRoom, tx ...*sql.Tx) (*ChatRoomRow, error)
		GetSystemRoomID(ctx context.Context, systemKind string, tx ...*sql.Tx) (uuid.UUID, error)
		FindDMRoomByPair(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error)
		CreateDMRoom(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error)
		AddDMMembers(ctx context.Context, roomID, userA, userB uuid.UUID, tx ...*sql.Tx) error
		RejoinDMMembers(ctx context.Context, roomID, userA, userB uuid.UUID, tx ...*sql.Tx) error
		AddMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		AddMemberWithRole(ctx context.Context, roomID, userID uuid.UUID, role string, ghost bool, tx ...*sql.Tx) error
		IsGhostMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		HasGhostMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (bool, error)
		SetMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string, tx ...*sql.Tx) error
		RemoveMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		CountRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error)
		DeleteRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListMessageMediaURLs(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListRoomMemberAvatarURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]ChatRoomRow, error)
		ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, includeArchived bool, limit, offset int, tx ...*sql.Tx) ([]ChatRoomRow, int, error)
		GetRoomByID(ctx context.Context, roomID, viewerID uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error)
		GetRoomSendContext(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (*ChatRoomSendContext, error)
		GetRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatRoomMemberRow, error)
		GetMemberRole(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		IsMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		SetMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool, tx ...*sql.Tx) error
		IsMuted(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		SetVoiceForceMuted(ctx context.Context, roomID, userID, mutedBy uuid.UUID, muted bool, tx ...*sql.Tx) error
		IsVoiceForceMuted(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		ClearVoiceForceMutes(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, includeArchived bool, limit, offset int, tx ...*sql.Tx) ([]ChatRoomRow, int, error)
		FindDMRoom(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		UpdateRoom(ctx context.Context, spec UpdateChatRoom, tx ...*sql.Tx) error
		AddRoomTags(ctx context.Context, roomID uuid.UUID, tags []string, tx ...*sql.Tx) error
		ReplaceRoomTags(ctx context.Context, roomID uuid.UUID, tags []string, tx ...*sql.Tx) error
		GetRoomTags(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)

		InsertMessageRow(ctx context.Context, spec NewChatMessage, tx ...*sql.Tx) (*ChatMessageRow, error)
		TouchRoomActivityForMessage(ctx context.Context, roomID uuid.UUID, isSystem bool, tx ...*sql.Tx) error
		EditMessage(ctx context.Context, messageID uuid.UUID, body string, tx ...*sql.Tx) error
		GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]ChatMessageRow, int, error)
		GetMessagesForMember(ctx context.Context, roomID, viewerID uuid.UUID, limit int, tx ...*sql.Tx) ([]ChatMessageRow, error)
		GetMessagesForViewer(ctx context.Context, roomID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]ChatMessageRow, int, error)
		SearchMessagesForViewer(ctx context.Context, viewerID, roomID uuid.UUID, query string, limit, offset int, tx ...*sql.Tx) ([]SearchResult, int, error)
		GetMessagesBefore(ctx context.Context, roomID, viewerID uuid.UUID, before string, limit int, tx ...*sql.Tx) ([]ChatMessageRow, error)
		GetMessageByID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (*ChatMessageRow, error)
		DeleteMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		DeleteMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error
		GetMessageSenderID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetMessageRoomID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		AddMessageMedia(ctx context.Context, spec NewChatMessageMedia, tx ...*sql.Tx) (int64, error)
		UpdateMessageMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error
		UpdateMessageMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error
		GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]dto.PostMediaResponse, error)

		TouchRoomActivity(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error)
		MarkRoomRead(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)

		SetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string, tx ...*sql.Tx) error
		SetMemberNicknameWithLock(ctx context.Context, roomID, userID uuid.UUID, nickname string, locked bool, tx ...*sql.Tx) error
		IsMemberNicknameLocked(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		SetMemberAvatar(ctx context.Context, roomID, userID uuid.UUID, avatarURL string, tx ...*sql.Tx) error
		SetMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, until string, byStaff bool, tx ...*sql.Tx) error
		ClearMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error
		GetMemberTimeoutState(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, string, bool, error)
		HasActiveMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		PinMessage(ctx context.Context, messageID, pinnedBy uuid.UUID, tx ...*sql.Tx) error
		UnpinMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error
		ListPinnedMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatMessageRow, error)
		AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string, tx ...*sql.Tx) (bool, error)
		RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string, tx ...*sql.Tx) (bool, error)
		CountReactions(ctx context.Context, messageID uuid.UUID, emoji string, tx ...*sql.Tx) (int, error)
		GetReactionsBatch(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]ReactionGroup, error)
	}

	ChatRepository interface {
		ChatDAO

		CreateDMRoomAtomic(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error)
		CreateGroupRoom(ctx context.Context, spec NewChatGroupRoom, tx ...*sql.Tx) (*ChatRoomRow, error)
		UpdateGroupRoom(ctx context.Context, spec UpdateChatRoom, tx ...*sql.Tx) error
		CreateSystemRoomWithHost(ctx context.Context, spec NewChatSystemRoom, tx ...*sql.Tx) (*ChatRoomRow, error)
		CreateSystemRooms(ctx context.Context, specs []NewChatSystemRoom, tx ...*sql.Tx) error
		SyncSystemRoomMembership(ctx context.Context, targets []SystemRoomMembership, tx ...*sql.Tx) ([]SystemRoomMembershipChange, error)
		AddMemberWithSystemMessage(ctx context.Context, member NewChatRoomMember, message NewChatMessage, tx ...*sql.Tx) (*ChatMessageRow, error)
		DeleteRoomWithMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		DeleteMessageWithMedia(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		InsertMessageAndMarkRead(ctx context.Context, spec NewChatMessage, tx ...*sql.Tx) (*ChatMessageRow, error)
		InsertSystemMessage(ctx context.Context, roomID, senderID uuid.UUID, body string, tx ...*sql.Tx) (*ChatMessageRow, error)
	}
)

type chatRepository struct {
	db    *sql.DB
	dao   ChatDAO
	audit AuditLogRepository
}

func NewChatRepo(database *sql.DB, dao ChatDAO, audit AuditLogRepository) ChatRepository {
	return &chatRepository{db: database, dao: dao, audit: audit}
}

func (r *chatRepository) CreateRoom(ctx context.Context, spec NewChatRoom, tx ...*sql.Tx) (*ChatRoomRow, error) {
	return r.dao.CreateRoom(ctx, spec, tx...)
}

func (r *chatRepository) CreateSystemRoom(ctx context.Context, spec NewChatSystemRoom, tx ...*sql.Tx) (*ChatRoomRow, error) {
	return r.dao.CreateSystemRoom(ctx, spec, tx...)
}

func (r *chatRepository) CreateGroupRoom(ctx context.Context, spec NewChatGroupRoom, tx ...*sql.Tx) (*ChatRoomRow, error) {
	var created *ChatRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateRoom(ctx, NewChatRoom{
			Name:        spec.Name,
			Description: spec.Description,
			Type:        string(dto.RoomTypeGroup),
			IsPublic:    spec.IsPublic,
			IsRP:        spec.IsRP,
			CreatedBy:   spec.CreatedBy,
		}, tx)
		if err != nil {
			return fmt.Errorf("create group room: %w", err)
		}

		if len(spec.Tags) > 0 {
			if err := r.dao.AddRoomTags(ctx, created.ID, spec.Tags, tx); err != nil {
				return fmt.Errorf("add room tags: %w", err)
			}
		}

		if err := r.dao.AddMemberWithRole(ctx, created.ID, spec.CreatedBy, "host", false, tx); err != nil {
			return fmt.Errorf("add creator to group: %w", err)
		}

		for i := range spec.MemberIDs {
			if err := r.dao.AddMemberWithRole(ctx, created.ID, spec.MemberIDs[i], "member", false, tx); err != nil {
				return fmt.Errorf("add member to group: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatRepository) UpdateGroupRoom(ctx context.Context, spec UpdateChatRoom, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.UpdateRoom(ctx, spec, tx); err != nil {
			return fmt.Errorf("update group room: %w", err)
		}

		if err := r.dao.ReplaceRoomTags(ctx, spec.RoomID, spec.Tags, tx); err != nil {
			return fmt.Errorf("replace room tags: %w", err)
		}

		return nil
	})
}

func (r *chatRepository) CreateSystemRoomWithHost(ctx context.Context, spec NewChatSystemRoom, tx ...*sql.Tx) (*ChatRoomRow, error) {
	var created *ChatRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.CreateSystemRoom(ctx, spec, tx)
		if err != nil {
			return err
		}

		return r.dao.AddMemberWithRole(ctx, created.ID, spec.CreatedBy, "host", false, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatRepository) CreateSystemRooms(ctx context.Context, specs []NewChatSystemRoom, tx ...*sql.Tx) error {
	if len(specs) == 0 {
		return nil
	}

	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		for i := range specs {
			if _, err := r.dao.CreateSystemRoom(ctx, specs[i], tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *chatRepository) SyncSystemRoomMembership(ctx context.Context, targets []SystemRoomMembership, tx ...*sql.Tx) ([]SystemRoomMembershipChange, error) {
	var changes []SystemRoomMembershipChange

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		changes = nil

		for i := range targets {
			target := targets[i]
			if target.RoomID == uuid.Nil {
				continue
			}

			currentRole, err := r.dao.GetMemberRole(ctx, target.RoomID, target.UserID, tx)
			if err != nil {
				return fmt.Errorf("get current role: %w", err)
			}

			wasMember := currentRole != ""

			switch {
			case target.ShouldBeMember && !wasMember:
				if err := r.dao.AddMemberWithRole(ctx, target.RoomID, target.UserID, target.DesiredRole, false, tx); err != nil {
					return err
				}

				changes = append(changes, SystemRoomMembershipChange{RoomID: target.RoomID, Joined: true})

			case !target.ShouldBeMember && wasMember:
				if err := r.dao.RemoveMember(ctx, target.RoomID, target.UserID, tx); err != nil {
					return err
				}

				changes = append(changes, SystemRoomMembershipChange{RoomID: target.RoomID, Left: true})

			case target.ShouldBeMember && wasMember && currentRole != target.DesiredRole:
				if err := r.dao.SetMemberRole(ctx, target.RoomID, target.UserID, target.DesiredRole, tx); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return changes, nil
}

func (r *chatRepository) AddMemberWithSystemMessage(ctx context.Context, member NewChatRoomMember, message NewChatMessage, tx ...*sql.Tx) (*ChatMessageRow, error) {
	var created *ChatMessageRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.AddMemberWithRole(ctx, member.RoomID, member.UserID, member.Role, member.Ghost, tx); err != nil {
			return err
		}

		if message.Body == "" {
			return nil
		}

		msg, err := r.dao.InsertMessageRow(ctx, message, tx)
		if err != nil {
			return err
		}

		if err := r.dao.TouchRoomActivityForMessage(ctx, message.RoomID, message.IsSystem, tx); err != nil {
			return err
		}

		created = msg

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatRepository) DeleteRoomWithMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		paths = nil

		mediaURLs, err := r.dao.ListRoomMediaURLs(ctx, roomID, tx)
		if err != nil {
			return fmt.Errorf("list room media urls: %w", err)
		}

		avatarURLs, err := r.dao.ListRoomMemberAvatarURLs(ctx, roomID, tx)
		if err != nil {
			return fmt.Errorf("list room member avatar urls: %w", err)
		}

		paths = append(paths, mediaURLs...)
		paths = append(paths, avatarURLs...)

		if err := r.dao.DeleteMessages(ctx, roomID, tx); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}

		if err := r.dao.DeleteRoom(ctx, roomID, tx); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *chatRepository) DeleteMessageWithMedia(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		paths, err = r.dao.ListMessageMediaURLs(ctx, messageID, tx)
		if err != nil {
			return fmt.Errorf("list message media urls: %w", err)
		}

		if err := r.dao.DeleteMessage(ctx, messageID, tx); err != nil {
			return fmt.Errorf("delete message: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *chatRepository) GetSystemRoomID(ctx context.Context, systemKind string, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetSystemRoomID(ctx, systemKind, tx...)
}

func (r *chatRepository) CreateDMRoomAtomic(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error) {
	var result *ChatRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		existing, err := r.dao.FindDMRoomByPair(ctx, userA, userB, tx)
		if err != nil {
			return err
		}

		if existing != nil {
			if err := r.dao.RejoinDMMembers(ctx, existing.ID, userA, userB, tx); err != nil {
				return err
			}

			result = existing

			return nil
		}

		created, err := r.dao.CreateDMRoom(ctx, userA, userB, tx)
		if err != nil {
			return err
		}

		if err := r.dao.AddDMMembers(ctx, created.ID, userA, userB, tx); err != nil {
			return err
		}

		result = created

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *chatRepository) FindDMRoomByPair(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error) {
	return r.dao.FindDMRoomByPair(ctx, userA, userB, tx...)
}

func (r *chatRepository) CreateDMRoom(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error) {
	return r.dao.CreateDMRoom(ctx, userA, userB, tx...)
}

func (r *chatRepository) AddDMMembers(ctx context.Context, roomID, userA, userB uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.AddDMMembers(ctx, roomID, userA, userB, tx...)
}

func (r *chatRepository) RejoinDMMembers(ctx context.Context, roomID, userA, userB uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.RejoinDMMembers(ctx, roomID, userA, userB, tx...)
}

func (r *chatRepository) AddMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.AddMember(ctx, roomID, userID, tx...)
}

func (r *chatRepository) AddMemberWithRole(ctx context.Context, roomID, userID uuid.UUID, role string, ghost bool, tx ...*sql.Tx) error {
	return r.dao.AddMemberWithRole(ctx, roomID, userID, role, ghost, tx...)
}

func (r *chatRepository) IsGhostMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsGhostMember(ctx, roomID, userID, tx...)
}

func (r *chatRepository) HasGhostMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.HasGhostMembers(ctx, roomID, tx...)
}

func (r *chatRepository) SetMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string, tx ...*sql.Tx) error {
	return r.dao.SetMemberRole(ctx, roomID, userID, role, tx...)
}

func (r *chatRepository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.RemoveMember(ctx, roomID, userID, tx...)
}

func (r *chatRepository) CountRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountRoomMembers(ctx, roomID, tx...)
}

func (r *chatRepository) DeleteRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteRoom(ctx, roomID, tx...)
}

func (r *chatRepository) ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.ListRoomMediaURLs(ctx, roomID, tx...)
}

func (r *chatRepository) ListMessageMediaURLs(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.ListMessageMediaURLs(ctx, messageID, tx...)
}

func (r *chatRepository) ListRoomMemberAvatarURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.ListRoomMemberAvatarURLs(ctx, roomID, tx...)
}

func (r *chatRepository) GetRoomsByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]ChatRoomRow, error) {
	return r.dao.GetRoomsByUser(ctx, userID, tx...)
}

func (r *chatRepository) ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, includeArchived bool, limit, offset int, tx ...*sql.Tx) ([]ChatRoomRow, int, error) {
	return r.dao.ListUserGroupRooms(ctx, userID, search, isRPOnly, tag, role, includeArchived, limit, offset, tx...)
}

func (r *chatRepository) GetRoomByID(ctx context.Context, roomID, viewerID uuid.UUID, tx ...*sql.Tx) (*ChatRoomRow, error) {
	return r.dao.GetRoomByID(ctx, roomID, viewerID, tx...)
}

func (r *chatRepository) GetRoomSendContext(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (*ChatRoomSendContext, error) {
	return r.dao.GetRoomSendContext(ctx, roomID, tx...)
}

func (r *chatRepository) GetRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetRoomMembers(ctx, roomID, tx...)
}

func (r *chatRepository) GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatRoomMemberRow, error) {
	return r.dao.GetRoomMembersDetailed(ctx, roomID, tx...)
}

func (r *chatRepository) GetMemberRole(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetMemberRole(ctx, roomID, userID, tx...)
}

func (r *chatRepository) GetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetMemberNickname(ctx, roomID, userID, tx...)
}

func (r *chatRepository) IsMember(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsMember(ctx, roomID, userID, tx...)
}

func (r *chatRepository) SetMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool, tx ...*sql.Tx) error {
	return r.dao.SetMuted(ctx, roomID, userID, muted, tx...)
}

func (r *chatRepository) IsMuted(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsMuted(ctx, roomID, userID, tx...)
}

func (r *chatRepository) GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetRoomMembersUnmuted(ctx, roomID, tx...)
}

func (r *chatRepository) SetVoiceForceMuted(ctx context.Context, roomID, userID, mutedBy uuid.UUID, muted bool, tx ...*sql.Tx) error {
	return r.dao.SetVoiceForceMuted(ctx, roomID, userID, mutedBy, muted, tx...)
}

func (r *chatRepository) IsVoiceForceMuted(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsVoiceForceMuted(ctx, roomID, userID, tx...)
}

func (r *chatRepository) ClearVoiceForceMutes(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.ClearVoiceForceMutes(ctx, roomID, tx...)
}

func (r *chatRepository) ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, includeArchived bool, limit, offset int, tx ...*sql.Tx) ([]ChatRoomRow, int, error) {
	return r.dao.ListPublicRooms(ctx, search, isRPOnly, tag, viewerID, excludeUserIDs, includeArchived, limit, offset, tx...)
}

func (r *chatRepository) FindDMRoom(ctx context.Context, userA, userB uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.FindDMRoom(ctx, userA, userB, tx...)
}

func (r *chatRepository) UpdateRoom(ctx context.Context, spec UpdateChatRoom, tx ...*sql.Tx) error {
	return r.dao.UpdateRoom(ctx, spec, tx...)
}

func (r *chatRepository) AddRoomTags(ctx context.Context, roomID uuid.UUID, tags []string, tx ...*sql.Tx) error {
	return r.dao.AddRoomTags(ctx, roomID, tags, tx...)
}

func (r *chatRepository) ReplaceRoomTags(ctx context.Context, roomID uuid.UUID, tags []string, tx ...*sql.Tx) error {
	return r.dao.ReplaceRoomTags(ctx, roomID, tags, tx...)
}

func (r *chatRepository) GetRoomTags(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	return r.dao.GetRoomTags(ctx, roomID, tx...)
}

func (r *chatRepository) GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	return r.dao.GetRoomTagsBatch(ctx, roomIDs, tx...)
}

func (r *chatRepository) InsertMessageAndMarkRead(ctx context.Context, spec NewChatMessage, tx ...*sql.Tx) (*ChatMessageRow, error) {
	var result *ChatMessageRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		msg, err := r.insertMessage(ctx, spec, tx)
		if err != nil {
			return fmt.Errorf("insert message: %w", err)
		}

		if err := r.dao.MarkRoomRead(ctx, spec.RoomID, spec.SenderID, tx); err != nil {
			return fmt.Errorf("mark sender read: %w", err)
		}

		result = msg

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *chatRepository) InsertSystemMessage(ctx context.Context, roomID, senderID uuid.UUID, body string, tx ...*sql.Tx) (*ChatMessageRow, error) {
	spec := NewChatMessage{RoomID: roomID, SenderID: senderID, Body: body, IsSystem: true}

	var result *ChatMessageRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		msg, err := r.insertMessage(ctx, spec, tx)
		if err != nil {
			return err
		}

		result = msg

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *chatRepository) insertMessage(ctx context.Context, spec NewChatMessage, tx *sql.Tx) (*ChatMessageRow, error) {
	msg, err := r.dao.InsertMessageRow(ctx, spec, tx)
	if err != nil {
		return nil, err
	}

	if err := r.dao.TouchRoomActivityForMessage(ctx, spec.RoomID, spec.IsSystem, tx); err != nil {
		return nil, err
	}

	return msg, nil
}

func (r *chatRepository) InsertMessageRow(ctx context.Context, spec NewChatMessage, tx ...*sql.Tx) (*ChatMessageRow, error) {
	return r.dao.InsertMessageRow(ctx, spec, tx...)
}

func (r *chatRepository) TouchRoomActivityForMessage(ctx context.Context, roomID uuid.UUID, isSystem bool, tx ...*sql.Tx) error {
	return r.dao.TouchRoomActivityForMessage(ctx, roomID, isSystem, tx...)
}

func (r *chatRepository) EditMessage(ctx context.Context, messageID uuid.UUID, body string, tx ...*sql.Tx) error {
	return r.dao.EditMessage(ctx, messageID, body, tx...)
}

func (r *chatRepository) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]ChatMessageRow, int, error) {
	return r.dao.GetMessages(ctx, roomID, limit, offset, tx...)
}

func (r *chatRepository) GetMessagesForMember(ctx context.Context, roomID, viewerID uuid.UUID, limit int, tx ...*sql.Tx) ([]ChatMessageRow, error) {
	return r.dao.GetMessagesForMember(ctx, roomID, viewerID, limit, tx...)
}

func (r *chatRepository) GetMessagesForViewer(ctx context.Context, roomID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]ChatMessageRow, int, error) {
	return r.dao.GetMessagesForViewer(ctx, roomID, viewerID, limit, offset, tx...)
}

func (r *chatRepository) SearchMessagesForViewer(ctx context.Context, viewerID, roomID uuid.UUID, query string, limit, offset int, tx ...*sql.Tx) ([]SearchResult, int, error) {
	return r.dao.SearchMessagesForViewer(ctx, viewerID, roomID, query, limit, offset, tx...)
}

func (r *chatRepository) GetMessagesBefore(ctx context.Context, roomID, viewerID uuid.UUID, before string, limit int, tx ...*sql.Tx) ([]ChatMessageRow, error) {
	return r.dao.GetMessagesBefore(ctx, roomID, viewerID, before, limit, tx...)
}

func (r *chatRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (*ChatMessageRow, error) {
	return r.dao.GetMessageByID(ctx, messageID, tx...)
}

func (r *chatRepository) DeleteMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteMessages(ctx, roomID, tx...)
}

func (r *chatRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteMessage(ctx, messageID, tx...)
}

func (r *chatRepository) GetMessageSenderID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetMessageSenderID(ctx, messageID, tx...)
}

func (r *chatRepository) GetMessageRoomID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetMessageRoomID(ctx, messageID, tx...)
}

func (r *chatRepository) AddMessageMedia(ctx context.Context, spec NewChatMessageMedia, tx ...*sql.Tx) (int64, error) {
	return r.dao.AddMessageMedia(ctx, spec, tx...)
}

func (r *chatRepository) UpdateMessageMediaURL(ctx context.Context, id int64, mediaURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMessageMediaURL(ctx, id, mediaURL, tx...)
}

func (r *chatRepository) UpdateMessageMediaThumbnail(ctx context.Context, id int64, thumbnailURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateMessageMediaThumbnail(ctx, id, thumbnailURL, tx...)
}

func (r *chatRepository) GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]dto.PostMediaResponse, error) {
	return r.dao.GetMessageMediaBatch(ctx, messageIDs, tx...)
}

func (r *chatRepository) TouchRoomActivity(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.TouchRoomActivity(ctx, roomID, tx...)
}

func (r *chatRepository) ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.ArchiveStaleGroupRooms(ctx, cutoff, tx...)
}

func (r *chatRepository) MarkRoomRead(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkRoomRead(ctx, roomID, userID, tx...)
}

func (r *chatRepository) CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.CountUnreadRoomsForUser(ctx, userID, tx...)
}

func (r *chatRepository) SetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string, tx ...*sql.Tx) error {
	return r.dao.SetMemberNickname(ctx, roomID, userID, nickname, tx...)
}

func (r *chatRepository) SetMemberNicknameWithLock(ctx context.Context, roomID, userID uuid.UUID, nickname string, locked bool, tx ...*sql.Tx) error {
	return r.dao.SetMemberNicknameWithLock(ctx, roomID, userID, nickname, locked, tx...)
}

func (r *chatRepository) IsMemberNicknameLocked(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsMemberNicknameLocked(ctx, roomID, userID, tx...)
}

func (r *chatRepository) SetMemberAvatar(ctx context.Context, roomID, userID uuid.UUID, avatarURL string, tx ...*sql.Tx) error {
	return r.dao.SetMemberAvatar(ctx, roomID, userID, avatarURL, tx...)
}

func (r *chatRepository) SetMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, until string, byStaff bool, tx ...*sql.Tx) error {
	return r.dao.SetMemberTimeout(ctx, roomID, userID, until, byStaff, tx...)
}

func (r *chatRepository) ClearMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.ClearMemberTimeout(ctx, roomID, userID, tx...)
}

func (r *chatRepository) GetMemberTimeoutState(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, string, bool, error) {
	return r.dao.GetMemberTimeoutState(ctx, roomID, userID, tx...)
}

func (r *chatRepository) HasActiveMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.HasActiveMemberTimeout(ctx, roomID, userID, tx...)
}

func (r *chatRepository) PinMessage(ctx context.Context, messageID, pinnedBy uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.PinMessage(ctx, messageID, pinnedBy, tx...)
}

func (r *chatRepository) UnpinMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnpinMessage(ctx, messageID, tx...)
}

func (r *chatRepository) ListPinnedMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]ChatMessageRow, error) {
	return r.dao.ListPinnedMessages(ctx, roomID, tx...)
}

func (r *chatRepository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string, tx ...*sql.Tx) (bool, error) {
	return r.dao.AddReaction(ctx, messageID, userID, emoji, tx...)
}

func (r *chatRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string, tx ...*sql.Tx) (bool, error) {
	return r.dao.RemoveReaction(ctx, messageID, userID, emoji, tx...)
}

func (r *chatRepository) CountReactions(ctx context.Context, messageID uuid.UUID, emoji string, tx ...*sql.Tx) (int, error) {
	return r.dao.CountReactions(ctx, messageID, emoji, tx...)
}

func (r *chatRepository) GetReactionsBatch(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]ReactionGroup, error) {
	return r.dao.GetReactionsBatch(ctx, messageIDs, viewerID, tx...)
}
