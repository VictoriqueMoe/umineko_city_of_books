package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeed_BuildsAConsistentDataset(t *testing.T) {
	// given
	db := daotest.NewDB(t)
	ctx := context.Background()
	cfg := seedConfig{Users: 30, Posts: 60, Likes: 300, Comments: 90, FollowsPerUser: 5, RoomsPerUser: 2, RoomSize: 10, MessagesEach: 4, Password: "pw"}

	// when
	users, err := seed(ctx, db, cfg)

	// then
	require.NoError(t, err)
	require.Len(t, users, cfg.Users)

	count := func(query string, args ...any) int {
		t.Helper()

		var n int
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&n))

		return n
	}

	t.Run("every user can write and holds a working session", func(t *testing.T) {
		assert.Equal(t, cfg.Users, count(`SELECT COUNT(*) FROM users WHERE starts_with(username, 'loadtest_') AND email_verified AND approved_at IS NOT NULL AND created_at < NOW() - INTERVAL '1 day'`))

		for _, u := range users {
			sum := sha256.Sum256([]byte(u.Token))
			assert.Equal(t, 1, count(`SELECT COUNT(*) FROM sessions WHERE token = $1 AND user_id = $2::uuid AND expires_at > NOW()`, hex.EncodeToString(sum[:]), u.ID), u.Username)
		}
	})

	t.Run("each user follows the configured number of other users", func(t *testing.T) {
		assert.Equal(t, cfg.Users*cfg.FollowsPerUser, count(`SELECT COUNT(*) FROM follows`))
		assert.Zero(t, count(`SELECT COUNT(*) FROM follows WHERE follower_id = following_id`))
	})

	t.Run("posts carry engagement counts and hot scores that match their rows", func(t *testing.T) {
		assert.Equal(t, cfg.Posts, count(`SELECT COUNT(*) FROM posts`))
		assert.Equal(t, cfg.Comments, count(`SELECT COUNT(*) FROM post_comments`))
		assert.Positive(t, count(`SELECT COUNT(*) FROM post_likes`))
		assert.Zero(t, count(`
			SELECT COUNT(*) FROM posts p
			WHERE p.like_count <> (SELECT COUNT(*) FROM post_likes l WHERE l.post_id = p.id)
			   OR p.comment_count <> (SELECT COUNT(*) FROM post_comments c WHERE c.post_id = p.id)
			   OR abs(p.hot_score - (log((1 + p.like_count + 2 * p.comment_count)::double precision) + (EXTRACT(EPOCH FROM p.created_at) / 45000)::double precision)) > 1e-9`))
	})

	t.Run("the engagement triggers are switched back on", func(t *testing.T) {
		assert.Equal(t, 2, count(`SELECT COUNT(*) FROM pg_trigger WHERE tgname IN ('post_likes_hot_score', 'post_comments_hot_score') AND tgenabled = 'O'`))
	})

	t.Run("chat rooms stay within their size, each with one host and its messages", func(t *testing.T) {
		rooms := count(`SELECT COUNT(*) FROM chat_rooms WHERE starts_with(name, 'Load test room ')`)
		assert.Equal(t, (cfg.Users*cfg.RoomsPerUser+cfg.RoomSize-1)/cfg.RoomSize, rooms)
		assert.Zero(t, count(`SELECT COUNT(*) FROM (SELECT room_id FROM chat_room_members GROUP BY room_id HAVING COUNT(*) > $1) over_full`, cfg.RoomSize))
		assert.Zero(t, count(`SELECT COUNT(*) FROM chat_rooms r WHERE (SELECT COUNT(*) FROM chat_room_members m WHERE m.room_id = r.id AND m.role = 'host') <> 1`))
		assert.Zero(t, count(`SELECT COUNT(*) FROM users u WHERE starts_with(u.username, 'loadtest_') AND (SELECT COUNT(*) FROM chat_room_members m WHERE m.user_id = u.id) <> $1`, cfg.RoomsPerUser))
		assert.Equal(t, rooms*cfg.MessagesEach, count(`SELECT COUNT(*) FROM chat_messages`))
	})
}

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		configured string
		wantErr    bool
	}{
		{name: "a separate load test database is allowed", target: "umineko_loadtest", configured: "umineko_city_of_books", wantErr: false},
		{name: "a name without loadtest is refused", target: "umineko_city_of_books", configured: "other", wantErr: true},
		{name: "the configured database is refused even when named loadtest", target: "umineko_loadtest", configured: "umineko_loadtest", wantErr: true},
		{name: "a name that would need quoting is refused", target: "loadtest; DROP DATABASE x", configured: "other", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := validateTarget(tc.target, tc.configured)

			// then
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
