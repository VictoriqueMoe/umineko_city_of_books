-- +goose Up
CREATE TABLE fanfic_tags (
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (fanfic_id, tag)
);

CREATE INDEX idx_fanfic_tags_tag ON fanfic_tags(tag);

-- +goose Down
DROP INDEX IF EXISTS idx_fanfic_tags_tag;
DROP TABLE IF EXISTS fanfic_tags;
