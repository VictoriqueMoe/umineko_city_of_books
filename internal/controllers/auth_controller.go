package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/session"
	usersvc "umineko_city_of_books/internal/user"

	"github.com/gofiber/fiber/v3"
)

var rulesSettings = map[string]*config.SiteSettingDef{
	"theories":             config.SettingRulesTheories,
	"theories_higurashi":   config.SettingRulesTheoriesHigurashi,
	"mysteries":            config.SettingRulesMysteries,
	"ships":                config.SettingRulesShips,
	"game_board":           config.SettingRulesGameBoard,
	"game_board_umineko":   config.SettingRulesGameBoardUmineko,
	"game_board_higurashi": config.SettingRulesGameBoardHigurashi,
	"game_board_ciconia":   config.SettingRulesGameBoardCiconia,
	"game_board_higanbana": config.SettingRulesGameBoardHiganbana,
	"game_board_roseguns":  config.SettingRulesGameBoardRoseguns,
	"gallery":              config.SettingRulesGallery,
	"gallery_umineko":      config.SettingRulesGalleryUmineko,
	"gallery_higurashi":    config.SettingRulesGalleryHigurashi,
	"gallery_ciconia":      config.SettingRulesGalleryCiconia,
	"fanfiction":           config.SettingRulesFanfiction,
	"journals":             config.SettingRulesJournals,
	"suggestions":          config.SettingRulesSuggestions,
	"chat_rooms":           config.SettingRulesChatRooms,
}

func (s *Service) getAllAuthRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupRegisterRoute,
		s.setupLoginRoute,
		s.setupLogoutRoute,
		s.setupSessionRoute,
		s.setupSiteInfoRoute,
		s.setupGetRulesRoute,
	}
}

func (s *Service) setupRegisterRoute(r fiber.Router) {
	r.Post("/auth/register", middleware.RequireTurnstile(s.SettingsService), s.register)
}

func (s *Service) setupLoginRoute(r fiber.Router) {
	r.Post("/auth/login", middleware.RequireTurnstile(s.SettingsService), s.login)
}

func (s *Service) setupLogoutRoute(r fiber.Router) {
	r.Post("/auth/logout", s.logout)
}

func (s *Service) setupSessionRoute(r fiber.Router) {
	r.Get("/auth/session", middleware.RequireAuth(s.AuthSession, s.AuthzService), s.getSession)
}

func (s *Service) getSession(ctx fiber.Ctx) error {
	userID := utils.UserID(ctx)
	user, err := s.UserRepo.GetByID(ctx.Context(), userID)
	if err != nil || user == nil {
		return utils.Unauthorized(ctx, "not authenticated")
	}
	return ctx.JSON(fiber.Map{"username": user.Username})
}

func (s *Service) setSessionCookie(ctx fiber.Ctx, token string) {
	days := s.SettingsService.GetInt(ctx.Context(), config.SettingSessionDurationDays)
	if days < 1 {
		days = 30
	}

	baseURL := s.SettingsService.Get(ctx.Context(), config.SettingBaseURL)
	secure := strings.HasPrefix(baseURL, "https://")
	ctx.Cookie(&fiber.Cookie{
		Name:     session.CookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		MaxAge:   days * 24 * 60 * 60,
		Path:     "/",
	})
}

func (s *Service) clearSessionCookie(ctx fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     session.CookieName,
		Value:    "",
		HTTPOnly: true,
		SameSite: "Lax",
		MaxAge:   -1,
		Path:     "/",
	})
}

func validateCredentials(creds dto.Credentials) error {
	if creds.GetUsername() == "" || creds.GetPassword() == "" {
		return fiber.NewError(fiber.StatusBadRequest, "username and password are required")
	}
	return nil
}

func (s *Service) register(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.RegisterRequest](ctx)
	if !ok {
		return nil
	}
	if err := validateCredentials(&req); err != nil {
		return utils.BadRequest(ctx, err.Error())
	}

	user, token, err := s.AuthService.Register(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidUsername) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, auth.ErrRegistrationDisabled) {
			return utils.Forbidden(ctx, err.Error())
		}
		if errors.Is(err, auth.ErrInviteRequired) || errors.Is(err, auth.ErrInvalidInvite) {
			return utils.BadRequest(ctx, err.Error())
		}
		if errors.Is(err, auth.ErrPasswordTooShort) {
			minLen := s.SettingsService.GetInt(ctx.Context(), config.SettingMinPasswordLength)
			return utils.BadRequest(ctx, fmt.Sprintf("password must be at least %d characters", minLen))
		}
		if errors.Is(err, usersvc.ErrUsernameTaken) {
			return utils.Conflict(ctx, "username already taken")
		}
		return utils.InternalError(ctx, "failed to register")
	}

	ip, _ := ctx.Locals("client_ip").(string)
	go func() {
		_ = s.UserRepo.UpdateIP(context.Background(), user.ID, ip)
	}()

	s.setSessionCookie(ctx, token)
	return ctx.Status(fiber.StatusCreated).JSON(user)
}

func (s *Service) login(ctx fiber.Ctx) error {
	req, ok := utils.BindJSON[dto.LoginRequest](ctx)
	if !ok {
		return nil
	}
	if err := validateCredentials(&req); err != nil {
		return utils.BadRequest(ctx, err.Error())
	}

	user, token, err := s.AuthService.Login(ctx.Context(), req)
	if err != nil {
		if errors.Is(err, usersvc.ErrInvalidCredentials) {
			return utils.Unauthorized(ctx, "invalid username or password")
		}
		if errors.Is(err, auth.ErrUserBanned) {
			return utils.Forbidden(ctx, "your account has been banned")
		}
		return utils.InternalError(ctx, "failed to login")
	}

	ip, _ := ctx.Locals("client_ip").(string)
	go func() {
		_ = s.UserRepo.UpdateIP(context.Background(), user.ID, ip)
	}()

	s.setSessionCookie(ctx, token)
	return ctx.JSON(user)
}

func (s *Service) logout(ctx fiber.Ctx) error {
	cookie := ctx.Cookies(session.CookieName)
	if err := s.AuthService.Logout(ctx.Context(), cookie); err != nil {
		return utils.InternalError(ctx, "failed to logout")
	}
	s.clearSessionCookie(ctx)
	return utils.OK(ctx)
}

func (s *Service) setupSiteInfoRoute(r fiber.Router) {
	r.Get("/site-info", s.siteInfo)
}

func (s *Service) siteInfo(ctx fiber.Ctx) error {
	topDetectives, _ := s.MysteryService.GetTopDetectiveIDs(ctx.Context())
	topGMs, _ := s.MysteryService.GetTopGMIDs(ctx.Context())

	vanityRoles, _ := s.VanityRoleRepo.List(ctx.Context())
	manualAssignments, _ := s.VanityRoleRepo.GetAllAssignments(ctx.Context())

	assignments := make(map[string][]string)
	for uid, roleIDs := range manualAssignments {
		assignments[uid] = roleIDs
	}
	for _, uid := range topDetectives {
		assignments[uid] = append(assignments[uid], "system_top_detective")
	}
	for _, uid := range topGMs {
		assignments[uid] = append(assignments[uid], "system_top_gm")
	}

	vrList := make([]dto.SiteInfoVanityRole, len(vanityRoles))
	for i, vr := range vanityRoles {
		vrList[i] = dto.SiteInfoVanityRole{
			ID:        vr.ID,
			Label:     vr.Label,
			Color:     vr.Color,
			IsSystem:  vr.IsSystem,
			SortOrder: vr.SortOrder,
		}
	}

	return ctx.JSON(dto.SiteInfoResponse{
		SiteName:              s.SettingsService.Get(ctx.Context(), config.SettingSiteName),
		SiteDescription:       s.SettingsService.Get(ctx.Context(), config.SettingSiteDescription),
		RegistrationType:      s.SettingsService.Get(ctx.Context(), config.SettingRegistrationType),
		AnnouncementBanner:    s.SettingsService.Get(ctx.Context(), config.SettingAnnouncementBanner),
		DefaultTheme:          s.SettingsService.Get(ctx.Context(), config.SettingDefaultTheme),
		MaintenanceMode:       s.SettingsService.GetBool(ctx.Context(), config.SettingMaintenanceMode),
		MaintenanceTitle:      s.SettingsService.Get(ctx.Context(), config.SettingMaintenanceTitle),
		MaintenanceMessage:    s.SettingsService.Get(ctx.Context(), config.SettingMaintenanceMessage),
		TurnstileEnabled:      s.SettingsService.GetBool(ctx.Context(), config.SettingTurnstileEnabled),
		TurnstileSiteKey:      s.SettingsService.Get(ctx.Context(), config.SettingTurnstileSiteKey),
		MaxImageSize:          s.SettingsService.GetInt(ctx.Context(), config.SettingMaxImageSize),
		MaxVideoSize:          s.SettingsService.GetInt(ctx.Context(), config.SettingMaxVideoSize),
		TopDetectiveIDs:       topDetectives,
		TopGMIDs:              topGMs,
		VanityRoles:           vrList,
		VanityRoleAssignments: assignments,
		Version:               config.Version,
	})
}

func (s *Service) setupGetRulesRoute(r fiber.Router) {
	r.Get("/rules/:page", s.getRules)
}

func (s *Service) getRules(ctx fiber.Ctx) error {
	page := ctx.Params("page")
	def, ok := rulesSettings[page]
	if !ok {
		return utils.NotFound(ctx, "unknown page")
	}

	return ctx.JSON(fiber.Map{
		"page":  page,
		"rules": s.SettingsService.Get(ctx.Context(), def),
	})
}
