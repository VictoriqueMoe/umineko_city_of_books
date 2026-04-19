-- +goose Up

-- +goose StatementBegin
CREATE TABLE audit_log_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT,
    details TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO audit_log_new (id, actor_id, action, target_type, target_id, details, created_at)
SELECT id, actor_id, action, target_type, target_id, details, created_at
FROM audit_log;

DROP TABLE audit_log;
ALTER TABLE audit_log_new RENAME TO audit_log;

CREATE INDEX idx_audit_log_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at DESC);
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
CREATE TABLE audit_log_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT,
    details TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO audit_log_old (id, actor_id, action, target_type, target_id, details, created_at)
SELECT id, actor_id, action, target_type, target_id, details, created_at
FROM audit_log
WHERE actor_id IS NOT NULL;

DROP TABLE audit_log;
ALTER TABLE audit_log_old RENAME TO audit_log;

CREATE INDEX idx_audit_log_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at DESC);
-- +goose StatementEnd
