package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type (
	seedConfig struct {
		Users          int
		Posts          int
		Likes          int
		Comments       int
		FollowsPerUser int
		RoomsPerUser   int
		RoomSize       int
		MessagesEach   int
		Password       string
	}

	seededUser struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Token    string `json:"token"`
	}
)

const (
	usernamePrefix = "loadtest_"
	roomNamePrefix = "Load test room "
)

var (
	seedCorners = []string{"general", "general", "general", "umineko", "umineko", "higurashi", "ciconia", "higanbana", "roseguns"}

	seedPhrases = []string{
		"Without love, it cannot be seen.",
		"The golden land is closer than you think.",
		"Has anyone worked out the epitaph yet?",
		"Rereading the fourth arc and noticing new things.",
		"Hot take: the tea party scenes are the best part.",
		"Which twilight do you think is the hardest to solve?",
		"Just finished the latest chapter, no spoilers please.",
		"The music in this scene is unbelievable.",
		"I keep coming back to the first game.",
		"Does the red truth ever lie? Discuss.",
	}
)

func seed(ctx context.Context, db *sql.DB, cfg seedConfig) ([]seededUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.MinCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	if err := execStep(ctx, db, "insert users", `
		INSERT INTO users (username, display_name, password_hash, email_verified, created_at, approved_at)
		SELECT $1 || lpad(g::text, 5, '0'), 'Load Tester ' || g, $2, TRUE, NOW() - INTERVAL '60 days', NOW()
		FROM generate_series(1, $3::int) AS g`,
		usernamePrefix, string(hash), cfg.Users); err != nil {
		return nil, err
	}

	users, err := mintSessions(ctx, db)
	if err != nil {
		return nil, err
	}

	if err := execStep(ctx, db, "insert follows", `
		INSERT INTO follows (follower_id, following_id)
		SELECT u.id, f.id
		FROM users u
		CROSS JOIN LATERAL (
			SELECT o.id
			FROM users o
			WHERE starts_with(o.username, $1) AND o.id <> u.id
			ORDER BY random()
			LIMIT $2::int
		) f
		WHERE starts_with(u.username, $1)`,
		usernamePrefix, cfg.FollowsPerUser); err != nil {
		return nil, err
	}

	if err := execStep(ctx, db, "insert posts", `
		WITH ids AS (
			SELECT array_agg(id) AS a FROM users WHERE starts_with(username, $1)
		), corners AS (
			SELECT string_to_array($2, ',') AS a
		), phrases AS (
			SELECT string_to_array($3, '|') AS a
		)
		INSERT INTO posts (user_id, corner, body, created_at)
		SELECT ids.a[1 + floor(random() * cardinality(ids.a))::int],
		       corners.a[1 + floor(random() * cardinality(corners.a))::int],
		       'Load test post ' || g || ': ' || phrases.a[1 + floor(random() * cardinality(phrases.a))::int],
		       NOW() - random() * INTERVAL '30 days'
		FROM ids, corners, phrases, generate_series(1, $4::int) AS g`,
		usernamePrefix, strings.Join(seedCorners, ","), strings.Join(seedPhrases, "|"), cfg.Posts); err != nil {
		return nil, err
	}

	if err := seedEngagement(ctx, db, cfg); err != nil {
		return nil, err
	}

	if err := seedChat(ctx, db, cfg); err != nil {
		return nil, err
	}

	return users, nil
}

func mintSessions(ctx context.Context, db *sql.DB) ([]seededUser, error) {
	rows, err := db.QueryContext(ctx, `SELECT id::text, username FROM users WHERE starts_with(username, $1) ORDER BY username`, usernamePrefix)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []seededUser
	for rows.Next() {
		var u seededUser
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	hashed := make([]string, 0, len(users))
	ids := make([]string, 0, len(users))
	for i, u := range users {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("generate token: %w", err)
		}

		users[i].Token = hex.EncodeToString(raw)
		sum := sha256.Sum256([]byte(users[i].Token))
		hashed = append(hashed, hex.EncodeToString(sum[:]))
		ids = append(ids, u.ID)
	}

	if err := execStep(ctx, db, "insert sessions", `
		INSERT INTO sessions (token, user_id, expires_at)
		SELECT t, u::uuid, NOW() + INTERVAL '30 days'
		FROM unnest(string_to_array($1, ','), string_to_array($2, ',')) AS x(t, u)`,
		strings.Join(hashed, ","), strings.Join(ids, ",")); err != nil {
		return nil, err
	}

	return users, nil
}

func seedEngagement(ctx context.Context, db *sql.DB, cfg seedConfig) (err error) {
	if err := execStep(ctx, db, "pause like trigger", `ALTER TABLE post_likes DISABLE TRIGGER post_likes_hot_score`); err != nil {
		return err
	}

	if err := execStep(ctx, db, "pause comment trigger", `ALTER TABLE post_comments DISABLE TRIGGER post_comments_hot_score`); err != nil {
		return err
	}

	defer func() {
		likeErr := execStep(ctx, db, "resume like trigger", `ALTER TABLE post_likes ENABLE TRIGGER post_likes_hot_score`)
		commentErr := execStep(ctx, db, "resume comment trigger", `ALTER TABLE post_comments ENABLE TRIGGER post_comments_hot_score`)
		if err == nil {
			err = errors.Join(likeErr, commentErr)
		}
	}()

	if err := execStep(ctx, db, "insert likes", `
		WITH u AS (
			SELECT array_agg(id) AS a FROM users WHERE starts_with(username, $1)
		), p AS (
			SELECT array_agg(id ORDER BY created_at DESC) AS a FROM posts
		)
		INSERT INTO post_likes (user_id, post_id, created_at)
		SELECT u.a[1 + floor(random() * cardinality(u.a))::int],
		       p.a[1 + floor(power(random(), 3) * cardinality(p.a))::int],
		       NOW() - random() * INTERVAL '30 days'
		FROM u, p, generate_series(1, $2::int)
		ON CONFLICT DO NOTHING`,
		usernamePrefix, cfg.Likes); err != nil {
		return err
	}

	if err := execStep(ctx, db, "insert comments", `
		WITH u AS (
			SELECT array_agg(id) AS a FROM users WHERE starts_with(username, $1)
		), p AS (
			SELECT array_agg(id ORDER BY created_at DESC) AS a FROM posts
		), phrases AS (
			SELECT string_to_array($2, '|') AS a
		)
		INSERT INTO post_comments (post_id, user_id, body, created_at)
		SELECT p.a[1 + floor(power(random(), 2) * cardinality(p.a))::int],
		       u.a[1 + floor(random() * cardinality(u.a))::int],
		       phrases.a[1 + floor(random() * cardinality(phrases.a))::int],
		       NOW() - random() * INTERVAL '30 days'
		FROM u, p, phrases, generate_series(1, $3::int)`,
		usernamePrefix, strings.Join(seedPhrases, "|"), cfg.Comments); err != nil {
		return err
	}

	return execStep(ctx, db, "recount engagement", `SELECT refresh_post_engagement(id) FROM posts`)
}

func seedChat(ctx context.Context, db *sql.DB, cfg seedConfig) error {
	rooms := max(1, (cfg.Users*cfg.RoomsPerUser+cfg.RoomSize-1)/cfg.RoomSize)
	perUser := min(cfg.RoomsPerUser, rooms)
	stride := max(1, rooms/perUser)

	if err := execStep(ctx, db, "insert chat rooms", `
		INSERT INTO chat_rooms (name, type, is_public, description, last_message_at)
		SELECT $1 || g, 'group', TRUE, 'Seeded for load testing', NOW()
		FROM generate_series(1, $2::int) AS g`,
		roomNamePrefix, rooms); err != nil {
		return err
	}

	if err := execStep(ctx, db, "insert chat members", `
		WITH r AS (
			SELECT array_agg(id ORDER BY id) AS a FROM chat_rooms WHERE starts_with(name, $1)
		), u AS (
			SELECT id, row_number() OVER (ORDER BY username) - 1 AS n FROM users WHERE starts_with(username, $2)
		)
		INSERT INTO chat_room_members (room_id, user_id, role)
		SELECT r.a[1 + ((u.n + j * $3::int) % cardinality(r.a))::int], u.id, 'member'
		FROM r, u, generate_series(0, $4::int - 1) AS j
		ON CONFLICT DO NOTHING`,
		roomNamePrefix, usernamePrefix, stride, perUser); err != nil {
		return err
	}

	if err := execStep(ctx, db, "assign chat hosts", `
		UPDATE chat_rooms r
		SET created_by = (SELECT m.user_id FROM chat_room_members m WHERE m.room_id = r.id ORDER BY m.user_id LIMIT 1)
		WHERE starts_with(r.name, $1)`,
		roomNamePrefix); err != nil {
		return err
	}

	if err := execStep(ctx, db, "promote chat hosts", `
		UPDATE chat_room_members m
		SET role = 'host'
		FROM chat_rooms r
		WHERE r.id = m.room_id AND r.created_by = m.user_id AND starts_with(r.name, $1)`,
		roomNamePrefix); err != nil {
		return err
	}

	return execStep(ctx, db, "insert chat messages", `
		WITH rm AS (
			SELECT m.room_id, array_agg(m.user_id) AS a
			FROM chat_room_members m
			JOIN chat_rooms r ON r.id = m.room_id
			WHERE starts_with(r.name, $1)
			GROUP BY m.room_id
		), phrases AS (
			SELECT string_to_array($2, '|') AS a
		)
		INSERT INTO chat_messages (room_id, sender_id, body, created_at)
		SELECT rm.room_id,
		       rm.a[1 + floor(random() * cardinality(rm.a))::int],
		       phrases.a[1 + floor(random() * cardinality(phrases.a))::int],
		       NOW() - random() * INTERVAL '2 days'
		FROM rm, phrases, generate_series(1, $3::int)`,
		roomNamePrefix, strings.Join(seedPhrases, "|"), cfg.MessagesEach)
}

func execStep(ctx context.Context, db *sql.DB, step string, query string, args ...any) error {
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}

	return nil
}
