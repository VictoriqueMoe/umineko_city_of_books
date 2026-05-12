-- +goose Up

UPDATE users
SET display_name = COALESCE(
    NULLIF(TRIM(
        LEFT(
            regexp_replace(
                regexp_replace(display_name, '<[^>]*>', '', 'g'),
                '\s+', ' ', 'g'
            ),
            40
        )
    ), ''),
    username
)
WHERE display_name IS NOT NULL
  AND (
      display_name ~ '<[^>]*>'
      OR length(display_name) > 40
      OR display_name ~ '\s\s'
      OR display_name <> TRIM(display_name)
  );

-- +goose Down

SELECT 1;
