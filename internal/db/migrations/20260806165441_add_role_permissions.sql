-- +goose Up
-- +goose StatementBegin
CREATE TABLE role_permissions (
    role TEXT NOT NULL,
    permission TEXT NOT NULL,
    PRIMARY KEY (role, permission),
    CONSTRAINT role_permissions_editable_role_only CHECK (role = 'moderator')
);

CREATE TABLE vanity_role_permissions (
    vanity_role_id TEXT NOT NULL REFERENCES vanity_roles(id) ON DELETE CASCADE,
    permission TEXT NOT NULL,
    PRIMARY KEY (vanity_role_id, permission)
);

INSERT INTO role_permissions (role, permission) VALUES
    ('moderator', 'view_admin_panel'),
    ('moderator', 'view_stats'),
    ('moderator', 'view_users'),
    ('moderator', 'delete_any_theory'),
    ('moderator', 'delete_any_response'),
    ('moderator', 'delete_any_post'),
    ('moderator', 'delete_any_comment'),
    ('moderator', 'edit_any_theory'),
    ('moderator', 'edit_any_post'),
    ('moderator', 'edit_any_comment'),
    ('moderator', 'ban_user'),
    ('moderator', 'edit_mystery_score'),
    ('moderator', 'edit_any_journal'),
    ('moderator', 'delete_any_journal'),
    ('moderator', 'manage_user_account'),
    ('moderator', 'use_chatbot')
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE vanity_role_permissions;
DROP TABLE role_permissions;
-- +goose StatementEnd
