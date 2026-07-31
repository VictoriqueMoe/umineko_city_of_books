-- +goose Up

ALTER TABLE user_roles VALIDATE CONSTRAINT user_roles_role_check;

ALTER TABLE live_streams ADD CONSTRAINT live_streams_status_check
    CHECK (status IN ('offline', 'starting', 'live'));
ALTER TABLE live_streams ADD CONSTRAINT live_streams_viewer_count_check
    CHECK (viewer_count >= 0);

ALTER TABLE mysteries ADD CONSTRAINT mysteries_unsolved_state_check
    CHECK (solved OR (solved_at IS NULL AND winner_id IS NULL));
ALTER TABLE mysteries ADD CONSTRAINT mysteries_unpaused_state_check
    CHECK (paused OR paused_at IS NULL);
ALTER TABLE mysteries ADD CONSTRAINT mysteries_paused_duration_check
    CHECK (paused_duration_seconds >= 0);

ALTER TABLE theories ADD CONSTRAINT theories_credibility_score_check
    CHECK (credibility_score >= 0 AND credibility_score <= 100);

ALTER TABLE post_polls ADD CONSTRAINT post_polls_duration_check
    CHECK (duration_seconds > 0);
ALTER TABLE post_polls ADD CONSTRAINT post_polls_expiry_check
    CHECK (expires_at > created_at);

ALTER TABLE users ADD CONSTRAINT users_banner_position_check
    CHECK (banner_position >= 0 AND banner_position <= 100);
ALTER TABLE users ADD CONSTRAINT users_progress_check
    CHECK (episode_progress >= 0 AND higurashi_arc_progress >= 0 AND ciconia_chapter_progress >= 0);

ALTER TABLE posts ADD CONSTRAINT posts_view_count_check CHECK (view_count >= 0);
ALTER TABLE art ADD CONSTRAINT art_view_count_check CHECK (view_count >= 0);

ALTER TABLE fanfics ADD CONSTRAINT fanfics_counts_check
    CHECK (word_count >= 0 AND favourite_count >= 0 AND view_count >= 0 AND comment_count >= 0);
ALTER TABLE fanfic_chapters ADD CONSTRAINT fanfic_chapters_number_check
    CHECK (chapter_number >= 1 AND word_count >= 0);
ALTER TABLE fanfic_reading_progress ADD CONSTRAINT fanfic_reading_progress_chapter_check
    CHECK (chapter_number >= 1);

ALTER TABLE journal_entries ADD CONSTRAINT journal_entries_number_check
    CHECK (entry_number >= 1 AND word_count >= 0);

ALTER TABLE mystery_attachments ADD CONSTRAINT mystery_attachments_file_size_check
    CHECK (file_size >= 0);

ALTER TABLE announcement_comment_media ADD CONSTRAINT announcement_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE art_comment_media ADD CONSTRAINT art_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE chat_message_media ADD CONSTRAINT chat_message_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE fanfic_characters ADD CONSTRAINT fanfic_characters_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE fanfic_comment_media ADD CONSTRAINT fanfic_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE fanfic_genres ADD CONSTRAINT fanfic_genres_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE journal_comment_media ADD CONSTRAINT journal_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE journal_entry_media ADD CONSTRAINT journal_entry_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE mystery_clues ADD CONSTRAINT mystery_clues_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE mystery_comment_media ADD CONSTRAINT mystery_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE mystery_media ADD CONSTRAINT mystery_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE oc_comment_media ADD CONSTRAINT oc_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE oc_images ADD CONSTRAINT oc_images_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE post_comment_media ADD CONSTRAINT post_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE post_media ADD CONSTRAINT post_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE post_poll_options ADD CONSTRAINT post_poll_options_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE response_evidence ADD CONSTRAINT response_evidence_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE secret_comment_media ADD CONSTRAINT secret_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE ship_characters ADD CONSTRAINT ship_characters_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE ship_comment_media ADD CONSTRAINT ship_comment_media_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE theory_evidence ADD CONSTRAINT theory_evidence_sort_order_check CHECK (sort_order >= 0);
ALTER TABLE vanity_roles ADD CONSTRAINT vanity_roles_sort_order_check CHECK (sort_order >= 0);

-- +goose Down

ALTER TABLE vanity_roles DROP CONSTRAINT vanity_roles_sort_order_check;
ALTER TABLE theory_evidence DROP CONSTRAINT theory_evidence_sort_order_check;
ALTER TABLE ship_comment_media DROP CONSTRAINT ship_comment_media_sort_order_check;
ALTER TABLE ship_characters DROP CONSTRAINT ship_characters_sort_order_check;
ALTER TABLE secret_comment_media DROP CONSTRAINT secret_comment_media_sort_order_check;
ALTER TABLE response_evidence DROP CONSTRAINT response_evidence_sort_order_check;
ALTER TABLE post_poll_options DROP CONSTRAINT post_poll_options_sort_order_check;
ALTER TABLE post_media DROP CONSTRAINT post_media_sort_order_check;
ALTER TABLE post_comment_media DROP CONSTRAINT post_comment_media_sort_order_check;
ALTER TABLE oc_images DROP CONSTRAINT oc_images_sort_order_check;
ALTER TABLE oc_comment_media DROP CONSTRAINT oc_comment_media_sort_order_check;
ALTER TABLE mystery_media DROP CONSTRAINT mystery_media_sort_order_check;
ALTER TABLE mystery_comment_media DROP CONSTRAINT mystery_comment_media_sort_order_check;
ALTER TABLE mystery_clues DROP CONSTRAINT mystery_clues_sort_order_check;
ALTER TABLE journal_entry_media DROP CONSTRAINT journal_entry_media_sort_order_check;
ALTER TABLE journal_comment_media DROP CONSTRAINT journal_comment_media_sort_order_check;
ALTER TABLE fanfic_genres DROP CONSTRAINT fanfic_genres_sort_order_check;
ALTER TABLE fanfic_comment_media DROP CONSTRAINT fanfic_comment_media_sort_order_check;
ALTER TABLE fanfic_characters DROP CONSTRAINT fanfic_characters_sort_order_check;
ALTER TABLE chat_message_media DROP CONSTRAINT chat_message_media_sort_order_check;
ALTER TABLE art_comment_media DROP CONSTRAINT art_comment_media_sort_order_check;
ALTER TABLE announcement_comment_media DROP CONSTRAINT announcement_comment_media_sort_order_check;

ALTER TABLE mystery_attachments DROP CONSTRAINT mystery_attachments_file_size_check;
ALTER TABLE journal_entries DROP CONSTRAINT journal_entries_number_check;
ALTER TABLE fanfic_reading_progress DROP CONSTRAINT fanfic_reading_progress_chapter_check;
ALTER TABLE fanfic_chapters DROP CONSTRAINT fanfic_chapters_number_check;
ALTER TABLE fanfics DROP CONSTRAINT fanfics_counts_check;
ALTER TABLE art DROP CONSTRAINT art_view_count_check;
ALTER TABLE posts DROP CONSTRAINT posts_view_count_check;
ALTER TABLE users DROP CONSTRAINT users_progress_check;
ALTER TABLE users DROP CONSTRAINT users_banner_position_check;
ALTER TABLE post_polls DROP CONSTRAINT post_polls_expiry_check;
ALTER TABLE post_polls DROP CONSTRAINT post_polls_duration_check;
ALTER TABLE theories DROP CONSTRAINT theories_credibility_score_check;
ALTER TABLE mysteries DROP CONSTRAINT mysteries_paused_duration_check;
ALTER TABLE mysteries DROP CONSTRAINT mysteries_unpaused_state_check;
ALTER TABLE mysteries DROP CONSTRAINT mysteries_unsolved_state_check;
ALTER TABLE live_streams DROP CONSTRAINT live_streams_viewer_count_check;
ALTER TABLE live_streams DROP CONSTRAINT live_streams_status_check;
