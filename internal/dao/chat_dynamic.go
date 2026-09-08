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
)

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
	row, err := genQueries(r.db, tx).GetChatRoomByID(ctx, sqlcgen.GetChatRoomByIDParams{
		UserID: s.ViewerID,
		ID:     s.RoomID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room by id: %w", err)
	}

	out := toChatRoomForViewer(row)
	out.Tags, _ = r.GetRoomTags(ctx, s.RoomID, tx...)

	return &out, nil
}

func (r *chatDAO) GetRoomSendContext(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) (*model.ChatRoomSendContext, error) {
	row, err := genQueries(r.db, tx).GetChatRoomSendContext(ctx, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room send context: %w", err)
	}

	out := model.ChatRoomSendContext{
		ID:            row.ID,
		Name:          row.Name,
		Type:          dto.RoomType(row.Type),
		IsPublic:      row.IsPublic,
		IsSystem:      row.IsSystem,
		LastMessageAt: nullTimeToString(row.LastMessageAt),
	}

	if row.SystemKind.Valid {
		out.SystemKind = row.SystemKind.String
	}

	if row.CreatedBy != nil {
		out.CreatedBy = *row.CreatedBy
	}

	return &out, nil
}

func (r *chatDAO) GetRoomMembersDetailed(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]model.ChatRoomMemberRow, error) {
	rows, err := genQueries(r.db, tx).GetChatRoomMembersDetailed(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members detailed: %w", err)
	}

	var result []model.ChatRoomMemberRow
	for _, row := range rows {
		result = append(result, model.ChatRoomMemberRow{
			UserID:          row.UserID,
			Username:        row.Username,
			DisplayName:     row.DisplayName,
			AvatarURL:       row.AvatarUrl,
			Role:            row.Role,
			AuthorRole:      row.AuthorRole,
			AuthorRoleTyped: role.Role(row.AuthorRole),
			JoinedAt:        row.JoinedAt.UTC().Format(time.RFC3339),
			Nickname:        row.Nickname,
			NicknameLocked:  row.NicknameLocked,
			MemberAvatarURL: row.MemberAvatarUrl,
			TimeoutUntil:    row.TimeoutUntil,
			TimeoutByStaff:  row.TimeoutByStaff,
			Ghost:           row.Ghost,
		})
	}

	return result, nil
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
