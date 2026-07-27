-- +goose Up
ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_role_check
    CHECK (role IN ('super_admin', 'admin', 'moderator')) NOT VALID;

-- +goose Down
ALTER TABLE user_roles
    DROP CONSTRAINT user_roles_role_check;
