package controllers

import (
	"umineko_city_of_books/internal/admin"
	announcementsvc "umineko_city_of_books/internal/announcement"
	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/chat"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/follow"
	"umineko_city_of_books/internal/gameroom"
	"umineko_city_of_books/internal/giphy"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/media"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/notification"
	ocsvc "umineko_city_of_books/internal/oc"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/report"
	"umineko_city_of_books/internal/repository"
	searchsvc "umineko_city_of_books/internal/search"
	secretsvc "umineko_city_of_books/internal/secret"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	shipsvc "umineko_city_of_books/internal/ship"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/upload"
	usersvc "umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/usersecret"
	"umineko_city_of_books/internal/vanityrole"
	"umineko_city_of_books/internal/ws"
)

type (
	Service struct {
		AuthService           auth.Service
		ProfileService        profile.Service
		TheoryService         theory.Service
		NotificationService   notification.Service
		AdminService          admin.Service
		AuthzService          authz.Service
		SettingsService       settings.Service
		ChatService           chat.Service
		ReportService         report.Service
		PostService           postsvc.Service
		FollowService         follow.Service
		ArtService            artsvc.Service
		BlockService          block.Service
		AnnouncementService   announcementsvc.Service
		MysteryService        mysterysvc.Service
		FanficService         fanficsvc.Service
		JournalService        journal.Service
		SecretService         secretsvc.Service
		UserRepo              repository.UserRepository
		UserService           usersvc.Service
		ShipService           shipsvc.Service
		OCService             ocsvc.Service
		UploadService         upload.Service
		MediaProcessor        *media.Processor
		VanityRoleService     vanityrole.Service
		UserSecretService     usersecret.Service
		AuthSession           *session.Manager
		Hub                   *ws.Hub
		GiphyService          giphy.Service
		GiphyFavouriteService giphyfavourite.Service
		GameRoomService       gameroom.Service
		HomeFeedService       homefeed.Service
		SidebarService        sidebar.Service
		SearchService         searchsvc.Service
		HTMLContent           string
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
	announcementService announcementsvc.Service,
	mysteryService mysterysvc.Service,
	userRepo repository.UserRepository,
	userService usersvc.Service,
	shipService shipsvc.Service,
	ocService ocsvc.Service,
	fanficService fanficsvc.Service,
	journalService journal.Service,
	secretService secretsvc.Service,
	uploadService upload.Service,
	mediaProcessor *media.Processor,
	vanityRoleService vanityrole.Service,
	userSecretService usersecret.Service,
	authSession *session.Manager,
	hub *ws.Hub,
	giphyService giphy.Service,
	giphyFavouriteService giphyfavourite.Service,
	gameRoomService gameroom.Service,
	homeFeedService homefeed.Service,
	sidebarService sidebar.Service,
	searchService searchsvc.Service,
	htmlContent string,
) Service {
	return Service{
		AuthService:           authService,
		ProfileService:        profileService,
		TheoryService:         theoryService,
		NotificationService:   notificationService,
		AdminService:          adminService,
		AuthzService:          authzService,
		SettingsService:       settingsService,
		ChatService:           chatService,
		ReportService:         reportService,
		PostService:           postService,
		FollowService:         followService,
		ArtService:            artService,
		BlockService:          blockService,
		AnnouncementService:   announcementService,
		MysteryService:        mysteryService,
		UserRepo:              userRepo,
		UserService:           userService,
		ShipService:           shipService,
		OCService:             ocService,
		FanficService:         fanficService,
		JournalService:        journalService,
		SecretService:         secretService,
		UploadService:         uploadService,
		MediaProcessor:        mediaProcessor,
		VanityRoleService:     vanityRoleService,
		UserSecretService:     userSecretService,
		AuthSession:           authSession,
		Hub:                   hub,
		GiphyService:          giphyService,
		GiphyFavouriteService: giphyFavouriteService,
		GameRoomService:       gameRoomService,
		HomeFeedService:       homeFeedService,
		SidebarService:        sidebarService,
		SearchService:         searchService,
		HTMLContent:           htmlContent,
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
	all = append(all, s.getAllOCRoutes()...)
	all = append(all, s.getAllFanficRoutes()...)
	all = append(all, s.getAllJournalRoutes()...)
	all = append(all, s.getAllSecretRoutes()...)
	all = append(all, s.getAllUserPreferencesRoutes()...)
	all = append(all, s.getAllGiphyRoutes()...)
	all = append(all, s.getAllGameRoomRoutes()...)
	all = append(all, s.getAllHomeRoutes()...)
	all = append(all, s.getAllSearchRoutes()...)
	return all
}

func (s *Service) GetPageRoutes() []FSetupRoute {
	return nil
}
