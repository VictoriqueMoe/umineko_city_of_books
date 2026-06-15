-- +goose Up
CREATE TYPE stream_default_mode AS ENUM ('webrtc', 'hls');
ALTER TABLE live_streams ADD COLUMN egress_id TEXT NOT NULL DEFAULT '';
ALTER TABLE live_streams ADD COLUMN hls_playlist_url TEXT NOT NULL DEFAULT '';
ALTER TABLE live_streams ADD COLUMN default_mode stream_default_mode NOT NULL DEFAULT 'webrtc';

-- +goose Down
ALTER TABLE live_streams DROP COLUMN default_mode;
ALTER TABLE live_streams DROP COLUMN hls_playlist_url;
ALTER TABLE live_streams DROP COLUMN egress_id;
DROP TYPE stream_default_mode;
