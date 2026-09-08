-- name: SitemapListTheories :many
SELECT t.id, t.created_at FROM theories t ORDER BY t.created_at DESC;

-- name: SitemapListPosts :many
SELECT p.id, p.created_at FROM posts p ORDER BY p.created_at DESC;

-- name: SitemapListArt :many
SELECT a.id, a.created_at FROM art a ORDER BY a.created_at DESC;

-- name: SitemapListMysteries :many
SELECT m.id, m.created_at FROM mysteries m ORDER BY m.created_at DESC;

-- name: SitemapListShips :many
SELECT s.id, s.created_at FROM ships s ORDER BY s.created_at DESC;

-- name: SitemapListFanfics :many
SELECT f.id, f.created_at FROM fanfics f WHERE f.status != 'draft' ORDER BY f.created_at DESC;

-- name: SitemapListUsernames :many
SELECT u.username FROM users u WHERE NOT u.is_bot ORDER BY u.created_at DESC;

-- name: SitemapListJournalRows :many
SELECT j.id, COALESCE(j.updated_at, j.created_at)::timestamptz AS journal_updated_at, e.entry_number, e.updated_at
FROM journals j
LEFT JOIN journal_entries e ON e.journal_id = j.id AND NOT e.is_draft
WHERE j.archived_at IS NULL
ORDER BY j.id, e.entry_number;
