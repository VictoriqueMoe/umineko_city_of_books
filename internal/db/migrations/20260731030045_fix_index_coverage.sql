-- +goose Up

CREATE INDEX idx_chat_messages_sender_id ON chat_messages (sender_id);
CREATE INDEX idx_chat_message_reactions_user_id ON chat_message_reactions (user_id);
CREATE INDEX idx_art_comments_user_id ON art_comments (user_id);
CREATE INDEX idx_mystery_comments_user_id ON mystery_comments (user_id);
CREATE INDEX idx_ship_comments_user_id ON ship_comments (user_id);
CREATE INDEX idx_announcement_comments_user_id ON announcement_comments (user_id);
CREATE INDEX idx_fanfic_comments_user_id ON fanfic_comments (user_id);
CREATE INDEX idx_journal_comments_user_id ON journal_comments (user_id);
CREATE INDEX idx_secret_comments_user_id ON secret_comments (user_id);
CREATE INDEX idx_oc_comments_user_id ON oc_comments (user_id);
CREATE INDEX idx_mystery_attempts_user_id ON mystery_attempts (user_id);
CREATE INDEX idx_reports_reporter_id ON reports (reporter_id);
CREATE INDEX idx_game_room_moves_user_id ON game_room_moves (user_id);
CREATE INDEX idx_game_rooms_created_by ON game_rooms (created_by);
CREATE INDEX idx_fanfic_oc_characters_created_by ON fanfic_oc_characters (created_by);
CREATE INDEX idx_chat_watch_party_participants_user_id ON chat_watch_party_participants (user_id);

CREATE INDEX idx_theory_votes_theory_id ON theory_votes (theory_id);
CREATE INDEX idx_response_votes_response_id ON response_votes (response_id);
CREATE INDEX idx_oc_votes_oc_id ON oc_votes (oc_id);
CREATE INDEX idx_mystery_attempt_votes_attempt_id ON mystery_attempt_votes (attempt_id);
CREATE INDEX idx_fanfic_reading_progress_fanfic_id ON fanfic_reading_progress (fanfic_id);
CREATE INDEX idx_user_vanity_roles_vanity_role_id ON user_vanity_roles (vanity_role_id);
CREATE INDEX idx_post_poll_votes_user_id ON post_poll_votes (user_id);
CREATE INDEX idx_oc_comments_parent_id ON oc_comments (parent_id);
CREATE INDEX idx_oc_comment_media_comment_id ON oc_comment_media (comment_id);
CREATE INDEX idx_oc_comment_likes_comment_id ON oc_comment_likes (comment_id);

CREATE INDEX idx_notifications_actor_id ON notifications (actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_users_banned_by ON users (banned_by) WHERE banned_by IS NOT NULL;
CREATE INDEX idx_users_locked_by ON users (locked_by) WHERE locked_by IS NOT NULL;
CREATE INDEX idx_invites_used_by ON invites (used_by) WHERE used_by IS NOT NULL;
CREATE INDEX idx_galleries_cover_art_id ON galleries (cover_art_id) WHERE cover_art_id IS NOT NULL;
CREATE INDEX idx_mysteries_winner_id ON mysteries (winner_id) WHERE winner_id IS NOT NULL;
CREATE INDEX idx_chat_messages_pinned_by ON chat_messages (pinned_by) WHERE pinned_by IS NOT NULL;
CREATE INDEX idx_chat_room_bans_banned_by ON chat_room_bans (banned_by) WHERE banned_by IS NOT NULL;
CREATE INDEX idx_chat_banned_words_created_by ON chat_banned_words (created_by) WHERE created_by IS NOT NULL;
CREATE INDEX idx_reports_resolved_by ON reports (resolved_by) WHERE resolved_by IS NOT NULL;
CREATE INDEX idx_site_settings_updated_by ON site_settings (updated_by) WHERE updated_by IS NOT NULL;
CREATE INDEX idx_game_rooms_turn_user_id ON game_rooms (turn_user_id) WHERE turn_user_id IS NOT NULL;
CREATE INDEX idx_game_rooms_winner_user_id ON game_rooms (winner_user_id) WHERE winner_user_id IS NOT NULL;
CREATE INDEX idx_chat_rooms_created_by ON chat_rooms (created_by) WHERE created_by IS NOT NULL;
CREATE INDEX idx_suggestion_resolved_resolved_by ON suggestion_resolved (resolved_by) WHERE resolved_by IS NOT NULL;
CREATE INDEX idx_announcements_author_id ON announcements (author_id) WHERE author_id IS NOT NULL;
CREATE INDEX idx_chat_watch_party_sessions_started_by ON chat_watch_party_sessions (started_by) WHERE started_by IS NOT NULL;
CREATE INDEX idx_chat_watch_party_sessions_controller_id ON chat_watch_party_sessions (controller_id) WHERE controller_id IS NOT NULL;
CREATE INDEX idx_chat_watch_party_messages_sender_id ON chat_watch_party_messages (sender_id) WHERE sender_id IS NOT NULL;

DROP INDEX idx_follows_follower_id;
DROP INDEX idx_post_polls_post_id;
DROP INDEX idx_journal_entries_journal;
DROP INDEX idx_fanfic_chapters_fanfic_id;
DROP INDEX idx_ocs_user_id;

DROP INDEX idx_post_poll_options_poll_id;
DROP INDEX idx_responses_theory_id;
DROP INDEX idx_post_comments_post_id;
DROP INDEX idx_art_comments_art_id;
DROP INDEX idx_mystery_attempts_mystery_id;
DROP INDEX idx_mystery_comments_mystery_id;
DROP INDEX idx_ship_comments_ship_id;
DROP INDEX idx_announcement_comments_announcement_id;
DROP INDEX idx_fanfic_comments_fanfic_id;
DROP INDEX idx_journal_comments_journal_id;
DROP INDEX idx_secret_comments_secret_id;

-- +goose Down

CREATE INDEX idx_secret_comments_secret_id ON secret_comments (secret_id);
CREATE INDEX idx_journal_comments_journal_id ON journal_comments (journal_id);
CREATE INDEX idx_fanfic_comments_fanfic_id ON fanfic_comments (fanfic_id);
CREATE INDEX idx_announcement_comments_announcement_id ON announcement_comments (announcement_id);
CREATE INDEX idx_ship_comments_ship_id ON ship_comments (ship_id);
CREATE INDEX idx_mystery_comments_mystery_id ON mystery_comments (mystery_id);
CREATE INDEX idx_mystery_attempts_mystery_id ON mystery_attempts (mystery_id);
CREATE INDEX idx_art_comments_art_id ON art_comments (art_id);
CREATE INDEX idx_post_comments_post_id ON post_comments (post_id);
CREATE INDEX idx_responses_theory_id ON responses (theory_id);
CREATE INDEX idx_post_poll_options_poll_id ON post_poll_options (poll_id);

CREATE INDEX idx_ocs_user_id ON ocs (user_id);
CREATE INDEX idx_fanfic_chapters_fanfic_id ON fanfic_chapters (fanfic_id);
CREATE INDEX idx_journal_entries_journal ON journal_entries (journal_id, entry_number);
CREATE INDEX idx_post_polls_post_id ON post_polls (post_id);
CREATE INDEX idx_follows_follower_id ON follows (follower_id);

DROP INDEX idx_chat_watch_party_messages_sender_id;
DROP INDEX idx_chat_watch_party_sessions_controller_id;
DROP INDEX idx_chat_watch_party_sessions_started_by;
DROP INDEX idx_announcements_author_id;
DROP INDEX idx_suggestion_resolved_resolved_by;
DROP INDEX idx_chat_rooms_created_by;
DROP INDEX idx_game_rooms_winner_user_id;
DROP INDEX idx_game_rooms_turn_user_id;
DROP INDEX idx_site_settings_updated_by;
DROP INDEX idx_reports_resolved_by;
DROP INDEX idx_chat_banned_words_created_by;
DROP INDEX idx_chat_room_bans_banned_by;
DROP INDEX idx_chat_messages_pinned_by;
DROP INDEX idx_mysteries_winner_id;
DROP INDEX idx_galleries_cover_art_id;
DROP INDEX idx_invites_used_by;
DROP INDEX idx_users_locked_by;
DROP INDEX idx_users_banned_by;
DROP INDEX idx_notifications_actor_id;

DROP INDEX idx_oc_comment_likes_comment_id;
DROP INDEX idx_oc_comment_media_comment_id;
DROP INDEX idx_oc_comments_parent_id;
DROP INDEX idx_post_poll_votes_user_id;
DROP INDEX idx_user_vanity_roles_vanity_role_id;
DROP INDEX idx_fanfic_reading_progress_fanfic_id;
DROP INDEX idx_mystery_attempt_votes_attempt_id;
DROP INDEX idx_oc_votes_oc_id;
DROP INDEX idx_response_votes_response_id;
DROP INDEX idx_theory_votes_theory_id;

DROP INDEX idx_chat_watch_party_participants_user_id;
DROP INDEX idx_fanfic_oc_characters_created_by;
DROP INDEX idx_game_rooms_created_by;
DROP INDEX idx_game_room_moves_user_id;
DROP INDEX idx_reports_reporter_id;
DROP INDEX idx_mystery_attempts_user_id;
DROP INDEX idx_oc_comments_user_id;
DROP INDEX idx_secret_comments_user_id;
DROP INDEX idx_journal_comments_user_id;
DROP INDEX idx_fanfic_comments_user_id;
DROP INDEX idx_announcement_comments_user_id;
DROP INDEX idx_ship_comments_user_id;
DROP INDEX idx_mystery_comments_user_id;
DROP INDEX idx_art_comments_user_id;
DROP INDEX idx_chat_message_reactions_user_id;
DROP INDEX idx_chat_messages_sender_id;
