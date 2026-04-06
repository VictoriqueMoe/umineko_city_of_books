package controllers

import (
	"umineko_city_of_books/internal/admin"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/media"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/notification"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/report"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	shipsvc "umineko_city_of_books/internal/ship"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
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
		BlockService        block.Service
		AnnouncementRepo    repository.AnnouncementRepository
		MysteryService      mysterysvc.Service
		UserRepo            repository.UserRepository
		ShipService         shipsvc.Service
		UploadService       upload.Service
		MediaProcessor      *media.Processor
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
	blockService block.Service,
	announcementRepo repository.AnnouncementRepository,
	mysteryService mysterysvc.Service,
	userRepo repository.UserRepository,
	shipService shipsvc.Service,
	uploadService upload.Service,
	mediaProcessor *media.Processor,
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
		BlockService:        blockService,
		AnnouncementRepo:    announcementRepo,
		MysteryService:      mysteryService,
		UserRepo:            userRepo,
		ShipService:         shipService,
		UploadService:       uploadService,
		MediaProcessor:      mediaProcessor,
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
	all = append(all, s.getAllBlockRoutes()...)
	all = append(all, s.getAllAnnouncementRoutes()...)
	all = append(all, s.getAllMysteryRoutes()...)
	all = append(all, s.getAllShipRoutes()...)
	return all
}

func (s *Service) GetPageRoutes() []FSetupRoute {
	return nil
}
