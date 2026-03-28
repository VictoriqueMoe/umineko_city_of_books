package controllers

import (
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/profile"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/theory"
)

type (
	Service struct {
		AuthService    auth.Service
		ProfileService profile.Service
		TheoryService  theory.Service
		AuthSession    *session.Manager
		HTMLContent    string
	}
)

func NewService(
	authService auth.Service,
	profileService profile.Service,
	theoryService theory.Service,
	authSession *session.Manager,
	htmlContent string,
) Service {
	return Service{
		AuthService:    authService,
		ProfileService: profileService,
		TheoryService:  theoryService,
		AuthSession:    authSession,
		HTMLContent:    htmlContent,
	}
}

func (s *Service) GetAPIRoutes() []FSetupRoute {
	var all []FSetupRoute
	all = append(all, s.getAllAuthRoutes()...)
	all = append(all, s.getAllProfileRoutes()...)
	all = append(all, s.getAllTheoryRoutes()...)
	return all
}

func (s *Service) GetPageRoutes() []FSetupRoute {
	return nil
}
