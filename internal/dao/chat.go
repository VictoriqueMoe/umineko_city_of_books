package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/utils"
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
)

const (
	hotScoreExpr = `(
		COALESCE((SELECT COUNT(*) + COUNT(DISTINCT sender_id) * 8
			FROM chat_messages
			WHERE room_id = cr.id AND is_system = FALSE
			  AND created_at >= NOW() - INTERVAL '24 hours'), 0)
		+ COALESCE((SELECT COUNT(*) * 3
			FROM chat_messages
			WHERE room_id = cr.id AND is_system = FALSE
			  AND created_at >= NOW() - INTERVAL '1 hour'), 0)
	)`

	createdRoomColumns = `id, name, description, type, is_public, is_rp, is_system, system_kind, created_by, created_at, last_message_at, archived_at`

	chatMessageMediaTable = `chat_message_media`
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

func scanCreatedRoom(row interface{ Scan(dest ...any) error }) (*model.ChatRoomRow, error) {
	var out model.ChatRoomRow
	var systemKind sql.NullString
	var createdAt time.Time
	var lastMessageAt, archivedAt sql.NullTime

	if err := row.Scan(&out.ID, &out.Name, &out.Description, &out.Type, &out.IsPublic, &out.IsRP, &out.IsSystem,
		&systemKind, &out.CreatedBy, &createdAt, &lastMessageAt, &archivedAt); err != nil {
		return nil, err
	}

	out.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	out.LastMessageAt = nullTimeToString(lastMessageAt)
	out.ArchivedAt = nullTimeToString(archivedAt)
	if systemKind.Valid {
		out.SystemKind = systemKind.String
	}

	return &out, nil
}

func (r *chatDAO) CreateRoom(ctx context.Context, s spec.NewChatRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	row, err := scanCreatedRoom(txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chat_rooms (name, description, type, is_public, is_rp, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+createdRoomColumns,
		s.Name, s.Description, s.Type, s.IsPublic, s.IsRP, s.CreatedBy,
	))
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	return row, nil
}

func (r *chatDAO) CreateSystemRoom(ctx context.Context, s spec.NewChatSystemRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	row, err := scanCreatedRoom(txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chat_rooms (id, name, description, type, is_public, is_rp, is_system, system_kind, created_by)
		 VALUES ($1, $2, $3, 'group', FALSE, FALSE, TRUE, $4, $5)
		 RETURNING `+createdRoomColumns,
		s.ID, s.Name, s.Description, s.SystemKind, s.CreatedBy,
	))
	if err != nil {
		return nil, fmt.Errorf("create system room: %w", err)
	}

	return row, nil
}

func (r *chatDAO) GetSystemRoomID(ctx context.Context, systemKind string, tx ...*sql.Tx) (uuid.UUID, error) {
	var id uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id FROM chat_rooms WHERE system_kind = $1 LIMIT 1`, systemKind,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get system room id: %w", err)
	}
	return id, nil
}

func (r *chatDAO) UpdateRoom(ctx context.Context, s spec.UpdateChatRoom, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_rooms SET name = $1, description = $2, is_public = $3, is_rp = $4
		 WHERE id = $5 AND type = 'group' AND is_system = FALSE`,
		s.Name, s.Description, s.IsPublic, s.IsRP, s.RoomID,
	)
	if err != nil {
		return fmt.Errorf("update room: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("room not found or not editable")
	}

	return nil
}

func (r *chatDAO) AddRoomTags(ctx context.Context, s spec.ChatRoomTags, tx ...*sql.Tx) error {
	if len(s.Tags) == 0 {
		return nil
	}
	for _, tag := range s.Tags {
		if tag == "" {
			continue
		}
		_, err := txOrDB(r.db, tx).ExecContext(ctx,
			`INSERT INTO chat_room_tags (room_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			s.RoomID, tag,
		)
		if err != nil {
			return fmt.Errorf("add room tag: %w", err)
		}
	}
	return nil
}

func (r *chatDAO) ReplaceRoomTags(ctx context.Context, s spec.ChatRoomTags, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chat_room_tags WHERE room_id = $1`, s.RoomID); err != nil {
		return fmt.Errorf("delete room tags: %w", err)
	}
	return r.AddRoomTags(ctx, s, tx...)
}

func (r *chatDAO) GetRoomTags(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT tag FROM chat_room_tags WHERE room_id = $1 ORDER BY tag`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room tags: %w", err)
	}

	return utils.ScanStrings(rows, "room tag")
}

func (r *chatDAO) GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(roomIDs) == 0 {
		return make(map[uuid.UUID][]string), nil
	}

	placeholders, args := utils.PlaceholderArgs(roomIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT room_id, tag FROM chat_room_tags WHERE room_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY tag`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get room tags batch: %w", err)
	}

	return utils.ScanGroups[uuid.UUID, string](rows, "room tag batch")
}

func (r *chatDAO) AddMemberWithRole(ctx context.Context, s spec.NewChatRoomMember, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id, role, ghost) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (room_id, user_id) DO UPDATE SET left_at = NULL, role = excluded.role, ghost = excluded.ghost, joined_at = NOW()`,
		s.RoomID, s.UserID, s.Role, s.Ghost,
	)
	if err != nil {
		return fmt.Errorf("add member with role: %w", err)
	}
	return nil
}

func (r *chatDAO) IsGhostMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	var g bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT ghost FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	).Scan(&g)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get ghost flag: %w", err)
	}
	return g, nil
}

func (r *chatDAO) HasGhostMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	var n int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(1) FROM chat_room_members WHERE room_id = $1 AND ghost = TRUE AND left_at IS NULL`,
		roomID,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("count ghost members: %w", err)
	}
	return n > 0, nil
}

func (r *chatDAO) SetMemberRole(ctx context.Context, s spec.ChatMemberRoleUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET role = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL`,
		s.Role, s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("set member role: %w", err)
	}
	return nil
}

func (r *chatDAO) GetMemberNickname(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (string, error) {
	var nickname string
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COALESCE(nickname, '') FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	).Scan(&nickname)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get member nickname: %w", err)
	}
	return nickname, nil
}

func (r *chatDAO) GetMemberRole(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (string, error) {
	var memberRole string
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT role FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	).Scan(&memberRole)
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
	existing, err := scanCreatedRoom(txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT `+createdRoomColumns+` FROM chat_rooms WHERE type = 'dm' AND dm_pair_key = $1`,
		dmPairKey(s.UserA, s.UserB),
	))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("create dm: lookup: %w", err)
	}

	return existing, nil
}

func (r *chatDAO) RejoinDMMembers(ctx context.Context, s spec.ChatDMMembers, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2), ($1, $3)
		 ON CONFLICT (room_id, user_id) DO UPDATE SET left_at = NULL, joined_at = NOW()
		 WHERE chat_room_members.left_at IS NOT NULL`,
		s.RoomID, s.UserA, s.UserB,
	)
	if err != nil {
		return fmt.Errorf("create dm: rejoin members: %w", err)
	}

	return nil
}

func (r *chatDAO) CreateDMRoom(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	created, err := scanCreatedRoom(txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chat_rooms (name, type, created_by, dm_pair_key)
		 VALUES ('', 'dm', $1, $2)
		 RETURNING `+createdRoomColumns,
		s.UserA, dmPairKey(s.UserA, s.UserB),
	))
	if err != nil {
		return nil, fmt.Errorf("create dm: insert room: %w", err)
	}

	return created, nil
}

func (r *chatDAO) AddDMMembers(ctx context.Context, s spec.ChatDMMembers, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2), ($1, $3)`,
		s.RoomID, s.UserA, s.UserB,
	)
	if err != nil {
		return fmt.Errorf("create dm: insert members: %w", err)
	}

	return nil
}

func (r *chatDAO) CountRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = $1 AND left_at IS NULL`, roomID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count room members: %w", err)
	}
	return count, nil
}

func (r *chatDAO) DeleteRoom(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chat_rooms WHERE id = $1`, roomID)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}
	return nil
}

func (r *chatDAO) ListRoomMediaURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cmm.media_url, cmm.thumbnail_url
		 FROM chat_message_media cmm
		 JOIN chat_messages cm ON cm.id = cmm.message_id
		 WHERE cm.room_id = $1`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list room media urls: %w", err)
	}

	return scanMediaPaths(rows, chatMessageMediaTable)
}

func (r *chatDAO) ListMessageMediaURLs(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT media_url, thumbnail_url FROM chat_message_media WHERE message_id = $1`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("list message media urls: %w", err)
	}

	return scanMediaPaths(rows, chatMessageMediaTable)
}

func (r *chatDAO) ListRoomMemberAvatarURLs(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT avatar_url FROM chat_room_members WHERE room_id = $1 AND avatar_url <> ''`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list room member avatar urls: %w", err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var avatarURL string
		if err := rows.Scan(&avatarURL); err != nil {
			return nil, fmt.Errorf("scan room member avatar url: %w", err)
		}
		urls = append(urls, avatarURL)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate room member avatar urls: %w", err)
	}

	return urls, nil
}

func (r *chatDAO) AddMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2)
		 ON CONFLICT (room_id, user_id) DO UPDATE SET left_at = NULL, joined_at = NOW()`,
		s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *chatDAO) RemoveMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET left_at = NOW() WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

func (r *chatDAO) GetRoomsByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, cr.archived_at, m.last_read_at, m.role, m.muted, m.ghost,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL)
		 FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
		 WHERE cr.system_kind IS DISTINCT FROM 'watch_party'
		 ORDER BY cr.is_system DESC, COALESCE(cr.last_message_at, cr.created_at) DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}
	defer rows.Close()

	var result []model.ChatRoomRow
	for rows.Next() {
		var row model.ChatRoomRow
		var systemKind sql.NullString
		var createdAt time.Time
		var lastMessageAt, archivedAt, lastReadAt sql.NullTime
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.Type, &row.IsPublic, &row.IsRP, &row.IsSystem, &systemKind, &row.CreatedBy, &createdAt, &lastMessageAt, &archivedAt, &lastReadAt, &row.ViewerRole, &row.ViewerMuted, &row.ViewerGhost, &row.MemberCount); err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.LastMessageAt = nullTimeToString(lastMessageAt)
		row.ArchivedAt = nullTimeToString(archivedAt)
		row.LastReadAt = nullTimeToString(lastReadAt)
		if systemKind.Valid {
			row.SystemKind = systemKind.String
		}
		row.IsMember = true
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	r.attachRoomTags(ctx, result, tx...)
	return result, nil
}

func (r *chatDAO) ListUserGroupRooms(ctx context.Context, q spec.ChatUserRoomFilter, tx ...*sql.Tx) ([]model.ChatRoomRow, int, error) {
	conditions := []string{"cr.type = 'group'", "m.user_id = $1", "m.left_at IS NULL", "cr.system_kind IS DISTINCT FROM 'watch_party'"}
	args := []any{q.UserID}
	idx := 2
	if !q.IncludeArchived {
		conditions = append(conditions, "cr.archived_at IS NULL")
	}
	if q.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(cr.name ILIKE $%d OR cr.description ILIKE $%d)", idx, idx+1))
		wc := "%" + q.Search + "%"
		args = append(args, wc, wc)
		idx += 2
	}
	if q.IsRPOnly {
		conditions = append(conditions, "cr.is_rp = TRUE")
	}
	if q.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS(SELECT 1 FROM chat_room_tags WHERE room_id = cr.id AND tag = $%d)", idx))
		args = append(args, q.Tag)
		idx++
	}
	if q.Role == "host" {
		conditions = append(conditions, "m.role = 'host'")
	} else if q.Role == "member" {
		conditions = append(conditions, "m.role != 'host'")
	}

	var where strings.Builder
	where.WriteString(" WHERE " + conditions[0])
	for _, c := range conditions[1:] {
		where.WriteString(" AND " + c)
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id`+where.String(), countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user group rooms: %w", err)
	}

	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, q.Limit, q.Offset)
	limitClause := fmt.Sprintf(" LIMIT $%d OFFSET $%d", idx, idx+1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, cr.archived_at, m.last_read_at, m.role, m.muted, m.ghost,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL),
		 `+hotScoreExpr+`
		 FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id`+where.String()+`
		 ORDER BY cr.is_system DESC, COALESCE(cr.last_message_at, cr.created_at) DESC`+limitClause, queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list user group rooms: %w", err)
	}
	defer rows.Close()

	var result []model.ChatRoomRow
	for rows.Next() {
		var row model.ChatRoomRow
		var systemKind sql.NullString
		var createdAt time.Time
		var lastMessageAt, archivedAt, lastReadAt sql.NullTime
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.Type, &row.IsPublic, &row.IsRP, &row.IsSystem, &systemKind, &row.CreatedBy, &createdAt, &lastMessageAt, &archivedAt, &lastReadAt, &row.ViewerRole, &row.ViewerMuted, &row.ViewerGhost, &row.MemberCount, &row.HotScore); err != nil {
			return nil, 0, fmt.Errorf("scan user group room: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.LastMessageAt = nullTimeToString(lastMessageAt)
		row.ArchivedAt = nullTimeToString(archivedAt)
		row.LastReadAt = nullTimeToString(lastReadAt)
		if systemKind.Valid {
			row.SystemKind = systemKind.String
		}
		row.IsMember = true
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	r.attachRoomTags(ctx, result, tx...)
	return result, total, nil
}

func (r *chatDAO) attachRoomTags(ctx context.Context, rooms []model.ChatRoomRow, tx ...*sql.Tx) {
	if len(rooms) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(rooms))
	for i := range rooms {
		ids[i] = rooms[i].ID
	}

	tagMap, _ := r.GetRoomTagsBatch(ctx, ids, tx...)
	for i := range rooms {
		rooms[i].Tags = tagMap[rooms[i].ID]
	}
}

func (r *chatDAO) GetRoomByID(ctx context.Context, s spec.ChatRoomViewer, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	var row model.ChatRoomRow
	var systemKind sql.NullString
	var viewerRole sql.NullString
	var viewerMuted, viewerGhost sql.NullBool
	var createdAt time.Time
	var lastMessageAt, archivedAt, lastReadAt sql.NullTime
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, cr.archived_at, m.last_read_at, m.role, m.muted, m.ghost,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL)
		 FROM chat_rooms cr
		 LEFT JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
		 WHERE cr.id = $2`,
		s.ViewerID, s.RoomID,
	).Scan(&row.ID, &row.Name, &row.Description, &row.Type, &row.IsPublic, &row.IsRP, &row.IsSystem, &systemKind, &row.CreatedBy, &createdAt, &lastMessageAt, &archivedAt, &lastReadAt, &viewerRole, &viewerMuted, &viewerGhost, &row.MemberCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room by id: %w", err)
	}
	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	row.LastMessageAt = nullTimeToString(lastMessageAt)
	row.ArchivedAt = nullTimeToString(archivedAt)
	row.LastReadAt = nullTimeToString(lastReadAt)
	if systemKind.Valid {
		row.SystemKind = systemKind.String
	}
	if viewerRole.Valid {
		row.ViewerRole = viewerRole.String
		row.IsMember = true
	}
	if viewerMuted.Valid {
		row.ViewerMuted = viewerMuted.Bool
	}
	if viewerGhost.Valid {
		row.ViewerGhost = viewerGhost.Bool
	}
	row.Tags, _ = r.GetRoomTags(ctx, s.RoomID, tx...)
	return &row, nil
}

func (r *chatDAO) GetRoomSendContext(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (*model.ChatRoomSendContext, error) {
	var row model.ChatRoomSendContext
	var systemKind sql.NullString
	var lastMessageAt sql.NullTime

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id, name, type, is_public, is_system, system_kind, created_by, last_message_at FROM chat_rooms WHERE id = $1`,
		roomID,
	).Scan(&row.ID, &row.Name, &row.Type, &row.IsPublic, &row.IsSystem, &systemKind, &row.CreatedBy, &lastMessageAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room send context: %w", err)
	}

	row.LastMessageAt = nullTimeToString(lastMessageAt)

	if systemKind.Valid {
		row.SystemKind = systemKind.String
	}

	return &row, nil
}

func (r *chatDAO) GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomMemberRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT m.user_id, u.username, u.display_name, u.avatar_url, m.role, COALESCE(ur.role, ''), m.joined_at, m.nickname, m.nickname_locked, m.avatar_url,
		 CASE WHEN m.timeout_until > NOW() THEN to_char(m.timeout_until AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') ELSE '' END,
		 CASE WHEN m.timeout_until > NOW() THEN m.timeout_set_by_staff ELSE FALSE END,
		 m.ghost
		 FROM chat_room_members m
		 JOIN users u ON m.user_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 WHERE m.room_id = $1 AND m.left_at IS NULL
		 ORDER BY CASE m.role WHEN 'host' THEN 0 ELSE 1 END, m.joined_at ASC`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room members detailed: %w", err)
	}
	defer rows.Close()

	var result []model.ChatRoomMemberRow
	for rows.Next() {
		var m model.ChatRoomMemberRow
		var joinedAt time.Time
		if err := rows.Scan(&m.UserID, &m.Username, &m.DisplayName, &m.AvatarURL, &m.Role, &m.AuthorRole, &joinedAt, &m.Nickname, &m.NicknameLocked, &m.MemberAvatarURL, &m.TimeoutUntil, &m.TimeoutByStaff, &m.Ghost); err != nil {
			return nil, fmt.Errorf("scan member detailed: %w", err)
		}
		m.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		m.AuthorRoleTyped = role.Role(m.AuthorRole)
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *chatDAO) ListPublicRooms(ctx context.Context, q spec.ChatPublicRoomFilter, tx ...*sql.Tx) ([]model.ChatRoomRow, int, error) {
	conditions := []string{"cr.type = 'group'", "cr.is_public = TRUE", "cr.is_system = FALSE"}
	if !q.IncludeArchived {
		conditions = append(conditions, "cr.archived_at IS NULL")
	}
	var countArgs []any
	idx := 1
	if q.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(cr.name ILIKE $%d OR cr.description ILIKE $%d)", idx, idx+1))
		wc := "%" + q.Search + "%"
		countArgs = append(countArgs, wc, wc)
		idx += 2
	}
	if q.IsRPOnly {
		conditions = append(conditions, "cr.is_rp = TRUE")
	}
	if q.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS(SELECT 1 FROM chat_room_tags WHERE room_id = cr.id AND tag = $%d)", idx))
		countArgs = append(countArgs, q.Tag)
		idx++
	}
	if q.ViewerID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("NOT EXISTS(SELECT 1 FROM chat_room_members WHERE room_id = cr.id AND user_id = $%d AND left_at IS NULL)", idx))
		countArgs = append(countArgs, q.ViewerID)
		idx++
	}
	countExclSQL, countExclArgs := ExcludeClauseNullable("cr.created_by", q.ExcludeUserIDs, idx)
	countArgs = append(countArgs, countExclArgs...)

	var whereCount strings.Builder
	whereCount.WriteString(" WHERE " + conditions[0])
	for _, c := range conditions[1:] {
		whereCount.WriteString(" AND " + c)
	}
	whereCount.WriteString(countExclSQL)

	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM chat_rooms cr"+whereCount.String(), countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count public rooms: %w", err)
	}

	queryArgs := []any{q.ViewerID}
	qConditions := []string{"cr.type = 'group'", "cr.is_public = TRUE", "cr.is_system = FALSE"}
	if !q.IncludeArchived {
		qConditions = append(qConditions, "cr.archived_at IS NULL")
	}
	qIdx := 2
	if q.Search != "" {
		qConditions = append(qConditions, fmt.Sprintf("(cr.name ILIKE $%d OR cr.description ILIKE $%d)", qIdx, qIdx+1))
		wc := "%" + q.Search + "%"
		queryArgs = append(queryArgs, wc, wc)
		qIdx += 2
	}
	if q.IsRPOnly {
		qConditions = append(qConditions, "cr.is_rp = TRUE")
	}
	if q.Tag != "" {
		qConditions = append(qConditions, fmt.Sprintf("EXISTS(SELECT 1 FROM chat_room_tags WHERE room_id = cr.id AND tag = $%d)", qIdx))
		queryArgs = append(queryArgs, q.Tag)
		qIdx++
	}
	if q.ViewerID != uuid.Nil {
		qConditions = append(qConditions, fmt.Sprintf("NOT EXISTS(SELECT 1 FROM chat_room_members WHERE room_id = cr.id AND user_id = $%d AND left_at IS NULL)", qIdx))
		queryArgs = append(queryArgs, q.ViewerID)
		qIdx++
	}
	qExclSQL, qExclArgs := ExcludeClauseNullable("cr.created_by", q.ExcludeUserIDs, qIdx)
	queryArgs = append(queryArgs, qExclArgs...)
	qIdx += len(qExclArgs)

	var whereQuery strings.Builder
	whereQuery.WriteString(" WHERE " + qConditions[0])
	for _, c := range qConditions[1:] {
		whereQuery.WriteString(" AND " + c)
	}
	whereQuery.WriteString(qExclSQL)

	limitClause := fmt.Sprintf(" LIMIT $%d OFFSET $%d", qIdx, qIdx+1)
	queryArgs = append(queryArgs, q.Limit, q.Offset)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, cr.archived_at,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL),
		 EXISTS(SELECT 1 FROM chat_room_members WHERE room_id = cr.id AND user_id = $1 AND left_at IS NULL),
		 `+hotScoreExpr+`
		 FROM chat_rooms cr`+whereQuery.String()+`
		 ORDER BY COALESCE(cr.last_message_at, cr.created_at) DESC`+limitClause,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list public rooms: %w", err)
	}
	defer rows.Close()

	var result []model.ChatRoomRow
	for rows.Next() {
		var row model.ChatRoomRow
		var systemKind sql.NullString
		var createdAt time.Time
		var lastMessageAt, archivedAt sql.NullTime
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.Type, &row.IsPublic, &row.IsRP, &row.IsSystem, &systemKind, &row.CreatedBy, &createdAt, &lastMessageAt, &archivedAt, &row.MemberCount, &row.IsMember, &row.HotScore); err != nil {
			return nil, 0, fmt.Errorf("scan public room: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.LastMessageAt = nullTimeToString(lastMessageAt)
		row.ArchivedAt = nullTimeToString(archivedAt)
		if systemKind.Valid {
			row.SystemKind = systemKind.String
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	r.attachRoomTags(ctx, result, tx...)
	return result, total, nil
}

func (r *chatDAO) GetRoomMembers(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT user_id FROM chat_room_members WHERE room_id = $1 AND left_at IS NULL`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}
	return utils.ScanIDs(rows, "member")
}

func (r *chatDAO) IsMember(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`, s.RoomID, s.UserID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return count > 0, nil
}

func (r *chatDAO) SetMuted(ctx context.Context, s spec.ChatMemberMuteUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET muted = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL`, s.Muted, s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("set muted: %w", err)
	}
	return nil
}

func (r *chatDAO) IsMuted(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	var muted bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT muted FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`, s.RoomID, s.UserID,
	).Scan(&muted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check muted: %w", err)
	}
	return muted, nil
}

func (r *chatDAO) GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT user_id FROM chat_room_members WHERE room_id = $1 AND muted = FALSE AND left_at IS NULL`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get unmuted members: %w", err)
	}
	return utils.ScanIDs(rows, "unmuted member")
}

func (r *chatDAO) SetVoiceForceMuted(ctx context.Context, s spec.ChatVoiceForceMuteUpdate, tx ...*sql.Tx) error {
	if !s.Muted {
		_, err := txOrDB(r.db, tx).ExecContext(ctx,
			`DELETE FROM chat_voice_force_mutes WHERE room_id = $1 AND user_id = $2`, s.RoomID, s.UserID,
		)
		if err != nil {
			return fmt.Errorf("clear voice force mute: %w", err)
		}
		return nil
	}

	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_voice_force_mutes (room_id, user_id, muted_by) VALUES ($1, $2, $3)
		 ON CONFLICT (room_id, user_id) DO UPDATE SET muted_by = EXCLUDED.muted_by`,
		s.RoomID, s.UserID, s.MutedBy,
	)
	if err != nil {
		return fmt.Errorf("set voice force mute: %w", err)
	}
	return nil
}

func (r *chatDAO) IsVoiceForceMuted(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	var muted bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM chat_voice_force_mutes WHERE room_id = $1 AND user_id = $2)`, s.RoomID, s.UserID,
	).Scan(&muted)
	if err != nil {
		return false, fmt.Errorf("check voice force mute: %w", err)
	}
	return muted, nil
}

func (r *chatDAO) ClearVoiceForceMutes(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM chat_voice_force_mutes WHERE room_id = $1`, roomID,
	)
	if err != nil {
		return fmt.Errorf("clear voice force mutes: %w", err)
	}
	return nil
}

func (r *chatDAO) FindDMRoom(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (uuid.UUID, error) {
	var id uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT cr.id FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
		 WHERE cr.type = 'dm' AND cr.dm_pair_key = $2
		 LIMIT 1`,
		s.UserA, dmPairKey(s.UserA, s.UserB),
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find dm room: %w", err)
	}
	return id, nil
}

func (r *chatDAO) InsertMessageRow(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	msg, err := scanMessageRow(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH ins AS (
			INSERT INTO chat_messages (room_id, sender_id, body, reply_to_id, is_system)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING *
		)
		SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 COALESCE(ur.role, ''),
		 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
		 parent.sender_id, COALESCE(NULLIF(pmem.nickname, ''), NULLIF(pu.display_name, ''), pu.username), parent.body,
		 cm.pinned_at, cm.pinned_by, cm.edited_at,
		 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
		 FROM ins cm
		 JOIN users u ON cm.sender_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
		 LEFT JOIN users pu ON parent.sender_id = pu.id
		 LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
		 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id`,
		s.RoomID, s.SenderID, s.Body, s.ReplyToID, s.IsSystem,
	))
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	return &msg, nil
}

func (r *chatDAO) TouchRoomActivityForMessage(ctx context.Context, s spec.ChatRoomActivityTouch, tx ...*sql.Tx) error {
	var err error

	if s.IsSystem {
		_, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE chat_rooms SET last_message_at = NOW() WHERE id = $1`,
			s.RoomID,
		)
	} else {
		_, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE chat_rooms SET last_message_at = NOW(), archived_at = NULL WHERE id = $1`,
			s.RoomID,
		)
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

const visibleToViewer = `
	AND cm.created_at >= COALESCE(
		(SELECT m.joined_at
		   FROM chat_room_members m
		   JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
		  WHERE m.room_id = cm.room_id AND m.user_id = $2),
		cm.created_at)`

func (r *chatDAO) getMessages(ctx context.Context, q spec.ChatMessagePage, tx ...*sql.Tx) ([]model.ChatMessageRow, int, error) {
	var total int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_messages cm WHERE cm.room_id = $1`+visibleToViewer, q.RoomID, q.ViewerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT * FROM (
			SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
			 COALESCE(ur.role, ''),
			 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
			 parent.sender_id, COALESCE(NULLIF(pmem.nickname, ''), NULLIF(pu.display_name, ''), pu.username), parent.body,
			 cm.pinned_at, cm.pinned_by, cm.edited_at,
			 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
			 FROM chat_messages cm
			 JOIN users u ON cm.sender_id = u.id
			 LEFT JOIN user_roles ur ON ur.user_id = u.id
			 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
			 LEFT JOIN users pu ON parent.sender_id = pu.id
			 LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
			 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
			 WHERE cm.room_id = $1`+visibleToViewer+`
			 ORDER BY cm.created_at DESC, cm.id DESC
			 LIMIT $3
		) sub ORDER BY sub.created_at ASC, sub.id ASC`,
		q.RoomID, q.ViewerID, q.Limit,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []model.ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, 0, err
		}
		messages = append(messages, msg)
	}
	return messages, total, rows.Err()
}

func (r *chatDAO) SearchMessagesForViewer(ctx context.Context, s spec.ChatMessageSearch, tx ...*sql.Tx) ([]model.SearchResult, int, error) {
	const (
		fromWhere = `FROM chat_messages cm
		 JOIN chat_room_members crm ON crm.room_id = cm.room_id AND crm.user_id = $1 AND crm.left_at IS NULL
		 JOIN chat_rooms cr ON cr.id = cm.room_id
		 JOIN users u ON cm.sender_id = u.id
		 CROSS JOIN q
		 WHERE cm.is_system = false
		   AND cr.system_kind IS DISTINCT FROM 'watch_party'
		   AND u.banned_at IS NULL AND u.locked_at IS NULL
		   AND ($3 = '00000000-0000-0000-0000-000000000000'::uuid OR cm.room_id = $3)
		   AND (cm.search_vector @@ q.tsq OR cm.body % q.qstr)`
		cte = `WITH q AS (SELECT websearch_to_tsquery('english', $2) AS tsq, $2 AS qstr)`
	)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, cte+`
		SELECT 'chat_message' AS entity_type, cm.id::text, cm.room_id::text,
		 COALESCE(NULLIF(cr.name, ''), 'Direct message') AS parent_title,
		 COALESCE(NULLIF(cr.name, ''), 'Direct message') AS title,
		 ts_headline('english', cm.body, q.tsq, `+SearchHeadlineOptions+`) AS snippet,
		 u.id::text, u.username, u.display_name, u.avatar_url,
		 cm.created_at,
		 (ts_rank_cd(cm.search_vector, q.tsq) + COALESCE(similarity(cm.body, q.qstr), 0))::float8 AS rank,
		 COUNT(*) OVER () AS total_count
		 `+fromWhere+`
		 ORDER BY rank DESC, cm.created_at DESC
		 LIMIT $4 OFFSET $5`,
		s.ViewerID, s.Query, s.RoomID, s.Limit, s.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search chat messages: %w", err)
	}
	defer rows.Close()

	return scanSearchRowsWithTotal(rows, s.Limit)
}

func scanMessageRow(row interface{ Scan(dest ...any) error }) (model.ChatMessageRow, error) {
	var msg model.ChatMessageRow
	var pinnedAt, editedAt sql.NullTime
	var pinnedBy uuid.NullUUID
	var createdAt time.Time
	if err := row.Scan(
		&msg.ID, &msg.RoomID, &msg.SenderID,
		&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL,
		&msg.SenderRole,
		&msg.Body, &msg.IsSystem, &createdAt, &msg.ReplyToID,
		&msg.ReplyToSenderID, &msg.ReplyToSenderName, &msg.ReplyToBody,
		&pinnedAt, &pinnedBy, &editedAt,
		&msg.SenderNickname, &msg.SenderMemberAvatar,
	); err != nil {
		return msg, fmt.Errorf("scan message: %w", err)
	}
	msg.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	if pinnedAt.Valid {
		msg.PinnedAt = new(pinnedAt.Time.UTC().Format(time.RFC3339))
	}
	if pinnedBy.Valid {
		msg.PinnedBy = new(pinnedBy.UUID)
	}
	if editedAt.Valid {
		msg.EditedAt = new(editedAt.Time.UTC().Format(time.RFC3339))
	}
	msg.SenderRoleTyped = role.Role(msg.SenderRole)
	return msg, nil
}

func attachmentPredicate(kind model.AttachmentKind) (string, error) {
	switch kind {
	case model.AttachmentKindMedia:
		return ` AND EXISTS (SELECT 1 FROM chat_message_media cmm WHERE cmm.message_id = cm.id)`, nil
	case model.AttachmentKindLinks:
		return ` AND cm.body LIKE '%http%'`, nil
	}

	return "", fmt.Errorf("list room attachments: unknown kind %q", kind)
}

func (r *chatDAO) ListRoomAttachments(ctx context.Context, q spec.ChatRoomAttachmentQuery, tx ...*sql.Tx) ([]model.ChatMessageRow, error) {
	predicate, err := attachmentPredicate(q.Kind)
	if err != nil {
		return nil, err
	}

	beforeTime, beforeID, err := splitMessageCursor(q.Before)
	if err != nil {
		return nil, fmt.Errorf("list room attachments: %w", err)
	}

	var cursor *time.Time
	if q.Before != "" {
		cursor = &beforeTime
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 COALESCE(ur.role, ''),
		 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
		 parent.sender_id, COALESCE(NULLIF(pmem.nickname, ''), NULLIF(pu.display_name, ''), pu.username), parent.body,
		 cm.pinned_at, cm.pinned_by, cm.edited_at,
		 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
		 LEFT JOIN users pu ON parent.sender_id = pu.id
		 LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
		 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
		 WHERE cm.room_id = $1 AND cm.is_system = FALSE AND (
			$2::timestamptz IS NULL
			OR cm.created_at < $2
			OR ($3 != '' AND cm.created_at = $2 AND cm.id::text < $3)
		 )`+predicate+strings.ReplaceAll(visibleToViewer, "$2", "$5")+`
		 ORDER BY cm.created_at DESC, cm.id DESC
		 LIMIT $4`,
		q.RoomID, cursor, beforeID, q.Limit, q.ViewerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list room attachments: %w", err)
	}
	defer rows.Close()

	var messages []model.ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
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

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT * FROM (
			SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
			 COALESCE(ur.role, ''),
			 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
			 parent.sender_id, COALESCE(NULLIF(pmem.nickname, ''), NULLIF(pu.display_name, ''), pu.username), parent.body,
			 cm.pinned_at, cm.pinned_by, cm.edited_at,
			 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
			 FROM chat_messages cm
			 JOIN users u ON cm.sender_id = u.id
			 LEFT JOIN user_roles ur ON ur.user_id = u.id
			 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
			 LEFT JOIN users pu ON parent.sender_id = pu.id
			 LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
			 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
			 WHERE cm.room_id = $1 AND (
				cm.created_at < $2 OR ($3 != '' AND cm.created_at = $2 AND cm.id::text < $3)
			 )`+strings.ReplaceAll(visibleToViewer, "$2", "$5")+`
			 ORDER BY cm.created_at DESC, cm.id DESC
			 LIMIT $4
		) sub ORDER BY sub.created_at ASC, sub.id ASC`,
		q.RoomID, beforeTime, beforeID, q.Limit, q.ViewerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages before: %w", err)
	}
	defer rows.Close()

	var messages []model.ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *chatDAO) GetMessageByID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	var msg model.ChatMessageRow
	var pinnedAt, editedAt sql.NullTime
	var pinnedBy uuid.NullUUID
	var createdAt time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 COALESCE(ur.role, ''),
		 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
		 parent.sender_id, COALESCE(NULLIF(pmem.nickname, ''), NULLIF(pu.display_name, ''), pu.username), parent.body,
		 cm.pinned_at, cm.pinned_by, cm.edited_at,
		 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
		 LEFT JOIN users pu ON parent.sender_id = pu.id
		 LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
		 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
		 WHERE cm.id = $1`,
		messageID,
	).Scan(
		&msg.ID, &msg.RoomID, &msg.SenderID,
		&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL,
		&msg.SenderRole,
		&msg.Body, &msg.IsSystem, &createdAt, &msg.ReplyToID,
		&msg.ReplyToSenderID, &msg.ReplyToSenderName, &msg.ReplyToBody,
		&pinnedAt, &pinnedBy, &editedAt,
		&msg.SenderNickname, &msg.SenderMemberAvatar,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	msg.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	if pinnedAt.Valid {
		msg.PinnedAt = new(pinnedAt.Time.UTC().Format(time.RFC3339))
	}
	if pinnedBy.Valid {
		msg.PinnedBy = new(pinnedBy.UUID)
	}
	if editedAt.Valid {
		msg.EditedAt = new(editedAt.Time.UTC().Format(time.RFC3339))
	}
	msg.SenderRoleTyped = role.Role(msg.SenderRole)
	return &msg, nil
}

func (r *chatDAO) GetMessageRoomID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var roomID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT room_id FROM chat_messages WHERE id = $1`, messageID,
	).Scan(&roomID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get message room id: %w", err)
	}
	return roomID, nil
}

func (r *chatDAO) DeleteMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chat_messages WHERE room_id = $1`, roomID)
	if err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	return nil
}

func (r *chatDAO) DeleteMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM chat_messages WHERE id = $1`, messageID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}

func (r *chatDAO) EditMessage(ctx context.Context, s spec.ChatMessageUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_messages SET body = $1, edited_at = NOW() WHERE id = $2`,
		s.Body, s.MessageID,
	)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	return nil
}

func (r *chatDAO) TouchRoomActivity(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_rooms SET last_message_at = NOW() WHERE id = $1`,
		roomID,
	)
	if err != nil {
		return fmt.Errorf("touch room activity: %w", err)
	}
	return nil
}

func (r *chatDAO) ArchiveStaleGroupRooms(ctx context.Context, cutoff time.Time, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cr.id FROM chat_rooms cr
		 WHERE cr.type = 'group'
		   AND cr.is_system = FALSE
		   AND cr.archived_at IS NULL
		   AND COALESCE(
		       (SELECT MAX(cm.created_at) FROM chat_messages cm WHERE cm.room_id = cr.id AND cm.is_system = FALSE),
		       cr.created_at
		   ) < $1`,
		cutoff.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("find stale chat rooms: %w", err)
	}
	ids, err := utils.ScanIDs(rows, "stale chat room id")
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	placeholders, args := utils.PlaceholderArgs(ids, 1)

	_, err = txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_rooms SET archived_at = NOW() WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("archive stale chat rooms: %w", err)
	}
	return ids, nil
}

func (r *chatDAO) MarkRoomRead(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET last_read_at = NOW() WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}
	return nil
}

func (r *chatDAO) GetMessageSenderID(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var senderID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT sender_id FROM chat_messages WHERE id = $1`, messageID,
	).Scan(&senderID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get message sender: %w", err)
	}
	return senderID, nil
}

func (r *chatDAO) AddMessageMedia(ctx context.Context, s spec.NewChatMessageMedia, tx ...*sql.Tx) (int64, error) {
	var id int64
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO chat_message_media (message_id, media_url, media_type, thumbnail_url, filename, sort_order, width, height, is_spoiler) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		s.TargetID, s.MediaURL, s.MediaType, s.ThumbnailURL, s.Filename, s.SortOrder, s.Width, s.Height, s.IsSpoiler,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add message media: %w", err)
	}
	return id, nil
}

func (r *chatDAO) UpdateMessageMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_message_media SET media_url = $1 WHERE id = $2`, s.URL, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update message media url: %w", err)
	}
	return nil
}

func (r *chatDAO) UpdateMessageMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_message_media SET thumbnail_url = $1 WHERE id = $2`, s.URL, s.ID,
	)
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

	placeholders, args := utils.PlaceholderArgs(messageIDs, 1)

	query := `SELECT id, message_id, media_url, media_type, thumbnail_url, COALESCE(filename, ''), sort_order, width, height, is_spoiler
	          FROM chat_message_media WHERE message_id IN (` + strings.Join(placeholders, ",") + `)
	          ORDER BY sort_order ASC, id ASC`

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get message media batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var msgID uuid.UUID
		var mediaURL, mediaType, thumbURL, filename string
		var sortOrder, width, height int
		var isSpoiler bool
		if err := rows.Scan(&id, &msgID, &mediaURL, &mediaType, &thumbURL, &filename, &sortOrder, &width, &height, &isSpoiler); err != nil {
			return nil, fmt.Errorf("scan message media: %w", err)
		}
		result[msgID] = append(result[msgID], dto.PostMediaResponse{
			ID:           int(id),
			MediaURL:     mediaURL,
			MediaType:    mediaType,
			ThumbnailURL: thumbURL,
			Filename:     filename,
			SortOrder:    sortOrder,
			Width:        width,
			Height:       height,
			IsSpoiler:    isSpoiler,
		})
	}
	return result, rows.Err()
}

func (r *chatDAO) SetMemberNickname(ctx context.Context, s spec.ChatMemberNicknameUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET nickname = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL`,
		s.Nickname, s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("set member nickname: %w", err)
	}
	return nil
}

func (r *chatDAO) SetMemberNicknameWithLock(ctx context.Context, s spec.ChatMemberNicknameUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET nickname = $1, nickname_locked = $2 WHERE room_id = $3 AND user_id = $4 AND left_at IS NULL`,
		s.Nickname, s.Locked, s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("set member nickname with lock: %w", err)
	}
	return nil
}

func (r *chatDAO) IsMemberNicknameLocked(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	var locked bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT nickname_locked FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check nickname locked: %w", err)
	}
	return locked, nil
}

func (r *chatDAO) SetMemberAvatar(ctx context.Context, s spec.ChatMemberAvatarUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET avatar_url = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL`,
		s.AvatarURL, s.RoomID, s.UserID,
	)
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
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET timeout_until = $1, timeout_set_by_staff = $2 WHERE room_id = $3 AND user_id = $4 AND left_at IS NULL`,
		t, s.ByStaff, s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("set member timeout: %w", err)
	}
	return nil
}

func (r *chatDAO) ClearMemberTimeout(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_room_members SET timeout_until = NULL, timeout_set_by_staff = FALSE WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("clear member timeout: %w", err)
	}
	return nil
}

func (r *chatDAO) HasActiveMemberTimeout(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, error) {
	var active bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT timeout_until > NOW()
		 FROM chat_room_members
		 WHERE room_id = $1 AND user_id = $2`,
		s.RoomID, s.UserID,
	).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check active member timeout: %w", err)
	}
	return active, nil
}

func (r *chatDAO) GetMemberTimeoutState(ctx context.Context, s spec.ChatMemberRef, tx ...*sql.Tx) (bool, string, bool, error) {
	var active sql.NullBool
	var until sql.NullTime
	var byStaff bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT timeout_until > NOW(),
		 timeout_until,
		 timeout_set_by_staff
		 FROM chat_room_members
		 WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`,
		s.RoomID, s.UserID,
	).Scan(&active, &until, &byStaff)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", false, nil
	}
	if err != nil {
		return false, "", false, fmt.Errorf("get member timeout state: %w", err)
	}
	if !until.Valid {
		return false, "", byStaff, nil
	}
	return active.Valid && active.Bool, until.Time.UTC().Format(time.RFC3339), byStaff, nil
}

func (r *chatDAO) PinMessage(ctx context.Context, s spec.ChatMessagePin, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_messages SET pinned_at = NOW(), pinned_by = $1 WHERE id = $2`,
		s.PinnedBy, s.MessageID,
	)
	if err != nil {
		return fmt.Errorf("pin message: %w", err)
	}
	return nil
}

func (r *chatDAO) UnpinMessage(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE chat_messages SET pinned_at = NULL, pinned_by = NULL WHERE id = $1`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("unpin message: %w", err)
	}
	return nil
}

func (r *chatDAO) ListPinnedMessages(ctx context.Context, s spec.ChatRoomViewer, tx ...*sql.Tx) ([]model.ChatMessageRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 COALESCE(ur.role, ''),
		 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
		 parent.sender_id, COALESCE(NULLIF(pmem.nickname, ''), NULLIF(pu.display_name, ''), pu.username), parent.body,
		 cm.pinned_at, cm.pinned_by, cm.edited_at,
		 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
		 LEFT JOIN users pu ON parent.sender_id = pu.id
		 LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
		 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
		 WHERE cm.room_id = $1 AND cm.pinned_at IS NOT NULL`+visibleToViewer+`
		 ORDER BY cm.pinned_at DESC`,
		s.RoomID, s.ViewerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pinned messages: %w", err)
	}
	defer rows.Close()

	var messages []model.ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *chatDAO) AddReaction(ctx context.Context, s spec.ChatMessageReaction, tx ...*sql.Tx) (bool, error) {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO chat_message_reactions (message_id, user_id, emoji) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		s.MessageID, s.UserID, s.Emoji,
	)
	if err != nil {
		return false, fmt.Errorf("add reaction: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("add reaction rows: %w", err)
	}
	return n > 0, nil
}

func (r *chatDAO) RemoveReaction(ctx context.Context, s spec.ChatMessageReaction, tx ...*sql.Tx) (bool, error) {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM chat_message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`,
		s.MessageID, s.UserID, s.Emoji,
	)
	if err != nil {
		return false, fmt.Errorf("remove reaction: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("remove reaction rows: %w", err)
	}
	return n > 0, nil
}

func (r *chatDAO) CountReactions(ctx context.Context, s spec.ChatReactionCount, tx ...*sql.Tx) (int, error) {
	var n int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_message_reactions WHERE message_id = $1 AND emoji = $2`,
		s.MessageID, s.Emoji,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count reactions: %w", err)
	}
	return n, nil
}

func (r *chatDAO) GetReactionsBatch(ctx context.Context, q spec.ChatReactionsQuery, tx ...*sql.Tx) (map[uuid.UUID][]model.ReactionGroup, error) {
	result := make(map[uuid.UUID][]model.ReactionGroup)
	if len(q.MessageIDs) == 0 {
		return result, nil
	}

	placeholders, idArgs := utils.PlaceholderArgs(q.MessageIDs, 2)
	args := append([]any{q.ViewerID}, idArgs...)

	query := `SELECT r.message_id, r.emoji, COUNT(*) AS cnt,
	          BOOL_OR(r.user_id = $1) AS viewer_reacted,
	          STRING_AGG(u.display_name, E'\n') AS names
	          FROM chat_message_reactions r
	          JOIN users u ON u.id = r.user_id
	          WHERE r.message_id IN (` + strings.Join(placeholders, ",") + `)
	          GROUP BY r.message_id, r.emoji
	          ORDER BY cnt DESC, r.emoji ASC`

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get reactions batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msgID uuid.UUID
		var emoji string
		var count int
		var viewerReacted bool
		var names sql.NullString
		if err := rows.Scan(&msgID, &emoji, &count, &viewerReacted, &names); err != nil {
			return nil, fmt.Errorf("scan reaction group: %w", err)
		}
		var displayNames []string
		if names.Valid && names.String != "" {
			displayNames = strings.Split(names.String, "\n")
		}
		result[msgID] = append(result[msgID], model.ReactionGroup{
			Emoji:         emoji,
			Count:         count,
			ViewerReacted: viewerReacted,
			DisplayNames:  displayNames,
		})
	}
	return result, rows.Err()
}

func (r *chatDAO) CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
		 WHERE cr.type = 'dm'
		   AND m.muted = FALSE
		   AND cr.last_message_at IS NOT NULL
		   AND (m.last_read_at IS NULL OR cr.last_message_at > m.last_read_at)`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread dm rooms: %w", err)
	}
	return count, nil
}
