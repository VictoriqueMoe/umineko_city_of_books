package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"
)

type (
	ChatDAO interface {
		CreateRoom(ctx context.Context, s spec.NewChatRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		CreateSystemRoom(ctx context.Context, s spec.NewChatSystemRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		GetSystemRoomID(ctx context.Context, systemKind string, tx ...*sql.Tx) (uuid.UUID, error)
		FindDMRoomByPair(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		CreateDMRoom(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		AddDMMembers(ctx context.Context, s spec.ChatDMMembers, tx ...*sql.Tx) error
		RejoinDMMembers(ctx context.Context, s spec.ChatDMMembers, tx ...*sql.Tx) error
		AddMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error
		AddMemberWithRole(ctx context.Context, s spec.NewChatRoomMember, tx ...*sql.Tx) error
		IsGhostMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		HasGhostMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (bool, error)
		SetMemberRole(ctx context.Context, s spec.ChatMemberRoleUpdate, tx ...*sql.Tx) error
		RemoveMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error
		CountRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error)
		DeleteRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListMessageMediaURLs(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListRoomMemberAvatarURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomRow, error)
		ListUserGroupRooms(ctx context.Context, q spec.ChatUserRoomFilter, tx ...*sql.Tx) ([]model.ChatRoomRow, int, error)
		GetRoomByID(ctx context.Context, s spec.ChatRoomViewer, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		GetRoomSendContext(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (*model.ChatRoomSendContext, error)
		GetRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomMemberRow, error)
		GetMemberRole(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (string, error)
		GetMemberNickname(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (string, error)
		IsMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		SetMuted(ctx context.Context, s spec.ChatMemberMuteUpdate, tx ...*sql.Tx) error
		IsMuted(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		SetVoiceForceMuted(ctx context.Context, s spec.ChatVoiceForceMuteUpdate, tx ...*sql.Tx) error
		IsVoiceForceMuted(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		ClearVoiceForceMutes(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		ListPublicRooms(ctx context.Context, q spec.ChatPublicRoomFilter, tx ...*sql.Tx) ([]model.ChatRoomRow, int, error)
		FindDMRoom(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (uuid.UUID, error)
		UpdateRoom(ctx context.Context, s spec.UpdateChatRoom, tx ...*sql.Tx) error
		AddRoomTags(ctx context.Context, s spec.ChatRoomTags, tx ...*sql.Tx) error
		ReplaceRoomTags(ctx context.Context, s spec.ChatRoomTags, tx ...*sql.Tx) error
		GetRoomTags(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)

		InsertMessageRow(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error)
		TouchRoomActivityForMessage(ctx context.Context, s spec.ChatRoomActivityTouch, tx ...*sql.Tx) error
		EditMessage(ctx context.Context, s spec.ChatMessageUpdate, tx ...*sql.Tx) error
		GetMessages(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, int, error)
		GetMessagesForMember(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, error)
		GetMessagesForViewer(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, int, error)
		SearchMessagesForViewer(ctx context.Context, s spec.ChatMessageSearch, tx ...*sql.Tx) ([]model.SearchResult, int, error)
		GetMessagesBefore(ctx context.Context, q spec.ChatMessageCursorPage, tx ...*sql.Tx) ([]model.ChatMessageRow, error)
		GetMessageByID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (*model.ChatMessageRow, error)
		DeleteMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		DeleteMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error
		GetMessageSenderID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetMessageRoomID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		AddMessageMedia(ctx context.Context, s spec.NewChatMessageMedia, tx ...*sql.Tx) (int64, error)
		UpdateMessageMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateMessageMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]dto.PostMediaResponse, error)

		TouchRoomActivity(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error
		ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error)
		MarkRoomRead(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error
		CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)

		SetMemberNickname(ctx context.Context, s spec.ChatMemberNicknameUpdate, tx ...*sql.Tx) error
		SetMemberNicknameWithLock(ctx context.Context, s spec.ChatMemberNicknameUpdate, tx ...*sql.Tx) error
		IsMemberNicknameLocked(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		SetMemberAvatar(ctx context.Context, s spec.ChatMemberAvatarUpdate, tx ...*sql.Tx) error
		SetMemberTimeout(ctx context.Context, s spec.ChatMemberTimeout, tx ...*sql.Tx) error
		ClearMemberTimeout(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error
		GetMemberTimeoutState(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, string, bool, error)
		HasActiveMemberTimeout(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error)
		PinMessage(ctx context.Context, s spec.ChatMessagePin, tx ...*sql.Tx) error
		UnpinMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error
		ListPinnedMessages(ctx context.Context, s spec.ChatRoomViewer, tx ...*sql.Tx) ([]model.ChatMessageRow, error)
		ListRoomAttachments(ctx context.Context, q spec.ChatRoomAttachmentQuery, tx ...*sql.Tx) ([]model.ChatMessageRow, error)
		AddReaction(ctx context.Context, s spec.ChatMessageReaction, tx ...*sql.Tx) (bool, error)
		RemoveReaction(ctx context.Context, s spec.ChatMessageReaction, tx ...*sql.Tx) (bool, error)
		CountReactions(ctx context.Context, s spec.ChatReactionCount, tx ...*sql.Tx) (int, error)
		GetReactionsBatch(ctx context.Context, q spec.ChatReactionsQuery, tx ...*sql.Tx) (map[uuid.UUID][]model.ReactionGroup, error)
	}

	chatDAO struct {
		db *sql.DB
	}

	chatRoomBaseRow = sqlcgen.CreateChatRoomRow

	chatMessageJoinRow = sqlcgen.GetChatMessageByIDRow
)

func nullTimeToString(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: nt.Time.UTC().Format(time.RFC3339)}
}

func parseTimestampInput(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp: %q", s)
}

func toChatRoomBase(row chatRoomBaseRow) model.ChatRoomRow {
	out := model.ChatRoomRow{
		ID:            row.ID,
		Name:          row.Name,
		Description:   row.Description,
		Type:          dto.RoomType(row.Type),
		IsPublic:      row.IsPublic,
		IsRP:          row.IsRp,
		IsSystem:      row.IsSystem,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
		LastMessageAt: nullTimeToString(row.LastMessageAt),
		ArchivedAt:    nullTimeToString(row.ArchivedAt),
	}

	if row.SystemKind.Valid {
		out.SystemKind = row.SystemKind.String
	}

	if row.CreatedBy != nil {
		out.CreatedBy = *row.CreatedBy
	}

	return out
}

func toChatRoomForMember(row sqlcgen.ListChatRoomsByUserRow) model.ChatRoomRow {
	out := toChatRoomBase(chatRoomBaseRow{
		ID:            row.ID,
		Name:          row.Name,
		Description:   row.Description,
		Type:          row.Type,
		IsPublic:      row.IsPublic,
		IsRp:          row.IsRp,
		IsSystem:      row.IsSystem,
		SystemKind:    row.SystemKind,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt,
		LastMessageAt: row.LastMessageAt,
		ArchivedAt:    row.ArchivedAt,
	})

	out.LastReadAt = nullTimeToString(row.LastReadAt)
	out.ViewerRole = row.ViewerRole
	out.ViewerMuted = row.ViewerMuted
	out.ViewerGhost = row.ViewerGhost
	out.MemberCount = int(row.MemberCount)
	out.IsMember = true

	return out
}

func toChatRoomForViewer(row sqlcgen.GetChatRoomByIDRow) model.ChatRoomRow {
	out := toChatRoomBase(chatRoomBaseRow{
		ID:            row.ID,
		Name:          row.Name,
		Description:   row.Description,
		Type:          row.Type,
		IsPublic:      row.IsPublic,
		IsRp:          row.IsRp,
		IsSystem:      row.IsSystem,
		SystemKind:    row.SystemKind,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt,
		LastMessageAt: row.LastMessageAt,
		ArchivedAt:    row.ArchivedAt,
	})

	out.LastReadAt = nullTimeToString(row.LastReadAt)
	out.MemberCount = int(row.MemberCount)

	if row.ViewerRole.Valid {
		out.ViewerRole = row.ViewerRole.String
		out.IsMember = true
	}

	if row.ViewerMuted.Valid {
		out.ViewerMuted = row.ViewerMuted.Bool
	}

	if row.ViewerGhost.Valid {
		out.ViewerGhost = row.ViewerGhost.Bool
	}

	return out
}

func replyToSenderName(nickname, displayName, username sql.NullString) *string {
	if nickname.Valid && nickname.String != "" {
		return new(nickname.String)
	}

	if displayName.Valid && displayName.String != "" {
		return new(displayName.String)
	}

	if username.Valid {
		return new(username.String)
	}

	return nil
}

func toChatMessageRow(row chatMessageJoinRow) model.ChatMessageRow {
	msg := model.ChatMessageRow{
		ID:                 row.ID,
		RoomID:             row.RoomID,
		SenderID:           row.SenderID,
		SenderUsername:     row.Username,
		SenderDisplayName:  row.DisplayName,
		SenderAvatarURL:    row.AvatarUrl,
		SenderRole:         row.SenderRole,
		SenderRoleTyped:    role.Role(row.SenderRole),
		Body:               row.Body,
		IsSystem:           row.IsSystem,
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339Nano),
		ReplyToID:          row.ReplyToID,
		ReplyToSenderID:    row.ReplyToSenderID,
		ReplyToSenderName:  replyToSenderName(row.ReplyMemberNickname, row.ReplyDisplayName, row.ReplyUsername),
		PinnedBy:           row.PinnedBy,
		SenderNickname:     row.SenderNickname,
		SenderMemberAvatar: row.SenderMemberAvatar,
	}

	if row.ReplyToBody.Valid {
		msg.ReplyToBody = new(row.ReplyToBody.String)
	}

	if row.PinnedAt.Valid {
		msg.PinnedAt = new(row.PinnedAt.Time.UTC().Format(time.RFC3339))
	}

	if row.EditedAt.Valid {
		msg.EditedAt = new(row.EditedAt.Time.UTC().Format(time.RFC3339))
	}

	return msg
}

func appendMediaPath(paths []string, mediaURL, thumbnailURL string) []string {
	if mediaURL != "" {
		paths = append(paths, mediaURL)
	}

	if thumbnailURL != "" {
		paths = append(paths, thumbnailURL)
	}

	return paths
}

func (r *chatDAO) CreateRoom(ctx context.Context, s spec.NewChatRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	created, err := genQueries(r.db, tx).CreateChatRoom(ctx, sqlcgen.CreateChatRoomParams{
		Name:        s.Name,
		Description: s.Description,
		Type:        s.Type,
		IsPublic:    s.IsPublic,
		IsRp:        s.IsRP,
		CreatedBy:   &s.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	return new(toChatRoomBase(created)), nil
}

func (r *chatDAO) CreateSystemRoom(ctx context.Context, s spec.NewChatSystemRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	created, err := genQueries(r.db, tx).CreateChatSystemRoom(ctx, sqlcgen.CreateChatSystemRoomParams{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		SystemKind:  sql.NullString{String: s.SystemKind, Valid: true},
		CreatedBy:   &s.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create system room: %w", err)
	}

	return new(toChatRoomBase(chatRoomBaseRow(created))), nil
}

func (r *chatDAO) GetSystemRoomID(ctx context.Context, systemKind string, tx ...*sql.Tx) (uuid.UUID, error) {
	id, err := genQueries(r.db, tx).GetChatSystemRoomID(ctx, sql.NullString{String: systemKind, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get system room id: %w", err)
	}

	return id, nil
}

func (r *chatDAO) UpdateRoom(ctx context.Context, s spec.UpdateChatRoom, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).UpdateChatRoom(ctx, sqlcgen.UpdateChatRoomParams{
		Name:        s.Name,
		Description: s.Description,
		IsPublic:    s.IsPublic,
		IsRp:        s.IsRP,
		ID:          s.RoomID,
	})
	if err != nil {
		return fmt.Errorf("update room: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("room not found or not editable")
	}

	return nil
}

func (r *chatDAO) AddRoomTags(ctx context.Context, s spec.ChatRoomTags, tx ...*sql.Tx) error {
	if len(s.Tags) == 0 {
		return nil
	}

	queries := genQueries(r.db, tx)
	for _, tag := range s.Tags {
		if tag == "" {
			continue
		}

		err := queries.AddChatRoomTag(ctx, sqlcgen.AddChatRoomTagParams{
			RoomID: s.RoomID,
			Tag:    tag,
		})
		if err != nil {
			return fmt.Errorf("add room tag: %w", err)
		}
	}

	return nil
}

func (r *chatDAO) ReplaceRoomTags(ctx context.Context, s spec.ChatRoomTags, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteChatRoomTags(ctx, s.RoomID); err != nil {
		return fmt.Errorf("delete room tags: %w", err)
	}

	return r.AddRoomTags(ctx, s, tx...)
}

func (r *chatDAO) GetRoomTags(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	tags, err := genQueries(r.db, tx).GetChatRoomTags(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room tags: %w", err)
	}

	return tags, nil
}

func (r *chatDAO) GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	result := make(map[uuid.UUID][]string)
	if len(roomIDs) == 0 {
		return result, nil
	}

	rows, err := genQueries(r.db, tx).GetChatRoomTagsBatch(ctx, joinUUIDs(roomIDs))
	if err != nil {
		return nil, fmt.Errorf("get room tags batch: %w", err)
	}

	for _, row := range rows {
		result[row.RoomID] = append(result[row.RoomID], row.Tag)
	}

	return result, nil
}

func (r *chatDAO) AddMemberWithRole(ctx context.Context, s spec.NewChatRoomMember, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AddChatRoomMemberWithRole(ctx, sqlcgen.AddChatRoomMemberWithRoleParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
		Role:   s.Role,
		Ghost:  s.Ghost,
	})
	if err != nil {
		return fmt.Errorf("add member with role: %w", err)
	}

	return nil
}

func (r *chatDAO) IsGhostMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	ghost, err := genQueries(r.db, tx).IsChatRoomGhostMember(ctx, sqlcgen.IsChatRoomGhostMemberParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get ghost flag: %w", err)
	}

	return ghost, nil
}

func (r *chatDAO) HasGhostMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	n, err := genQueries(r.db, tx).CountChatRoomGhostMembers(ctx, roomID)
	if err != nil {
		return false, fmt.Errorf("count ghost members: %w", err)
	}

	return n > 0, nil
}

func (r *chatDAO) SetMemberRole(ctx context.Context, s spec.ChatMemberRoleUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetChatMemberRole(ctx, sqlcgen.SetChatMemberRoleParams{
		Role:   s.Role,
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set member role: %w", err)
	}

	return nil
}

func (r *chatDAO) GetMemberNickname(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (string, error) {
	nickname, err := genQueries(r.db, tx).GetChatMemberNickname(ctx, sqlcgen.GetChatMemberNicknameParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get member nickname: %w", err)
	}

	return nickname, nil
}

func (r *chatDAO) GetMemberRole(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (string, error) {
	memberRole, err := genQueries(r.db, tx).GetChatMemberRole(ctx, sqlcgen.GetChatMemberRoleParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}

	return memberRole, nil
}

func dmPairKey(a, b uuid.UUID) string {
	sa, sb := a.String(), b.String()
	if sa > sb {
		sa, sb = sb, sa
	}
	return sa + ":" + sb
}

func (r *chatDAO) FindDMRoomByPair(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	existing, err := genQueries(r.db, tx).FindChatDMRoomByPair(ctx, sql.NullString{
		String: dmPairKey(s.UserA, s.UserB),
		Valid:  true,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("create dm: lookup: %w", err)
	}

	return new(toChatRoomBase(chatRoomBaseRow(existing))), nil
}

func (r *chatDAO) RejoinDMMembers(ctx context.Context, s spec.ChatDMMembers, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).RejoinChatDMMembers(ctx, sqlcgen.RejoinChatDMMembersParams{
		RoomID:   s.RoomID,
		UserID:   s.UserA,
		UserID_2: s.UserB,
	})
	if err != nil {
		return fmt.Errorf("create dm: rejoin members: %w", err)
	}

	return nil
}

func (r *chatDAO) CreateDMRoom(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	created, err := genQueries(r.db, tx).CreateChatDMRoom(ctx, sqlcgen.CreateChatDMRoomParams{
		CreatedBy: &s.UserA,
		DmPairKey: sql.NullString{String: dmPairKey(s.UserA, s.UserB), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create dm: insert room: %w", err)
	}

	return new(toChatRoomBase(chatRoomBaseRow(created))), nil
}

func (r *chatDAO) AddDMMembers(ctx context.Context, s spec.ChatDMMembers, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AddChatDMMembers(ctx, sqlcgen.AddChatDMMembersParams{
		RoomID:   s.RoomID,
		UserID:   s.UserA,
		UserID_2: s.UserB,
	})
	if err != nil {
		return fmt.Errorf("create dm: insert members: %w", err)
	}

	return nil
}

func (r *chatDAO) CountRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountChatRoomMembers(ctx, roomID)
	if err != nil {
		return 0, fmt.Errorf("count room members: %w", err)
	}

	return int(count), nil
}

func (r *chatDAO) DeleteRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteChatRoom(ctx, roomID); err != nil {
		return fmt.Errorf("delete room: %w", err)
	}

	return nil
}

func (r *chatDAO) ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).ListChatRoomMediaURLs(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list room media urls: %w", err)
	}

	var paths []string
	for _, row := range rows {
		paths = appendMediaPath(paths, row.MediaUrl, row.ThumbnailUrl)
	}

	return paths, nil
}

func (r *chatDAO) ListMessageMediaURLs(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).ListChatMessageMediaURLs(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("list message media urls: %w", err)
	}

	var paths []string
	for _, row := range rows {
		paths = appendMediaPath(paths, row.MediaUrl, row.ThumbnailUrl)
	}

	return paths, nil
}

func (r *chatDAO) ListRoomMemberAvatarURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	urls, err := genQueries(r.db, tx).ListChatRoomMemberAvatarURLs(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list room member avatar urls: %w", err)
	}

	return urls, nil
}

func (r *chatDAO) AddMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AddChatRoomMember(ctx, sqlcgen.AddChatRoomMemberParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return nil
}

func (r *chatDAO) RemoveMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).RemoveChatRoomMember(ctx, sqlcgen.RemoveChatRoomMemberParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

func (r *chatDAO) GetRoomsByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomRow, error) {
	rows, err := genQueries(r.db, tx).ListChatRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}

	var result []model.ChatRoomRow
	for _, row := range rows {
		result = append(result, toChatRoomForMember(row))
	}

	r.attachRoomTags(ctx, result, tx...)

	return result, nil
}

func (r *chatDAO) GetRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).GetChatRoomMemberIDs(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}

	return ids, nil
}

func (r *chatDAO) IsMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).IsChatRoomMember(ctx, sqlcgen.IsChatRoomMemberParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}

	return count > 0, nil
}

func (r *chatDAO) SetMuted(ctx context.Context, s spec.ChatMemberMuteUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetChatMemberMuted(ctx, sqlcgen.SetChatMemberMutedParams{
		Muted:  s.Muted,
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set muted: %w", err)
	}

	return nil
}

func (r *chatDAO) IsMuted(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	muted, err := genQueries(r.db, tx).IsChatMemberMuted(ctx, sqlcgen.IsChatMemberMutedParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check muted: %w", err)
	}

	return muted, nil
}

func (r *chatDAO) GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).GetChatRoomUnmutedMemberIDs(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get unmuted members: %w", err)
	}

	return ids, nil
}

func (r *chatDAO) SetVoiceForceMuted(ctx context.Context, s spec.ChatVoiceForceMuteUpdate, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	if !s.Muted {
		err := queries.DeleteChatVoiceForceMute(ctx, sqlcgen.DeleteChatVoiceForceMuteParams{
			RoomID: s.RoomID,
			UserID: s.UserID,
		})
		if err != nil {
			return fmt.Errorf("clear voice force mute: %w", err)
		}

		return nil
	}

	err := queries.SetChatVoiceForceMute(ctx, sqlcgen.SetChatVoiceForceMuteParams{
		RoomID:  s.RoomID,
		UserID:  s.UserID,
		MutedBy: &s.MutedBy,
	})
	if err != nil {
		return fmt.Errorf("set voice force mute: %w", err)
	}

	return nil
}

func (r *chatDAO) IsVoiceForceMuted(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	muted, err := genQueries(r.db, tx).IsChatVoiceForceMuted(ctx, sqlcgen.IsChatVoiceForceMutedParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return false, fmt.Errorf("check voice force mute: %w", err)
	}

	return muted, nil
}

func (r *chatDAO) ClearVoiceForceMutes(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).ClearChatVoiceForceMutes(ctx, roomID); err != nil {
		return fmt.Errorf("clear voice force mutes: %w", err)
	}

	return nil
}

func (r *chatDAO) FindDMRoom(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (uuid.UUID, error) {
	id, err := genQueries(r.db, tx).FindChatDMRoomForUser(ctx, sqlcgen.FindChatDMRoomForUserParams{
		UserID:    s.UserA,
		DmPairKey: sql.NullString{String: dmPairKey(s.UserA, s.UserB), Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find dm room: %w", err)
	}

	return id, nil
}

func (r *chatDAO) InsertMessageRow(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	msg, err := genQueries(r.db, tx).InsertChatMessage(ctx, sqlcgen.InsertChatMessageParams{
		RoomID:    s.RoomID,
		SenderID:  s.SenderID,
		Body:      s.Body,
		ReplyToID: s.ReplyToID,
		IsSystem:  s.IsSystem,
	})
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	return new(toChatMessageRow(chatMessageJoinRow(msg))), nil
}

func (r *chatDAO) TouchRoomActivityForMessage(ctx context.Context, s spec.ChatRoomActivityTouch, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	var err error
	if s.IsSystem {
		err = queries.TouchChatRoomActivity(ctx, s.RoomID)
	} else {
		err = queries.TouchChatRoomActivityUnarchive(ctx, s.RoomID)
	}

	if err != nil {
		return fmt.Errorf("touch room activity: %w", err)
	}

	return nil
}

func (r *chatDAO) GetMessages(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, int, error) {
	return r.getMessages(ctx, q, tx...)
}

func (r *chatDAO) GetMessagesForViewer(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, int, error) {
	return r.getMessages(ctx, q, tx...)
}

func (r *chatDAO) GetMessagesForMember(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, error) {
	rows, _, err := r.getMessages(ctx, q, tx...)

	return rows, err
}

func (r *chatDAO) getMessages(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountChatRoomMessages(ctx, sqlcgen.CountChatRoomMessagesParams{
		RoomID: q.RoomID,
		UserID: q.ViewerID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	rows, err := queries.ListChatRoomMessages(ctx, sqlcgen.ListChatRoomMessagesParams{
		RoomID: q.RoomID,
		UserID: q.ViewerID,
		Limit:  int32(q.Limit),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get messages: %w", err)
	}

	var messages []model.ChatMessageRow
	for _, row := range rows {
		messages = append(messages, toChatMessageRow(chatMessageJoinRow(row)))
	}

	return messages, int(total), nil
}

func (r *chatDAO) SearchMessagesForViewer(ctx context.Context, s spec.ChatMessageSearch, tx ...*sql.Tx) ([]model.SearchResult, int, error) {
	rows, err := genQueries(r.db, tx).SearchChatMessagesForViewer(ctx, sqlcgen.SearchChatMessagesForViewerParams{
		UserID:  s.ViewerID,
		Column2: s.Query,
		Column3: s.RoomID,
		Limit:   int32(s.Limit),
		Offset:  int32(s.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search chat messages: %w", err)
	}

	results := make([]model.SearchResult, 0, s.Limit)
	total := 0

	for _, row := range rows {
		results = append(results, model.SearchResult{
			EntityType:        model.SearchEntityType(row.EntityType),
			ID:                row.ID,
			ParentID:          new(row.ParentID),
			ParentTitle:       new(row.ParentTitle),
			Title:             row.Title,
			Snippet:           row.Snippet,
			AuthorID:          new(row.AuthorID),
			AuthorUsername:    row.Username,
			AuthorDisplayName: row.DisplayName,
			AuthorAvatarURL:   row.AvatarUrl,
			CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
			Rank:              row.Rank,
		})

		total = int(row.TotalCount)
	}

	return results, total, nil
}

func (r *chatDAO) ListRoomAttachments(ctx context.Context, q spec.ChatRoomAttachmentQuery, tx ...*sql.Tx) ([]model.ChatMessageRow, error) {
	beforeTime, beforeID, err := splitMessageCursor(q.Before)
	if err != nil {
		return nil, fmt.Errorf("list room attachments: %w", err)
	}

	params := sqlcgen.ListChatRoomMediaAttachmentsParams{
		RoomID:     q.RoomID,
		BeforeTime: sql.NullTime{Time: beforeTime, Valid: q.Before != ""},
		BeforeID:   beforeID,
		ViewerID:   q.ViewerID,
		RowLimit:   int32(q.Limit),
	}

	queries := genQueries(r.db, tx)

	var joined []chatMessageJoinRow
	switch q.Kind {
	case model.AttachmentKindMedia:
		rows, err := queries.ListChatRoomMediaAttachments(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("list room attachments: %w", err)
		}

		for _, row := range rows {
			joined = append(joined, chatMessageJoinRow(row))
		}
	case model.AttachmentKindLinks:
		rows, err := queries.ListChatRoomLinkAttachments(ctx, sqlcgen.ListChatRoomLinkAttachmentsParams(params))
		if err != nil {
			return nil, fmt.Errorf("list room attachments: %w", err)
		}

		for _, row := range rows {
			joined = append(joined, chatMessageJoinRow(row))
		}
	default:
		return nil, fmt.Errorf("list room attachments: unknown kind %q", q.Kind)
	}

	var messages []model.ChatMessageRow
	for _, row := range joined {
		messages = append(messages, toChatMessageRow(row))
	}

	return messages, nil
}

func splitMessageCursor(before string) (time.Time, string, error) {
	beforeTS := before
	beforeID := ""
	parts := strings.SplitN(before, "|", 2)
	if len(parts) > 0 {
		beforeTS = strings.TrimSpace(parts[0])
	}
	if len(parts) == 2 {
		candidate := strings.TrimSpace(parts[1])
		if _, err := uuid.Parse(candidate); err == nil {
			beforeID = candidate
		}
	}

	beforeTime, err := parseTimestampInput(beforeTS)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse before: %w", err)
	}

	return beforeTime, beforeID, nil
}

func (r *chatDAO) GetMessagesBefore(ctx context.Context, q spec.ChatMessageCursorPage, tx ...*sql.Tx) ([]model.ChatMessageRow, error) {
	beforeTime, beforeID, parseErr := splitMessageCursor(q.Before)
	if parseErr != nil {
		return nil, fmt.Errorf("get messages before: %w", parseErr)
	}

	rows, err := genQueries(r.db, tx).ListChatRoomMessagesBefore(ctx, sqlcgen.ListChatRoomMessagesBeforeParams{
		RoomID:  q.RoomID,
		Column2: beforeTime,
		Column3: beforeID,
		Limit:   int32(q.Limit),
		UserID:  q.ViewerID,
	})
	if err != nil {
		return nil, fmt.Errorf("get messages before: %w", err)
	}

	var messages []model.ChatMessageRow
	for _, row := range rows {
		messages = append(messages, toChatMessageRow(chatMessageJoinRow(row)))
	}

	return messages, nil
}

func (r *chatDAO) GetMessageByID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	row, err := genQueries(r.db, tx).GetChatMessageByID(ctx, messageID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get message by id: %w", err)
	}

	return new(toChatMessageRow(row)), nil
}

func (r *chatDAO) GetMessageRoomID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	roomID, err := genQueries(r.db, tx).GetChatMessageRoomID(ctx, messageID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get message room id: %w", err)
	}

	return roomID, nil
}

func (r *chatDAO) DeleteMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteChatRoomMessages(ctx, roomID); err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}

	return nil
}

func (r *chatDAO) DeleteMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteChatMessage(ctx, messageID); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return nil
}

func (r *chatDAO) EditMessage(ctx context.Context, s spec.ChatMessageUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).EditChatMessage(ctx, sqlcgen.EditChatMessageParams{
		Body: s.Body,
		ID:   s.MessageID,
	})
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return nil
}

func (r *chatDAO) TouchRoomActivity(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).TouchChatRoomActivity(ctx, roomID); err != nil {
		return fmt.Errorf("touch room activity: %w", err)
	}

	return nil
}

func (r *chatDAO) ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error) {
	queries := genQueries(r.db, tx)

	ids, err := queries.ListStaleChatRoomIDs(ctx, cutoff.UTC())
	if err != nil {
		return nil, fmt.Errorf("find stale chat rooms: %w", err)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	if err := queries.ArchiveChatRooms(ctx, joinUUIDs(ids)); err != nil {
		return nil, fmt.Errorf("archive stale chat rooms: %w", err)
	}

	return ids, nil
}

func (r *chatDAO) MarkRoomRead(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).MarkChatRoomRead(ctx, sqlcgen.MarkChatRoomReadParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}

	return nil
}

func (r *chatDAO) GetMessageSenderID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	senderID, err := genQueries(r.db, tx).GetChatMessageSenderID(ctx, messageID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get message sender: %w", err)
	}

	return senderID, nil
}

func (r *chatDAO) AddMessageMedia(ctx context.Context, s spec.NewChatMessageMedia, tx ...*sql.Tx) (int64, error) {
	id, err := genQueries(r.db, tx).AddChatMessageMedia(ctx, sqlcgen.AddChatMessageMediaParams{
		MessageID:    s.TargetID,
		MediaUrl:     s.MediaURL,
		MediaType:    s.MediaType,
		ThumbnailUrl: s.ThumbnailURL,
		Filename:     sql.NullString{String: s.Filename, Valid: true},
		SortOrder:    int32(s.SortOrder),
		Width:        int32(s.Width),
		Height:       int32(s.Height),
		IsSpoiler:    s.IsSpoiler,
	})
	if err != nil {
		return 0, fmt.Errorf("add message media: %w", err)
	}

	return id, nil
}

func (r *chatDAO) UpdateMessageMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateChatMessageMediaURL(ctx, sqlcgen.UpdateChatMessageMediaURLParams{
		MediaUrl: s.URL,
		ID:       s.ID,
	})
	if err != nil {
		return fmt.Errorf("update message media url: %w", err)
	}

	return nil
}

func (r *chatDAO) UpdateMessageMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateChatMessageMediaThumbnail(ctx, sqlcgen.UpdateChatMessageMediaThumbnailParams{
		ThumbnailUrl: s.URL,
		ID:           s.ID,
	})
	if err != nil {
		return fmt.Errorf("update message media thumbnail: %w", err)
	}

	return nil
}

func (r *chatDAO) GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]dto.PostMediaResponse, error) {
	result := make(map[uuid.UUID][]dto.PostMediaResponse)
	if len(messageIDs) == 0 {
		return result, nil
	}

	rows, err := genQueries(r.db, tx).GetChatMessageMediaBatch(ctx, joinUUIDs(messageIDs))
	if err != nil {
		return nil, fmt.Errorf("get message media batch: %w", err)
	}

	for _, row := range rows {
		result[row.MessageID] = append(result[row.MessageID], dto.PostMediaResponse{
			ID:           int(row.ID),
			MediaURL:     row.MediaUrl,
			MediaType:    row.MediaType,
			ThumbnailURL: row.ThumbnailUrl,
			Filename:     row.Filename,
			SortOrder:    int(row.SortOrder),
			Width:        int(row.Width),
			Height:       int(row.Height),
			IsSpoiler:    row.IsSpoiler,
		})
	}

	return result, nil
}

func (r *chatDAO) SetMemberNickname(ctx context.Context, s spec.ChatMemberNicknameUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetChatMemberNickname(ctx, sqlcgen.SetChatMemberNicknameParams{
		Nickname: s.Nickname,
		RoomID:   s.RoomID,
		UserID:   s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set member nickname: %w", err)
	}

	return nil
}

func (r *chatDAO) SetMemberNicknameWithLock(ctx context.Context, s spec.ChatMemberNicknameUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetChatMemberNicknameWithLock(ctx, sqlcgen.SetChatMemberNicknameWithLockParams{
		Nickname:       s.Nickname,
		NicknameLocked: s.Locked,
		RoomID:         s.RoomID,
		UserID:         s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set member nickname with lock: %w", err)
	}

	return nil
}

func (r *chatDAO) IsMemberNicknameLocked(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	locked, err := genQueries(r.db, tx).IsChatMemberNicknameLocked(ctx, sqlcgen.IsChatMemberNicknameLockedParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check nickname locked: %w", err)
	}

	return locked, nil
}

func (r *chatDAO) SetMemberAvatar(ctx context.Context, s spec.ChatMemberAvatarUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetChatMemberAvatar(ctx, sqlcgen.SetChatMemberAvatarParams{
		AvatarUrl: s.AvatarURL,
		RoomID:    s.RoomID,
		UserID:    s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set member avatar: %w", err)
	}

	return nil
}

func (r *chatDAO) SetMemberTimeout(ctx context.Context, s spec.ChatMemberTimeout, tx ...*sql.Tx) error {
	t, parseErr := parseTimestampInput(s.Until)
	if parseErr != nil {
		return fmt.Errorf("set member timeout: parse until: %w", parseErr)
	}

	err := genQueries(r.db, tx).SetChatMemberTimeout(ctx, sqlcgen.SetChatMemberTimeoutParams{
		TimeoutUntil:      sql.NullTime{Time: t, Valid: true},
		TimeoutSetByStaff: s.ByStaff,
		RoomID:            s.RoomID,
		UserID:            s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set member timeout: %w", err)
	}

	return nil
}

func (r *chatDAO) ClearMemberTimeout(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).ClearChatMemberTimeout(ctx, sqlcgen.ClearChatMemberTimeoutParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("clear member timeout: %w", err)
	}

	return nil
}

func (r *chatDAO) HasActiveMemberTimeout(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	active, err := genQueries(r.db, tx).HasActiveChatMemberTimeout(ctx, sqlcgen.HasActiveChatMemberTimeoutParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check active member timeout: %w", err)
	}

	return active, nil
}

func (r *chatDAO) GetMemberTimeoutState(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, string, bool, error) {
	row, err := genQueries(r.db, tx).GetChatMemberTimeoutState(ctx, sqlcgen.GetChatMemberTimeoutStateParams{
		RoomID: s.RoomID,
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", false, nil
	}
	if err != nil {
		return false, "", false, fmt.Errorf("get member timeout state: %w", err)
	}

	if !row.TimeoutUntil.Valid {
		return false, "", row.TimeoutSetByStaff, nil
	}

	return row.Active, row.TimeoutUntil.Time.UTC().Format(time.RFC3339), row.TimeoutSetByStaff, nil
}

func (r *chatDAO) PinMessage(ctx context.Context, s spec.ChatMessagePin, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).PinChatMessage(ctx, sqlcgen.PinChatMessageParams{
		PinnedBy: &s.PinnedBy,
		ID:       s.MessageID,
	})
	if err != nil {
		return fmt.Errorf("pin message: %w", err)
	}

	return nil
}

func (r *chatDAO) UnpinMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UnpinChatMessage(ctx, messageID); err != nil {
		return fmt.Errorf("unpin message: %w", err)
	}

	return nil
}

func (r *chatDAO) ListPinnedMessages(ctx context.Context, s spec.ChatRoomViewer, tx ...*sql.Tx) ([]model.ChatMessageRow, error) {
	rows, err := genQueries(r.db, tx).ListPinnedChatMessages(ctx, sqlcgen.ListPinnedChatMessagesParams{
		RoomID: s.RoomID,
		UserID: s.ViewerID,
	})
	if err != nil {
		return nil, fmt.Errorf("list pinned messages: %w", err)
	}

	var messages []model.ChatMessageRow
	for _, row := range rows {
		messages = append(messages, toChatMessageRow(chatMessageJoinRow(row)))
	}

	return messages, nil
}

func (r *chatDAO) AddReaction(ctx context.Context, s spec.ChatMessageReaction, tx ...*sql.Tx) (bool, error) {
	n, err := genQueries(r.db, tx).AddChatMessageReaction(ctx, sqlcgen.AddChatMessageReactionParams{
		MessageID: s.MessageID,
		UserID:    s.UserID,
		Emoji:     s.Emoji,
	})
	if err != nil {
		return false, fmt.Errorf("add reaction: %w", err)
	}

	return n > 0, nil
}

func (r *chatDAO) RemoveReaction(ctx context.Context, s spec.ChatMessageReaction, tx ...*sql.Tx) (bool, error) {
	n, err := genQueries(r.db, tx).RemoveChatMessageReaction(ctx, sqlcgen.RemoveChatMessageReactionParams{
		MessageID: s.MessageID,
		UserID:    s.UserID,
		Emoji:     s.Emoji,
	})
	if err != nil {
		return false, fmt.Errorf("remove reaction: %w", err)
	}

	return n > 0, nil
}

func (r *chatDAO) CountReactions(ctx context.Context, s spec.ChatReactionCount, tx ...*sql.Tx) (int, error) {
	n, err := genQueries(r.db, tx).CountChatMessageReactions(ctx, sqlcgen.CountChatMessageReactionsParams{
		MessageID: s.MessageID,
		Emoji:     s.Emoji,
	})
	if err != nil {
		return 0, fmt.Errorf("count reactions: %w", err)
	}

	return int(n), nil
}

func (r *chatDAO) GetReactionsBatch(ctx context.Context, q spec.ChatReactionsQuery, tx ...*sql.Tx) (map[uuid.UUID][]model.ReactionGroup, error) {
	result := make(map[uuid.UUID][]model.ReactionGroup)
	if len(q.MessageIDs) == 0 {
		return result, nil
	}

	rows, err := genQueries(r.db, tx).GetChatMessageReactionsBatch(ctx, sqlcgen.GetChatMessageReactionsBatchParams{
		UserID:  q.ViewerID,
		Column2: joinUUIDs(q.MessageIDs),
	})
	if err != nil {
		return nil, fmt.Errorf("get reactions batch: %w", err)
	}

	for _, row := range rows {
		var displayNames []string
		if len(row.Names) > 0 {
			displayNames = strings.Split(string(row.Names), "\n")
		}

		result[row.MessageID] = append(result[row.MessageID], model.ReactionGroup{
			Emoji:         row.Emoji,
			Count:         int(row.Cnt),
			ViewerReacted: row.ViewerReacted,
			DisplayNames:  displayNames,
		})
	}

	return result, nil
}

func (r *chatDAO) CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUnreadChatDMRoomsForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count unread dm rooms: %w", err)
	}

	return int(count), nil
}
