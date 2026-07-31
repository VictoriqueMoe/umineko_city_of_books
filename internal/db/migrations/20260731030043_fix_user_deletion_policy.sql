-- +goose Up

ALTER TABLE chat_rooms DROP CONSTRAINT chat_rooms_created_by_fkey;
ALTER TABLE chat_rooms ALTER COLUMN created_by DROP NOT NULL;
ALTER TABLE chat_rooms ADD CONSTRAINT chat_rooms_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE chat_watch_party_sessions DROP CONSTRAINT chat_watch_party_sessions_started_by_fkey;
ALTER TABLE chat_watch_party_sessions ALTER COLUMN started_by DROP NOT NULL;
ALTER TABLE chat_watch_party_sessions ADD CONSTRAINT chat_watch_party_sessions_started_by_fkey
    FOREIGN KEY (started_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE chat_watch_party_sessions DROP CONSTRAINT chat_watch_party_sessions_controller_id_fkey;
ALTER TABLE chat_watch_party_sessions ALTER COLUMN controller_id DROP NOT NULL;
ALTER TABLE chat_watch_party_sessions ADD CONSTRAINT chat_watch_party_sessions_controller_id_fkey
    FOREIGN KEY (controller_id) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE suggestion_resolved DROP CONSTRAINT suggestion_resolved_resolved_by_fkey;
ALTER TABLE suggestion_resolved ALTER COLUMN resolved_by DROP NOT NULL;
ALTER TABLE suggestion_resolved ADD CONSTRAINT suggestion_resolved_resolved_by_fkey
    FOREIGN KEY (resolved_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE announcements DROP CONSTRAINT announcements_author_id_fkey;
ALTER TABLE announcements ALTER COLUMN author_id DROP NOT NULL;
ALTER TABLE announcements ADD CONSTRAINT announcements_author_id_fkey
    FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE chat_watch_party_sessions DROP COLUMN hyperbeam_admin_token;

-- +goose Down

ALTER TABLE chat_watch_party_sessions ADD COLUMN hyperbeam_admin_token TEXT NOT NULL DEFAULT '';

DELETE FROM announcements WHERE author_id IS NULL;
ALTER TABLE announcements DROP CONSTRAINT announcements_author_id_fkey;
ALTER TABLE announcements ALTER COLUMN author_id SET NOT NULL;
ALTER TABLE announcements ADD CONSTRAINT announcements_author_id_fkey
    FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE CASCADE;

DELETE FROM suggestion_resolved WHERE resolved_by IS NULL;
ALTER TABLE suggestion_resolved DROP CONSTRAINT suggestion_resolved_resolved_by_fkey;
ALTER TABLE suggestion_resolved ALTER COLUMN resolved_by SET NOT NULL;
ALTER TABLE suggestion_resolved ADD CONSTRAINT suggestion_resolved_resolved_by_fkey
    FOREIGN KEY (resolved_by) REFERENCES users (id) ON DELETE CASCADE;

DELETE FROM chat_watch_party_sessions WHERE controller_id IS NULL OR started_by IS NULL;

ALTER TABLE chat_watch_party_sessions DROP CONSTRAINT chat_watch_party_sessions_controller_id_fkey;
ALTER TABLE chat_watch_party_sessions ALTER COLUMN controller_id SET NOT NULL;
ALTER TABLE chat_watch_party_sessions ADD CONSTRAINT chat_watch_party_sessions_controller_id_fkey
    FOREIGN KEY (controller_id) REFERENCES users (id) ON DELETE RESTRICT;

ALTER TABLE chat_watch_party_sessions DROP CONSTRAINT chat_watch_party_sessions_started_by_fkey;
ALTER TABLE chat_watch_party_sessions ALTER COLUMN started_by SET NOT NULL;
ALTER TABLE chat_watch_party_sessions ADD CONSTRAINT chat_watch_party_sessions_started_by_fkey
    FOREIGN KEY (started_by) REFERENCES users (id) ON DELETE RESTRICT;

DELETE FROM chat_rooms WHERE created_by IS NULL;
ALTER TABLE chat_rooms DROP CONSTRAINT chat_rooms_created_by_fkey;
ALTER TABLE chat_rooms ALTER COLUMN created_by SET NOT NULL;
ALTER TABLE chat_rooms ADD CONSTRAINT chat_rooms_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE;
