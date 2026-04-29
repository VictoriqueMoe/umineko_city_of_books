package homefeed

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
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
		repo repository.HomeFeedRepository
		hub  *ws.Hub
	}
)

func NewService(repo repository.HomeFeedRepository, hub *ws.Hub) Service {
	return &service{repo: repo, hub: hub}
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
