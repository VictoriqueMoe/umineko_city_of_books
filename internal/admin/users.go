package admin

import (
	"context"
	"fmt"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"
	userpkg "umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *service) assertCanGrantRole(ctx context.Context, actorID uuid.UUID, granted role.Role) error {
	grantedRank, known := roleRank[granted]
	if !known || granted == "" {
		return ErrUnknownRole
	}

	actorRole, err := s.authz.GetRole(ctx, actorID)
	if err != nil {
		return fmt.Errorf("get actor role: %w", err)
	}

	if grantedRank >= roleRank[actorRole] {
		return ErrRoleOutranksActor
	}

	return nil
}

func (s *service) SetUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	if err := s.assertCanGrantRole(ctx, actorID, r); err != nil {
		return err
	}

	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.roleRepo.SetRole(ctx, spec.UserRoleSpec{UserID: targetID, Role: r}); err != nil {
			return fmt.Errorf("set role: %w", err)
		}
		s.auditUserDetails(ctx, actorID, audit.ActionSetRole, targetID, string(r))
		if s.chatSync != nil {
			if err := s.chatSync.EnsureSystemRooms(ctx); err != nil {
				logger.Ctx(ctx).Error().Err(err).Msg("ensure system rooms after role change")
			}
			if err := s.chatSync.SyncSystemRoomMembership(ctx, targetID, r); err != nil {
				logger.Ctx(ctx).Error().Err(err).Str("user_id", targetID.String()).Msg("sync system rooms after role set")
			}
		}
		s.broadcastRoleChange(targetID, string(r))
		return nil
	})
}

func (s *service) RemoveUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.roleRepo.RemoveRole(ctx, spec.UserRoleSpec{UserID: targetID, Role: r}); err != nil {
			return fmt.Errorf("remove role: %w", err)
		}
		s.auditUserDetails(ctx, actorID, audit.ActionRemoveRole, targetID, string(r))
		if s.chatSync != nil {
			if err := s.chatSync.SyncSystemRoomMembership(ctx, targetID, ""); err != nil {
				logger.Ctx(ctx).Error().Err(err).Str("user_id", targetID.String()).Msg("sync system rooms after role remove")
			}
		}
		s.broadcastRoleChange(targetID, "")
		return nil
	})
}

func (s *service) broadcastRoleChange(userID uuid.UUID, newRole string) {
	s.hub.Broadcast(ws.Message{
		Type: "role_changed",
		Data: map[string]any{
			"user_id": userID,
			"role":    newRole,
		},
	})
}

func (s *service) BanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.userRepo.BanUser(ctx, spec.UserBan{UserID: targetID, BannedBy: actorID, Reason: reason}); err != nil {
			return fmt.Errorf("ban user: %w", err)
		}
		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			logger.Ctx(ctx).Error().Err(err).Str("user_id", targetID.String()).Msg("failed to invalidate sessions after ban")
		}
		s.auditUserDetails(ctx, actorID, audit.ActionBanUser, targetID, reason)
		s.broadcastBanChange(targetID, true, reason)
		return nil
	})
}

func (s *service) UnbanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.UnbanUser(ctx, targetID); err != nil {
			return fmt.Errorf("unban user: %w", err)
		}
		s.auditUser(ctx, actorID, audit.ActionUnbanUser, targetID)
		s.broadcastBanChange(targetID, false, "")
		return nil
	})
}

func (s *service) broadcastBanChange(userID uuid.UUID, banned bool, reason string) {
	s.hub.Broadcast(ws.Message{
		Type: "ban_changed",
		Data: map[string]any{
			"user_id":    userID,
			"banned":     banned,
			"ban_reason": reason,
		},
	})
}

func (s *service) LockUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.userRepo.LockUser(ctx, spec.UserLock{UserID: targetID, LockedBy: actorID, Reason: reason}); err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		s.auditUserDetails(ctx, actorID, audit.ActionLockUser, targetID, reason)
		s.broadcastLockChange(targetID, true, reason)
		return nil
	})
}

func (s *service) UnlockUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.UnlockUser(ctx, targetID); err != nil {
			return fmt.Errorf("unlock user: %w", err)
		}
		s.auditUser(ctx, actorID, audit.ActionUnlockUser, targetID)
		s.broadcastLockChange(targetID, false, "")
		return nil
	})
}

func (s *service) ApproveUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.userRepo.ApproveUser(ctx, spec.UserApproval{UserID: targetID, ApprovedBy: actorID}); err != nil {
			return fmt.Errorf("approve user: %w", err)
		}
		s.auditUser(ctx, actorID, audit.ActionApproveUser, targetID)
		return nil
	})
}

func (s *service) UnapproveUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.UnapproveUser(ctx, targetID); err != nil {
			return fmt.Errorf("unapprove user: %w", err)
		}
		s.auditUser(ctx, actorID, audit.ActionUnapproveUser, targetID)
		return nil
	})
}

func (s *service) broadcastLockChange(userID uuid.UUID, locked bool, reason string) {
	s.hub.SendToUser(userID, ws.Message{
		Type: "lock_changed",
		Data: map[string]any{
			"user_id":     userID,
			"locked":      locked,
			"lock_reason": reason,
		},
	})
}

func (s *service) DeleteUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	user, _ := s.userRepo.GetByID(ctx, targetID)

	return s.guardedAction(ctx, actorID, targetID, func() error {
		if user != nil && user.IsBot {
			return ErrBotAccountProtected
		}

		if err := s.userRepo.AdminDeleteAccount(ctx, targetID); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		details := ""
		if user != nil {
			s.uploadSvc.Delete(user.AvatarURL)
			s.uploadSvc.Delete(user.BannerURL)

			details = "username=" + user.Username
		}

		s.auditDetails(ctx, actorID, audit.ActionDeleteUser, audit.TargetUser, targetID.String(), details)
		return nil
	})
}

func (s *service) ResetUserPassword(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (string, error) {
	var newPassword string
	err := s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		generated, err := generatePassword()
		if err != nil {
			return fmt.Errorf("generate password: %w", err)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(generated), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		if err := s.userRepo.SetPasswordHash(ctx, spec.UserPasswordHashUpdate{UserID: targetID, PasswordHash: string(passwordHash)}); err != nil {
			return fmt.Errorf("set password: %w", err)
		}

		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("user_id", targetID.String()).Msg("failed to invalidate sessions after password reset")
		}
		s.auditUser(ctx, actorID, audit.ActionResetPassword, targetID)
		newPassword = generated
		return nil
	})
	if err != nil {
		return "", err
	}
	return newPassword, nil
}

func (s *service) SetUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, email string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		previous := s.currentEmail(ctx, targetID)

		if err := s.authSvc.SetEmailForUser(ctx, targetID, email); err != nil {
			return fmt.Errorf("set email: %w", err)
		}

		s.auditUserDetails(ctx, actorID, audit.ActionSetUserEmail, targetID, changeDetails(previous, email))
		return nil
	})
}

func (s *service) currentEmail(ctx context.Context, targetID uuid.UUID) string {
	usr, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil || usr == nil {
		return ""
	}
	return usr.Email
}

func (s *service) currentDisplayName(ctx context.Context, targetID uuid.UUID) string {
	usr, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil || usr == nil {
		return ""
	}
	return usr.DisplayName
}

func changeDetails(previous, next string) string {
	if previous == "" {
		return next
	}
	return fmt.Sprintf("%s -> %s", previous, next)
}

func (s *service) VerifyUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.authSvc.MarkEmailVerified(ctx, targetID); err != nil {
			return fmt.Errorf("verify email: %w", err)
		}

		s.auditUser(ctx, actorID, audit.ActionVerifyUserEmail, targetID)
		return nil
	})
}

func (s *service) UnverifyUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.authSvc.MarkEmailUnverified(ctx, targetID); err != nil {
			return fmt.Errorf("unverify email: %w", err)
		}

		s.auditUser(ctx, actorID, audit.ActionUnverifyUserEmail, targetID)
		return nil
	})
}

func (s *service) SetUserDisplayName(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, displayName string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		clamped := userpkg.ClampDisplayName(displayName)
		if clamped == "" {
			return ErrEmptyDisplayName
		}

		previous := s.currentDisplayName(ctx, targetID)

		if err := s.userRepo.SetDisplayName(ctx, spec.UserDisplayNameUpdate{UserID: targetID, DisplayName: clamped}); err != nil {
			return fmt.Errorf("set display name: %w", err)
		}

		s.auditUserDetails(ctx, actorID, audit.ActionSetDisplayName, targetID, changeDetails(previous, clamped))
		s.broadcastDisplayNameChange(targetID, clamped)
		return nil
	})
}

func (s *service) SetDisplayNameLocked(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, locked bool) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.SetDisplayNameLocked(ctx, spec.UserDisplayNameLockUpdate{UserID: targetID, Locked: locked}); err != nil {
			return fmt.Errorf("set display name lock: %w", err)
		}

		action := audit.ActionUnlockDisplayName
		if locked {
			action = audit.ActionLockDisplayName
		}
		s.auditUser(ctx, actorID, action, targetID)
		return nil
	})
}

func (s *service) broadcastDisplayNameChange(userID uuid.UUID, displayName string) {
	s.hub.Broadcast(ws.Message{
		Type: "profile_changed",
		Data: map[string]any{
			"user_id":      userID,
			"display_name": displayName,
		},
	})
}

func (s *service) ForceLogout(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			return fmt.Errorf("delete sessions: %w", err)
		}

		s.auditUser(ctx, actorID, audit.ActionForceLogout, targetID)
		return nil
	})
}

func (s *service) ListAccountsOnIP(ctx context.Context, targetID uuid.UUID) (*dto.AdminIPMatchesResponse, error) {
	target, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if target == nil {
		return nil, ErrUserNotFound
	}

	if target.IP == nil || *target.IP == "" {
		return &dto.AdminIPMatchesResponse{Users: []dto.AdminUserItem{}}, nil
	}

	users, err := s.userRepo.ListByIP(ctx, spec.UserIPFilter{IP: *target.IP, ExcludeUserID: targetID})
	if err != nil {
		return nil, fmt.Errorf("list users by ip: %w", err)
	}

	items := make([]dto.AdminUserItem, len(users))
	for i, u := range users {
		items[i] = dto.AdminUserItem{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        role.Role(u.Role),
			Banned:      u.BannedAt != nil,
			Locked:      u.LockedAt != nil,
			CreatedAt:   u.CreatedAt,
		}
	}

	return &dto.AdminIPMatchesResponse{IP: *target.IP, Users: items}, nil
}

func (s *service) GetUserAuditLog(ctx context.Context, targetID uuid.UUID, page bounds.Page) (*dto.AuditLogListResponse, error) {
	entries, total, err := s.auditRepo.ListForUser(ctx, spec.AuditLogUserListing{UserID: targetID, Page: page})
	if err != nil {
		return nil, fmt.Errorf("get user audit log: %w", err)
	}

	return &dto.AuditLogListResponse{
		Entries: toAuditLogEntries(entries),
		Total:   total,
		Limit:   page.Limit(),
		Offset:  page.Offset(),
	}, nil
}
