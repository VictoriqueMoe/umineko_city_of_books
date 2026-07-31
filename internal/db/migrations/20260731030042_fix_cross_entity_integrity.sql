-- +goose Up

ALTER TABLE post_poll_options ADD CONSTRAINT post_poll_options_poll_id_id_key UNIQUE (poll_id, id);
ALTER TABLE post_poll_votes DROP CONSTRAINT post_poll_votes_option_id_fkey;
ALTER TABLE post_poll_votes ADD CONSTRAINT post_poll_votes_option_same_poll_fkey
    FOREIGN KEY (poll_id, option_id) REFERENCES post_poll_options (poll_id, id) ON DELETE CASCADE;

ALTER TABLE responses ADD CONSTRAINT responses_theory_id_id_key UNIQUE (theory_id, id);
ALTER TABLE responses DROP CONSTRAINT responses_parent_id_fkey;
ALTER TABLE responses ADD CONSTRAINT responses_parent_same_theory_fkey
    FOREIGN KEY (theory_id, parent_id) REFERENCES responses (theory_id, id) ON DELETE CASCADE;

ALTER TABLE post_comments ADD CONSTRAINT post_comments_post_id_id_key UNIQUE (post_id, id);
ALTER TABLE post_comments DROP CONSTRAINT post_comments_parent_id_fkey;
ALTER TABLE post_comments ADD CONSTRAINT post_comments_parent_same_post_fkey
    FOREIGN KEY (post_id, parent_id) REFERENCES post_comments (post_id, id) ON DELETE CASCADE;

ALTER TABLE art_comments ADD CONSTRAINT art_comments_art_id_id_key UNIQUE (art_id, id);
ALTER TABLE art_comments DROP CONSTRAINT art_comments_parent_id_fkey;
ALTER TABLE art_comments ADD CONSTRAINT art_comments_parent_same_art_fkey
    FOREIGN KEY (art_id, parent_id) REFERENCES art_comments (art_id, id) ON DELETE CASCADE;

ALTER TABLE mystery_attempts ADD CONSTRAINT mystery_attempts_mystery_id_id_key UNIQUE (mystery_id, id);
ALTER TABLE mystery_attempts DROP CONSTRAINT mystery_attempts_parent_id_fkey;
ALTER TABLE mystery_attempts ADD CONSTRAINT mystery_attempts_parent_same_mystery_fkey
    FOREIGN KEY (mystery_id, parent_id) REFERENCES mystery_attempts (mystery_id, id) ON DELETE CASCADE;

ALTER TABLE mystery_comments ADD CONSTRAINT mystery_comments_mystery_id_id_key UNIQUE (mystery_id, id);
ALTER TABLE mystery_comments DROP CONSTRAINT mystery_comments_parent_id_fkey;
ALTER TABLE mystery_comments ADD CONSTRAINT mystery_comments_parent_same_mystery_fkey
    FOREIGN KEY (mystery_id, parent_id) REFERENCES mystery_comments (mystery_id, id) ON DELETE CASCADE;

ALTER TABLE ship_comments ADD CONSTRAINT ship_comments_ship_id_id_key UNIQUE (ship_id, id);
ALTER TABLE ship_comments DROP CONSTRAINT ship_comments_parent_id_fkey;
ALTER TABLE ship_comments ADD CONSTRAINT ship_comments_parent_same_ship_fkey
    FOREIGN KEY (ship_id, parent_id) REFERENCES ship_comments (ship_id, id) ON DELETE CASCADE;

ALTER TABLE announcement_comments ADD CONSTRAINT announcement_comments_announcement_id_id_key UNIQUE (announcement_id, id);
ALTER TABLE announcement_comments DROP CONSTRAINT announcement_comments_parent_id_fkey;
ALTER TABLE announcement_comments ADD CONSTRAINT announcement_comments_parent_same_announcement_fkey
    FOREIGN KEY (announcement_id, parent_id) REFERENCES announcement_comments (announcement_id, id) ON DELETE CASCADE;

ALTER TABLE fanfic_comments ADD CONSTRAINT fanfic_comments_fanfic_id_id_key UNIQUE (fanfic_id, id);
ALTER TABLE fanfic_comments DROP CONSTRAINT fanfic_comments_parent_id_fkey;
ALTER TABLE fanfic_comments ADD CONSTRAINT fanfic_comments_parent_same_fanfic_fkey
    FOREIGN KEY (fanfic_id, parent_id) REFERENCES fanfic_comments (fanfic_id, id) ON DELETE CASCADE;

ALTER TABLE journal_comments ADD CONSTRAINT journal_comments_journal_id_id_key UNIQUE (journal_id, id);
ALTER TABLE journal_comments DROP CONSTRAINT journal_comments_parent_id_fkey;
ALTER TABLE journal_comments ADD CONSTRAINT journal_comments_parent_same_journal_fkey
    FOREIGN KEY (journal_id, parent_id) REFERENCES journal_comments (journal_id, id) ON DELETE CASCADE;

ALTER TABLE secret_comments ADD CONSTRAINT secret_comments_secret_id_id_key UNIQUE (secret_id, id);
ALTER TABLE secret_comments DROP CONSTRAINT secret_comments_parent_id_fkey;
ALTER TABLE secret_comments ADD CONSTRAINT secret_comments_parent_same_secret_fkey
    FOREIGN KEY (secret_id, parent_id) REFERENCES secret_comments (secret_id, id) ON DELETE CASCADE;

ALTER TABLE oc_comments ADD CONSTRAINT oc_comments_oc_id_id_key UNIQUE (oc_id, id);
ALTER TABLE oc_comments DROP CONSTRAINT oc_comments_parent_id_fkey;
ALTER TABLE oc_comments ADD CONSTRAINT oc_comments_parent_same_oc_fkey
    FOREIGN KEY (oc_id, parent_id) REFERENCES oc_comments (oc_id, id) ON DELETE CASCADE;

ALTER TABLE chat_messages ADD CONSTRAINT chat_messages_room_id_id_key UNIQUE (room_id, id);
ALTER TABLE chat_messages DROP CONSTRAINT chat_messages_reply_to_id_fkey;
ALTER TABLE chat_messages ADD CONSTRAINT chat_messages_reply_same_room_fkey
    FOREIGN KEY (room_id, reply_to_id) REFERENCES chat_messages (room_id, id) ON DELETE SET NULL (reply_to_id);

-- +goose Down

ALTER TABLE chat_messages DROP CONSTRAINT chat_messages_reply_same_room_fkey;
ALTER TABLE chat_messages ADD CONSTRAINT chat_messages_reply_to_id_fkey
    FOREIGN KEY (reply_to_id) REFERENCES chat_messages (id) ON DELETE SET NULL;
ALTER TABLE chat_messages DROP CONSTRAINT chat_messages_room_id_id_key;

ALTER TABLE oc_comments DROP CONSTRAINT oc_comments_parent_same_oc_fkey;
ALTER TABLE oc_comments ADD CONSTRAINT oc_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES oc_comments (id) ON DELETE CASCADE;
ALTER TABLE oc_comments DROP CONSTRAINT oc_comments_oc_id_id_key;

ALTER TABLE secret_comments DROP CONSTRAINT secret_comments_parent_same_secret_fkey;
ALTER TABLE secret_comments ADD CONSTRAINT secret_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES secret_comments (id) ON DELETE CASCADE;
ALTER TABLE secret_comments DROP CONSTRAINT secret_comments_secret_id_id_key;

ALTER TABLE journal_comments DROP CONSTRAINT journal_comments_parent_same_journal_fkey;
ALTER TABLE journal_comments ADD CONSTRAINT journal_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES journal_comments (id) ON DELETE CASCADE;
ALTER TABLE journal_comments DROP CONSTRAINT journal_comments_journal_id_id_key;

ALTER TABLE fanfic_comments DROP CONSTRAINT fanfic_comments_parent_same_fanfic_fkey;
ALTER TABLE fanfic_comments ADD CONSTRAINT fanfic_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES fanfic_comments (id) ON DELETE CASCADE;
ALTER TABLE fanfic_comments DROP CONSTRAINT fanfic_comments_fanfic_id_id_key;

ALTER TABLE announcement_comments DROP CONSTRAINT announcement_comments_parent_same_announcement_fkey;
ALTER TABLE announcement_comments ADD CONSTRAINT announcement_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES announcement_comments (id) ON DELETE CASCADE;
ALTER TABLE announcement_comments DROP CONSTRAINT announcement_comments_announcement_id_id_key;

ALTER TABLE ship_comments DROP CONSTRAINT ship_comments_parent_same_ship_fkey;
ALTER TABLE ship_comments ADD CONSTRAINT ship_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES ship_comments (id) ON DELETE CASCADE;
ALTER TABLE ship_comments DROP CONSTRAINT ship_comments_ship_id_id_key;

ALTER TABLE mystery_comments DROP CONSTRAINT mystery_comments_parent_same_mystery_fkey;
ALTER TABLE mystery_comments ADD CONSTRAINT mystery_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES mystery_comments (id) ON DELETE CASCADE;
ALTER TABLE mystery_comments DROP CONSTRAINT mystery_comments_mystery_id_id_key;

ALTER TABLE mystery_attempts DROP CONSTRAINT mystery_attempts_parent_same_mystery_fkey;
ALTER TABLE mystery_attempts ADD CONSTRAINT mystery_attempts_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES mystery_attempts (id) ON DELETE CASCADE;
ALTER TABLE mystery_attempts DROP CONSTRAINT mystery_attempts_mystery_id_id_key;

ALTER TABLE art_comments DROP CONSTRAINT art_comments_parent_same_art_fkey;
ALTER TABLE art_comments ADD CONSTRAINT art_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES art_comments (id) ON DELETE CASCADE;
ALTER TABLE art_comments DROP CONSTRAINT art_comments_art_id_id_key;

ALTER TABLE post_comments DROP CONSTRAINT post_comments_parent_same_post_fkey;
ALTER TABLE post_comments ADD CONSTRAINT post_comments_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES post_comments (id) ON DELETE CASCADE;
ALTER TABLE post_comments DROP CONSTRAINT post_comments_post_id_id_key;

ALTER TABLE responses DROP CONSTRAINT responses_parent_same_theory_fkey;
ALTER TABLE responses ADD CONSTRAINT responses_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES responses (id) ON DELETE CASCADE;
ALTER TABLE responses DROP CONSTRAINT responses_theory_id_id_key;

ALTER TABLE post_poll_votes DROP CONSTRAINT post_poll_votes_option_same_poll_fkey;
ALTER TABLE post_poll_votes ADD CONSTRAINT post_poll_votes_option_id_fkey
    FOREIGN KEY (option_id) REFERENCES post_poll_options (id) ON DELETE CASCADE;
ALTER TABLE post_poll_options DROP CONSTRAINT post_poll_options_poll_id_id_key;
