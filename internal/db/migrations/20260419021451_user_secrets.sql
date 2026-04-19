-- +goose Up

CREATE TABLE user_secrets (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret_id TEXT NOT NULL,
    unlocked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, secret_id)
);

INSERT INTO vanity_roles (id, label, color, is_system, sort_order)
VALUES ('system_witch_hunter', 'Witch Hunter', '#e89ec0', 1, 100);

-- +goose Down

DELETE FROM vanity_roles WHERE id = 'system_witch_hunter';
DROP TABLE IF EXISTS user_secrets;
