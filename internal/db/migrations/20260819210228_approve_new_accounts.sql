-- +goose Up
ALTER TABLE users ADD COLUMN approved_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN approved_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN approved_by;
ALTER TABLE users DROP COLUMN approved_at;
