-- +goose Up
DELETE FROM banned_giphy b
USING banned_giphy o
WHERE b.kind = 'user'
  AND o.kind = 'user'
  AND lower(btrim(b.value)) = lower(btrim(o.value))
  AND (b.created_at, b.value) > (o.created_at, o.value);

UPDATE banned_giphy
SET value = lower(btrim(value))
WHERE kind = 'user' AND value <> lower(btrim(value));

-- +goose Down
