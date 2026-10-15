-- +goose Up
ALTER TABLE notifications ADD COLUMN count INTEGER NOT NULL DEFAULT 1;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY user_id, type, reference_id ORDER BY created_at DESC, id DESC) AS rn,
           COUNT(*) OVER (PARTITION BY user_id, type, reference_id) AS group_size
    FROM notifications
    WHERE read = FALSE AND type IN ('chat_room_message', 'chat_message')
),
kept AS (
    UPDATE notifications n
    SET count = r.group_size
    FROM ranked r
    WHERE n.id = r.id AND r.rn = 1 AND r.group_size > 1
)
DELETE FROM notifications n
USING ranked r
WHERE n.id = r.id AND r.rn > 1;

CREATE UNIQUE INDEX idx_notifications_unread_chat_group
    ON notifications (user_id, type, reference_id)
    WHERE read = FALSE AND type IN ('chat_room_message', 'chat_message');

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_unread_chat_group;

ALTER TABLE notifications DROP COLUMN IF EXISTS count;
