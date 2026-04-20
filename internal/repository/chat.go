package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
		MemberCount   int
		ViewerRole    string
		ViewerMuted   bool
		ViewerGhost   bool
		IsMember      bool
		Tags          []string
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
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]ChatRoomRow, error)
		ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, limit, offset int) ([]ChatRoomRow, int, error)
		GetRoomByID(ctx context.Context, roomID, viewerID uuid.UUID) (*ChatRoomRow, error)
		GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
		GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID) ([]ChatRoomMemberRow, error)
		GetMemberRole(ctx context.Context, roomID, userID uuid.UUID) (string, error)
		IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		SetMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error
		IsMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
		ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, limit, offset int) ([]ChatRoomRow, int, error)
		FindDMRoom(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, error)
		AddRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error
		ReplaceRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error
		GetRoomTags(ctx context.Context, roomID uuid.UUID) ([]string, error)
		GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]string, error)

		InsertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string, replyToID *uuid.UUID) error
		InsertSystemMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string) error
		EditMessage(ctx context.Context, messageID uuid.UUID, body string) error
		GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]ChatMessageRow, int, error)
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
		GetReactionsBatch(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]ReactionGroup, error)
	}

	chatRepository struct {
		db *sql.DB
	}
)

func (r *chatRepository) CreateRoom(ctx context.Context, id uuid.UUID, name, description, roomType string, isPublic, isRP bool, createdBy uuid.UUID) error {
	publicInt := 0
	if isPublic {
		publicInt = 1
	}
	rpInt := 0
	if isRP {
		rpInt = 1
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_rooms (id, name, description, type, is_public, is_rp, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, description, roomType, publicInt, rpInt, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

func (r *chatRepository) CreateSystemRoom(ctx context.Context, id uuid.UUID, name, description, systemKind string, createdBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_rooms (id, name, description, type, is_public, is_rp, is_system, system_kind, created_by) VALUES (?, ?, ?, 'group', 0, 0, 1, ?, ?)`,
		id, name, description, systemKind, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create system room: %w", err)
	}
	return nil
}

func (r *chatRepository) GetSystemRoomID(ctx context.Context, systemKind string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM chat_rooms WHERE system_kind = ? LIMIT 1`, systemKind,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get system room id: %w", err)
	}
	return id, nil
}

func (r *chatRepository) AddRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		_, err := r.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO chat_room_tags (room_id, tag) VALUES (?, ?)`,
			roomID, tag,
		)
		if err != nil {
			return fmt.Errorf("add room tag: %w", err)
		}
	}
	return nil
}

func (r *chatRepository) ReplaceRoomTags(ctx context.Context, roomID uuid.UUID, tags []string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM chat_room_tags WHERE room_id = ?`, roomID); err != nil {
		return fmt.Errorf("delete room tags: %w", err)
	}
	return r.AddRoomTags(ctx, roomID, tags)
}

func (r *chatRepository) GetRoomTags(ctx context.Context, roomID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT tag FROM chat_room_tags WHERE room_id = ? ORDER BY tag`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room tags: %w", err)
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan room tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *chatRepository) GetRoomTagsBatch(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	result := make(map[uuid.UUID][]string)
	if len(roomIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(roomIDs))
	args := make([]interface{}, len(roomIDs))
	for i := range roomIDs {
		placeholders[i] = "?"
		args[i] = roomIDs[i]
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT room_id, tag FROM chat_room_tags WHERE room_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY tag`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get room tags batch: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var roomID uuid.UUID
		var tag string
		if err := rows.Scan(&roomID, &tag); err != nil {
			return nil, fmt.Errorf("scan room tag batch: %w", err)
		}
		result[roomID] = append(result[roomID], tag)
	}
	return result, rows.Err()
}

func (r *chatRepository) AddMemberWithRole(ctx context.Context, roomID, userID uuid.UUID, role string, ghost bool) error {
	ghostInt := 0
	if ghost {
		ghostInt = 1
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id, role, ghost) VALUES (?, ?, ?, ?)
		 ON CONFLICT(room_id, user_id) DO UPDATE SET left_at = NULL, role = excluded.role, ghost = excluded.ghost, joined_at = CURRENT_TIMESTAMP`,
		roomID, userID, role, ghostInt,
	)
	if err != nil {
		return fmt.Errorf("add member with role: %w", err)
	}
	return nil
}

func (r *chatRepository) IsGhostMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var g int
	err := r.db.QueryRowContext(ctx,
		`SELECT ghost FROM chat_room_members WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	).Scan(&g)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get ghost flag: %w", err)
	}
	return g != 0, nil
}

func (r *chatRepository) HasGhostMembers(ctx context.Context, roomID uuid.UUID) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM chat_room_members WHERE room_id = ? AND ghost = 1 AND left_at IS NULL`,
		roomID,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("count ghost members: %w", err)
	}
	return n > 0, nil
}

func (r *chatRepository) SetMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET role = ? WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		role, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set member role: %w", err)
	}
	return nil
}

func (r *chatRepository) GetMemberRole(ctx context.Context, roomID, userID uuid.UUID) (string, error) {
	var role string
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM chat_room_members WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}
	return role, nil
}

func dmPairKey(a, b uuid.UUID) string {
	sa, sb := a.String(), b.String()
	if sa > sb {
		sa, sb = sb, sa
	}
	return sa + ":" + sb
}

func (r *chatRepository) CreateDMRoomAtomic(ctx context.Context, id, userA, userB uuid.UUID) (uuid.UUID, error) {
	pairKey := dmPairKey(userA, userB)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create dm: begin tx: %w", err)
	}
	defer tx.Rollback()

	var existing uuid.UUID
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM chat_rooms WHERE type = 'dm' AND dm_pair_key = ?`,
		pairKey,
	).Scan(&existing)
	if err == nil {
		if cErr := tx.Commit(); cErr != nil {
			return uuid.Nil, fmt.Errorf("create dm: commit existing: %w", cErr)
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("create dm: lookup: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO chat_rooms (id, name, type, created_by, dm_pair_key) VALUES (?, '', 'dm', ?, ?)`,
		id, userA, pairKey,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create dm: insert room: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id) VALUES (?, ?), (?, ?)`,
		id, userA, id, userB,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create dm: insert members: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("create dm: commit: %w", err)
	}
	return id, nil
}

func (r *chatRepository) CountRoomMembers(ctx context.Context, roomID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = ? AND left_at IS NULL`, roomID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count room members: %w", err)
	}
	return count, nil
}

func (r *chatRepository) DeleteRoom(ctx context.Context, roomID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_rooms WHERE id = ?`, roomID)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}
	return nil
}

func (r *chatRepository) AddMember(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_room_members (room_id, user_id) VALUES (?, ?)
		 ON CONFLICT(room_id, user_id) DO UPDATE SET left_at = NULL, joined_at = CURRENT_TIMESTAMP`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *chatRepository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET left_at = CURRENT_TIMESTAMP WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

func (r *chatRepository) GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]ChatRoomRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, m.last_read_at, m.role, m.muted, m.ghost,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL)
		 FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = ? AND m.left_at IS NULL
		 ORDER BY cr.is_system DESC, COALESCE(cr.last_message_at, cr.created_at) DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}
	defer rows.Close()

	var result []ChatRoomRow
	for rows.Next() {
		var row ChatRoomRow
		var publicInt, rpInt, systemInt int
		var mutedInt, ghostInt int
		var systemKind sql.NullString
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.Type, &publicInt, &rpInt, &systemInt, &systemKind, &row.CreatedBy, &row.CreatedAt, &row.LastMessageAt, &row.LastReadAt, &row.ViewerRole, &mutedInt, &ghostInt, &row.MemberCount); err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}
		row.IsPublic = publicInt != 0
		row.IsRP = rpInt != 0
		row.IsSystem = systemInt != 0
		row.ViewerMuted = mutedInt != 0
		row.ViewerGhost = ghostInt != 0
		if systemKind.Valid {
			row.SystemKind = systemKind.String
		}
		row.IsMember = true
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(result) > 0 {
		ids := make([]uuid.UUID, len(result))
		for i := range result {
			ids[i] = result[i].ID
		}
		tagMap, _ := r.GetRoomTagsBatch(ctx, ids)
		for i := range result {
			result[i].Tags = tagMap[result[i].ID]
		}
	}
	return result, nil
}

func (r *chatRepository) ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, limit, offset int) ([]ChatRoomRow, int, error) {
	conditions := []string{"cr.type = 'group'", "m.user_id = ?", "m.left_at IS NULL"}
	args := []interface{}{userID}
	if search != "" {
		conditions = append(conditions, "(cr.name LIKE ? OR cr.description LIKE ?)")
		wc := "%" + search + "%"
		args = append(args, wc, wc)
	}
	if isRPOnly {
		conditions = append(conditions, "cr.is_rp = 1")
	}
	if tag != "" {
		conditions = append(conditions, "EXISTS(SELECT 1 FROM chat_room_tags WHERE room_id = cr.id AND tag = ?)")
		args = append(args, tag)
	}
	if role == "host" {
		conditions = append(conditions, "m.role = 'host'")
	} else if role == "member" {
		conditions = append(conditions, "m.role != 'host'")
	}

	where := " WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		where += " AND " + c
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id`+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user group rooms: %w", err)
	}

	queryArgs := make([]interface{}, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, m.last_read_at, m.role, m.muted, m.ghost,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL)
		 FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id`+where+`
		 ORDER BY cr.is_system DESC, COALESCE(cr.last_message_at, cr.created_at) DESC
		 LIMIT ? OFFSET ?`, queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list user group rooms: %w", err)
	}
	defer rows.Close()

	var result []ChatRoomRow
	for rows.Next() {
		var row ChatRoomRow
		var publicInt, rpInt, systemInt int
		var mutedInt, ghostInt int
		var systemKind sql.NullString
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.Type, &publicInt, &rpInt, &systemInt, &systemKind, &row.CreatedBy, &row.CreatedAt, &row.LastMessageAt, &row.LastReadAt, &row.ViewerRole, &mutedInt, &ghostInt, &row.MemberCount); err != nil {
			return nil, 0, fmt.Errorf("scan user group room: %w", err)
		}
		row.IsPublic = publicInt != 0
		row.IsRP = rpInt != 0
		row.IsSystem = systemInt != 0
		row.ViewerMuted = mutedInt != 0
		row.ViewerGhost = ghostInt != 0
		if systemKind.Valid {
			row.SystemKind = systemKind.String
		}
		row.IsMember = true
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(result) > 0 {
		ids := make([]uuid.UUID, len(result))
		for i := range result {
			ids[i] = result[i].ID
		}
		tagMap, _ := r.GetRoomTagsBatch(ctx, ids)
		for i := range result {
			result[i].Tags = tagMap[result[i].ID]
		}
	}
	return result, total, nil
}

func (r *chatRepository) GetRoomByID(ctx context.Context, roomID, viewerID uuid.UUID) (*ChatRoomRow, error) {
	var row ChatRoomRow
	var publicInt, rpInt, systemInt int
	var systemKind sql.NullString
	var viewerRole sql.NullString
	var viewerMuted, viewerGhost sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, m.last_read_at, m.role, m.muted, m.ghost,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL)
		 FROM chat_rooms cr
		 LEFT JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = ? AND m.left_at IS NULL
		 WHERE cr.id = ?`,
		viewerID, roomID,
	).Scan(&row.ID, &row.Name, &row.Description, &row.Type, &publicInt, &rpInt, &systemInt, &systemKind, &row.CreatedBy, &row.CreatedAt, &row.LastMessageAt, &row.LastReadAt, &viewerRole, &viewerMuted, &viewerGhost, &row.MemberCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room by id: %w", err)
	}
	row.IsPublic = publicInt != 0
	row.IsRP = rpInt != 0
	row.IsSystem = systemInt != 0
	if systemKind.Valid {
		row.SystemKind = systemKind.String
	}
	if viewerRole.Valid {
		row.ViewerRole = viewerRole.String
		row.IsMember = true
	}
	if viewerMuted.Valid {
		row.ViewerMuted = viewerMuted.Int64 != 0
	}
	if viewerGhost.Valid {
		row.ViewerGhost = viewerGhost.Int64 != 0
	}
	row.Tags, _ = r.GetRoomTags(ctx, roomID)
	return &row, nil
}

func (r *chatRepository) GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID) ([]ChatRoomMemberRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.user_id, u.username, u.display_name, u.avatar_url, m.role, COALESCE(ur.role, ''), m.joined_at, m.nickname, m.nickname_locked, m.avatar_url,
		 COALESCE(CASE WHEN datetime(m.timeout_until) > CURRENT_TIMESTAMP THEN strftime('%Y-%m-%dT%H:%M:%SZ', m.timeout_until) ELSE '' END, ''),
		 CASE WHEN datetime(m.timeout_until) > CURRENT_TIMESTAMP THEN m.timeout_set_by_staff ELSE 0 END,
		 m.ghost
		 FROM chat_room_members m
		 JOIN users u ON m.user_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 WHERE m.room_id = ? AND m.left_at IS NULL
		 ORDER BY CASE m.role WHEN 'host' THEN 0 ELSE 1 END, m.joined_at ASC`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room members detailed: %w", err)
	}
	defer rows.Close()

	var result []ChatRoomMemberRow
	for rows.Next() {
		var m ChatRoomMemberRow
		var lockedInt int
		var timeoutByStaffInt int
		var ghostInt int
		if err := rows.Scan(&m.UserID, &m.Username, &m.DisplayName, &m.AvatarURL, &m.Role, &m.AuthorRole, &m.JoinedAt, &m.Nickname, &lockedInt, &m.MemberAvatarURL, &m.TimeoutUntil, &timeoutByStaffInt, &ghostInt); err != nil {
			return nil, fmt.Errorf("scan member detailed: %w", err)
		}
		m.NicknameLocked = lockedInt != 0
		m.TimeoutByStaff = timeoutByStaffInt != 0
		m.Ghost = ghostInt != 0
		m.AuthorRoleTyped = role.Role(m.AuthorRole)
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *chatRepository) ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, excludeUserIDs []uuid.UUID, limit, offset int) ([]ChatRoomRow, int, error) {
	conditions := []string{"cr.type = 'group'", "cr.is_public = 1", "cr.is_system = 0"}
	var args []interface{}
	if search != "" {
		conditions = append(conditions, "(cr.name LIKE ? OR cr.description LIKE ?)")
		wc := "%" + search + "%"
		args = append(args, wc, wc)
	}
	if isRPOnly {
		conditions = append(conditions, "cr.is_rp = 1")
	}
	if tag != "" {
		conditions = append(conditions, "EXISTS(SELECT 1 FROM chat_room_tags WHERE room_id = cr.id AND tag = ?)")
		args = append(args, tag)
	}
	if viewerID != uuid.Nil {
		conditions = append(conditions, "NOT EXISTS(SELECT 1 FROM chat_room_members WHERE room_id = cr.id AND user_id = ? AND left_at IS NULL)")
		args = append(args, viewerID)
	}
	exclSQL, exclArgs := ExcludeClause("cr.created_by", excludeUserIDs)
	args = append(args, exclArgs...)

	where := " WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		where += " AND " + c
	}
	where += exclSQL

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM chat_rooms cr"+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count public rooms: %w", err)
	}

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at,
		 (SELECT COUNT(*) FROM chat_room_members WHERE room_id = cr.id AND left_at IS NULL),
		 EXISTS(SELECT 1 FROM chat_room_members WHERE room_id = cr.id AND user_id = ? AND left_at IS NULL)
		 FROM chat_rooms cr`+where+`
		 ORDER BY COALESCE(cr.last_message_at, cr.created_at) DESC
		 LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list public rooms: %w", err)
	}
	defer rows.Close()

	var result []ChatRoomRow
	for rows.Next() {
		var row ChatRoomRow
		var publicInt, rpInt, systemInt, isMemberInt int
		var systemKind sql.NullString
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.Type, &publicInt, &rpInt, &systemInt, &systemKind, &row.CreatedBy, &row.CreatedAt, &row.LastMessageAt, &row.MemberCount, &isMemberInt); err != nil {
			return nil, 0, fmt.Errorf("scan public room: %w", err)
		}
		row.IsPublic = publicInt != 0
		row.IsRP = rpInt != 0
		row.IsSystem = systemInt != 0
		if systemKind.Valid {
			row.SystemKind = systemKind.String
		}
		row.IsMember = isMemberInt != 0
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(result) > 0 {
		ids := make([]uuid.UUID, len(result))
		for i := range result {
			ids[i] = result[i].ID
		}
		tagMap, _ := r.GetRoomTagsBatch(ctx, ids)
		for i := range result {
			result[i].Tags = tagMap[result[i].ID]
		}
	}
	return result, total, nil
}

func (r *chatRepository) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM chat_room_members WHERE room_id = ? AND left_at IS NULL`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}
	defer rows.Close()

	var members []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, uid)
	}
	return members, rows.Err()
}

func (r *chatRepository) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = ? AND user_id = ? AND left_at IS NULL`, roomID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return count > 0, nil
}

func (r *chatRepository) SetMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error {
	v := 0
	if muted {
		v = 1
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET muted = ? WHERE room_id = ? AND user_id = ? AND left_at IS NULL`, v, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set muted: %w", err)
	}
	return nil
}

func (r *chatRepository) IsMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var muted int
	err := r.db.QueryRowContext(ctx,
		`SELECT muted FROM chat_room_members WHERE room_id = ? AND user_id = ? AND left_at IS NULL`, roomID, userID,
	).Scan(&muted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check muted: %w", err)
	}
	return muted != 0, nil
}

func (r *chatRepository) GetRoomMembersUnmuted(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM chat_room_members WHERE room_id = ? AND muted = 0 AND left_at IS NULL`, roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("get unmuted members: %w", err)
	}
	defer rows.Close()

	var members []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan unmuted member: %w", err)
		}
		members = append(members, id)
	}
	return members, rows.Err()
}

func (r *chatRepository) FindDMRoom(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT cr.id FROM chat_rooms cr
		 JOIN chat_room_members m1 ON cr.id = m1.room_id AND m1.user_id = ? AND m1.left_at IS NULL
		 JOIN chat_room_members m2 ON cr.id = m2.room_id AND m2.user_id = ? AND m2.left_at IS NULL
		 WHERE cr.type = 'dm'
		 LIMIT 1`,
		userA, userB,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find dm room: %w", err)
	}
	return id, nil
}

func (r *chatRepository) InsertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string, replyToID *uuid.UUID) error {
	return r.insertMessage(ctx, id, roomID, senderID, body, replyToID, false)
}

func (r *chatRepository) InsertSystemMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string) error {
	return r.insertMessage(ctx, id, roomID, senderID, body, nil, true)
}

func (r *chatRepository) insertMessage(ctx context.Context, id, roomID, senderID uuid.UUID, body string, replyToID *uuid.UUID, isSystem bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("insert message: begin tx: %w", err)
	}
	defer tx.Rollback()

	isSystemInt := 0
	if isSystem {
		isSystemInt = 1
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO chat_messages (id, room_id, sender_id, body, reply_to_id, is_system) VALUES (?, ?, ?, ?, ?, ?)`,
		id, roomID, senderID, body, replyToID, isSystemInt,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE chat_rooms SET last_message_at = CURRENT_TIMESTAMP WHERE id = ?`,
		roomID,
	)
	if err != nil {
		return fmt.Errorf("touch room activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("insert message: commit: %w", err)
	}
	return nil
}

func (r *chatRepository) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]ChatMessageRow, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_messages WHERE room_id = ?`, roomID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT * FROM (
			SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
			 COALESCE(ur.role, ''),
			 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
			 parent.sender_id, pu.display_name, parent.body,
			 cm.pinned_at, cm.pinned_by, cm.edited_at,
			 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
			 FROM chat_messages cm
			 JOIN users u ON cm.sender_id = u.id
			 LEFT JOIN user_roles ur ON ur.user_id = u.id
			 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
			 LEFT JOIN users pu ON parent.sender_id = pu.id
			 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
			 WHERE cm.room_id = ?
			 ORDER BY datetime(cm.created_at) DESC, cm.id DESC
			 LIMIT ?
		) sub ORDER BY datetime(sub.created_at) ASC, sub.id ASC`,
		roomID, limit,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, 0, err
		}
		messages = append(messages, msg)
	}
	return messages, total, rows.Err()
}

func scanMessageRow(rows *sql.Rows) (ChatMessageRow, error) {
	var msg ChatMessageRow
	var pinnedAt sql.NullString
	var pinnedBy sql.NullString
	var editedAt sql.NullString
	var isSystemInt int
	if err := rows.Scan(
		&msg.ID, &msg.RoomID, &msg.SenderID,
		&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL,
		&msg.SenderRole,
		&msg.Body, &isSystemInt, &msg.CreatedAt, &msg.ReplyToID,
		&msg.ReplyToSenderID, &msg.ReplyToSenderName, &msg.ReplyToBody,
		&pinnedAt, &pinnedBy, &editedAt,
		&msg.SenderNickname, &msg.SenderMemberAvatar,
	); err != nil {
		return msg, fmt.Errorf("scan message: %w", err)
	}
	if pinnedAt.Valid {
		s := pinnedAt.String
		msg.PinnedAt = &s
	}
	if pinnedBy.Valid {
		if parsed, err := uuid.Parse(pinnedBy.String); err == nil {
			msg.PinnedBy = &parsed
		}
	}
	if editedAt.Valid {
		s := editedAt.String
		msg.EditedAt = &s
	}
	msg.IsSystem = isSystemInt != 0
	msg.SenderRoleTyped = role.Role(msg.SenderRole)
	return msg, nil
}

func (r *chatRepository) GetMessagesBefore(ctx context.Context, roomID uuid.UUID, before string, limit int) ([]ChatMessageRow, error) {
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

	rows, err := r.db.QueryContext(ctx,
		`SELECT * FROM (
			SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
			 COALESCE(ur.role, ''),
			 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
			 parent.sender_id, pu.display_name, parent.body,
			 cm.pinned_at, cm.pinned_by, cm.edited_at,
			 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
			 FROM chat_messages cm
			 JOIN users u ON cm.sender_id = u.id
			 LEFT JOIN user_roles ur ON ur.user_id = u.id
			 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
			 LEFT JOIN users pu ON parent.sender_id = pu.id
			 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
			 WHERE cm.room_id = ? AND (
				datetime(cm.created_at) < datetime(?) OR (? != '' AND datetime(cm.created_at) = datetime(?) AND cm.id < ?)
			 )
			 ORDER BY datetime(cm.created_at) DESC, cm.id DESC
			 LIMIT ?
		) sub ORDER BY datetime(sub.created_at) ASC, sub.id ASC`,
		roomID, beforeTS, beforeID, beforeTS, beforeID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages before: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *chatRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessageRow, error) {
	var msg ChatMessageRow
	var pinnedAt sql.NullString
	var pinnedBy sql.NullString
	var editedAt sql.NullString
	var isSystemInt int
	err := r.db.QueryRowContext(ctx,
		`SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 COALESCE(ur.role, ''),
		 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
		 parent.sender_id, pu.display_name, parent.body,
		 cm.pinned_at, cm.pinned_by, cm.edited_at,
		 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
		 LEFT JOIN users pu ON parent.sender_id = pu.id
		 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
		 WHERE cm.id = ?`,
		messageID,
	).Scan(
		&msg.ID, &msg.RoomID, &msg.SenderID,
		&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL,
		&msg.SenderRole,
		&msg.Body, &isSystemInt, &msg.CreatedAt, &msg.ReplyToID,
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
	if pinnedAt.Valid {
		s := pinnedAt.String
		msg.PinnedAt = &s
	}
	if pinnedBy.Valid {
		if parsed, err := uuid.Parse(pinnedBy.String); err == nil {
			msg.PinnedBy = &parsed
		}
	}
	if editedAt.Valid {
		s := editedAt.String
		msg.EditedAt = &s
	}
	msg.IsSystem = isSystemInt != 0
	msg.SenderRoleTyped = role.Role(msg.SenderRole)
	return &msg, nil
}

func (r *chatRepository) GetMessageRoomID(ctx context.Context, messageID uuid.UUID) (uuid.UUID, error) {
	var roomID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT room_id FROM chat_messages WHERE id = ?`, messageID,
	).Scan(&roomID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get message room id: %w", err)
	}
	return roomID, nil
}

func (r *chatRepository) DeleteMessages(ctx context.Context, roomID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_messages WHERE room_id = ?`, roomID)
	if err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	return nil
}

func (r *chatRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_messages WHERE id = ?`, messageID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}

func (r *chatRepository) EditMessage(ctx context.Context, messageID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_messages SET body = ?, edited_at = CURRENT_TIMESTAMP WHERE id = ?`,
		body, messageID,
	)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	return nil
}

func (r *chatRepository) TouchRoomActivity(ctx context.Context, roomID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_rooms SET last_message_at = CURRENT_TIMESTAMP WHERE id = ?`,
		roomID,
	)
	if err != nil {
		return fmt.Errorf("touch room activity: %w", err)
	}
	return nil
}

func (r *chatRepository) MarkRoomRead(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET last_read_at = CURRENT_TIMESTAMP WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}
	return nil
}

func (r *chatRepository) GetMessageSenderID(ctx context.Context, messageID uuid.UUID) (uuid.UUID, error) {
	var senderID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT sender_id FROM chat_messages WHERE id = ?`, messageID,
	).Scan(&senderID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get message sender: %w", err)
	}
	return senderID, nil
}

func (r *chatRepository) AddMessageMedia(ctx context.Context, messageID uuid.UUID, mediaURL, mediaType, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO chat_message_media (message_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		messageID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add message media: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("add message media: last insert id: %w", err)
	}
	return id, nil
}

func (r *chatRepository) UpdateMessageMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_message_media SET media_url = ? WHERE id = ?`, mediaURL, id,
	)
	if err != nil {
		return fmt.Errorf("update message media url: %w", err)
	}
	return nil
}

func (r *chatRepository) UpdateMessageMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_message_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id,
	)
	if err != nil {
		return fmt.Errorf("update message media thumbnail: %w", err)
	}
	return nil
}

func (r *chatRepository) GetMessageMediaBatch(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]dto.PostMediaResponse, error) {
	result := make(map[uuid.UUID][]dto.PostMediaResponse)
	if len(messageIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i := 0; i < len(messageIDs); i++ {
		placeholders[i] = "?"
		args[i] = messageIDs[i]
	}

	query := `SELECT id, message_id, media_url, media_type, thumbnail_url, sort_order
	          FROM chat_message_media WHERE message_id IN (` + strings.Join(placeholders, ",") + `)
	          ORDER BY sort_order ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get message media batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var msgID uuid.UUID
		var mediaURL, mediaType, thumbURL string
		var sortOrder int
		if err := rows.Scan(&id, &msgID, &mediaURL, &mediaType, &thumbURL, &sortOrder); err != nil {
			return nil, fmt.Errorf("scan message media: %w", err)
		}
		result[msgID] = append(result[msgID], dto.PostMediaResponse{
			ID:           int(id),
			MediaURL:     mediaURL,
			MediaType:    mediaType,
			ThumbnailURL: thumbURL,
			SortOrder:    sortOrder,
		})
	}
	return result, rows.Err()
}

func (r *chatRepository) SetMemberNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET nickname = ? WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		nickname, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set member nickname: %w", err)
	}
	return nil
}

func (r *chatRepository) SetMemberNicknameWithLock(ctx context.Context, roomID, userID uuid.UUID, nickname string, locked bool) error {
	lockedInt := 0
	if locked {
		lockedInt = 1
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET nickname = ?, nickname_locked = ? WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		nickname, lockedInt, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set member nickname with lock: %w", err)
	}
	return nil
}

func (r *chatRepository) IsMemberNicknameLocked(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var locked int
	err := r.db.QueryRowContext(ctx,
		`SELECT nickname_locked FROM chat_room_members WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check nickname locked: %w", err)
	}
	return locked != 0, nil
}

func (r *chatRepository) SetMemberAvatar(ctx context.Context, roomID, userID uuid.UUID, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET avatar_url = ? WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		avatarURL, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set member avatar: %w", err)
	}
	return nil
}

func (r *chatRepository) SetMemberTimeout(ctx context.Context, roomID, userID uuid.UUID, until string, byStaff bool) error {
	byStaffInt := 0
	if byStaff {
		byStaffInt = 1
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET timeout_until = ?, timeout_set_by_staff = ? WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		until, byStaffInt, roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("set member timeout: %w", err)
	}
	return nil
}

func (r *chatRepository) ClearMemberTimeout(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_room_members SET timeout_until = NULL, timeout_set_by_staff = 0 WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("clear member timeout: %w", err)
	}
	return nil
}

func (r *chatRepository) HasActiveMemberTimeout(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var activeInt int
	err := r.db.QueryRowContext(ctx,
		`SELECT CASE WHEN datetime(timeout_until) > CURRENT_TIMESTAMP THEN 1 ELSE 0 END
		 FROM chat_room_members
		 WHERE room_id = ? AND user_id = ?`,
		roomID, userID,
	).Scan(&activeInt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check active member timeout: %w", err)
	}
	return activeInt != 0, nil
}

func (r *chatRepository) GetMemberTimeoutState(ctx context.Context, roomID, userID uuid.UUID) (bool, string, bool, error) {
	var activeInt int
	var until sql.NullString
	var byStaffInt int
	err := r.db.QueryRowContext(ctx,
		`SELECT CASE WHEN datetime(timeout_until) > CURRENT_TIMESTAMP THEN 1 ELSE 0 END,
		 timeout_until,
		 timeout_set_by_staff
		 FROM chat_room_members
		 WHERE room_id = ? AND user_id = ? AND left_at IS NULL`,
		roomID, userID,
	).Scan(&activeInt, &until, &byStaffInt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", false, nil
	}
	if err != nil {
		return false, "", false, fmt.Errorf("get member timeout state: %w", err)
	}
	active := activeInt != 0
	if !until.Valid {
		return false, "", byStaffInt != 0, nil
	}
	return active, until.String, byStaffInt != 0, nil
}

func (r *chatRepository) PinMessage(ctx context.Context, messageID, pinnedBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_messages SET pinned_at = CURRENT_TIMESTAMP, pinned_by = ? WHERE id = ?`,
		pinnedBy, messageID,
	)
	if err != nil {
		return fmt.Errorf("pin message: %w", err)
	}
	return nil
}

func (r *chatRepository) UnpinMessage(ctx context.Context, messageID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE chat_messages SET pinned_at = NULL, pinned_by = NULL WHERE id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("unpin message: %w", err)
	}
	return nil
}

func (r *chatRepository) ListPinnedMessages(ctx context.Context, roomID uuid.UUID) ([]ChatMessageRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
		 COALESCE(ur.role, ''),
		 cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
		 parent.sender_id, pu.display_name, parent.body,
		 cm.pinned_at, cm.pinned_by, cm.edited_at,
		 COALESCE(mem.nickname, ''), COALESCE(mem.avatar_url, '')
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
		 LEFT JOIN users pu ON parent.sender_id = pu.id
		 LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
		 WHERE cm.room_id = ? AND cm.pinned_at IS NOT NULL
		 ORDER BY cm.pinned_at DESC`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pinned messages: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessageRow
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *chatRepository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO chat_message_reactions (message_id, user_id, emoji) VALUES (?, ?, ?)`,
		messageID, userID, emoji,
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

func (r *chatRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM chat_message_reactions WHERE message_id = ? AND user_id = ? AND emoji = ?`,
		messageID, userID, emoji,
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

func (r *chatRepository) GetReactionsBatch(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]ReactionGroup, error) {
	result := make(map[uuid.UUID][]ReactionGroup)
	if len(messageIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, 0, len(messageIDs)+1)
	args = append(args, viewerID)
	for i := range messageIDs {
		placeholders[i] = "?"
		args = append(args, messageIDs[i])
	}

	query := `SELECT r.message_id, r.emoji, COUNT(*) AS cnt,
	          MAX(CASE WHEN r.user_id = ? THEN 1 ELSE 0 END) AS viewer_reacted,
	          GROUP_CONCAT(u.display_name, char(10)) AS names
	          FROM chat_message_reactions r
	          JOIN users u ON u.id = r.user_id
	          WHERE r.message_id IN (` + strings.Join(placeholders, ",") + `)
	          GROUP BY r.message_id, r.emoji
	          ORDER BY cnt DESC, r.emoji ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get reactions batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msgID uuid.UUID
		var emoji string
		var count int
		var viewerReacted int
		var names sql.NullString
		if err := rows.Scan(&msgID, &emoji, &count, &viewerReacted, &names); err != nil {
			return nil, fmt.Errorf("scan reaction group: %w", err)
		}
		var displayNames []string
		if names.Valid && names.String != "" {
			displayNames = strings.Split(names.String, "\n")
		}
		result[msgID] = append(result[msgID], ReactionGroup{
			Emoji:         emoji,
			Count:         count,
			ViewerReacted: viewerReacted != 0,
			DisplayNames:  displayNames,
		})
	}
	return result, rows.Err()
}

func (r *chatRepository) CountUnreadRoomsForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_rooms cr
		 JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = ? AND m.left_at IS NULL
		 WHERE cr.type = 'dm'
		   AND cr.last_message_at IS NOT NULL
		   AND (m.last_read_at IS NULL OR cr.last_message_at > m.last_read_at)`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread dm rooms: %w", err)
	}
	return count, nil
}
