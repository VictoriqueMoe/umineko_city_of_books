package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	HomeActivityRow struct {
		Kind        string
		ID          uuid.UUID
		Title       string
		Body        string
		Corner      string
		CreatedAt   string
		AuthorID    uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
	}

	HomeMemberRow struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		CreatedAt   string
	}

	HomePublicRoomRow struct {
		ID            uuid.UUID
		Name          string
		Description   string
		MemberCount   int
		LastMessageAt *string
	}

	HomeCornerActivityRow struct {
		Corner        string
		PostCount     int
		UniquePosters int
		LastPostAt    *string
	}

	SidebarActivityEntry struct {
		Key      string
		LatestAt string
	}

	HomeFeedRepository interface {
		ListRecentActivity(ctx context.Context, limit int) ([]HomeActivityRow, error)
		ListRecentMembers(ctx context.Context, limit int) ([]HomeMemberRow, error)
		ListPublicRooms(ctx context.Context, limit int) ([]HomePublicRoomRow, error)
		ListCornerActivity24h(ctx context.Context) ([]HomeCornerActivityRow, error)
		ListSidebarActivity(ctx context.Context) ([]SidebarActivityEntry, error)
	}

	homeFeedRepository struct {
		db *sql.DB
	}
)

const homeActivitySQL = `
WITH feed AS (
    SELECT 'theory' AS kind, t.id AS id, t.title AS title, substr(t.body, 1, 200) AS body,
           t.series AS corner, t.created_at AS created_at, t.user_id AS author_id
    FROM theories t
    UNION ALL
    SELECT 'post' AS kind, p.id AS id, '' AS title, substr(p.body, 1, 200) AS body,
           p.corner AS corner, p.created_at AS created_at, p.user_id AS author_id
    FROM posts p
    UNION ALL
    SELECT 'journal' AS kind, j.id AS id, j.title AS title, substr(j.body, 1, 200) AS body,
           j.work AS corner, j.created_at AS created_at, j.user_id AS author_id
    FROM journals j
    WHERE j.archived_at IS NULL
    UNION ALL
    SELECT 'art' AS kind, a.id AS id, a.title AS title, substr(a.description, 1, 200) AS body,
           a.corner AS corner, a.created_at AS created_at, a.user_id AS author_id
    FROM art a
)
SELECT f.kind, f.id, f.title, f.body, f.corner, f.created_at,
       f.author_id, u.username, u.display_name, u.avatar_url
FROM feed f
JOIN users u ON u.id = f.author_id
WHERE u.banned_at IS NULL
ORDER BY f.created_at DESC
LIMIT $1
`

func (r *homeFeedRepository) ListRecentActivity(ctx context.Context, limit int) ([]HomeActivityRow, error) {
	rows, err := r.db.QueryContext(ctx, homeActivitySQL, limit)
	if err != nil {
		return nil, fmt.Errorf("home feed activity: %w", err)
	}
	defer rows.Close()

	var out []HomeActivityRow
	for rows.Next() {
		var row HomeActivityRow
		if err := rows.Scan(&row.Kind, &row.ID, &row.Title, &row.Body, &row.Corner, &row.CreatedAt,
			&row.AuthorID, &row.Username, &row.DisplayName, &row.AvatarURL); err != nil {
			return nil, fmt.Errorf("scan home activity: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *homeFeedRepository) ListRecentMembers(ctx context.Context, limit int) ([]HomeMemberRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, display_name, avatar_url, created_at
		 FROM users
		 WHERE banned_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("home feed members: %w", err)
	}
	defer rows.Close()

	var out []HomeMemberRow
	for rows.Next() {
		var m HomeMemberRow
		if err := rows.Scan(&m.ID, &m.Username, &m.DisplayName, &m.AvatarURL, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan home member: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *homeFeedRepository) ListCornerActivity24h(ctx context.Context) ([]HomeCornerActivityRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.corner,
		        COUNT(*) AS post_count,
		        COUNT(DISTINCT p.user_id) AS unique_posters,
		        MAX(p.created_at) AS last_post_at
		 FROM posts p
		 JOIN users u ON u.id = p.user_id
		 WHERE p.created_at > NOW() - INTERVAL '1 day' AND u.banned_at IS NULL
		 GROUP BY p.corner`,
	)
	if err != nil {
		return nil, fmt.Errorf("home feed corner activity: %w", err)
	}
	defer rows.Close()

	var out []HomeCornerActivityRow
	for rows.Next() {
		var c HomeCornerActivityRow
		if err := rows.Scan(&c.Corner, &c.PostCount, &c.UniquePosters, &c.LastPostAt); err != nil {
			return nil, fmt.Errorf("scan corner activity: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

const sidebarActivitySQL = `
SELECT 'game_board_' || corner AS key, MAX(created_at) AS latest_at FROM posts GROUP BY corner
UNION ALL
SELECT 'gallery_' || corner AS key, MAX(created_at) AS latest_at FROM art GROUP BY corner
UNION ALL
SELECT 'theories_' || series AS key, MAX(created_at) AS latest_at FROM theories GROUP BY series
UNION ALL
SELECT 'mysteries' AS key, MAX(created_at) AS latest_at FROM mysteries
UNION ALL
SELECT 'secrets' AS key, MAX(created_at) AS latest_at FROM secret_comments
UNION ALL
SELECT 'ships' AS key, MAX(created_at) AS latest_at FROM ships
UNION ALL
SELECT 'fanfiction' AS key, MAX(created_at) AS latest_at FROM fanfics
UNION ALL
SELECT 'journals' AS key, MAX(created_at) AS latest_at FROM journals WHERE archived_at IS NULL
UNION ALL
SELECT 'rooms' AS key, MAX(created_at) AS latest_at FROM chat_rooms WHERE type = 'group' AND is_public = TRUE AND is_system = FALSE
`

func (r *homeFeedRepository) ListSidebarActivity(ctx context.Context) ([]SidebarActivityEntry, error) {
	rows, err := r.db.QueryContext(ctx, sidebarActivitySQL)
	if err != nil {
		return nil, fmt.Errorf("sidebar activity: %w", err)
	}
	defer rows.Close()

	var out []SidebarActivityEntry
	for rows.Next() {
		var key string
		var latest sql.NullString
		if err := rows.Scan(&key, &latest); err != nil {
			return nil, fmt.Errorf("scan sidebar activity: %w", err)
		}
		if !latest.Valid {
			continue
		}
		out = append(out, SidebarActivityEntry{Key: key, LatestAt: latest.String})
	}
	return out, rows.Err()
}

func (r *homeFeedRepository) ListPublicRooms(ctx context.Context, limit int) ([]HomePublicRoomRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT cr.id, cr.name, cr.description,
		        (SELECT COUNT(*) FROM chat_room_members m WHERE m.room_id = cr.id) AS member_count,
		        cr.last_message_at
		 FROM chat_rooms cr
		 WHERE cr.type = 'group' AND cr.is_public = TRUE
		 ORDER BY COALESCE(cr.last_message_at, cr.created_at) DESC
		 LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("home feed public rooms: %w", err)
	}
	defer rows.Close()

	var out []HomePublicRoomRow
	for rows.Next() {
		var rr HomePublicRoomRow
		if err := rows.Scan(&rr.ID, &rr.Name, &rr.Description, &rr.MemberCount, &rr.LastMessageAt); err != nil {
			return nil, fmt.Errorf("scan public room: %w", err)
		}
		out = append(out, rr)
	}
	return out, rows.Err()
}
