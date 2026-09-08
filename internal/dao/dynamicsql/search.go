package dynamicsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	dbtx interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}

	searchDAO struct {
		db *sql.DB
	}
)

const SearchHeadlineOptions = `'MaxFragments=1, MaxWords=18, MinWords=5, ShortWord=3, HighlightAll=false, StartSel=<mark>, StopSel=</mark>'`

func NewSearch(db *sql.DB) *searchDAO {
	return &searchDAO{db: db}
}

func txOrDB(db *sql.DB, tx []*sql.Tx) dbtx {
	if len(tx) > 0 && tx[0] != nil {
		return tx[0]
	}

	return db
}

func genQueries(db *sql.DB, tx []*sql.Tx) *sqlcgen.Queries {
	if len(tx) > 0 && tx[0] != nil {
		return sqlcgen.New(tx[0])
	}

	return sqlcgen.New(db)
}

func (r *searchDAO) Search(ctx context.Context, s spec.SearchQuery, tx ...*sql.Tx) ([]model.SearchResult, int, error) {
	srcs := model.ResolveSearchTypes(s.Types)
	if len(srcs) == 0 {
		return nil, 0, nil
	}

	subqueries := make([]string, len(srcs))
	for i, src := range srcs {
		subqueries[i] = buildSearchSubquery(src)
	}
	union := strings.Join(subqueries, "\nUNION ALL\n")

	countSQL := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT COUNT(*) FROM (%s) results`, union)

	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, countSQL, s.Query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("search count: %w", err)
	}

	dataSQL := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT entity_type, id, parent_id, parent_title, title, snippet,
               author_id, author_username, author_display_name, author_avatar_url, created_at, rank
        FROM (%s) results
        ORDER BY rank DESC, created_at DESC
        LIMIT $2 OFFSET $3`, union)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, dataSQL, s.Query, s.Limit, s.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	results, err := scanSearchRows(rows, s.Limit)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func (r *searchDAO) QuickSearch(ctx context.Context, s spec.QuickSearchQuery, tx ...*sql.Tx) ([]model.SearchResult, error) {
	sources := model.SearchSources()

	subqueries := make([]string, len(sources))
	for i, src := range sources {
		subqueries[i] = fmt.Sprintf(`(SELECT * FROM (%s) sub ORDER BY rank DESC, created_at DESC LIMIT %d)`, buildSearchSubquery(src), s.PerTypeLimit)
	}
	union := strings.Join(subqueries, "\nUNION ALL\n")

	sqlStr := fmt.Sprintf(`WITH q AS (SELECT websearch_to_tsquery('english', $1) AS tsq, $1 AS qstr)
        SELECT entity_type, id, parent_id, parent_title, title, snippet,
               author_id, author_username, author_display_name, author_avatar_url, created_at, rank
        FROM (%s) results
        ORDER BY rank DESC, created_at DESC`, union)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, sqlStr, s.Query)
	if err != nil {
		return nil, fmt.Errorf("quick search: %w", err)
	}
	defer rows.Close()

	return scanSearchRows(rows, s.PerTypeLimit*len(sources))
}

func buildSearchSubquery(s model.SearchSource) string {
	parentIDExpr := s.ParentIDExpr
	if parentIDExpr == "" {
		parentIDExpr = "NULL::text"
	}

	parentTitleExpr := s.ParentTitleExpr
	if parentTitleExpr == "" {
		parentTitleExpr = "NULL::text"
	}

	var rankExpr strings.Builder
	rankExpr.WriteString("ts_rank_cd(" + s.SearchVector + ", q.tsq)")
	matchExpr := s.SearchVector + " @@ q.tsq"

	trigramExprs := append([]string(nil), s.TrigramExprs...)
	if s.TrigramOnTitle {
		trigramExprs = append([]string{s.TitleExpr}, trigramExprs...)
	}

	for _, expr := range trigramExprs {
		rankExpr.WriteString(" + COALESCE(similarity(" + expr + ", q.qstr), 0)")
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
            COALESCE(u.id::text, '') AS author_id, COALESCE(u.username, '') AS author_username, COALESCE(u.display_name, '') AS author_display_name, COALESCE(u.avatar_url, '') AS author_avatar_url,
            %s AS created_at,
            (%s)::float8 AS rank
        %s
        WHERE %s`,
		s.Type, s.IDExpr, parentIDExpr, parentTitleExpr,
		s.TitleExpr,
		s.BodyExpr, SearchHeadlineOptions,
		s.CreatedAt,
		rankExpr.String(),
		strings.Join(parts, "\n        "),
		strings.Join(whereParts, "\n          AND "),
	)
}

func scanSearchRows(rows *sql.Rows, capacity int) ([]model.SearchResult, error) {
	results := make([]model.SearchResult, 0, capacity)

	for rows.Next() {
		var (
			r         model.SearchResult
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

		r.EntityType = model.SearchEntityType(entityT)
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search rows: %w", err)
	}

	return results, nil
}
