-- name: RecordPostView :execrows
INSERT INTO post_views (post_id, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING;
