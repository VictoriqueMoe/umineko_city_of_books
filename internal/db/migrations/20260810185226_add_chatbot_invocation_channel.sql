-- +goose Up
UPDATE chatbot_invocations ci SET surface = COALESCE((SELECT r.type FROM chat_rooms r WHERE r.id = ci.room_id), 'group') WHERE ci.surface = 'chat';
CREATE TYPE chatbot_channel AS ENUM ('dm', 'group', 'post', 'post_comment');
ALTER TABLE chatbot_invocations RENAME COLUMN surface TO channel;
ALTER TABLE chatbot_invocations ALTER COLUMN channel DROP DEFAULT;
ALTER TABLE chatbot_invocations ALTER COLUMN channel TYPE chatbot_channel USING channel::chatbot_channel;

-- +goose Down
ALTER TABLE chatbot_invocations ALTER COLUMN channel TYPE TEXT USING channel::TEXT;
DROP TYPE chatbot_channel;
ALTER TABLE chatbot_invocations RENAME COLUMN channel TO surface;
UPDATE chatbot_invocations SET surface = 'chat' WHERE surface IN ('dm', 'group');
ALTER TABLE chatbot_invocations ALTER COLUMN surface SET DEFAULT 'chat';
