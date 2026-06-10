-- +goose Up
CREATE TABLE live_streams (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'offline',
    livekit_room TEXT NOT NULL DEFAULT '',
    ingress_id   TEXT NOT NULL DEFAULT '',
    whip_url     TEXT NOT NULL DEFAULT '',
    stream_key   TEXT NOT NULL DEFAULT '',
    viewer_count INTEGER NOT NULL DEFAULT 0,
    started_at   TIMESTAMPTZ,
    ended_at     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_live_streams_one_active_per_user
    ON live_streams (user_id)
    WHERE status <> 'offline';

CREATE INDEX idx_live_streams_status ON live_streams (status);

-- +goose Down
DROP TABLE live_streams;
