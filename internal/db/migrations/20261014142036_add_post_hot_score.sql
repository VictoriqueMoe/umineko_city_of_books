-- +goose Up
ALTER TABLE posts ADD COLUMN hot_score DOUBLE PRECISION NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) / 45000)::double precision;

-- +goose StatementBegin
CREATE FUNCTION refresh_post_hot_score(target UUID) RETURNS VOID AS $$
    UPDATE posts p
    SET hot_score = log((1
            + (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = target)
            + 2 * (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = target))::double precision)
        + (EXTRACT(EPOCH FROM p.created_at) / 45000)::double precision
    WHERE p.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION post_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_post_hot_score(OLD.post_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_post_hot_score(NEW.post_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER post_likes_hot_score AFTER INSERT OR DELETE ON post_likes
    FOR EACH ROW EXECUTE FUNCTION post_engagement_changed();

CREATE TRIGGER post_comments_hot_score AFTER INSERT OR DELETE ON post_comments
    FOR EACH ROW EXECUTE FUNCTION post_engagement_changed();

UPDATE posts p
SET hot_score = log((1
        + (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id)
        + 2 * (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id))::double precision)
    + (EXTRACT(EPOCH FROM p.created_at) / 45000)::double precision;

CREATE INDEX idx_posts_corner_hot_score ON posts (corner, hot_score DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_posts_corner_hot_score;
DROP TRIGGER IF EXISTS post_comments_hot_score ON post_comments;
DROP TRIGGER IF EXISTS post_likes_hot_score ON post_likes;
DROP FUNCTION IF EXISTS post_engagement_changed();
DROP FUNCTION IF EXISTS refresh_post_hot_score(UUID);
ALTER TABLE posts DROP COLUMN IF EXISTS hot_score;
