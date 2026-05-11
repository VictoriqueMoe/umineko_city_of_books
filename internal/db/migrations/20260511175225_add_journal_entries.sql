-- +goose Up

-- New per-entry table modelled after fanfic_chapters.
CREATE TABLE journal_entries (
    id UUID PRIMARY KEY,
    journal_id UUID NOT NULL REFERENCES journals(id) ON DELETE CASCADE,
    entry_number INTEGER NOT NULL,
    title TEXT,
    body TEXT NOT NULL DEFAULT '',
    word_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (journal_id, entry_number)
);
CREATE INDEX idx_journal_entries_journal ON journal_entries(journal_id, entry_number);

ALTER TABLE journal_entries ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED;
CREATE INDEX idx_journal_entries_search_vector ON journal_entries USING GIN(search_vector);

-- Scope journal_comments by optional entry id. NULL = title-page comment.
ALTER TABLE journal_comments
    ADD COLUMN entry_id UUID REFERENCES journal_entries(id) ON DELETE CASCADE;
CREATE INDEX idx_journal_comments_entry ON journal_comments(entry_id) WHERE entry_id IS NOT NULL;

-- Step 1: migrate every existing journal body into entry #1 (the intro).
INSERT INTO journal_entries (id, journal_id, entry_number, title, body, word_count, created_at, updated_at)
SELECT gen_random_uuid(),
       id,
       1,
       NULL,
       body,
       coalesce(array_length(regexp_split_to_array(coalesce(body, ''), '\s+'), 1), 0),
       created_at,
       coalesce(updated_at, created_at)
FROM journals
WHERE coalesce(body, '') <> '';

-- Step 2: each author top-level comment was historically an "update". Convert each
-- to its own entry (continuing the numbering from any body-derived entry #1), then
-- re-scope all of that comment's descendant replies to live under the new entry
-- and delete the original comment row.
-- +goose StatementBegin
DO $$
DECLARE
    j RECORD;
    c RECORD;
    next_num INT;
    new_entry_id UUID;
BEGIN
    FOR j IN SELECT id, user_id FROM journals LOOP
        SELECT COALESCE(MAX(entry_number), 0) + 1
          INTO next_num
          FROM journal_entries
          WHERE journal_id = j.id;

        FOR c IN
            SELECT id, body, created_at, updated_at
            FROM journal_comments
            WHERE journal_id = j.id
              AND parent_id IS NULL
              AND user_id = j.user_id
            ORDER BY created_at ASC
        LOOP
            new_entry_id := gen_random_uuid();

            INSERT INTO journal_entries
                (id, journal_id, entry_number, title, body, word_count, created_at, updated_at)
            VALUES (
                new_entry_id,
                j.id,
                next_num,
                NULL,
                c.body,
                COALESCE(array_length(regexp_split_to_array(COALESCE(c.body, ''), '\s+'), 1), 0),
                c.created_at,
                COALESCE(c.updated_at, c.created_at)
            );

            -- Re-scope every descendant of this comment (any nesting depth) to the new entry.
            WITH RECURSIVE descendants AS (
                SELECT id FROM journal_comments WHERE parent_id = c.id
                UNION ALL
                SELECT cc.id
                FROM journal_comments cc
                JOIN descendants d ON cc.parent_id = d.id
            )
            UPDATE journal_comments
               SET entry_id = new_entry_id
             WHERE id IN (SELECT id FROM descendants);

            -- The about-to-be-deleted comment was the parent of its immediate children.
            -- Detach them so they become top-level comments under the new entry.
            UPDATE journal_comments
               SET parent_id = NULL
             WHERE parent_id = c.id;

            -- Remove the original top-level comment; its content now lives in the entry.
            DELETE FROM journal_comments WHERE id = c.id;

            next_num := next_num + 1;
        END LOOP;
    END LOOP;
END $$;
-- +goose StatementEnd

-- The journals search vector currently references body; drop it so we can drop the column.
DROP INDEX IF EXISTS idx_journals_search_vector;
ALTER TABLE journals DROP COLUMN search_vector;

ALTER TABLE journals DROP COLUMN body;

-- Recreate the journals search vector against title only (entry content is indexed via journal_entries).
ALTER TABLE journals ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, ''))) STORED;
CREATE INDEX idx_journals_search_vector ON journals USING GIN(search_vector);

-- +goose Down

DROP INDEX IF EXISTS idx_journals_search_vector;
ALTER TABLE journals DROP COLUMN search_vector;

-- Re-add the body column. Best-effort restore: pull entry #1's body back into journals.body.
ALTER TABLE journals ADD COLUMN body TEXT NOT NULL DEFAULT '';
UPDATE journals j SET body = e.body
FROM journal_entries e
WHERE e.journal_id = j.id AND e.entry_number = 1;

ALTER TABLE journals ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED;
CREATE INDEX idx_journals_search_vector ON journals USING GIN(search_vector);

DROP INDEX IF EXISTS idx_journal_comments_entry;
ALTER TABLE journal_comments DROP COLUMN entry_id;

DROP INDEX IF EXISTS idx_journal_entries_search_vector;
DROP INDEX IF EXISTS idx_journal_entries_journal;
DROP TABLE journal_entries;
