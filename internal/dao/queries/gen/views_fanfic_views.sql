-- name: RecordFanficView :execrows
INSERT INTO fanfic_views (fanfic_id, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING;
