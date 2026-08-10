package homefeed

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

var echoWindows = []struct {
	ago   string
	label string
}{
	{ago: "1 year", label: "one year ago today"},
	{ago: "1 month", label: "one month ago today"},
}

const (
	echoCandidateLimit   = 8
	defaultActivityLimit = 10
	defaultMembersLimit  = 5
	defaultRoomsLimit    = 5
)

type (
	Service interface {
		HomeActivity(ctx context.Context) (*dto.HomeActivityResponse, error)
		SidebarActivity(ctx context.Context) (*dto.SidebarActivityResponse, error)
	}

	service struct {
		repo  repository.HomeFeedRepository
		hub   *ws.Hub
		cache *cache.Manager
	}
)

func NewService(repo repository.HomeFeedRepository, hub *ws.Hub, cacheMgr *cache.Manager) Service {
	return &service{repo: repo, hub: hub, cache: cacheMgr}
}

func (s *service) echoes(ctx context.Context) []dto.HomeEcho {
	key := cache.HomeEchoes.Key(time.Now().UTC().Format("2006-01-02"))
	if cached, err := cache.Get[[]dto.HomeEcho](ctx, s.cache, key); err == nil {
		return cached
	}

	echoes := s.buildEchoes(ctx)
	_ = cache.Set(ctx, s.cache, key, echoes, cache.HomeEchoes.TTL)

	return echoes
}

func (s *service) buildEchoes(ctx context.Context) []dto.HomeEcho {
	for _, window := range echoWindows {
		rows, err := s.repo.ListEchoes(ctx, window.ago, echoCandidateLimit)
		if err != nil {
			logger.Log.Warn().Err(err).Str("ago", window.ago).Msg("list echoes failed")
			return nil
		}
		if len(rows) == 0 {
			continue
		}

		out := make([]dto.HomeEcho, len(rows))
		for i, r := range rows {
			out[i] = dto.HomeEcho{
				Kind:      r.Kind,
				ID:        r.ID,
				Title:     r.Title,
				Excerpt:   r.Body,
				Corner:    r.Corner,
				Episode:   r.Episode,
				IsSpoiler: r.IsSpoiler,
				URL:       activityURL(r.Kind, r.ID),
				Age:       window.label,
				CreatedAt: r.CreatedAt,
				Author: dto.HomeActivityAuthor{
					ID:          r.AuthorID,
					Username:    r.Username,
					DisplayName: r.DisplayName,
					AvatarURL:   r.AvatarURL,
				},
			}
		}
		return out
	}

	return nil
}

func (s *service) HomeActivity(ctx context.Context) (*dto.HomeActivityResponse, error) {
	activity, err := s.repo.ListRecentActivity(ctx, defaultActivityLimit)
	if err != nil {
		return nil, fmt.Errorf("activity: %w", err)
	}
	members, err := s.repo.ListRecentMembers(ctx, defaultMembersLimit)
	if err != nil {
		return nil, fmt.Errorf("members: %w", err)
	}
	rooms, err := s.repo.ListPublicRooms(ctx, defaultRoomsLimit)
	if err != nil {
		return nil, fmt.Errorf("rooms: %w", err)
	}
	corners, err := s.repo.ListCornerActivity24h(ctx)
	if err != nil {
		return nil, fmt.Errorf("corners: %w", err)
	}

	resp := &dto.HomeActivityResponse{
		OnlineCount:    s.hub.OnlineCount(),
		RecentActivity: make([]dto.HomeActivityEntry, len(activity)),
		RecentMembers:  make([]dto.HomeMember, len(members)),
		PublicRooms:    make([]dto.HomePublicRoom, len(rooms)),
		CornerActivity: make([]dto.HomeCornerActivity, len(corners)),
		Echoes:         s.echoes(ctx),
	}

	for i, a := range activity {
		resp.RecentActivity[i] = dto.HomeActivityEntry{
			Kind:      a.Kind,
			ID:        a.ID,
			Title:     a.Title,
			Excerpt:   a.Body,
			Corner:    a.Corner,
			URL:       activityURL(a.Kind, a.ID),
			CreatedAt: a.CreatedAt,
			Author: dto.HomeActivityAuthor{
				ID:          a.AuthorID,
				Username:    a.Username,
				DisplayName: a.DisplayName,
				AvatarURL:   a.AvatarURL,
			},
		}
	}
	for i, m := range members {
		resp.RecentMembers[i] = dto.HomeMember{
			ID:          m.ID,
			Username:    m.Username,
			DisplayName: m.DisplayName,
			AvatarURL:   m.AvatarURL,
			CreatedAt:   m.CreatedAt,
		}
	}
	for i, rr := range rooms {
		resp.PublicRooms[i] = dto.HomePublicRoom{
			ID:            rr.ID,
			Name:          rr.Name,
			Description:   rr.Description,
			MemberCount:   rr.MemberCount,
			LastMessageAt: rr.LastMessageAt,
		}
	}
	for i, cc := range corners {
		resp.CornerActivity[i] = dto.HomeCornerActivity{
			Corner:        cc.Corner,
			PostCount:     cc.PostCount,
			UniquePosters: cc.UniquePosters,
			LastPostAt:    cc.LastPostAt,
		}
	}
	return resp, nil
}

func (s *service) SidebarActivity(ctx context.Context) (*dto.SidebarActivityResponse, error) {
	entries, err := s.repo.ListSidebarActivity(ctx)
	if err != nil {
		return nil, err
	}
	activity := make(map[string]string, len(entries))
	for _, e := range entries {
		activity[e.Key] = e.LatestAt
	}
	return &dto.SidebarActivityResponse{Activity: activity}, nil
}

func activityURL(kind string, id uuid.UUID) string {
	switch kind {
	case "theory":
		return fmt.Sprintf("/theory/%s", id)
	case "post":
		return fmt.Sprintf("/game-board/%s", id)
	case "journal":
		return fmt.Sprintf("/journals/%s", id)
	case "art":
		return fmt.Sprintf("/gallery/art/%s", id)
	default:
		return "/"
	}
}
