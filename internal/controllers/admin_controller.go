package controllers

import (
	"errors"

	"umineko_city_of_books/internal/admin"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/role"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *Service) getAllAdminRoutes() []FSetupRoute {
	return []FSetupRoute{
		s.setupAdminGetStats,
		s.setupAdminListUsers,
		s.setupAdminGetUser,
		s.setupAdminSetRole,
		s.setupAdminRemoveRole,
		s.setupAdminBanUser,
		s.setupAdminUnbanUser,
		s.setupAdminDeleteUser,
		s.setupAdminGetSettings,
		s.setupAdminUpdateSettings,
		s.setupAdminGetAuditLog,
		s.setupAdminCreateInvite,
		s.setupAdminListInvites,
		s.setupAdminDeleteInvite,
		s.setupAdminUpdateMysteryScore,
	}
}

func (s *Service) requirePerm(perm authz.Permission) fiber.Handler {
	return middleware.RequirePermission(s.AuthSession, s.AuthzService, perm)
}

func (s *Service) setupAdminGetStats(r fiber.Router) {
	r.Get("/admin/stats", s.requirePerm(authz.PermViewStats), s.adminGetStats)
}

func (s *Service) setupAdminListUsers(r fiber.Router) {
	r.Get("/admin/users", s.requirePerm(authz.PermViewUsers), s.adminListUsers)
}

func (s *Service) setupAdminGetUser(r fiber.Router) {
	r.Get("/admin/users/:id", s.requirePerm(authz.PermViewUsers), s.adminGetUser)
}

func (s *Service) setupAdminSetRole(r fiber.Router) {
	r.Post("/admin/users/:id/role", s.requirePerm(authz.PermManageRoles), s.adminSetRole)
}

func (s *Service) setupAdminRemoveRole(r fiber.Router) {
	r.Delete("/admin/users/:id/role", s.requirePerm(authz.PermManageRoles), s.adminRemoveRole)
}

func (s *Service) setupAdminBanUser(r fiber.Router) {
	r.Post("/admin/users/:id/ban", s.requirePerm(authz.PermBanUser), s.adminBanUser)
}

func (s *Service) setupAdminUnbanUser(r fiber.Router) {
	r.Post("/admin/users/:id/unban", s.requirePerm(authz.PermBanUser), s.adminUnbanUser)
}

func (s *Service) setupAdminDeleteUser(r fiber.Router) {
	r.Delete("/admin/users/:id", s.requirePerm(authz.PermDeleteAnyUser), s.adminDeleteUser)
}

func (s *Service) setupAdminGetSettings(r fiber.Router) {
	r.Get("/admin/settings", s.requirePerm(authz.PermManageSettings), s.adminGetSettings)
}

func (s *Service) setupAdminUpdateSettings(r fiber.Router) {
	r.Put("/admin/settings", s.requirePerm(authz.PermManageSettings), s.adminUpdateSettings)
}

func (s *Service) setupAdminGetAuditLog(r fiber.Router) {
	r.Get("/admin/audit-log", s.requirePerm(authz.PermViewAuditLog), s.adminGetAuditLog)
}

func (s *Service) adminGetStats(ctx fiber.Ctx) error {
	result, err := s.AdminService.GetStats(ctx.Context())
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminListUsers(ctx fiber.Ctx) error {
	search := ctx.Query("search")
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.AdminService.ListUsers(ctx.Context(), search, limit, offset)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminGetUser(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
	}

	result, err := s.AdminService.GetUser(ctx.Context(), targetID)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminSetRole(ctx fiber.Ctx) error {
	actorID, targetID, err := actorAndTarget(ctx)
	if err != nil {
		return err
	}

	var req dto.SetRoleRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.AdminService.SetUserRole(ctx.Context(), actorID, targetID, role.Role(req.Role)); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) adminRemoveRole(ctx fiber.Ctx) error {
	actorID, targetID, err := actorAndTarget(ctx)
	if err != nil {
		return err
	}

	var req dto.SetRoleRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.AdminService.RemoveUserRole(ctx.Context(), actorID, targetID, role.Role(req.Role)); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) adminBanUser(ctx fiber.Ctx) error {
	actorID, targetID, err := actorAndTarget(ctx)
	if err != nil {
		return err
	}

	var req dto.BanUserRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.AdminService.BanUser(ctx.Context(), actorID, targetID, req.Reason); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) adminUnbanUser(ctx fiber.Ctx) error {
	actorID, targetID, err := actorAndTarget(ctx)
	if err != nil {
		return err
	}

	if err := s.AdminService.UnbanUser(ctx.Context(), actorID, targetID); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) adminDeleteUser(ctx fiber.Ctx) error {
	actorID, targetID, err := actorAndTarget(ctx)
	if err != nil {
		return err
	}

	if err := s.AdminService.DeleteUser(ctx.Context(), actorID, targetID); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) adminGetSettings(ctx fiber.Ctx) error {
	result, err := s.AdminService.GetSettings(ctx.Context())
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminUpdateSettings(ctx fiber.Ctx) error {
	actorID := ctx.Locals("userID").(uuid.UUID)

	var req dto.UpdateSettingsRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := s.AdminService.UpdateSettings(ctx.Context(), actorID, req.Settings); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func (s *Service) adminGetAuditLog(ctx fiber.Ctx) error {
	action := ctx.Query("action")
	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.AdminService.GetAuditLog(ctx.Context(), action, limit, offset)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) setupAdminCreateInvite(r fiber.Router) {
	r.Post("/admin/invites", s.requirePerm(authz.PermManageRoles), s.adminCreateInvite)
}

func (s *Service) setupAdminListInvites(r fiber.Router) {
	r.Get("/admin/invites", s.requirePerm(authz.PermManageRoles), s.adminListInvites)
}

func (s *Service) setupAdminDeleteInvite(r fiber.Router) {
	r.Delete("/admin/invites/:code", s.requirePerm(authz.PermManageRoles), s.adminDeleteInvite)
}

func (s *Service) setupAdminUpdateMysteryScore(r fiber.Router) {
	r.Put("/admin/users/:id/mystery-score", s.requirePerm(authz.PermEditMysteryScore), s.adminUpdateMysteryScore)
	r.Put("/admin/users/:id/gm-score", s.requirePerm(authz.PermEditMysteryScore), s.adminUpdateGMScore)
}

func (s *Service) adminUpdateMysteryScore(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
	}
	var req struct {
		DesiredScore int `json:"desired_score"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	rawScore, _ := s.UserRepo.GetDetectiveRawScore(ctx.Context(), targetID)
	adjustment := req.DesiredScore - rawScore
	if err := s.UserRepo.UpdateMysteryScoreAdjustment(ctx.Context(), targetID, adjustment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) adminUpdateGMScore(ctx fiber.Ctx) error {
	targetID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
	}
	var req struct {
		DesiredScore int `json:"desired_score"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	rawScore, _ := s.UserRepo.GetGMRawScore(ctx.Context(), targetID)
	adjustment := req.DesiredScore - rawScore
	if err := s.UserRepo.UpdateGMScoreAdjustment(ctx.Context(), targetID, adjustment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) adminCreateInvite(ctx fiber.Ctx) error {
	actorID := ctx.Locals("userID").(uuid.UUID)

	result, err := s.AdminService.CreateInvite(ctx.Context(), actorID)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) adminListInvites(ctx fiber.Ctx) error {
	limit := fiber.Query[int](ctx, "limit", 50)
	offset := fiber.Query[int](ctx, "offset", 0)

	result, err := s.AdminService.ListInvites(ctx.Context(), limit, offset)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminDeleteInvite(ctx fiber.Ctx) error {
	actorID := ctx.Locals("userID").(uuid.UUID)
	code := ctx.Params("code")

	if err := s.AdminService.DeleteInvite(ctx.Context(), actorID, code); err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func handleAdminError(ctx fiber.Ctx, err error) error {
	if errors.Is(err, admin.ErrUserNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	if errors.Is(err, admin.ErrProtectedUser) {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "this user cannot be modified"})
	}
	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}
