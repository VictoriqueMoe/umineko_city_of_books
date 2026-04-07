-- +goose Up

CREATE TABLE fanfic_series (
    name TEXT PRIMARY KEY COLLATE NOCASE
);

INSERT INTO fanfic_series (name) VALUES ('Umineko'), ('Higurashi'), ('Ciconia');

CREATE TABLE fanfic_languages (
    name TEXT PRIMARY KEY COLLATE NOCASE
);

INSERT INTO fanfic_languages (name) VALUES
    ('English'), ('Spanish'), ('French'), ('German'), ('Portuguese'),
    ('Italian'), ('Japanese'), ('Chinese'), ('Korean'), ('Russian'),
    ('Polish'), ('Dutch'), ('Indonesian'), ('Turkish'), ('Arabic'),
    ('Thai'), ('Filipino'), ('Vietnamese'), ('Hindi');

CREATE TABLE fanfics (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    series TEXT NOT NULL DEFAULT 'Umineko',
    rating TEXT NOT NULL DEFAULT 'K',
    language TEXT NOT NULL DEFAULT 'English',
    status TEXT NOT NULL DEFAULT 'in_progress',
    is_oneshot INTEGER NOT NULL DEFAULT 0,
    cover_image_url TEXT NOT NULL DEFAULT '',
    cover_thumbnail_url TEXT NOT NULL DEFAULT '',
    word_count INTEGER NOT NULL DEFAULT 0,
    favourite_count INTEGER NOT NULL DEFAULT 0,
    view_count INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    published_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fanfics_user_id ON fanfics(user_id);
CREATE INDEX idx_fanfics_updated_at ON fanfics(updated_at DESC);
CREATE INDEX idx_fanfics_published_at ON fanfics(published_at DESC);
CREATE INDEX idx_fanfics_series ON fanfics(series);
CREATE INDEX idx_fanfics_rating ON fanfics(rating);
CREATE INDEX idx_fanfics_language ON fanfics(language);
CREATE INDEX idx_fanfics_status ON fanfics(status);

CREATE TABLE fanfic_chapters (
    id TEXT PRIMARY KEY,
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    chapter_number INTEGER NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    word_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(fanfic_id, chapter_number)
);

CREATE INDEX idx_fanfic_chapters_fanfic_id ON fanfic_chapters(fanfic_id);

CREATE TABLE fanfic_genres (
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    genre TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (fanfic_id, genre)
);

CREATE INDEX idx_fanfic_genres_genre ON fanfic_genres(genre);

CREATE TABLE fanfic_characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    series TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    character_name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_pairing INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_fanfic_characters_fanfic_id ON fanfic_characters(fanfic_id);
CREATE INDEX idx_fanfic_characters_lookup ON fanfic_characters(series, character_id);
CREATE INDEX idx_fanfic_characters_name ON fanfic_characters(character_name);

CREATE TABLE fanfic_oc_characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fanfic_oc_characters_name ON fanfic_oc_characters(name COLLATE NOCASE);

CREATE TABLE fanfic_favourites (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, fanfic_id)
);

CREATE INDEX idx_fanfic_favourites_fanfic_id ON fanfic_favourites(fanfic_id);

CREATE TABLE fanfic_views (
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    viewer_hash TEXT NOT NULL,
    viewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (fanfic_id, viewer_hash)
);

CREATE TABLE fanfic_comments (
    id TEXT PRIMARY KEY,
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES fanfic_comments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);

CREATE INDEX idx_fanfic_comments_fanfic_id ON fanfic_comments(fanfic_id);
CREATE INDEX idx_fanfic_comments_parent_id ON fanfic_comments(parent_id);

CREATE TABLE fanfic_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES fanfic_comments(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_fanfic_comment_likes_comment_id ON fanfic_comment_likes(comment_id);

CREATE TABLE fanfic_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES fanfic_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    thumbnail_url TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_fanfic_comment_media_comment_id ON fanfic_comment_media(comment_id);

-- +goose Down
DROP TABLE IF EXISTS fanfic_comment_media;
DROP TABLE IF EXISTS fanfic_comment_likes;
DROP TABLE IF EXISTS fanfic_comments;
DROP TABLE IF EXISTS fanfic_views;
DROP TABLE IF EXISTS fanfic_favourites;
DROP TABLE IF EXISTS fanfic_oc_characters;
DROP TABLE IF EXISTS fanfic_characters;
DROP TABLE IF EXISTS fanfic_genres;
DROP TABLE IF EXISTS fanfic_chapters;
DROP TABLE IF EXISTS fanfics;
DROP TABLE IF EXISTS fanfic_languages;
DROP TABLE IF EXISTS fanfic_series;
