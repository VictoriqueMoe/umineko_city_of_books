-- +goose Up
INSERT OR IGNORE INTO fanfic_oc_characters (name, created_by, created_at)
SELECT TRIM(fc.character_name), f.user_id, MIN(f.created_at)
FROM fanfic_characters fc
JOIN fanfics f ON f.id = fc.fanfic_id
WHERE (fc.character_id IS NULL OR fc.character_id = '')
  AND TRIM(fc.character_name) != ''
GROUP BY LOWER(TRIM(fc.character_name));

-- +goose Down
DELETE FROM fanfic_oc_characters;
