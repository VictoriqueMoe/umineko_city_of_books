-- +goose Up
ALTER TABLE chat_rooms ADD COLUMN is_public INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chat_rooms ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_room_members ADD COLUMN role TEXT NOT NULL DEFAULT 'member';
CREATE INDEX idx_chat_rooms_public ON chat_rooms(is_public, last_message_at) WHERE type = 'group';
UPDATE chat_room_members
   SET role = 'host'
 WHERE (room_id, user_id) IN (
   SELECT cr.id, cr.created_by FROM chat_rooms cr WHERE cr.type = 'group'
 );

-- +goose Down
DROP INDEX IF EXISTS idx_chat_rooms_public;
ALTER TABLE chat_room_members DROP COLUMN role;
ALTER TABLE chat_rooms DROP COLUMN description;
ALTER TABLE chat_rooms DROP COLUMN is_public;
