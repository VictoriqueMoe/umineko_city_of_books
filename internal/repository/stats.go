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
		NewUsers24h     int
		NewUsers7d      int
		NewUsers30d     int
		NewTheories24h  int
		NewTheories7d   int
		NewTheories30d  int
		NewResponses24h int
		NewResponses7d  int
		NewResponses30d int
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

	periods := []struct {
		interval  string
		users     *int
		theories  *int
		responses *int
	}{
		{"-1 day", &s.NewUsers24h, &s.NewTheories24h, &s.NewResponses24h},
		{"-7 days", &s.NewUsers7d, &s.NewTheories7d, &s.NewResponses7d},
		{"-30 days", &s.NewUsers30d, &s.NewTheories30d, &s.NewResponses30d},
	}

	for _, p := range periods {
		r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users WHERE created_at > datetime('now', ?)`, p.interval,
		).Scan(p.users)
		r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM theories WHERE created_at > datetime('now', ?)`, p.interval,
		).Scan(p.theories)
		r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM responses WHERE created_at > datetime('now', ?)`, p.interval,
		).Scan(p.responses)
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
		 ) actions
		 JOIN users u ON actions.user_id = u.id
		 GROUP BY u.id
		 ORDER BY action_count DESC
		 LIMIT ?`, limit,
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
