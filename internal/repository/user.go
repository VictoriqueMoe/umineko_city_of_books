package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/secrets"

	"github.com/google/uuid"
)

var (
	leaderboardGameTypes = []dto.GameType{
		dto.GameTypeChess,
		dto.GameTypeCheckers,
		dto.GameTypeOthello,
		dto.GameTypeMinesweeper,
		dto.GameTypeSnakesLadders,
	}
)

type (
	NewUser struct {
		Username      string
		Email         string
		PasswordHash  string
		DisplayName   string
		AvatarURL     string
		HomePage      string
		IsBot         bool
		DMsEnabled    bool
		EmailVerified bool
	}

	NewAccount struct {
		User NewUser
		Role role.Role
	}

	NewRegistration struct {
		Account               NewAccount
		InviteCode            string
		VerificationHash      string
		VerificationExpiresAt time.Time
		SessionToken          string
		SessionExpiresAt      time.Time
	}

	PasswordUpdate struct {
		UserID       uuid.UUID
		PasswordHash string
		TokenHash    string
	}

	UserDAO interface {
		Create(ctx context.Context, spec NewUser, tx ...*sql.Tx) (*model.User, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID, tx ...*sql.Tx) ([]model.User, error)
		GetByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, error)
		GetByUsernames(ctx context.Context, usernames []string, tx ...*sql.Tx) ([]model.User, error)
		ExistsByUsername(ctx context.Context, username string, tx ...*sql.Tx) (bool, error)
		Count(ctx context.Context, tx ...*sql.Tx) (int, error)
		UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest, tx ...*sql.Tx) error
		UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string, tx ...*sql.Tx) error
		UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string, tx ...*sql.Tx) error
		UpdateIP(ctx context.Context, userID uuid.UUID, ip string, tx ...*sql.Tx) error
		UpdateGameBoardSort(ctx context.Context, userID uuid.UUID, sort string, tx ...*sql.Tx) error
		UpdateAppearance(ctx context.Context, userID uuid.UUID, theme, font string, wideLayout bool, tx ...*sql.Tx) error
		UpdateMysteryScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int, tx ...*sql.Tx) error
		UpdateGMScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int, tx ...*sql.Tx) error
		GetDetectiveRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetGMRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetPasswordHash(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		SetPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string, tx ...*sql.Tx) error
		SetEmail(ctx context.Context, userID uuid.UUID, email string, tx ...*sql.Tx) error
		SetDisplayName(ctx context.Context, userID uuid.UUID, displayName string, tx ...*sql.Tx) error
		SetDisplayNameLocked(ctx context.Context, userID uuid.UUID, locked bool, tx ...*sql.Tx) error
		ListByIP(ctx context.Context, ip string, excludeUserID uuid.UUID, tx ...*sql.Tx) ([]model.User, error)
		MarkEmailVerified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		MarkEmailUnverified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		EmailInUse(ctx context.Context, email string, excludeUserID uuid.UUID, tx ...*sql.Tx) (bool, error)
		RequiresEmailVerification(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		DeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		GetProfileByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, *model.UserStats, error)
		GetProfileByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, *model.UserStats, error)
		ListAll(ctx context.Context, search string, limit, offset int, tx ...*sql.Tx) ([]model.User, int, error)
		ListPublic(ctx context.Context, tx ...*sql.Tx) ([]model.User, error)
		SearchByName(ctx context.Context, query string, limit int, tx ...*sql.Tx) ([]model.User, error)
		BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string, tx ...*sql.Tx) error
		UnbanUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		IsBanned(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		LockUser(ctx context.Context, userID uuid.UUID, lockedBy uuid.UUID, reason string, tx ...*sql.Tx) error
		UnlockUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		IsLocked(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		AdminDeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	UserRepository interface {
		UserDAO

		RegisterAccount(ctx context.Context, spec NewRegistration, tx ...*sql.Tx) (*model.User, error)
		ResetPassword(ctx context.Context, spec PasswordUpdate, tx ...*sql.Tx) error
		SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool, tx ...*sql.Tx) error
		ConfirmEmailVerification(ctx context.Context, userID uuid.UUID, tokenHash string, tx ...*sql.Tx) error
	}
)

type userRepository struct {
	db            *sql.DB
	dao           UserDAO
	cache         *cache.Manager
	roles         RoleRepository
	audit         AuditLogRepository
	verifications EmailVerificationRepository
	invites       InviteRepository
	sessions      SessionRepository
	resets        PasswordResetRepository
}

func NewUserRepo(database *sql.DB, dao UserDAO, c *cache.Manager, roles RoleRepository, audit AuditLogRepository, verifications EmailVerificationRepository, invites InviteRepository, sessions SessionRepository, resets PasswordResetRepository) UserRepository {
	return &userRepository{
		db:            database,
		dao:           dao,
		cache:         c,
		roles:         roles,
		audit:         audit,
		verifications: verifications,
		invites:       invites,
		sessions:      sessions,
		resets:        resets,
	}
}

func (r *userRepository) RegisterAccount(ctx context.Context, spec NewRegistration, tx ...*sql.Tx) (*model.User, error) {
	var created *model.User

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.dao.Create(ctx, spec.Account.User, tx)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		if spec.Account.Role != "" {
			if err := r.roles.SetRole(ctx, created.ID, spec.Account.Role, tx); err != nil {
				return fmt.Errorf("assign initial role: %w", err)
			}
		}

		verification := NewEmailVerification{
			TokenHash: spec.VerificationHash,
			UserID:    created.ID,
			ExpiresAt: spec.VerificationExpiresAt,
		}

		if err := r.verifications.Issue(ctx, verification, tx); err != nil {
			return fmt.Errorf("store verification token: %w", err)
		}

		entry := NewAuditEntry{
			ActorID:    created.ID,
			Action:     AuditActionUserCreated,
			TargetType: AuditTargetUser,
			TargetID:   created.ID.String(),
			Details:    "username=" + spec.Account.User.Username,
			SubjectID:  created.ID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("write user_created audit log: %w", err)
		}

		if spec.InviteCode != "" {
			if err := r.invites.MarkUsed(ctx, spec.InviteCode, created.ID, tx); err != nil {
				if errors.Is(err, ErrInviteUnavailable) {
					return err
				}

				return fmt.Errorf("mark invite as used: %w", err)
			}
		}

		if err := r.sessions.Create(ctx, spec.SessionToken, created.ID, spec.SessionExpiresAt, tx); err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if spec.Account.Role != "" {
		logger.Log.Info().Str("user_id", created.ID.String()).Str("username", spec.Account.User.Username).Msg("first user created, assigned super admin role")
	}

	return created, nil
}

func (r *userRepository) ResetPassword(ctx context.Context, spec PasswordUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.SetPasswordHash(ctx, spec.UserID, spec.PasswordHash, tx); err != nil {
			return fmt.Errorf("set password: %w", err)
		}

		if err := r.resets.MarkUsed(ctx, spec.TokenHash, tx); err != nil {
			return fmt.Errorf("mark reset token used: %w", err)
		}

		if err := r.sessions.DeleteAllForUser(ctx, spec.UserID, tx); err != nil {
			return fmt.Errorf("invalidate sessions after password reset: %w", err)
		}

		entry := NewAuditEntry{
			ActorID:    spec.UserID,
			Action:     AuditActionPasswordReset,
			TargetType: AuditTargetUser,
			TargetID:   spec.UserID.String(),
			SubjectID:  spec.UserID,
		}

		if err := r.audit.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("write password_reset audit log: %w", err)
		}

		return nil
	})
}

func (r *userRepository) SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if verified {
			if err := r.dao.MarkEmailVerified(ctx, userID, tx); err != nil {
				return fmt.Errorf("mark email verified: %w", err)
			}
		} else {
			if err := r.dao.MarkEmailUnverified(ctx, userID, tx); err != nil {
				return fmt.Errorf("mark email unverified: %w", err)
			}
		}

		if err := r.verifications.DeleteUnusedForUser(ctx, userID, tx); err != nil {
			return fmt.Errorf("clear verification tokens: %w", err)
		}

		return nil
	})
}

func (r *userRepository) ConfirmEmailVerification(ctx context.Context, userID uuid.UUID, tokenHash string, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.MarkEmailVerified(ctx, userID, tx); err != nil {
			return fmt.Errorf("mark email verified: %w", err)
		}

		if err := r.verifications.MarkUsed(ctx, tokenHash, tx); err != nil {
			return fmt.Errorf("mark verification token used: %w", err)
		}

		return nil
	})
}

func (r *userRepository) Create(ctx context.Context, spec NewUser, tx ...*sql.Tx) (*model.User, error) {
	return r.dao.Create(ctx, spec, tx...)
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *userRepository) GetByIDs(ctx context.Context, ids []uuid.UUID, tx ...*sql.Tx) ([]model.User, error) {
	return r.dao.GetByIDs(ctx, ids, tx...)
}

func (r *userRepository) GetByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, error) {
	return r.dao.GetByUsername(ctx, username, tx...)
}

func (r *userRepository) GetByUsernames(ctx context.Context, usernames []string, tx ...*sql.Tx) ([]model.User, error) {
	return r.dao.GetByUsernames(ctx, usernames, tx...)
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string, tx ...*sql.Tx) (bool, error) {
	return r.dao.ExistsByUsername(ctx, username, tx...)
}

func (r *userRepository) Count(ctx context.Context, tx ...*sql.Tx) (int, error) {
	return r.dao.Count(ctx, tx...)
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest, tx ...*sql.Tx) error {
	return r.dao.UpdateProfile(ctx, userID, req, tx...)
}

func (r *userRepository) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateAvatarURL(ctx, userID, avatarURL, tx...)
}

func (r *userRepository) UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string, tx ...*sql.Tx) error {
	return r.dao.UpdateBannerURL(ctx, userID, bannerURL, tx...)
}

func (r *userRepository) UpdateIP(ctx context.Context, userID uuid.UUID, ip string, tx ...*sql.Tx) error {
	return r.dao.UpdateIP(ctx, userID, ip, tx...)
}

func (r *userRepository) UpdateGameBoardSort(ctx context.Context, userID uuid.UUID, sort string, tx ...*sql.Tx) error {
	return r.dao.UpdateGameBoardSort(ctx, userID, sort, tx...)
}

func (r *userRepository) UpdateAppearance(ctx context.Context, userID uuid.UUID, theme, font string, wideLayout bool, tx ...*sql.Tx) error {
	return r.dao.UpdateAppearance(ctx, userID, theme, font, wideLayout, tx...)
}

func (r *userRepository) UpdateMysteryScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int, tx ...*sql.Tx) error {
	if err := r.dao.UpdateMysteryScoreAdjustment(ctx, userID, adjustment, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key())
}

func (r *userRepository) UpdateGMScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int, tx ...*sql.Tx) error {
	if err := r.dao.UpdateGMScoreAdjustment(ctx, userID, adjustment, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopGMs.Key())
}

func (r *userRepository) GetDetectiveRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetDetectiveRawScore(ctx, userID, tx...)
}

func (r *userRepository) GetGMRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.GetGMRawScore(ctx, userID, tx...)
}

func (r *userRepository) GetPasswordHash(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetPasswordHash(ctx, userID, tx...)
}

func (r *userRepository) SetPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string, tx ...*sql.Tx) error {
	return r.dao.SetPasswordHash(ctx, userID, passwordHash, tx...)
}

func (r *userRepository) SetEmail(ctx context.Context, userID uuid.UUID, email string, tx ...*sql.Tx) error {
	return r.dao.SetEmail(ctx, userID, email, tx...)
}

func (r *userRepository) SetDisplayName(ctx context.Context, userID uuid.UUID, displayName string, tx ...*sql.Tx) error {
	return r.dao.SetDisplayName(ctx, userID, displayName, tx...)
}

func (r *userRepository) SetDisplayNameLocked(ctx context.Context, userID uuid.UUID, locked bool, tx ...*sql.Tx) error {
	return r.dao.SetDisplayNameLocked(ctx, userID, locked, tx...)
}

func (r *userRepository) ListByIP(ctx context.Context, ip string, excludeUserID uuid.UUID, tx ...*sql.Tx) ([]model.User, error) {
	return r.dao.ListByIP(ctx, ip, excludeUserID, tx...)
}

func (r *userRepository) MarkEmailVerified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkEmailVerified(ctx, userID, tx...)
}

func (r *userRepository) MarkEmailUnverified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkEmailUnverified(ctx, userID, tx...)
}

func (r *userRepository) EmailInUse(ctx context.Context, email string, excludeUserID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.EmailInUse(ctx, email, excludeUserID, tx...)
}

func (r *userRepository) RequiresEmailVerification(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.RequiresEmailVerification(ctx, userID, tx...)
}

func (r *userRepository) DeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.DeleteAccount(ctx, userID, tx...); err != nil {
		return err
	}

	r.invalidateAfterUserDelete(ctx, userID)

	return nil
}

func (r *userRepository) invalidateAfterUserDelete(ctx context.Context, userID uuid.UUID) {
	keys := []string{
		cache.MysteryTopDetectives.Key(),
		cache.MysteryTopGMs.Key(),
		cache.VanityAssignments.Key(),
		cache.UserVanityRoleIDs.Key(userID.String()),
		cache.UserRole.Key(userID.String()),
	}

	for _, gameType := range leaderboardGameTypes {
		keys = append(keys, cache.GameTopWinners.Key(string(gameType)))
	}

	for _, spec := range secrets.All() {
		keys = append(keys, cache.SecretHolders.Key(string(spec.ID)), cache.SecretSolved.Key(string(spec.ID)))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to invalidate caches after deleting a user")
	}
}

func (r *userRepository) GetProfileByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, *model.UserStats, error) {
	return r.dao.GetProfileByUsername(ctx, username, tx...)
}

func (r *userRepository) GetProfileByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, *model.UserStats, error) {
	return r.dao.GetProfileByID(ctx, id, tx...)
}

func (r *userRepository) ListAll(ctx context.Context, search string, limit, offset int, tx ...*sql.Tx) ([]model.User, int, error) {
	return r.dao.ListAll(ctx, search, limit, offset, tx...)
}

func (r *userRepository) ListPublic(ctx context.Context, tx ...*sql.Tx) ([]model.User, error) {
	return r.dao.ListPublic(ctx, tx...)
}

func (r *userRepository) SearchByName(ctx context.Context, query string, limit int, tx ...*sql.Tx) ([]model.User, error) {
	return r.dao.SearchByName(ctx, query, limit, tx...)
}

func (r *userRepository) BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string, tx ...*sql.Tx) error {
	return r.dao.BanUser(ctx, userID, bannedBy, reason, tx...)
}

func (r *userRepository) UnbanUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnbanUser(ctx, userID, tx...)
}

func (r *userRepository) IsBanned(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsBanned(ctx, userID, tx...)
}

func (r *userRepository) LockUser(ctx context.Context, userID uuid.UUID, lockedBy uuid.UUID, reason string, tx ...*sql.Tx) error {
	return r.dao.LockUser(ctx, userID, lockedBy, reason, tx...)
}

func (r *userRepository) UnlockUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.UnlockUser(ctx, userID, tx...)
}

func (r *userRepository) IsLocked(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsLocked(ctx, userID, tx...)
}

func (r *userRepository) AdminDeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.AdminDeleteAccount(ctx, userID, tx...); err != nil {
		return err
	}

	r.invalidateAfterUserDelete(ctx, userID)

	return nil
}
