-- +goose Up

ALTER TABLE users ADD COLUMN ciconia_chapter_progress INTEGER DEFAULT 0;

-- +goose Down

ALTER TABLE users DROP COLUMN ciconia_chapter_progress;
