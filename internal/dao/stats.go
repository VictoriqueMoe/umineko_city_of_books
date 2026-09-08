package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
)

type (
	StatsDAO interface {
		GetOverview(ctx context.Context, tx ...*sql.Tx) (*model.SiteStats, error)
		GetMostActiveUsers(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.ActiveUser, error)
	}

	statsDAO struct {
		db *sql.DB
	}

	statsRecentRow = sqlcgen.StatsRecentUsersRow
)

func applyRecentCounts(row statsRecentRow, err error, day, week, month *int) {
	if err != nil {
		return
	}

	*day = int(row.DayCount)
	*week = int(row.WeekCount)
	*month = int(row.MonthCount)
}

func (r *statsDAO) GetOverview(ctx context.Context, tx ...*sql.Tx) (*model.SiteStats, error) {
	queries := genQueries(r.db, tx)

	var s model.SiteStats

	users, err := queries.StatsCountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	s.TotalUsers = int(users)

	theories, err := queries.StatsCountTheories(ctx)
	if err != nil {
		return nil, fmt.Errorf("count theories: %w", err)
	}
	s.TotalTheories = int(theories)

	responses, err := queries.StatsCountResponses(ctx)
	if err != nil {
		return nil, fmt.Errorf("count responses: %w", err)
	}
	s.TotalResponses = int(responses)

	votes, err := queries.StatsCountVotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("count votes: %w", err)
	}
	s.TotalVotes = int(votes)

	if posts, postsErr := queries.StatsCountPosts(ctx); postsErr == nil {
		s.TotalPosts = int(posts)
	}

	if comments, commentsErr := queries.StatsCountComments(ctx); commentsErr == nil {
		s.TotalComments = int(comments)
	}

	recentUsers, err := queries.StatsRecentUsers(ctx)
	applyRecentCounts(recentUsers, err, &s.NewUsers24h, &s.NewUsers7d, &s.NewUsers30d)

	recentTheories, err := queries.StatsRecentTheories(ctx)
	applyRecentCounts(statsRecentRow(recentTheories), err, &s.NewTheories24h, &s.NewTheories7d, &s.NewTheories30d)

	recentResponses, err := queries.StatsRecentResponses(ctx)
	applyRecentCounts(statsRecentRow(recentResponses), err, &s.NewResponses24h, &s.NewResponses7d, &s.NewResponses30d)

	recentPosts, err := queries.StatsRecentPosts(ctx)
	applyRecentCounts(statsRecentRow(recentPosts), err, &s.NewPosts24h, &s.NewPosts7d, &s.NewPosts30d)

	s.PostsByCorner = make(map[string]int)

	corners, err := queries.StatsPostsByCorner(ctx)
	if err == nil {
		for _, row := range corners {
			s.PostsByCorner[row.Corner] = int(row.PostCount)
		}
	}

	return &s, nil
}

func (r *statsDAO) GetMostActiveUsers(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.ActiveUser, error) {
	rows, err := genQueries(r.db, tx).StatsMostActiveUsers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("most active users: %w", err)
	}

	var users []model.ActiveUser
	for _, row := range rows {
		users = append(users, model.ActiveUser{
			ID:          row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
			ActionCount: int(row.ActionCount),
		})
	}

	return users, nil
}
