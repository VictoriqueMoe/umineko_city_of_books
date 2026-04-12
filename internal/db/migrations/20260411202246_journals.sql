-- +goose Up
CREATE TABLE journals (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    work TEXT NOT NULL DEFAULT 'general',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME,
    last_author_activity_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at DATETIME
);
CREATE INDEX idx_journals_user_id ON journals(user_id);
CREATE INDEX idx_journals_work ON journals(work);
CREATE INDEX idx_journals_last_author_activity ON journals(last_author_activity_at);
CREATE INDEX idx_journals_archived_at ON journals(archived_at);

CREATE TABLE journal_comments (
    id TEXT PRIMARY KEY,
    journal_id TEXT NOT NULL REFERENCES journals(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES journal_comments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);
CREATE INDEX idx_journal_comments_journal_id ON journal_comments(journal_id);
CREATE INDEX idx_journal_comments_parent_id ON journal_comments(parent_id);

CREATE TABLE journal_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES journal_comments(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, comment_id)
);
CREATE INDEX idx_journal_comment_likes_comment_id ON journal_comment_likes(comment_id);

CREATE TABLE journal_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES journal_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    thumbnail_url TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_journal_comment_media_comment_id ON journal_comment_media(comment_id);

CREATE TABLE journal_follows (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    journal_id TEXT NOT NULL REFERENCES journals(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, journal_id)
);
CREATE INDEX idx_journal_follows_journal_id ON journal_follows(journal_id);

-- +goose Down
DROP TABLE IF EXISTS journal_follows;
DROP TABLE IF EXISTS journal_comment_media;
DROP TABLE IF EXISTS journal_comment_likes;
DROP TABLE IF EXISTS journal_comments;
DROP TABLE IF EXISTS journals;
