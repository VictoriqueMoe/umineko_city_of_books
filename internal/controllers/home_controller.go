package controllers

import (
	"fmt"

	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	homeActivityLimit      = 10
	homeMembersLimit       = 5
	homeRoomsLimit         = 5
	sidebarVisitedKeyLimit = 100
)

func (s *Service) getAllHomeRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupGetHomeActivity,
		s.setupGetSidebarActivity,
		s.setupGetSidebarLastVisited,
		s.setupMarkSidebarVisited,
	}
}

func (s *Service) setupGetHomeActivity(r fiber.Router) {
	r.Get("/home/activity", s.getHomeActivity)
}

func (s *Service) setupGetSidebarActivity(r fiber.Router) {
	r.Get("/sidebar/activity", s.getSidebarActivity)
}

func (s *Service) setupGetSidebarLastVisited(r fiber.Router) {
	r.Get("/sidebar/last-visited", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getSidebarLastVisited)
}

func (s *Service) setupMarkSidebarVisited(r fiber.Router) {
	r.Post("/sidebar/last-visited", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.markSidebarVisited)
}

func (s *Service) getSidebarLastVisited(ctx fiber.Ctx) error {
	userID := ctrlutils.UserID(ctx)
	visited, err := s.SidebarVisitedRepo.ListForUser(ctx.Context(), userID)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load sidebar last visited")
	}
	return ctx.JSON(dto.SidebarLastVisitedResponse{Visited: visited})
}

func (s *Service) markSidebarVisited(ctx fiber.Ctx) error {
	userID := ctrlutils.UserID(ctx)
	var body dto.MarkSidebarVisitedRequest
	if err := ctx.Bind().JSON(&body); err != nil {
		return ctrlutils.BadRequest(ctx, "invalid request body")
	}
	if body.Key == "" {
		return ctrlutils.BadRequest(ctx, "key is required")
	}
	if len(body.Key) > sidebarVisitedKeyLimit {
		return ctrlutils.BadRequest(ctx, "key too long")
	}
	if err := s.SidebarVisitedRepo.Upsert(ctx.Context(), userID, body.Key); err != nil {
		return ctrlutils.InternalError(ctx, "failed to mark sidebar visited")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) getSidebarActivity(ctx fiber.Ctx) error {
	entries, err := s.HomeFeedRepo.ListSidebarActivity(ctx.Context())
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load sidebar activity")
	}
	activity := make(map[string]string, len(entries))
	for i := 0; i < len(entries); i++ {
		activity[entries[i].Key] = entries[i].LatestAt
	}
	return ctx.JSON(dto.SidebarActivityResponse{Activity: activity})
}

func (s *Service) getHomeActivity(ctx fiber.Ctx) error {
	c := ctx.Context()

	activity, err := s.HomeFeedRepo.ListRecentActivity(c, homeActivityLimit)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load activity")
	}
	members, err := s.HomeFeedRepo.ListRecentMembers(c, homeMembersLimit)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load members")
	}
	rooms, err := s.HomeFeedRepo.ListPublicRooms(c, homeRoomsLimit)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load public rooms")
	}
	corners, err := s.HomeFeedRepo.ListCornerActivity24h(c)
	if err != nil {
		return ctrlutils.InternalError(ctx, "failed to load corner activity")
	}

	resp := dto.HomeActivityResponse{
		OnlineCount:    s.Hub.OnlineCount(),
		RecentActivity: make([]dto.HomeActivityEntry, len(activity)),
		RecentMembers:  make([]dto.HomeMember, len(members)),
		PublicRooms:    make([]dto.HomePublicRoom, len(rooms)),
		CornerActivity: make([]dto.HomeCornerActivity, len(corners)),
	}

	for i := 0; i < len(activity); i++ {
		a := activity[i]
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

	for i := 0; i < len(members); i++ {
		m := members[i]
		resp.RecentMembers[i] = dto.HomeMember{
			ID:          m.ID,
			Username:    m.Username,
			DisplayName: m.DisplayName,
			AvatarURL:   m.AvatarURL,
			CreatedAt:   m.CreatedAt,
		}
	}

	for i := 0; i < len(rooms); i++ {
		rr := rooms[i]
		resp.PublicRooms[i] = dto.HomePublicRoom{
			ID:            rr.ID,
			Name:          rr.Name,
			Description:   rr.Description,
			MemberCount:   rr.MemberCount,
			LastMessageAt: rr.LastMessageAt,
		}
	}

	for i := 0; i < len(corners); i++ {
		cc := corners[i]
		resp.CornerActivity[i] = dto.HomeCornerActivity{
			Corner:        cc.Corner,
			PostCount:     cc.PostCount,
			UniquePosters: cc.UniquePosters,
			LastPostAt:    cc.LastPostAt,
		}
	}

	return ctx.JSON(resp)
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
