-- +goose Up

ALTER TABLE users ADD COLUMN higurashi_arc_progress INTEGER DEFAULT 0;

-- +goose Down

ALTER TABLE users DROP COLUMN higurashi_arc_progress;
