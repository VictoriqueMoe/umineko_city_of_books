-- +goose Up
ALTER TABLE mysteries
    ADD COLUMN keep_open_after_solve BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE mysteries
    DROP COLUMN keep_open_after_solve;
