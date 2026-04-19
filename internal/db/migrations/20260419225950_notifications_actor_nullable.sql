-- +goose Up

-- +goose StatementBegin
CREATE TABLE notifications_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    reference_type TEXT NOT NULL DEFAULT 'theory',
    actor_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    read INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    message TEXT DEFAULT ''
);

INSERT INTO notifications_new (id, user_id, type, reference_id, reference_type, actor_id, read, created_at, message)
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type,
       CASE WHEN EXISTS (SELECT 1 FROM users u WHERE u.id = n.actor_id) THEN n.actor_id ELSE NULL END,
       n.read, n.created_at, n.message
FROM notifications n
WHERE EXISTS (SELECT 1 FROM users u WHERE u.id = n.user_id);

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
    reference_type TEXT NOT NULL DEFAULT 'theory',
    actor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    message TEXT DEFAULT ''
);

INSERT INTO notifications_old (id, user_id, type, reference_id, reference_type, actor_id, read, created_at, message)
SELECT id, user_id, type, reference_id, reference_type, actor_id, read, created_at, message
FROM notifications
WHERE actor_id IS NOT NULL;

DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_read ON notifications(user_id, read);
-- +goose StatementEnd
