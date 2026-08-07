-- +goose Up
CREATE TABLE chat_voice_force_mutes (
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

CREATE INDEX idx_chat_voice_force_mutes_user ON chat_voice_force_mutes(user_id);

-- +goose Down
DROP TABLE chat_voice_force_mutes;
