-- +goose Up
UPDATE users
SET password_hash = '!'
WHERE id = '00000000-0000-4000-8000-000000000001'
  AND password_hash = '$2a$10$T0RCi7ybdIJraglqpfRiwOi4diB1y2/reIVa6YfWGtneJan4KwxLm';

-- +goose Down
UPDATE users
SET password_hash = '$2a$10$T0RCi7ybdIJraglqpfRiwOi4diB1y2/reIVa6YfWGtneJan4KwxLm'
WHERE id = '00000000-0000-4000-8000-000000000001'
  AND password_hash = '!';
