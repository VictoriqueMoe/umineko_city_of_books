-- name: RecordArtView :execrows
INSERT INTO art_views (art_id, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING;
