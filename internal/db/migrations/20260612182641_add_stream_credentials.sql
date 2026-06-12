-- +goose Up
CREATE TABLE stream_credentials (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    ingress_id TEXT NOT NULL,
    whip_url TEXT NOT NULL,
    stream_key TEXT NOT NULL,
    room TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE stream_credentials;
