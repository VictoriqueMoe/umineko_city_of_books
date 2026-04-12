-- +goose Up
CREATE TABLE vanity_roles (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#888888',
    is_system INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_vanity_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vanity_role_id TEXT NOT NULL REFERENCES vanity_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, vanity_role_id)
);

INSERT INTO vanity_roles (id, label, color, is_system, sort_order)
VALUES ('system_top_detective', 'True Detective', '#38bdf8', 1, 0);

INSERT INTO vanity_roles (id, label, color, is_system, sort_order)
VALUES ('system_top_gm', 'Game Master', '#ef5350', 1, 1);

-- +goose Down
DROP TABLE IF EXISTS user_vanity_roles;
DROP TABLE IF EXISTS vanity_roles;
