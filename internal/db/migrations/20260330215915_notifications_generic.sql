-- +goose Up

-- +goose StatementBegin
CREATE TABLE notifications_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    reference_type TEXT NOT NULL DEFAULT 'theory',
    actor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO notifications_new (id, user_id, type, reference_id, reference_type, actor_id, read, created_at)
SELECT id, user_id, type, reference_id, 'theory', actor_id, read, created_at
FROM notifications;

DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_read ON notifications(user_id, read);
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
CREATE TABLE notifications_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    theory_id TEXT NOT NULL,
    actor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO notifications_old (id, user_id, type, reference_id, theory_id, actor_id, read, created_at)
SELECT id, user_id, type, reference_id, reference_id, actor_id, read, created_at
FROM notifications;

DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_read ON notifications(user_id, read);
-- +goose StatementEnd
