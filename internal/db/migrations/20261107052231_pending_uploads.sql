-- +goose Up
CREATE TYPE pending_upload_status AS ENUM ('processing', 'ready', 'failed');

CREATE TABLE pending_uploads (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type    TEXT NOT NULL,
    filename      TEXT NOT NULL,
    file_size     BIGINT NOT NULL,
    status        pending_upload_status NOT NULL DEFAULT 'processing',
    media_url     TEXT NOT NULL DEFAULT '',
    thumbnail_url TEXT NOT NULL DEFAULT '',
    width         INT NOT NULL DEFAULT 0,
    height        INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pending_uploads_user_id ON pending_uploads(user_id);
CREATE INDEX idx_pending_uploads_created_at ON pending_uploads(created_at);

-- +goose Down
DROP TABLE pending_uploads;
DROP TYPE pending_upload_status;
