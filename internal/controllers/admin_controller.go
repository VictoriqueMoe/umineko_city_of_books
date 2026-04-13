package controllers

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/admin"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/role"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	roleMutation func(ctx context.Context, actorID, targetID uuid.UUID, r role.Role) error
	scoreReader  func(ctx context.Context, userID uuid.UUID) (int, error)
	scoreUpdater func(ctx context.Context, userID uuid.UUID, adjustment int) error
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
		s.setupAdminListVanityRoles,
		s.setupAdminCreateVanityRole,
		s.setupAdminUpdateVanityRole,
		s.setupAdminDeleteVanityRole,
		s.setupAdminGetVanityRoleUsers,
		s.setupAdminAssignVanityRole,
		s.setupAdminUnassignVanityRole,
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
	targetID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}

	result, err := s.AdminService.GetUser(ctx.Context(), targetID)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) handleRoleMutation(ctx fiber.Ctx, op roleMutation) error {
	actorID, targetID, ok := utils.ActorAndTarget(ctx)
	if !ok {
		return nil
	}

	req, ok := utils.BindJSON[dto.SetRoleRequest](ctx)
	if !ok {
		return nil
	}

	if err := op(ctx.Context(), actorID, targetID, role.Role(req.Role)); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminSetRole(ctx fiber.Ctx) error {
	return s.handleRoleMutation(ctx, s.AdminService.SetUserRole)
}

func (s *Service) adminRemoveRole(ctx fiber.Ctx) error {
	return s.handleRoleMutation(ctx, s.AdminService.RemoveUserRole)
}

func (s *Service) adminBanUser(ctx fiber.Ctx) error {
	actorID, targetID, ok := utils.ActorAndTarget(ctx)
	if !ok {
		return nil
	}

	req, ok := utils.BindJSON[dto.BanUserRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.AdminService.BanUser(ctx.Context(), actorID, targetID, req.Reason); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminUnbanUser(ctx fiber.Ctx) error {
	actorID, targetID, ok := utils.ActorAndTarget(ctx)
	if !ok {
		return nil
	}

	if err := s.AdminService.UnbanUser(ctx.Context(), actorID, targetID); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminDeleteUser(ctx fiber.Ctx) error {
	actorID, targetID, ok := utils.ActorAndTarget(ctx)
	if !ok {
		return nil
	}

	if err := s.AdminService.DeleteUser(ctx.Context(), actorID, targetID); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminGetSettings(ctx fiber.Ctx) error {
	result, err := s.AdminService.GetSettings(ctx.Context())
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminUpdateSettings(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)

	req, ok := utils.BindJSON[dto.UpdateSettingsRequest](ctx)
	if !ok {
		return nil
	}

	if err := s.AdminService.UpdateSettings(ctx.Context(), actorID, req.Settings); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
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

func (s *Service) handleScoreUpdate(ctx fiber.Ctx, getRaw scoreReader, setAdjustment scoreUpdater) error {
	targetID, ok := utils.ParseID(ctx)
	if !ok {
		return nil
	}
	var req struct {
		DesiredScore int `json:"desired_score"`
	}
	if err := ctx.Bind().JSON(&req); err != nil {
		return utils.BadRequest(ctx, "invalid request")
	}
	rawScore, _ := getRaw(ctx.Context(), targetID)
	if err := setAdjustment(ctx.Context(), targetID, req.DesiredScore-rawScore); err != nil {
		return utils.InternalError(ctx, "failed to update")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Service) adminUpdateMysteryScore(ctx fiber.Ctx) error {
	return s.handleScoreUpdate(ctx, s.UserRepo.GetDetectiveRawScore, s.UserRepo.UpdateMysteryScoreAdjustment)
}

func (s *Service) adminUpdateGMScore(ctx fiber.Ctx) error {
	return s.handleScoreUpdate(ctx, s.UserRepo.GetGMRawScore, s.UserRepo.UpdateGMScoreAdjustment)
}

func (s *Service) adminCreateInvite(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)

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
	actorID := utils.UserID(ctx)
	code := ctx.Params("code")

	if err := s.AdminService.DeleteInvite(ctx.Context(), actorID, code); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func handleAdminError(ctx fiber.Ctx, err error) error {
	if errors.Is(err, admin.ErrUserNotFound) {
		return utils.NotFound(ctx, "user not found")
	}
	if errors.Is(err, admin.ErrProtectedUser) {
		return utils.Forbidden(ctx, "this user cannot be modified")
	}
	if errors.Is(err, admin.ErrVanityRoleNotFound) {
		return utils.NotFound(ctx, "vanity role not found")
	}
	if errors.Is(err, admin.ErrSystemRole) {
		return utils.Forbidden(ctx, "cannot modify system role assignments")
	}
	return utils.InternalError(ctx, err.Error())
}

func (s *Service) setupAdminListVanityRoles(r fiber.Router) {
	r.Get("/admin/vanity-roles", s.requirePerm(authz.PermManageVanityRoles), s.adminListVanityRoles)
}

func (s *Service) setupAdminCreateVanityRole(r fiber.Router) {
	r.Post("/admin/vanity-roles", s.requirePerm(authz.PermManageVanityRoles), s.adminCreateVanityRole)
}

func (s *Service) setupAdminUpdateVanityRole(r fiber.Router) {
	r.Put("/admin/vanity-roles/:id", s.requirePerm(authz.PermManageVanityRoles), s.adminUpdateVanityRole)
}

func (s *Service) setupAdminDeleteVanityRole(r fiber.Router) {
	r.Delete("/admin/vanity-roles/:id", s.requirePerm(authz.PermManageVanityRoles), s.adminDeleteVanityRole)
}

func (s *Service) setupAdminGetVanityRoleUsers(r fiber.Router) {
	r.Get("/admin/vanity-roles/:id/users", s.requirePerm(authz.PermManageVanityRoles), s.adminGetVanityRoleUsers)
}

func (s *Service) setupAdminAssignVanityRole(r fiber.Router) {
	r.Post("/admin/vanity-roles/:id/users", s.requirePerm(authz.PermManageVanityRoles), s.adminAssignVanityRole)
}

func (s *Service) setupAdminUnassignVanityRole(r fiber.Router) {
	r.Delete("/admin/vanity-roles/:id/users/:userId", s.requirePerm(authz.PermManageVanityRoles), s.adminUnassignVanityRole)
}

func (s *Service) adminListVanityRoles(ctx fiber.Ctx) error {
	roles, err := s.AdminService.ListVanityRoles(ctx.Context())
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(roles)
}

func (s *Service) adminCreateVanityRole(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	req, ok := utils.BindJSON[dto.CreateVanityRoleRequest](ctx)
	if !ok {
		return nil
	}
	result, err := s.AdminService.CreateVanityRole(ctx.Context(), actorID, req)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(result)
}

func (s *Service) adminUpdateVanityRole(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	id := ctx.Params("id")
	req, ok := utils.BindJSON[dto.UpdateVanityRoleRequest](ctx)
	if !ok {
		return nil
	}
	if err := s.AdminService.UpdateVanityRole(ctx.Context(), actorID, id, req); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminDeleteVanityRole(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	id := ctx.Params("id")
	if err := s.AdminService.DeleteVanityRole(ctx.Context(), actorID, id); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminGetVanityRoleUsers(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	search := ctx.Query("search")
	limit := fiber.Query[int](ctx, "limit", 20)
	offset := fiber.Query[int](ctx, "offset", 0)
	result, err := s.AdminService.GetVanityRoleUsers(ctx.Context(), id, search, limit, offset)
	if err != nil {
		return handleAdminError(ctx, err)
	}
	return ctx.JSON(result)
}

func (s *Service) adminAssignVanityRole(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roleID := ctx.Params("id")
	req, ok := utils.BindJSON[dto.AssignVanityRoleRequest](ctx)
	if !ok {
		return nil
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return utils.BadRequest(ctx, "invalid user id")
	}
	if err := s.AdminService.AssignVanityRole(ctx.Context(), actorID, roleID, userID); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}

func (s *Service) adminUnassignVanityRole(ctx fiber.Ctx) error {
	actorID := utils.UserID(ctx)
	roleID := ctx.Params("id")
	userID, ok := utils.ParseIDParam(ctx, "userId")
	if !ok {
		return nil
	}
	if err := s.AdminService.UnassignVanityRole(ctx.Context(), actorID, roleID, userID); err != nil {
		return handleAdminError(ctx, err)
	}
	return utils.OK(ctx)
}
