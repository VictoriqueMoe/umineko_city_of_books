-- +goose Up
CREATE TYPE storage_backend AS ENUM ('local', 's3');

CREATE TABLE stored_files (
    key        TEXT PRIMARY KEY,
    backend    storage_backend NOT NULL,
    location   TEXT NOT NULL,
    size       BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stored_files_backend ON stored_files(backend);

-- +goose Down
DROP TABLE stored_files;
DROP TYPE storage_backend;
