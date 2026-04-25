package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	SiteStats struct {
		TotalUsers      int
		TotalTheories   int
		TotalResponses  int
		TotalVotes      int
		TotalPosts      int
		TotalComments   int
		NewUsers24h     int
		NewUsers7d      int
		NewUsers30d     int
		NewTheories24h  int
		NewTheories7d   int
		NewTheories30d  int
		NewResponses24h int
		NewResponses7d  int
		NewResponses30d int
		NewPosts24h     int
		NewPosts7d      int
		NewPosts30d     int
		PostsByCorner   map[string]int
	}

	ActiveUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		ActionCount int
	}

	StatsRepository interface {
		GetOverview(ctx context.Context) (*SiteStats, error)
		GetMostActiveUsers(ctx context.Context, limit int) ([]ActiveUser, error)
	}

	statsRepository struct {
		db *sql.DB
	}
)

func (r *statsRepository) GetOverview(ctx context.Context) (*SiteStats, error) {
	var s SiteStats

	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&s.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM theories`).Scan(&s.TotalTheories)
	if err != nil {
		return nil, fmt.Errorf("count theories: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM responses`).Scan(&s.TotalResponses)
	if err != nil {
		return nil, fmt.Errorf("count responses: %w", err)
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM theory_votes) + (SELECT COUNT(*) FROM response_votes)`,
	).Scan(&s.TotalVotes)
	if err != nil {
		return nil, fmt.Errorf("count votes: %w", err)
	}

	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM posts`).Scan(&s.TotalPosts)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_comments`).Scan(&s.TotalComments)

	periods := []struct {
		interval  string
		users     *int
		theories  *int
		responses *int
		posts     *int
	}{
		{"1 day", &s.NewUsers24h, &s.NewTheories24h, &s.NewResponses24h, &s.NewPosts24h},
		{"7 days", &s.NewUsers7d, &s.NewTheories7d, &s.NewResponses7d, &s.NewPosts7d},
		{"30 days", &s.NewUsers30d, &s.NewTheories30d, &s.NewResponses30d, &s.NewPosts30d},
	}

	for _, p := range periods {
		_ = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users WHERE created_at > NOW() - $1::interval`, p.interval,
		).Scan(p.users)
		_ = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM theories WHERE created_at > NOW() - $1::interval`, p.interval,
		).Scan(p.theories)
		_ = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM responses WHERE created_at > NOW() - $1::interval`, p.interval,
		).Scan(p.responses)
		_ = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM posts WHERE created_at > NOW() - $1::interval`, p.interval,
		).Scan(p.posts)
	}

	s.PostsByCorner = make(map[string]int)
	cornerRows, err := r.db.QueryContext(ctx, `SELECT corner, COUNT(*) FROM posts GROUP BY corner`)
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

func (r *statsRepository) GetMostActiveUsers(ctx context.Context, limit int) ([]ActiveUser, error) {
	rows, err := r.db.QueryContext(ctx,
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

	var users []ActiveUser
	for rows.Next() {
		var u ActiveUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.ActionCount); err != nil {
			return nil, fmt.Errorf("scan active user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
