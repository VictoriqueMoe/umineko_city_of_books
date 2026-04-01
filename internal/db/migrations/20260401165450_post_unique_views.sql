-- +goose Up
CREATE TABLE post_views (
    post_id TEXT NOT NULL,
    viewer_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, viewer_hash),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE post_views;
