package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository"
)

const (
	recentCountsQuery = `SELECT
		COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 day'),
		COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '7 days'),
		COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '30 days')
	 FROM %s`
)

type (
	statsDAO struct {
		db *sql.DB
	}
)

func (r *statsDAO) GetOverview(ctx context.Context, tx ...*sql.Tx) (*repository.SiteStats, error) {
	var s repository.SiteStats

	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE NOT is_bot`).Scan(&s.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	err = txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM theories`).Scan(&s.TotalTheories)
	if err != nil {
		return nil, fmt.Errorf("count theories: %w", err)
	}

	err = txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM responses`).Scan(&s.TotalResponses)
	if err != nil {
		return nil, fmt.Errorf("count responses: %w", err)
	}

	err = txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM theory_votes) + (SELECT COUNT(*) FROM response_votes)`,
	).Scan(&s.TotalVotes)
	if err != nil {
		return nil, fmt.Errorf("count votes: %w", err)
	}

	_ = txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM posts`).Scan(&s.TotalPosts)
	_ = txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM post_comments`).Scan(&s.TotalComments)

	recent := []struct {
		from  string
		day   *int
		week  *int
		month *int
	}{
		{"users WHERE NOT is_bot", &s.NewUsers24h, &s.NewUsers7d, &s.NewUsers30d},
		{"theories", &s.NewTheories24h, &s.NewTheories7d, &s.NewTheories30d},
		{"responses", &s.NewResponses24h, &s.NewResponses7d, &s.NewResponses30d},
		{"posts", &s.NewPosts24h, &s.NewPosts7d, &s.NewPosts30d},
	}

	for i := range recent {
		c := recent[i]
		_ = txOrDB(r.db, tx).QueryRowContext(ctx, fmt.Sprintf(recentCountsQuery, c.from)).
			Scan(c.day, c.week, c.month)
	}

	s.PostsByCorner = make(map[string]int)
	cornerRows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT corner, COUNT(*) FROM posts GROUP BY corner`)
	if err == nil {
		defer cornerRows.Close()
		for cornerRows.Next() {
			var corner string
			var count int
			if cornerRows.Scan(&corner, &count) == nil {
				s.PostsByCorner[corner] = count
			}
		}
	}

	return &s, nil
}

func (r *statsDAO) GetMostActiveUsers(ctx context.Context, limit int, tx ...*sql.Tx) ([]repository.ActiveUser, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COUNT(*) as action_count
		 FROM (
			SELECT user_id FROM theories
			UNION ALL
			SELECT user_id FROM responses
			UNION ALL
			SELECT user_id FROM posts
			UNION ALL
			SELECT user_id FROM post_comments
		 ) actions
		 JOIN users u ON actions.user_id = u.id
		 GROUP BY u.id
		 ORDER BY action_count DESC
		 LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("most active users: %w", err)
	}
	defer rows.Close()

	var users []repository.ActiveUser
	for rows.Next() {
		var u repository.ActiveUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.ActionCount); err != nil {
			return nil, fmt.Errorf("scan active user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
