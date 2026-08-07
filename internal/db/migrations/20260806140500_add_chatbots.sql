-- +goose Up
ALTER TABLE users ADD COLUMN is_bot BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_users_is_bot ON users(is_bot) WHERE is_bot;

CREATE TABLE chatbots (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    system_prompt TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    reasoning_effort TEXT NOT NULL DEFAULT '',
    max_output_tokens INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE chatbot_invocation_status AS ENUM ('pending', 'replied', 'refused', 'failed', 'quota');

CREATE TABLE chatbot_invocations (
    id UUID PRIMARY KEY,
    bot_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID REFERENCES chat_rooms(id) ON DELETE SET NULL,
    message_id UUID NOT NULL,
    surface TEXT NOT NULL DEFAULT 'chat',
    model TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    cached_prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    status chatbot_invocation_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chatbot_invocations_user_created ON chatbot_invocations(user_id, created_at DESC);
CREATE INDEX idx_chatbot_invocations_created ON chatbot_invocations(created_at DESC);
CREATE INDEX idx_chatbot_invocations_bot_created ON chatbot_invocations(bot_user_id, created_at DESC);

INSERT INTO vanity_roles (id, label, color, is_system, sort_order)
VALUES ('bot', 'Bot', '#8b5cf6', TRUE, 50)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE chatbot_invocations;

DROP TYPE chatbot_invocation_status;

DROP TABLE chatbots;

DELETE FROM vanity_roles WHERE id = 'bot';

DROP INDEX idx_users_is_bot;

ALTER TABLE users DROP COLUMN is_bot;
