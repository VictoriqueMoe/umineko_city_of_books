-- +goose Up
-- +goose StatementBegin
CREATE TABLE banned_giphy (
    kind TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT,
    reason TEXT,
    PRIMARY KEY (kind, value)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE banned_giphy;
-- +goose StatementEnd
