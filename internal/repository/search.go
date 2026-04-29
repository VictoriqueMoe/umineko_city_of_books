package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type (
	SearchEntityType string

	SearchResult struct {
		EntityType        SearchEntityType
		ID                string
		ParentID          *string
		ParentTitle       *string
		Title             string
		Snippet           string
		AuthorID          *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		CreatedAt         string
		Rank              float64
	}

	SearchSource struct {
		Type SearchEntityType

		From         string
		AuthorJoin   string
		ParentJoin   string
		IDExpr       string
		TitleExpr    string
		BodyExpr     string
		SearchVector string
		CreatedAt    string

		ParentIDExpr    string
		ParentTitleExpr string

		TrigramOnTitle bool
		TrigramExprs   []string

		ExtraWhere string
	}

	SearchRepository interface {
		Search(ctx context.Context, query string, types []SearchEntityType, limit, offset int) ([]SearchResult, int, error)
		QuickSearch(ctx context.Context, query string, perTypeLimit int) ([]SearchResult, error)
	}

	searchRepository struct {
		db *sql.DB
	}
)

const (
	SearchEntityTheory              SearchEntityType = "theory"
	SearchEntityResponse            SearchEntityType = "response"
	SearchEntityPost                SearchEntityType = "post"
	SearchEntityPostComment         SearchEntityType = "post_comment"
	SearchEntityArt                 SearchEntityType = "art"
	SearchEntityArtComment          SearchEntityType = "art_comment"
	SearchEntityMystery             SearchEntityType = "mystery"
	SearchEntityMysteryAttempt      SearchEntityType = "mystery_attempt"
	SearchEntityMysteryComment      SearchEntityType = "mystery_comment"
	SearchEntityShip                SearchEntityType = "ship"
	SearchEntityShipComment         SearchEntityType = "ship_comment"
	SearchEntityAnnouncement        SearchEntityType = "announcement"
	SearchEntityAnnouncementComment SearchEntityType = "announcement_comment"
	SearchEntityFanfic              SearchEntityType = "fanfic"
	SearchEntityFanficComment       SearchEntityType = "fanfic_comment"
	SearchEntityJournal             SearchEntityType = "journal"
	SearchEntityJournalComment      SearchEntityType = "journal_comment"
	SearchEntityUser                SearchEntityType = "user"
)

const searchHeadlineOptions = `'MaxFragments=1, MaxWords=18, MinWords=5, ShortWord=3, HighlightAll=false, StartSel=<mark>, StopSel=</mark>'`

var searchSources = []SearchSource{
	{
		Type:           SearchEntityTheory,
		From:           "theories t",
		AuthorJoin:     "JOIN users u ON t.user_id = u.id",
		IDExpr:         "t.id::text",
		TitleExpr:      "t.title",
		BodyExpr:       "t.body",
		SearchVector:   "t.search_vector",
		CreatedAt:      "t.created_at",
		TrigramOnTitle: true,
	},
	{
		Type:            SearchEntityResponse,
		From:            "responses r",
		AuthorJoin:      "JOIN users u ON r.user_id = u.id",
		ParentJoin:      "JOIN theories t ON r.theory_id = t.id",
		IDExpr:          "r.id::text",
		TitleExpr:       "t.title",
		BodyExpr:        "r.body",
		SearchVector:    "r.search_vector",
		CreatedAt:       "r.created_at",
		ParentIDExpr:    "r.theory_id::text",
		ParentTitleExpr: "t.title",
	},
	{
		Type:         SearchEntityPost,
		From:         "posts p",
		AuthorJoin:   "JOIN users u ON p.user_id = u.id",
		IDExpr:       "p.id::text",
		TitleExpr:    "COALESCE(NULLIF(LEFT(p.body, 80), ''), 'Game Board post')",
		BodyExpr:     "p.body",
		SearchVector: "p.search_vector",
		CreatedAt:    "p.created_at",
	},
	{
		Type:            SearchEntityPostComment,
		From:            "post_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN posts p ON c.post_id = p.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "COALESCE(NULLIF(LEFT(p.body, 60), ''), 'Game Board post')",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.post_id::text",
		ParentTitleExpr: "COALESCE(NULLIF(LEFT(p.body, 60), ''), 'Game Board post')",
	},
	{
		Type:           SearchEntityArt,
		From:           "art a",
		AuthorJoin:     "JOIN users u ON a.user_id = u.id",
		IDExpr:         "a.id::text",
		TitleExpr:      "a.title",
		BodyExpr:       "a.description",
		SearchVector:   "a.search_vector",
		CreatedAt:      "a.created_at",
		TrigramOnTitle: true,
	},
	{
		Type:            SearchEntityArtComment,
		From:            "art_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN art a ON c.art_id = a.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "a.title",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.art_id::text",
		ParentTitleExpr: "a.title",
	},
	{
		Type:           SearchEntityMystery,
		From:           "mysteries m",
		AuthorJoin:     "JOIN users u ON m.user_id = u.id",
		IDExpr:         "m.id::text",
		TitleExpr:      "m.title",
		BodyExpr:       "m.body",
		SearchVector:   "m.search_vector",
		CreatedAt:      "m.created_at",
		TrigramOnTitle: true,
	},
	{
		Type:            SearchEntityMysteryAttempt,
		From:            "mystery_attempts a",
		AuthorJoin:      "JOIN users u ON a.user_id = u.id",
		ParentJoin:      "JOIN mysteries m ON a.mystery_id = m.id",
		IDExpr:          "a.id::text",
		TitleExpr:       "m.title",
		BodyExpr:        "a.body",
		SearchVector:    "a.search_vector",
		CreatedAt:       "a.created_at",
		ParentIDExpr:    "a.mystery_id::text",
		ParentTitleExpr: "m.title",
	},
	{
		Type:            SearchEntityMysteryComment,
		From:            "mystery_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN mysteries m ON c.mystery_id = m.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "m.title",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.mystery_id::text",
		ParentTitleExpr: "m.title",
	},
	{
		Type:           SearchEntityShip,
		From:           "ships s",
		AuthorJoin:     "JOIN users u ON s.user_id = u.id",
		IDExpr:         "s.id::text",
		TitleExpr:      "s.title",
		BodyExpr:       "s.description",
		SearchVector:   "s.search_vector",
		CreatedAt:      "s.created_at",
		TrigramOnTitle: true,
	},
	{
		Type:            SearchEntityShipComment,
		From:            "ship_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN ships s ON c.ship_id = s.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "s.title",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.ship_id::text",
		ParentTitleExpr: "s.title",
	},
	{
		Type:           SearchEntityAnnouncement,
		From:           "announcements a",
		AuthorJoin:     "JOIN users u ON a.author_id = u.id",
		IDExpr:         "a.id::text",
		TitleExpr:      "a.title",
		BodyExpr:       "a.body",
		SearchVector:   "a.search_vector",
		CreatedAt:      "a.created_at",
		TrigramOnTitle: true,
	},
	{
		Type:            SearchEntityAnnouncementComment,
		From:            "announcement_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN announcements a ON c.announcement_id = a.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "a.title",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.announcement_id::text",
		ParentTitleExpr: "a.title",
	},
	{
		Type:           SearchEntityFanfic,
		From:           "fanfics f",
		AuthorJoin:     "JOIN users u ON f.user_id = u.id",
		IDExpr:         "f.id::text",
		TitleExpr:      "f.title",
		BodyExpr:       "f.summary",
		SearchVector:   "f.search_vector",
		CreatedAt:      "f.published_at",
		TrigramOnTitle: true,
		ExtraWhere:     "f.status != 'draft'",
	},
	{
		Type:            SearchEntityFanficComment,
		From:            "fanfic_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN fanfics f ON c.fanfic_id = f.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "f.title",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.fanfic_id::text",
		ParentTitleExpr: "f.title",
		ExtraWhere:      "f.status != 'draft'",
	},
	{
		Type:           SearchEntityJournal,
		From:           "journals j",
		AuthorJoin:     "JOIN users u ON j.user_id = u.id",
		IDExpr:         "j.id::text",
		TitleExpr:      "j.title",
		BodyExpr:       "j.body",
		SearchVector:   "j.search_vector",
		CreatedAt:      "j.created_at",
		TrigramOnTitle: true,
		ExtraWhere:     "j.archived_at IS NULL",
	},
	{
		Type:            SearchEntityJournalComment,
		From:            "journal_comments c",
		AuthorJoin:      "JOIN users u ON c.user_id = u.id",
		ParentJoin:      "JOIN journals j ON c.journal_id = j.id",
		IDExpr:          "c.id::text",
		TitleExpr:       "j.title",
		BodyExpr:        "c.body",
		SearchVector:    "c.search_vector",
		CreatedAt:       "c.created_at",
		ParentIDExpr:    "c.journal_id::text",
		ParentTitleExpr: "j.title",
		ExtraWhere:      "j.archived_at IS NULL",
	},
	{
		Type:           SearchEntityUser,
		From:           "users u",
		IDExpr:         "u.id::text",
		TitleExpr:      "u.display_name",
		BodyExpr:       "u.bio",
		SearchVector:   "u.search_vector",
		CreatedAt:      "u.created_at",
		TrigramOnTitle: true,
		TrigramExprs:   []string{"u.username"},
	},
}

var searchSourcesByType = func() map[SearchEntityType]SearchSource {
	m := make(map[SearchEntityType]SearchSource, len(searchSources))
	for _, s := range searchSources {
		m[s.Type] = s
	}
	return m
}()

func SearchSourceFor(t SearchEntityType) (SearchSource, bool) {
	src, ok := searchSourcesByType[t]
	return src, ok
}

func SearchSources() []SearchSource {
	return searchSources
}

func resolveSearchTypes(types []SearchEntityType) []SearchSource {
	if len(types) == 0 {
		return searchSources
	}
	out := make([]SearchSource, 0, len(types))
	seen := make(map[SearchEntityType]bool, len(types))
	for _, t := range types {
		if seen[t] {
			continue
		}
		src, ok := searchSourcesByType[t]
		if !ok {
			continue
		}
		seen[t] = true
		out = append(out, src)
	}
	return out
}

func (s SearchSource) buildSubquery() string {
	parentIDExpr := s.ParentIDExpr
	if parentIDExpr == "" {
		parentIDExpr = "NULL::text"
	}
	parentTitleExpr := s.ParentTitleExpr
	if parentTitleExpr == "" {
		parentTitleExpr = "NULL::text"
	}

	rankExpr := "ts_rank_cd(" + s.SearchVector + ", q.tsq)"
	matchExpr := s.SearchVector + " @@ q.tsq"

	trigramExprs := append([]string(nil), s.TrigramExprs...)
	if s.TrigramOnTitle {
		trigramExprs = append([]string{s.TitleExpr}, trigramExprs...)
	}
	for _, expr := range trigramExprs {
		rankExpr += " + COALESCE(similarity(" + expr + ", q.qstr), 0)"
		matchExpr += " OR " + expr + " % q.qstr"
	}
	if len(trigramExprs) > 0 {
		matchExpr = "(" + matchExpr + ")"
	}

	var parts []string
	parts = append(parts, "FROM "+s.From)
	if s.ParentJoin != "" {
		parts = append(parts, s.ParentJoin)
	}
	if s.AuthorJoin != "" {
		parts = append(parts, s.AuthorJoin)
	}
	parts = append(parts, "CROSS JOIN q")

	whereParts := []string{"u.banned_at IS NULL", "u.locked_at IS NULL"}
	if s.ExtraWhere != "" {
		whereParts = append(whereParts, s.ExtraWhere)
	}
	whereParts = append(whereParts, matchExpr)

	return fmt.Sprintf(`SELECT '%s' AS entity_type, %s AS id, %s AS parent_id, %s AS parent_title,
            %s AS title,
            ts_headline('english', %s, q.tsq, %s) AS snippet,
            u.id::text AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
            %s AS created_at,
            (%s)::float8 AS rank
        %s
        WHERE %s`,
		s.Type, s.IDExpr, parentIDExpr, parentTitleExpr,
		s.TitleExpr,
		s.BodyExpr, searchHeadlineOptions,
		s.CreatedAt,
		rankExpr,
		strings.Join(parts, "\n        "),
		strings.Join(whereParts, "\n          AND "),
	)
}

func (r *searchRepository) Search(ctx context.Context, query string, types []SearchEntityType, limit, offset int) ([]SearchResult, int, error) {
	srcs := resolveSearchTypes(types)
	if len(srcs) == 0 {
		return nil, 0, nil
	}

	subqueries := make([]string, len(srcs))
	for i, src := range srcs {
		subqueries[i] = src.buildSubquery()
	}
	union := strings.Join(subqueries, "\nUNION ALL\n")

	countSQL := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT COUNT(*) FROM (%s) results`, union)

	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("search count: %w", err)
	}

	dataSQL := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT entity_type, id, parent_id, parent_title, title, snippet,
               author_id, author_username, author_display_name, author_avatar_url, created_at, rank
        FROM (%s) results
        ORDER BY rank DESC, created_at DESC
        LIMIT $2 OFFSET $3`, union)

	rows, err := r.db.QueryContext(ctx, dataSQL, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	results, err := scanSearchRows(rows, limit)
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *searchRepository) QuickSearch(ctx context.Context, query string, perTypeLimit int) ([]SearchResult, error) {
	subqueries := make([]string, len(searchSources))
	for i, src := range searchSources {
		subqueries[i] = fmt.Sprintf(`(SELECT * FROM (%s) sub ORDER BY rank DESC, created_at DESC LIMIT %d)`, src.buildSubquery(), perTypeLimit)
	}
	union := strings.Join(subqueries, "\nUNION ALL\n")

	sqlStr := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT entity_type, id, parent_id, parent_title, title, snippet,
               author_id, author_username, author_display_name, author_avatar_url, created_at, rank
        FROM (%s) results
        ORDER BY rank DESC, created_at DESC`, union)

	rows, err := r.db.QueryContext(ctx, sqlStr, query)
	if err != nil {
		return nil, fmt.Errorf("quick search: %w", err)
	}
	defer rows.Close()

	return scanSearchRows(rows, perTypeLimit*len(searchSources))
}

func scanSearchRows(rows *sql.Rows, capacity int) ([]SearchResult, error) {
	results := make([]SearchResult, 0, capacity)
	for rows.Next() {
		var (
			r         SearchResult
			createdAt time.Time
			entityT   string
		)
		if err := rows.Scan(
			&entityT, &r.ID, &r.ParentID, &r.ParentTitle, &r.Title, &r.Snippet,
			&r.AuthorID, &r.AuthorUsername, &r.AuthorDisplayName, &r.AuthorAvatarURL,
			&createdAt, &r.Rank,
		); err != nil {
			return nil, fmt.Errorf("search scan: %w", err)
		}
		r.EntityType = SearchEntityType(entityT)
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search rows: %w", err)
	}
	return results, nil
}
