package repository

import (
	"context"

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
)

type statsRepository struct {
	dao StatsRepository
}

func NewStatsRepo(dao StatsRepository) StatsRepository {
	return &statsRepository{dao: dao}
}

func (r *statsRepository) GetOverview(ctx context.Context) (*SiteStats, error) {
	return r.dao.GetOverview(ctx)
}

func (r *statsRepository) GetMostActiveUsers(ctx context.Context, limit int) ([]ActiveUser, error) {
	return r.dao.GetMostActiveUsers(ctx, limit)
}
