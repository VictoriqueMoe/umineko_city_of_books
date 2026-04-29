-- +goose Up

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE theories ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED;
CREATE INDEX idx_theories_search_vector ON theories USING GIN(search_vector);
CREATE INDEX idx_theories_title_trgm ON theories USING GIN(title gin_trgm_ops);

ALTER TABLE responses ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_responses_search_vector ON responses USING GIN(search_vector);

ALTER TABLE posts ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_posts_search_vector ON posts USING GIN(search_vector);

ALTER TABLE post_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_post_comments_search_vector ON post_comments USING GIN(search_vector);

ALTER TABLE art ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_art_search_vector ON art USING GIN(search_vector);
CREATE INDEX idx_art_title_trgm ON art USING GIN(title gin_trgm_ops);

ALTER TABLE art_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_art_comments_search_vector ON art_comments USING GIN(search_vector);

ALTER TABLE mysteries ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED;
CREATE INDEX idx_mysteries_search_vector ON mysteries USING GIN(search_vector);
CREATE INDEX idx_mysteries_title_trgm ON mysteries USING GIN(title gin_trgm_ops);

ALTER TABLE mystery_attempts ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_mystery_attempts_search_vector ON mystery_attempts USING GIN(search_vector);

ALTER TABLE mystery_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_mystery_comments_search_vector ON mystery_comments USING GIN(search_vector);

ALTER TABLE ships ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_ships_search_vector ON ships USING GIN(search_vector);
CREATE INDEX idx_ships_title_trgm ON ships USING GIN(title gin_trgm_ops);

ALTER TABLE ship_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_ship_comments_search_vector ON ship_comments USING GIN(search_vector);

ALTER TABLE announcements ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED;
CREATE INDEX idx_announcements_search_vector ON announcements USING GIN(search_vector);
CREATE INDEX idx_announcements_title_trgm ON announcements USING GIN(title gin_trgm_ops);

ALTER TABLE announcement_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_announcement_comments_search_vector ON announcement_comments USING GIN(search_vector);

ALTER TABLE fanfics ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(summary, '')), 'B')
    ) STORED;
CREATE INDEX idx_fanfics_search_vector ON fanfics USING GIN(search_vector);
CREATE INDEX idx_fanfics_title_trgm ON fanfics USING GIN(title gin_trgm_ops);

ALTER TABLE fanfic_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_fanfic_comments_search_vector ON fanfic_comments USING GIN(search_vector);

ALTER TABLE journals ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED;
CREATE INDEX idx_journals_search_vector ON journals USING GIN(search_vector);
CREATE INDEX idx_journals_title_trgm ON journals USING GIN(title gin_trgm_ops);

ALTER TABLE journal_comments ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_journal_comments_search_vector ON journal_comments USING GIN(search_vector);

ALTER TABLE users ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(display_name, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(username, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(bio, '')), 'B')
    ) STORED;
CREATE INDEX idx_users_search_vector ON users USING GIN(search_vector);
CREATE INDEX idx_users_display_name_trgm ON users USING GIN(display_name gin_trgm_ops);
CREATE INDEX idx_users_username_trgm ON users USING GIN(username gin_trgm_ops);

-- +goose Down

DROP INDEX IF EXISTS idx_users_username_trgm;
DROP INDEX IF EXISTS idx_users_display_name_trgm;
DROP INDEX IF EXISTS idx_users_search_vector;
ALTER TABLE users DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_journal_comments_search_vector;
ALTER TABLE journal_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_journals_title_trgm;
DROP INDEX IF EXISTS idx_journals_search_vector;
ALTER TABLE journals DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_fanfic_comments_search_vector;
ALTER TABLE fanfic_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_fanfics_title_trgm;
DROP INDEX IF EXISTS idx_fanfics_search_vector;
ALTER TABLE fanfics DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_announcement_comments_search_vector;
ALTER TABLE announcement_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_announcements_title_trgm;
DROP INDEX IF EXISTS idx_announcements_search_vector;
ALTER TABLE announcements DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_ship_comments_search_vector;
ALTER TABLE ship_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_ships_title_trgm;
DROP INDEX IF EXISTS idx_ships_search_vector;
ALTER TABLE ships DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_mystery_comments_search_vector;
ALTER TABLE mystery_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_mystery_attempts_search_vector;
ALTER TABLE mystery_attempts DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_mysteries_title_trgm;
DROP INDEX IF EXISTS idx_mysteries_search_vector;
ALTER TABLE mysteries DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_art_comments_search_vector;
ALTER TABLE art_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_art_title_trgm;
DROP INDEX IF EXISTS idx_art_search_vector;
ALTER TABLE art DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_post_comments_search_vector;
ALTER TABLE post_comments DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_posts_search_vector;
ALTER TABLE posts DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_responses_search_vector;
ALTER TABLE responses DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_theories_title_trgm;
DROP INDEX IF EXISTS idx_theories_search_vector;
ALTER TABLE theories DROP COLUMN IF EXISTS search_vector;
