-- +goose Up

CREATE TABLE site_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

-- +goose Down

DROP TABLE IF EXISTS site_settings;
