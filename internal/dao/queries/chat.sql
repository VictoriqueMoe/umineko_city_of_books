-- name: CreateChatRoom :one
INSERT INTO chat_rooms (name, description, type, is_public, is_rp, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, type, is_public, is_rp, is_system, system_kind, created_by, created_at, last_message_at, archived_at;

-- name: CreateChatSystemRoom :one
INSERT INTO chat_rooms (id, name, description, type, is_public, is_rp, is_system, system_kind, created_by)
VALUES ($1, $2, $3, 'group', FALSE, FALSE, TRUE, $4, $5)
RETURNING id, name, description, type, is_public, is_rp, is_system, system_kind, created_by, created_at, last_message_at, archived_at;

-- name: CreateChatDMRoom :one
INSERT INTO chat_rooms (name, type, created_by, dm_pair_key)
VALUES ('', 'dm', $1, $2)
RETURNING id, name, description, type, is_public, is_rp, is_system, system_kind, created_by, created_at, last_message_at, archived_at;

-- name: FindChatDMRoomByPair :one
SELECT id, name, description, type, is_public, is_rp, is_system, system_kind, created_by, created_at, last_message_at, archived_at
FROM chat_rooms
WHERE type = 'dm' AND dm_pair_key = $1;

-- name: FindChatDMRoomForUser :one
SELECT cr.id
FROM chat_rooms cr
JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
WHERE cr.type = 'dm' AND cr.dm_pair_key = $2
LIMIT 1;

-- name: GetChatSystemRoomID :one
SELECT id FROM chat_rooms WHERE system_kind = $1 LIMIT 1;

-- name: UpdateChatRoom :execrows
UPDATE chat_rooms SET name = $1, description = $2, is_public = $3, is_rp = $4
WHERE id = $5 AND type = 'group' AND is_system = FALSE;

-- name: DeleteChatRoom :exec
DELETE FROM chat_rooms WHERE id = $1;

-- name: GetChatRoomSendContext :one
SELECT id, name, type, is_public, is_system, system_kind, created_by, last_message_at
FROM chat_rooms
WHERE id = $1;

-- name: GetChatRoomByID :one
SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, cr.archived_at,
       m.last_read_at, m.role AS viewer_role, m.muted AS viewer_muted, m.ghost AS viewer_ghost,
       (SELECT COUNT(*) FROM chat_room_members mc WHERE mc.room_id = cr.id AND mc.left_at IS NULL) AS member_count
FROM chat_rooms cr
LEFT JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
WHERE cr.id = $2;

-- name: ListChatRoomsByUser :many
SELECT cr.id, cr.name, cr.description, cr.type, cr.is_public, cr.is_rp, cr.is_system, cr.system_kind, cr.created_by, cr.created_at, cr.last_message_at, cr.archived_at,
       m.last_read_at, m.role AS viewer_role, m.muted AS viewer_muted, m.ghost AS viewer_ghost,
       (SELECT COUNT(*) FROM chat_room_members mc WHERE mc.room_id = cr.id AND mc.left_at IS NULL) AS member_count
FROM chat_rooms cr
JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
WHERE cr.system_kind IS DISTINCT FROM 'watch_party'
ORDER BY cr.is_system DESC, COALESCE(cr.last_message_at, cr.created_at) DESC;

-- name: TouchChatRoomActivity :exec
UPDATE chat_rooms SET last_message_at = NOW() WHERE id = $1;

-- name: TouchChatRoomActivityUnarchive :exec
UPDATE chat_rooms SET last_message_at = NOW(), archived_at = NULL WHERE id = $1;

-- name: ListStaleChatRoomIDs :many
SELECT cr.id FROM chat_rooms cr
WHERE cr.type = 'group'
  AND cr.is_system = FALSE
  AND cr.archived_at IS NULL
  AND COALESCE(
      (SELECT MAX(cm.created_at) FROM chat_messages cm WHERE cm.room_id = cr.id AND cm.is_system = FALSE),
      cr.created_at
  ) < $1::timestamptz;

-- name: ArchiveChatRooms :exec
UPDATE chat_rooms SET archived_at = NOW() WHERE id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: CountUnreadChatDMRoomsForUser :one
SELECT COUNT(*) FROM chat_rooms cr
JOIN chat_room_members m ON cr.id = m.room_id AND m.user_id = $1 AND m.left_at IS NULL
WHERE cr.type = 'dm'
  AND m.muted = FALSE
  AND cr.last_message_at IS NOT NULL
  AND (m.last_read_at IS NULL OR cr.last_message_at > m.last_read_at);

-- name: AddChatRoomTag :exec
INSERT INTO chat_room_tags (room_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteChatRoomTags :exec
DELETE FROM chat_room_tags WHERE room_id = $1;

-- name: GetChatRoomTags :many
SELECT tag FROM chat_room_tags WHERE room_id = $1 ORDER BY tag;

-- name: GetChatRoomTagsBatch :many
SELECT room_id, tag FROM chat_room_tags
WHERE room_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY tag;

-- name: AddChatRoomMember :exec
INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2)
ON CONFLICT (room_id, user_id) DO UPDATE SET left_at = NULL, joined_at = NOW();

-- name: AddChatRoomMemberWithRole :exec
INSERT INTO chat_room_members (room_id, user_id, role, ghost) VALUES ($1, $2, $3, $4)
ON CONFLICT (room_id, user_id) DO UPDATE SET left_at = NULL, role = excluded.role, ghost = excluded.ghost, joined_at = NOW();

-- name: AddChatDMMembers :exec
INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2), ($1, $3);

-- name: RejoinChatDMMembers :exec
INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2), ($1, $3)
ON CONFLICT (room_id, user_id) DO UPDATE SET left_at = NULL, joined_at = NOW()
WHERE chat_room_members.left_at IS NOT NULL;

-- name: RemoveChatRoomMember :exec
UPDATE chat_room_members SET left_at = NOW() WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: CountChatRoomMembers :one
SELECT COUNT(*) FROM chat_room_members WHERE room_id = $1 AND left_at IS NULL;

-- name: IsChatRoomMember :one
SELECT COUNT(*) FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: IsChatRoomGhostMember :one
SELECT ghost FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: CountChatRoomGhostMembers :one
SELECT COUNT(1) FROM chat_room_members WHERE room_id = $1 AND ghost = TRUE AND left_at IS NULL;

-- name: SetChatMemberRole :exec
UPDATE chat_room_members SET role = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL;

-- name: GetChatMemberRole :one
SELECT role FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: GetChatMemberNickname :one
SELECT COALESCE(nickname, '')::text AS nickname FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: GetChatRoomMemberIDs :many
SELECT user_id FROM chat_room_members WHERE room_id = $1 AND left_at IS NULL;

-- name: GetChatRoomUnmutedMemberIDs :many
SELECT user_id FROM chat_room_members WHERE room_id = $1 AND muted = FALSE AND left_at IS NULL;

-- name: ListChatRoomMemberAvatarURLs :many
SELECT avatar_url FROM chat_room_members WHERE room_id = $1 AND avatar_url <> '';

-- name: GetChatRoomMembersDetailed :many
SELECT m.user_id, u.username, u.display_name, u.avatar_url, m.role, COALESCE(ur.role, '')::text AS author_role,
       m.joined_at, m.nickname, m.nickname_locked, m.avatar_url AS member_avatar_url,
       (CASE WHEN m.timeout_until > NOW() THEN to_char(m.timeout_until AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') ELSE '' END)::text AS timeout_until,
       (CASE WHEN m.timeout_until > NOW() THEN m.timeout_set_by_staff ELSE FALSE END)::boolean AS timeout_by_staff,
       m.ghost
FROM chat_room_members m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
WHERE m.room_id = $1 AND m.left_at IS NULL
ORDER BY CASE m.role WHEN 'host' THEN 0 ELSE 1 END, m.joined_at ASC;

-- name: SetChatMemberMuted :exec
UPDATE chat_room_members SET muted = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL;

-- name: IsChatMemberMuted :one
SELECT muted FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: MarkChatRoomRead :exec
UPDATE chat_room_members SET last_read_at = NOW() WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: SetChatMemberNickname :exec
UPDATE chat_room_members SET nickname = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL;

-- name: SetChatMemberNicknameWithLock :exec
UPDATE chat_room_members SET nickname = $1, nickname_locked = $2 WHERE room_id = $3 AND user_id = $4 AND left_at IS NULL;

-- name: IsChatMemberNicknameLocked :one
SELECT nickname_locked FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: SetChatMemberAvatar :exec
UPDATE chat_room_members SET avatar_url = $1 WHERE room_id = $2 AND user_id = $3 AND left_at IS NULL;

-- name: SetChatMemberTimeout :exec
UPDATE chat_room_members SET timeout_until = $1, timeout_set_by_staff = $2 WHERE room_id = $3 AND user_id = $4 AND left_at IS NULL;

-- name: ClearChatMemberTimeout :exec
UPDATE chat_room_members SET timeout_until = NULL, timeout_set_by_staff = FALSE WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: HasActiveChatMemberTimeout :one
SELECT (timeout_until > NOW())::boolean AS active
FROM chat_room_members
WHERE room_id = $1 AND user_id = $2;

-- name: GetChatMemberTimeoutState :one
SELECT COALESCE(timeout_until > NOW(), FALSE)::boolean AS active,
       timeout_until,
       timeout_set_by_staff
FROM chat_room_members
WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: DeleteChatVoiceForceMute :exec
DELETE FROM chat_voice_force_mutes WHERE room_id = $1 AND user_id = $2;

-- name: SetChatVoiceForceMute :exec
INSERT INTO chat_voice_force_mutes (room_id, user_id, muted_by) VALUES ($1, $2, $3)
ON CONFLICT (room_id, user_id) DO UPDATE SET muted_by = EXCLUDED.muted_by;

-- name: IsChatVoiceForceMuted :one
SELECT EXISTS (SELECT 1 FROM chat_voice_force_mutes WHERE room_id = $1 AND user_id = $2) AS force_muted;

-- name: ClearChatVoiceForceMutes :exec
DELETE FROM chat_voice_force_mutes WHERE room_id = $1;

-- name: InsertChatMessage :one
WITH ins AS (
    INSERT INTO chat_messages (room_id, sender_id, body, reply_to_id, is_system)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, room_id, sender_id, body, reply_to_id, pinned_at, pinned_by, is_system, edited_at, created_at
)
SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
       COALESCE(ur.role, '')::text AS sender_role,
       cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
       parent.sender_id AS reply_to_sender_id,
       pmem.nickname AS reply_member_nickname,
       pu.display_name AS reply_display_name,
       pu.username AS reply_username,
       parent.body AS reply_to_body,
       cm.pinned_at, cm.pinned_by, cm.edited_at,
       COALESCE(mem.nickname, '')::text AS sender_nickname,
       COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
FROM ins cm
JOIN users u ON cm.sender_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
LEFT JOIN users pu ON parent.sender_id = pu.id
LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id;

-- name: CountChatRoomMessages :one
SELECT COUNT(*) FROM chat_messages cm
WHERE cm.room_id = $1
  AND cm.created_at >= COALESCE(
      (SELECT m.joined_at
         FROM chat_room_members m
         JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
        WHERE m.room_id = cm.room_id AND m.user_id = $2),
      cm.created_at);

-- name: ListChatRoomMessages :many
SELECT sub.id, sub.room_id, sub.sender_id, sub.username, sub.display_name, sub.avatar_url,
       sub.sender_role, sub.body, sub.is_system, sub.created_at, sub.reply_to_id,
       sub.reply_to_sender_id, sub.reply_member_nickname, sub.reply_display_name, sub.reply_username, sub.reply_to_body,
       sub.pinned_at, sub.pinned_by, sub.edited_at, sub.sender_nickname, sub.sender_member_avatar
FROM (
    SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
           COALESCE(ur.role, '')::text AS sender_role,
           cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
           parent.sender_id AS reply_to_sender_id,
           pmem.nickname AS reply_member_nickname,
           pu.display_name AS reply_display_name,
           pu.username AS reply_username,
           parent.body AS reply_to_body,
           cm.pinned_at, cm.pinned_by, cm.edited_at,
           COALESCE(mem.nickname, '')::text AS sender_nickname,
           COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
    FROM chat_messages cm
    JOIN users u ON cm.sender_id = u.id
    LEFT JOIN user_roles ur ON ur.user_id = u.id
    LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
    LEFT JOIN users pu ON parent.sender_id = pu.id
    LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
    LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
    WHERE cm.room_id = $1
      AND cm.created_at >= COALESCE(
          (SELECT m.joined_at
             FROM chat_room_members m
             JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
            WHERE m.room_id = cm.room_id AND m.user_id = $2),
          cm.created_at)
    ORDER BY cm.created_at DESC, cm.id DESC
    LIMIT $3
) sub
ORDER BY sub.created_at ASC, sub.id ASC;

-- name: ListChatRoomMessagesBefore :many
SELECT sub.id, sub.room_id, sub.sender_id, sub.username, sub.display_name, sub.avatar_url,
       sub.sender_role, sub.body, sub.is_system, sub.created_at, sub.reply_to_id,
       sub.reply_to_sender_id, sub.reply_member_nickname, sub.reply_display_name, sub.reply_username, sub.reply_to_body,
       sub.pinned_at, sub.pinned_by, sub.edited_at, sub.sender_nickname, sub.sender_member_avatar
FROM (
    SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
           COALESCE(ur.role, '')::text AS sender_role,
           cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
           parent.sender_id AS reply_to_sender_id,
           pmem.nickname AS reply_member_nickname,
           pu.display_name AS reply_display_name,
           pu.username AS reply_username,
           parent.body AS reply_to_body,
           cm.pinned_at, cm.pinned_by, cm.edited_at,
           COALESCE(mem.nickname, '')::text AS sender_nickname,
           COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
    FROM chat_messages cm
    JOIN users u ON cm.sender_id = u.id
    LEFT JOIN user_roles ur ON ur.user_id = u.id
    LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
    LEFT JOIN users pu ON parent.sender_id = pu.id
    LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
    LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
    WHERE cm.room_id = $1 AND (
        cm.created_at < $2::timestamptz
        OR ($3::text != '' AND cm.created_at = $2::timestamptz AND cm.id::text < $3::text)
    )
      AND cm.created_at >= COALESCE(
          (SELECT m.joined_at
             FROM chat_room_members m
             JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
            WHERE m.room_id = cm.room_id AND m.user_id = $5),
          cm.created_at)
    ORDER BY cm.created_at DESC, cm.id DESC
    LIMIT $4
) sub
ORDER BY sub.created_at ASC, sub.id ASC;

-- name: ListChatRoomMediaAttachments :many
SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
       COALESCE(ur.role, '')::text AS sender_role,
       cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
       parent.sender_id AS reply_to_sender_id,
       pmem.nickname AS reply_member_nickname,
       pu.display_name AS reply_display_name,
       pu.username AS reply_username,
       parent.body AS reply_to_body,
       cm.pinned_at, cm.pinned_by, cm.edited_at,
       COALESCE(mem.nickname, '')::text AS sender_nickname,
       COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
FROM chat_messages cm
JOIN users u ON cm.sender_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
LEFT JOIN users pu ON parent.sender_id = pu.id
LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
WHERE cm.room_id = sqlc.arg(room_id) AND cm.is_system = FALSE AND (
    sqlc.narg(before_time)::timestamptz IS NULL
    OR cm.created_at < sqlc.narg(before_time)::timestamptz
    OR (sqlc.arg(before_id)::text != '' AND cm.created_at = sqlc.narg(before_time)::timestamptz AND cm.id::text < sqlc.arg(before_id)::text)
)
  AND EXISTS (SELECT 1 FROM chat_message_media cmm WHERE cmm.message_id = cm.id)
  AND cm.created_at >= COALESCE(
      (SELECT m.joined_at
         FROM chat_room_members m
         JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
        WHERE m.room_id = cm.room_id AND m.user_id = sqlc.arg(viewer_id)),
      cm.created_at)
ORDER BY cm.created_at DESC, cm.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListChatRoomLinkAttachments :many
SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
       COALESCE(ur.role, '')::text AS sender_role,
       cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
       parent.sender_id AS reply_to_sender_id,
       pmem.nickname AS reply_member_nickname,
       pu.display_name AS reply_display_name,
       pu.username AS reply_username,
       parent.body AS reply_to_body,
       cm.pinned_at, cm.pinned_by, cm.edited_at,
       COALESCE(mem.nickname, '')::text AS sender_nickname,
       COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
FROM chat_messages cm
JOIN users u ON cm.sender_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
LEFT JOIN users pu ON parent.sender_id = pu.id
LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
WHERE cm.room_id = sqlc.arg(room_id) AND cm.is_system = FALSE AND (
    sqlc.narg(before_time)::timestamptz IS NULL
    OR cm.created_at < sqlc.narg(before_time)::timestamptz
    OR (sqlc.arg(before_id)::text != '' AND cm.created_at = sqlc.narg(before_time)::timestamptz AND cm.id::text < sqlc.arg(before_id)::text)
)
  AND cm.body LIKE '%http%'
  AND cm.created_at >= COALESCE(
      (SELECT m.joined_at
         FROM chat_room_members m
         JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
        WHERE m.room_id = cm.room_id AND m.user_id = sqlc.arg(viewer_id)),
      cm.created_at)
ORDER BY cm.created_at DESC, cm.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListPinnedChatMessages :many
SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
       COALESCE(ur.role, '')::text AS sender_role,
       cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
       parent.sender_id AS reply_to_sender_id,
       pmem.nickname AS reply_member_nickname,
       pu.display_name AS reply_display_name,
       pu.username AS reply_username,
       parent.body AS reply_to_body,
       cm.pinned_at, cm.pinned_by, cm.edited_at,
       COALESCE(mem.nickname, '')::text AS sender_nickname,
       COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
FROM chat_messages cm
JOIN users u ON cm.sender_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
LEFT JOIN users pu ON parent.sender_id = pu.id
LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
WHERE cm.room_id = $1 AND cm.pinned_at IS NOT NULL
  AND cm.created_at >= COALESCE(
      (SELECT m.joined_at
         FROM chat_room_members m
         JOIN chat_rooms cr ON cr.id = m.room_id AND cr.type = 'dm'
        WHERE m.room_id = cm.room_id AND m.user_id = $2),
      cm.created_at)
ORDER BY cm.pinned_at DESC;

-- name: GetChatMessageByID :one
SELECT cm.id, cm.room_id, cm.sender_id, u.username, u.display_name, u.avatar_url,
       COALESCE(ur.role, '')::text AS sender_role,
       cm.body, cm.is_system, cm.created_at, cm.reply_to_id,
       parent.sender_id AS reply_to_sender_id,
       pmem.nickname AS reply_member_nickname,
       pu.display_name AS reply_display_name,
       pu.username AS reply_username,
       parent.body AS reply_to_body,
       cm.pinned_at, cm.pinned_by, cm.edited_at,
       COALESCE(mem.nickname, '')::text AS sender_nickname,
       COALESCE(mem.avatar_url, '')::text AS sender_member_avatar
FROM chat_messages cm
JOIN users u ON cm.sender_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN chat_messages parent ON cm.reply_to_id = parent.id
LEFT JOIN users pu ON parent.sender_id = pu.id
LEFT JOIN chat_room_members pmem ON pmem.room_id = cm.room_id AND pmem.user_id = parent.sender_id
LEFT JOIN chat_room_members mem ON mem.room_id = cm.room_id AND mem.user_id = cm.sender_id
WHERE cm.id = $1;

-- name: SearchChatMessagesForViewer :many
WITH q AS (SELECT websearch_to_tsquery('english', $2::text) AS tsq, $2::text AS qstr)
SELECT 'chat_message'::text AS entity_type, cm.id::text AS id, cm.room_id::text AS parent_id,
       COALESCE(NULLIF(cr.name, ''), 'Direct message')::text AS parent_title,
       COALESCE(NULLIF(cr.name, ''), 'Direct message')::text AS title,
       ts_headline('english', cm.body, q.tsq, 'MaxFragments=1, MaxWords=18, MinWords=5, ShortWord=3, HighlightAll=false, StartSel=<mark>, StopSel=</mark>')::text AS snippet,
       u.id::text AS author_id, u.username, u.display_name, u.avatar_url,
       cm.created_at,
       (ts_rank_cd(cm.search_vector, q.tsq) + COALESCE(similarity(cm.body, q.qstr), 0))::float8 AS rank,
       COUNT(*) OVER () AS total_count
FROM chat_messages cm
JOIN chat_room_members crm ON crm.room_id = cm.room_id AND crm.user_id = $1 AND crm.left_at IS NULL
JOIN chat_rooms cr ON cr.id = cm.room_id
JOIN users u ON cm.sender_id = u.id
CROSS JOIN q
WHERE cm.is_system = false
  AND cr.system_kind IS DISTINCT FROM 'watch_party'
  AND u.banned_at IS NULL AND u.locked_at IS NULL
  AND ($3::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR cm.room_id = $3::uuid)
  AND (cm.search_vector @@ q.tsq OR cm.body % q.qstr)
ORDER BY rank DESC, cm.created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetChatMessageRoomID :one
SELECT room_id FROM chat_messages WHERE id = $1;

-- name: GetChatMessageSenderID :one
SELECT sender_id FROM chat_messages WHERE id = $1;

-- name: DeleteChatRoomMessages :exec
DELETE FROM chat_messages WHERE room_id = $1;

-- name: DeleteChatMessage :exec
DELETE FROM chat_messages WHERE id = $1;

-- name: EditChatMessage :exec
UPDATE chat_messages SET body = $1, edited_at = NOW() WHERE id = $2;

-- name: PinChatMessage :exec
UPDATE chat_messages SET pinned_at = NOW(), pinned_by = $1 WHERE id = $2;

-- name: UnpinChatMessage :exec
UPDATE chat_messages SET pinned_at = NULL, pinned_by = NULL WHERE id = $1;

-- name: AddChatMessageMedia :one
INSERT INTO chat_message_media (message_id, media_url, media_type, thumbnail_url, filename, sort_order, width, height, is_spoiler)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: UpdateChatMessageMediaURL :exec
UPDATE chat_message_media SET media_url = $1 WHERE id = $2;

-- name: UpdateChatMessageMediaThumbnail :exec
UPDATE chat_message_media SET thumbnail_url = $1 WHERE id = $2;

-- name: GetChatMessageMediaBatch :many
SELECT id, message_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename,
       sort_order, width, height, is_spoiler
FROM chat_message_media
WHERE message_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order ASC, id ASC;

-- name: ListChatRoomMediaURLs :many
SELECT cmm.media_url, cmm.thumbnail_url
FROM chat_message_media cmm
JOIN chat_messages cm ON cm.id = cmm.message_id
WHERE cm.room_id = $1;

-- name: ListChatMessageMediaURLs :many
SELECT media_url, thumbnail_url FROM chat_message_media WHERE message_id = $1;

-- name: AddChatMessageReaction :execrows
INSERT INTO chat_message_reactions (message_id, user_id, emoji) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: RemoveChatMessageReaction :execrows
DELETE FROM chat_message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3;

-- name: CountChatMessageReactions :one
SELECT COUNT(*) FROM chat_message_reactions WHERE message_id = $1 AND emoji = $2;

-- name: GetChatMessageReactionsBatch :many
SELECT r.message_id, r.emoji, COUNT(*) AS cnt,
       BOOL_OR(r.user_id = $1) AS viewer_reacted,
       STRING_AGG(u.display_name, E'\n') AS names
FROM chat_message_reactions r
JOIN users u ON u.id = r.user_id
WHERE r.message_id = ANY(string_to_array($2::text, ',')::uuid[])
GROUP BY r.message_id, r.emoji
ORDER BY cnt DESC, r.emoji ASC;
