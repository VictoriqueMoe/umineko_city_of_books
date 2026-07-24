-- +goose NO TRANSACTION

-- +goose Up
ALTER INDEX idx_announcement_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_announcements_search_vector SET (fastupdate = off);
ALTER INDEX idx_announcements_title_trgm SET (fastupdate = off);
ALTER INDEX idx_art_search_vector SET (fastupdate = off);
ALTER INDEX idx_art_title_trgm SET (fastupdate = off);
ALTER INDEX idx_art_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_chat_messages_search_vector SET (fastupdate = off);
ALTER INDEX idx_chat_messages_body_trgm SET (fastupdate = off);
ALTER INDEX idx_fanfic_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_fanfics_search_vector SET (fastupdate = off);
ALTER INDEX idx_fanfics_title_trgm SET (fastupdate = off);
ALTER INDEX idx_journal_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_journal_entries_search_vector SET (fastupdate = off);
ALTER INDEX idx_journals_search_vector SET (fastupdate = off);
ALTER INDEX idx_journals_title_trgm SET (fastupdate = off);
ALTER INDEX idx_live_streams_search_vector SET (fastupdate = off);
ALTER INDEX idx_mysteries_title_trgm SET (fastupdate = off);
ALTER INDEX idx_mysteries_search_vector SET (fastupdate = off);
ALTER INDEX idx_mystery_attempts_search_vector SET (fastupdate = off);
ALTER INDEX idx_mystery_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_oc_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_ocs_name_trgm SET (fastupdate = off);
ALTER INDEX idx_ocs_search_vector SET (fastupdate = off);
ALTER INDEX idx_post_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_posts_search_vector SET (fastupdate = off);
ALTER INDEX idx_responses_search_vector SET (fastupdate = off);
ALTER INDEX idx_ship_comments_search_vector SET (fastupdate = off);
ALTER INDEX idx_ships_search_vector SET (fastupdate = off);
ALTER INDEX idx_ships_title_trgm SET (fastupdate = off);
ALTER INDEX idx_theories_search_vector SET (fastupdate = off);
ALTER INDEX idx_theories_title_trgm SET (fastupdate = off);
ALTER INDEX idx_users_display_name_trgm SET (fastupdate = off);
ALTER INDEX idx_users_search_vector SET (fastupdate = off);
ALTER INDEX idx_users_username_trgm SET (fastupdate = off);

ALTER TABLE chat_messages SET (
    autovacuum_vacuum_insert_scale_factor = 0,
    autovacuum_vacuum_insert_threshold = 1000
);

VACUUM chat_messages;

-- +goose Down
ALTER TABLE chat_messages RESET (
    autovacuum_vacuum_insert_scale_factor,
    autovacuum_vacuum_insert_threshold
);

ALTER INDEX idx_announcement_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_announcements_search_vector RESET (fastupdate);
ALTER INDEX idx_announcements_title_trgm RESET (fastupdate);
ALTER INDEX idx_art_search_vector RESET (fastupdate);
ALTER INDEX idx_art_title_trgm RESET (fastupdate);
ALTER INDEX idx_art_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_chat_messages_search_vector RESET (fastupdate);
ALTER INDEX idx_chat_messages_body_trgm RESET (fastupdate);
ALTER INDEX idx_fanfic_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_fanfics_search_vector RESET (fastupdate);
ALTER INDEX idx_fanfics_title_trgm RESET (fastupdate);
ALTER INDEX idx_journal_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_journal_entries_search_vector RESET (fastupdate);
ALTER INDEX idx_journals_search_vector RESET (fastupdate);
ALTER INDEX idx_journals_title_trgm RESET (fastupdate);
ALTER INDEX idx_live_streams_search_vector RESET (fastupdate);
ALTER INDEX idx_mysteries_title_trgm RESET (fastupdate);
ALTER INDEX idx_mysteries_search_vector RESET (fastupdate);
ALTER INDEX idx_mystery_attempts_search_vector RESET (fastupdate);
ALTER INDEX idx_mystery_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_oc_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_ocs_name_trgm RESET (fastupdate);
ALTER INDEX idx_ocs_search_vector RESET (fastupdate);
ALTER INDEX idx_post_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_posts_search_vector RESET (fastupdate);
ALTER INDEX idx_responses_search_vector RESET (fastupdate);
ALTER INDEX idx_ship_comments_search_vector RESET (fastupdate);
ALTER INDEX idx_ships_search_vector RESET (fastupdate);
ALTER INDEX idx_ships_title_trgm RESET (fastupdate);
ALTER INDEX idx_theories_search_vector RESET (fastupdate);
ALTER INDEX idx_theories_title_trgm RESET (fastupdate);
ALTER INDEX idx_users_display_name_trgm RESET (fastupdate);
ALTER INDEX idx_users_search_vector RESET (fastupdate);
ALTER INDEX idx_users_username_trgm RESET (fastupdate);
