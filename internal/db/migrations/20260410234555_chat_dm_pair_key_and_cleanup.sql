-- +goose Up

DELETE FROM chat_rooms
WHERE type = 'dm'
  AND id NOT IN (SELECT DISTINCT room_id FROM chat_messages);

UPDATE chat_rooms SET last_message_at = NULL
WHERE id NOT IN (SELECT DISTINCT room_id FROM chat_messages);

ALTER TABLE chat_rooms ADD COLUMN dm_pair_key TEXT;

UPDATE chat_rooms
SET dm_pair_key = (
    SELECT MIN(user_id) || ':' || MAX(user_id)
    FROM chat_room_members
    WHERE room_id = chat_rooms.id
)
WHERE type = 'dm';

CREATE UNIQUE INDEX idx_chat_rooms_dm_pair_key ON chat_rooms(dm_pair_key) WHERE type = 'dm';

-- +goose Down

DROP INDEX IF EXISTS idx_chat_rooms_dm_pair_key;
ALTER TABLE chat_rooms DROP COLUMN dm_pair_key;
