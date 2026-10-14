-- +goose Up
ALTER TABLE posts ADD COLUMN like_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE posts ADD COLUMN comment_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE art ADD COLUMN like_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ocs ADD COLUMN vote_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ocs ADD COLUMN favourite_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ocs ADD COLUMN comment_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE theories ADD COLUMN vote_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE theories ADD COLUMN vote_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ships ADD COLUMN vote_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ships ADD COLUMN controversy BIGINT NOT NULL DEFAULT 0;
ALTER TABLE ships ADD COLUMN comment_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE journals ADD COLUMN follower_count INTEGER NOT NULL DEFAULT 0;

-- +goose StatementBegin
CREATE FUNCTION refresh_post_engagement(target UUID) RETURNS VOID AS $$
    UPDATE posts p
    SET like_count = c.likes,
        comment_count = c.comments,
        hot_score = log((1 + c.likes + 2 * c.comments)::double precision)
            + (EXTRACT(EPOCH FROM p.created_at) / 45000)::double precision
    FROM (
        SELECT (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = target) AS likes,
               (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = target) AS comments
    ) c
    WHERE p.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION post_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_post_engagement(OLD.post_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_post_engagement(NEW.post_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP FUNCTION refresh_post_hot_score(UUID);

-- +goose StatementBegin
CREATE FUNCTION refresh_art_engagement(target UUID) RETURNS VOID AS $$
    UPDATE art a
    SET like_count = (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = target)
    WHERE a.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION art_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_art_engagement(OLD.art_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_art_engagement(NEW.art_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION refresh_oc_engagement(target UUID) RETURNS VOID AS $$
    UPDATE ocs o
    SET vote_score = COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = target), 0),
        favourite_count = (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = target),
        comment_count = (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = target)
    WHERE o.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION oc_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_oc_engagement(OLD.oc_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_oc_engagement(NEW.oc_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION refresh_theory_engagement(target UUID) RETURNS VOID AS $$
    UPDATE theories t
    SET vote_score = COALESCE((SELECT SUM(v.value) FROM theory_votes v WHERE v.theory_id = target), 0),
        vote_count = (SELECT COUNT(*) FROM theory_votes v WHERE v.theory_id = target)
    WHERE t.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION theory_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_theory_engagement(OLD.theory_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_theory_engagement(NEW.theory_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION refresh_ship_engagement(target UUID) RETURNS VOID AS $$
    UPDATE ships s
    SET vote_score = COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = target), 0),
        controversy = (SELECT COUNT(*) FROM ship_votes v WHERE v.ship_id = target AND v.value = 1)
            * (SELECT COUNT(*) FROM ship_votes v WHERE v.ship_id = target AND v.value = -1),
        comment_count = (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = target)
    WHERE s.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION ship_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_ship_engagement(OLD.ship_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_ship_engagement(NEW.ship_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION refresh_journal_engagement(target UUID) RETURNS VOID AS $$
    UPDATE journals j
    SET follower_count = (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = target)
    WHERE j.id = target;
$$ LANGUAGE sql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION journal_engagement_changed() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_journal_engagement(OLD.journal_id);
        RETURN OLD;
    END IF;

    PERFORM refresh_journal_engagement(NEW.journal_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER art_likes_engagement AFTER INSERT OR DELETE ON art_likes
    FOR EACH ROW EXECUTE FUNCTION art_engagement_changed();

CREATE TRIGGER oc_votes_engagement AFTER INSERT OR UPDATE OR DELETE ON oc_votes
    FOR EACH ROW EXECUTE FUNCTION oc_engagement_changed();

CREATE TRIGGER oc_favourites_engagement AFTER INSERT OR DELETE ON oc_favourites
    FOR EACH ROW EXECUTE FUNCTION oc_engagement_changed();

CREATE TRIGGER oc_comments_engagement AFTER INSERT OR DELETE ON oc_comments
    FOR EACH ROW EXECUTE FUNCTION oc_engagement_changed();

CREATE TRIGGER theory_votes_engagement AFTER INSERT OR UPDATE OR DELETE ON theory_votes
    FOR EACH ROW EXECUTE FUNCTION theory_engagement_changed();

CREATE TRIGGER ship_votes_engagement AFTER INSERT OR UPDATE OR DELETE ON ship_votes
    FOR EACH ROW EXECUTE FUNCTION ship_engagement_changed();

CREATE TRIGGER ship_comments_engagement AFTER INSERT OR DELETE ON ship_comments
    FOR EACH ROW EXECUTE FUNCTION ship_engagement_changed();

CREATE TRIGGER journal_follows_engagement AFTER INSERT OR DELETE ON journal_follows
    FOR EACH ROW EXECUTE FUNCTION journal_engagement_changed();

UPDATE posts p
SET like_count = (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id),
    comment_count = (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id);

UPDATE art a
SET like_count = (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id);

UPDATE ocs o
SET vote_score = COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0),
    favourite_count = (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id),
    comment_count = (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id);

UPDATE theories t
SET vote_score = COALESCE((SELECT SUM(v.value) FROM theory_votes v WHERE v.theory_id = t.id), 0),
    vote_count = (SELECT COUNT(*) FROM theory_votes v WHERE v.theory_id = t.id);

UPDATE ships s
SET vote_score = COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0),
    controversy = (SELECT COUNT(*) FROM ship_votes v WHERE v.ship_id = s.id AND v.value = 1)
        * (SELECT COUNT(*) FROM ship_votes v WHERE v.ship_id = s.id AND v.value = -1),
    comment_count = (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id);

UPDATE journals j
SET follower_count = (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id);

CREATE INDEX idx_posts_corner_created_at ON posts (corner, created_at DESC);
CREATE INDEX idx_posts_corner_like_count ON posts (corner, like_count DESC, created_at DESC);
CREATE INDEX idx_posts_corner_comment_count ON posts (corner, comment_count DESC, created_at DESC);
CREATE INDEX idx_posts_corner_view_count ON posts (corner, view_count DESC, created_at DESC);
CREATE INDEX idx_art_corner_created_at ON art (corner, created_at DESC);
CREATE INDEX idx_art_corner_like_count ON art (corner, like_count DESC, created_at DESC);
CREATE INDEX idx_art_corner_view_count ON art (corner, view_count DESC, created_at DESC);
CREATE INDEX idx_ocs_created_at ON ocs (created_at DESC);
CREATE INDEX idx_ocs_name_lower ON ocs (lower(name));
CREATE INDEX idx_ocs_vote_score_top ON ocs (vote_score DESC, created_at DESC);
CREATE INDEX idx_ocs_vote_score_crack ON ocs (vote_score ASC, created_at DESC);
CREATE INDEX idx_ocs_favourite_count ON ocs (favourite_count DESC, created_at DESC);
CREATE INDEX idx_ocs_comment_count ON ocs (comment_count DESC, created_at DESC);
CREATE INDEX idx_theories_vote_score ON theories (vote_score DESC, created_at DESC);
CREATE INDEX idx_theories_vote_count ON theories (vote_count DESC, created_at DESC);
DROP INDEX idx_theories_credibility;
CREATE INDEX idx_theories_credibility ON theories (credibility_score DESC, created_at DESC);
CREATE INDEX idx_ships_vote_score_top ON ships (vote_score DESC, created_at DESC);
CREATE INDEX idx_ships_vote_score_crack ON ships (vote_score ASC, created_at DESC);
CREATE INDEX idx_ships_controversy ON ships (controversy DESC, created_at DESC);
CREATE INDEX idx_ships_comment_count ON ships (comment_count DESC, created_at DESC);
CREATE INDEX idx_journals_follower_count ON journals (follower_count DESC, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_journals_follower_count;
DROP INDEX IF EXISTS idx_ships_comment_count;
DROP INDEX IF EXISTS idx_ships_controversy;
DROP INDEX IF EXISTS idx_ships_vote_score_crack;
DROP INDEX IF EXISTS idx_ships_vote_score_top;
DROP INDEX IF EXISTS idx_theories_credibility;
CREATE INDEX idx_theories_credibility ON theories (credibility_score DESC);
DROP INDEX IF EXISTS idx_theories_vote_count;
DROP INDEX IF EXISTS idx_theories_vote_score;
DROP INDEX IF EXISTS idx_ocs_comment_count;
DROP INDEX IF EXISTS idx_ocs_favourite_count;
DROP INDEX IF EXISTS idx_ocs_vote_score_crack;
DROP INDEX IF EXISTS idx_ocs_vote_score_top;
DROP INDEX IF EXISTS idx_ocs_name_lower;
DROP INDEX IF EXISTS idx_ocs_created_at;
DROP INDEX IF EXISTS idx_art_corner_view_count;
DROP INDEX IF EXISTS idx_art_corner_like_count;
DROP INDEX IF EXISTS idx_art_corner_created_at;
DROP INDEX IF EXISTS idx_posts_corner_view_count;
DROP INDEX IF EXISTS idx_posts_corner_comment_count;
DROP INDEX IF EXISTS idx_posts_corner_like_count;
DROP INDEX IF EXISTS idx_posts_corner_created_at;

DROP TRIGGER IF EXISTS journal_follows_engagement ON journal_follows;
DROP TRIGGER IF EXISTS ship_comments_engagement ON ship_comments;
DROP TRIGGER IF EXISTS ship_votes_engagement ON ship_votes;
DROP TRIGGER IF EXISTS theory_votes_engagement ON theory_votes;
DROP TRIGGER IF EXISTS oc_comments_engagement ON oc_comments;
DROP TRIGGER IF EXISTS oc_favourites_engagement ON oc_favourites;
DROP TRIGGER IF EXISTS oc_votes_engagement ON oc_votes;
DROP TRIGGER IF EXISTS art_likes_engagement ON art_likes;

DROP FUNCTION IF EXISTS journal_engagement_changed();
DROP FUNCTION IF EXISTS refresh_journal_engagement(UUID);
DROP FUNCTION IF EXISTS ship_engagement_changed();
DROP FUNCTION IF EXISTS refresh_ship_engagement(UUID);
DROP FUNCTION IF EXISTS theory_engagement_changed();
DROP FUNCTION IF EXISTS refresh_theory_engagement(UUID);
DROP FUNCTION IF EXISTS oc_engagement_changed();
DROP FUNCTION IF EXISTS refresh_oc_engagement(UUID);
DROP FUNCTION IF EXISTS art_engagement_changed();
DROP FUNCTION IF EXISTS refresh_art_engagement(UUID);

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
CREATE OR REPLACE FUNCTION post_engagement_changed() RETURNS TRIGGER AS $$
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

DROP FUNCTION IF EXISTS refresh_post_engagement(UUID);

ALTER TABLE journals DROP COLUMN IF EXISTS follower_count;
ALTER TABLE ships DROP COLUMN IF EXISTS comment_count;
ALTER TABLE ships DROP COLUMN IF EXISTS controversy;
ALTER TABLE ships DROP COLUMN IF EXISTS vote_score;
ALTER TABLE theories DROP COLUMN IF EXISTS vote_count;
ALTER TABLE theories DROP COLUMN IF EXISTS vote_score;
ALTER TABLE ocs DROP COLUMN IF EXISTS comment_count;
ALTER TABLE ocs DROP COLUMN IF EXISTS favourite_count;
ALTER TABLE ocs DROP COLUMN IF EXISTS vote_score;
ALTER TABLE art DROP COLUMN IF EXISTS like_count;
ALTER TABLE posts DROP COLUMN IF EXISTS comment_count;
ALTER TABLE posts DROP COLUMN IF EXISTS like_count;
