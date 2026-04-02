package controllers

import (
	"umineko_city_of_books/internal/admin"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/notification"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/report"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/ws"
)

type (
	Service struct {
		AuthService         auth.Service
		ProfileService      profile.Service
		TheoryService       theory.Service
		NotificationService notification.Service
		AdminService        admin.Service
		AuthzService        authz.Service
		SettingsService     settings.Service
		ChatService         chat.Service
		ReportService       report.Service
		PostService         postsvc.Service
		FollowService       follow.Service
		ArtService          artsvc.Service
		AuthSession         *session.Manager
		Hub                 *ws.Hub
		HTMLContent         string
	}
)

func NewService(
	authService auth.Service,
	profileService profile.Service,
	theoryService theory.Service,
	notificationService notification.Service,
	adminService admin.Service,
	authzService authz.Service,
	settingsService settings.Service,
	chatService chat.Service,
	reportService report.Service,
	postService postsvc.Service,
	followService follow.Service,
	artService artsvc.Service,
	authSession *session.Manager,
	hub *ws.Hub,
	htmlContent string,
) Service {
	return Service{
		AuthService:         authService,
		ProfileService:      profileService,
		TheoryService:       theoryService,
		NotificationService: notificationService,
		AdminService:        adminService,
		AuthzService:        authzService,
		SettingsService:     settingsService,
		ChatService:         chatService,
		ReportService:       reportService,
		PostService:         postService,
		FollowService:       followService,
		ArtService:          artService,
		AuthSession:         authSession,
		Hub:                 hub,
		HTMLContent:         htmlContent,
	}
}

func (s *Service) GetAPIRoutes() []FSetupRoute {
	var all []FSetupRoute
	all = append(all, s.getAllAuthRoutes()...)
	all = append(all, s.getAllProfileRoutes()...)
	all = append(all, s.getAllTheoryRoutes()...)
	all = append(all, s.getAllNotificationRoutes()...)
	all = append(all, s.getAllAdminRoutes()...)
	all = append(all, s.getAllChatRoutes()...)
	all = append(all, s.getAllReportRoutes()...)
	all = append(all, s.getAllPostRoutes()...)
	all = append(all, s.getAllArtRoutes()...)
	return all
}

func (s *Service) GetPageRoutes() []FSetupRoute {
	return nil
}
