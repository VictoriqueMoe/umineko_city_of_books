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
