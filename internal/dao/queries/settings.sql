-- name: GetSiteSetting :one
SELECT value FROM site_settings WHERE key = $1;

-- name: GetAllSiteSettings :many
SELECT key, value FROM site_settings;

-- name: UpsertSiteSetting :exec
INSERT INTO site_settings (key, value, updated_by, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = NOW();

-- name: DeleteSiteSetting :exec
DELETE FROM site_settings WHERE key = $1;
