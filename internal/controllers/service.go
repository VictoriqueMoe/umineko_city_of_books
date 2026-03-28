package controllers

import (
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/theory"
)

type (
	Service struct {
		AuthService         auth.Service
		ProfileService      profile.Service
		TheoryService       theory.Service
		NotificationService notification.Service
		AuthSession         *session.Manager
		HTMLContent         string
	}
)

func NewService(
	authService auth.Service,
	profileService profile.Service,
	theoryService theory.Service,
	notificationService notification.Service,
	authSession *session.Manager,
	htmlContent string,
) Service {
	return Service{
		AuthService:         authService,
		ProfileService:      profileService,
		TheoryService:       theoryService,
		NotificationService: notificationService,
		AuthSession:         authSession,
		HTMLContent:         htmlContent,
	}
}

func (s *Service) GetAPIRoutes() []FSetupRoute {
	var all []FSetupRoute
	all = append(all, s.getAllAuthRoutes()...)
	all = append(all, s.getAllProfileRoutes()...)
	all = append(all, s.getAllTheoryRoutes()...)
	all = append(all, s.getAllNotificationRoutes()...)
	return all
}

func (s *Service) GetPageRoutes() []FSetupRoute {
	return nil
}
