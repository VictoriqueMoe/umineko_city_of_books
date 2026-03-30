package controllers

import (
	"umineko_city_of_books/internal/admin"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/profile"
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
	return all
}

func (s *Service) GetPageRoutes() []FSetupRoute {
	return nil
}
