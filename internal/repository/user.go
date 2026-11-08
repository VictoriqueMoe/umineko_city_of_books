package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/secrets"

	"github.com/google/uuid"
)

type (
	UserRepository interface {
		dao.UserDAO

		RegisterAccount(ctx context.Context, registration spec.NewRegistration, tx ...*sql.Tx) (*model.User, error)
		ResetPassword(ctx context.Context, update spec.PasswordUpdate, tx ...*sql.Tx) error
		SetEmailVerified(ctx context.Context, s spec.UserEmailVerification, tx ...*sql.Tx) error
		ConfirmEmailVerification(ctx context.Context, s spec.UserEmailConfirmation, tx ...*sql.Tx) error
	}

	userRepository struct {
		dao.UserDAO

		db            *sql.DB
		cache         *cache.Manager
		roles         RoleRepository
		auditLogs     AuditLogRepository
		verifications EmailVerificationRepository
		invites       InviteRepository
		sessions      SessionRepository
		resets        PasswordResetRepository
		blocks        BlockRepository
		permissions   PermissionRepository
		mysteries     MysteryRepository
		posts         PostRepository
		art           ArtRepository
		gameRooms     GameRoomRepository
		userSecrets   UserSecretRepository
	}
)

func NewUserRepo(
	database *sql.DB,
	d dao.UserDAO,
	c *cache.Manager,
	roles RoleRepository,
	auditLogs AuditLogRepository,
	verifications EmailVerificationRepository,
	invites InviteRepository,
	sessions SessionRepository,
	resets PasswordResetRepository,
	blocks BlockRepository,
	permissions PermissionRepository,
	mysteries MysteryRepository,
	posts PostRepository,
	art ArtRepository,
	gameRooms GameRoomRepository,
	userSecrets UserSecretRepository,
) UserRepository {
	return &userRepository{
		UserDAO:       d,
		db:            database,
		cache:         c,
		roles:         roles,
		auditLogs:     auditLogs,
		verifications: verifications,
		invites:       invites,
		sessions:      sessions,
		resets:        resets,
		blocks:        blocks,
		permissions:   permissions,
		mysteries:     mysteries,
		posts:         posts,
		art:           art,
		gameRooms:     gameRooms,
		userSecrets:   userSecrets,
	}
}

func (r *userRepository) RegisterAccount(ctx context.Context, registration spec.NewRegistration, tx ...*sql.Tx) (*model.User, error) {
	var created *model.User

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.UserDAO.Create(ctx, registration.Account.User, tx)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		if registration.Account.Role != "" {
			assignment := spec.UserRoleSpec{
				UserID: created.ID,
				Role:   registration.Account.Role,
			}

			if err := r.roles.SetRole(ctx, assignment, tx); err != nil {
				return fmt.Errorf("assign initial role: %w", err)
			}
		}

		verification := spec.NewEmailVerification{
			TokenHash: registration.VerificationHash,
			UserID:    created.ID,
			ExpiresAt: registration.VerificationExpiresAt,
		}

		if err := r.verifications.Issue(ctx, verification, tx); err != nil {
			return fmt.Errorf("store verification token: %w", err)
		}

		entry := audit.NewEntry{
			ActorID:    created.ID,
			Action:     audit.ActionUserCreated,
			TargetType: audit.TargetUser,
			TargetID:   created.ID.String(),
			Details:    "username=" + registration.Account.User.Username,
			SubjectID:  created.ID,
		}

		if err := r.auditLogs.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("write user_created audit log: %w", err)
		}

		if registration.InviteCode != "" {
			if err := r.invites.MarkUsed(ctx, spec.InviteRedemption{Code: registration.InviteCode, UsedBy: created.ID}, tx); err != nil {
				if errors.Is(err, daoerr.ErrInviteUnavailable) {
					return err
				}

				return fmt.Errorf("mark invite as used: %w", err)
			}
		}

		if err := r.sessions.Create(ctx, spec.NewSession{Token: registration.SessionToken, UserID: created.ID, ExpiresAt: registration.SessionExpiresAt}, tx); err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if registration.Account.Role != "" {
		logger.Ctx(ctx).Info().Str("user_id", created.ID.String()).Str("username", registration.Account.User.Username).Msg("first user created, assigned super admin role")
	}

	return created, nil
}

func (r *userRepository) ResetPassword(ctx context.Context, update spec.PasswordUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		password := spec.UserPasswordHashUpdate{
			UserID:       update.UserID,
			PasswordHash: update.PasswordHash,
		}

		if err := r.UserDAO.SetPasswordHash(ctx, password, tx); err != nil {
			return fmt.Errorf("set password: %w", err)
		}

		if err := r.resets.MarkUsed(ctx, update.TokenHash, tx); err != nil {
			return fmt.Errorf("mark reset token used: %w", err)
		}

		if err := r.sessions.DeleteAllForUser(ctx, update.UserID, tx); err != nil {
			return fmt.Errorf("invalidate sessions after password reset: %w", err)
		}

		entry := audit.NewEntry{
			ActorID:    update.UserID,
			Action:     audit.ActionPasswordReset,
			TargetType: audit.TargetUser,
			TargetID:   update.UserID.String(),
			SubjectID:  update.UserID,
		}

		if err := r.auditLogs.Create(ctx, entry, tx); err != nil {
			return fmt.Errorf("write password_reset audit log: %w", err)
		}

		return nil
	})
}

func (r *userRepository) SetEmailVerified(ctx context.Context, s spec.UserEmailVerification, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if s.Verified {
			if err := r.UserDAO.MarkEmailVerified(ctx, s.UserID, tx); err != nil {
				return fmt.Errorf("mark email verified: %w", err)
			}
		} else {
			if err := r.UserDAO.MarkEmailUnverified(ctx, s.UserID, tx); err != nil {
				return fmt.Errorf("mark email unverified: %w", err)
			}
		}

		if err := r.verifications.DeleteUnusedForUser(ctx, s.UserID, tx); err != nil {
			return fmt.Errorf("clear verification tokens: %w", err)
		}

		return nil
	})
}

func (r *userRepository) ConfirmEmailVerification(ctx context.Context, s spec.UserEmailConfirmation, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.UserDAO.MarkEmailVerified(ctx, s.UserID, tx); err != nil {
			return fmt.Errorf("mark email verified: %w", err)
		}

		if err := r.verifications.MarkUsed(ctx, s.TokenHash, tx); err != nil {
			return fmt.Errorf("mark verification token used: %w", err)
		}

		return nil
	})
}

func (r *userRepository) GetIP(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	load := func(ctx context.Context) (string, error) {
		return r.UserDAO.GetIP(ctx, userID, tx...)
	}

	return r.cache.Load(ctx, cache.UserIP.Key(userID.String()), load)
}

func (r *userRepository) UpdateIP(ctx context.Context, s spec.UserIPUpdate, tx ...*sql.Tx) error {
	if err := r.UserDAO.UpdateIP(ctx, s, tx...); err != nil {
		return err
	}

	if err := r.cache.Del(ctx, cache.UserIP.Key(s.UserID.String())); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("user_id", s.UserID.String()).Msg("failed to invalidate the cached ip after updating it")
	}

	return nil
}

func (r *userRepository) UpdateMysteryScoreAdjustment(ctx context.Context, s spec.UserMysteryScoreUpdate, tx ...*sql.Tx) error {
	if err := r.UserDAO.UpdateMysteryScoreAdjustment(ctx, s, tx...); err != nil {
		return err
	}

	r.mysteries.InvalidateLeaderboards(ctx)

	return nil
}

func (r *userRepository) UpdateGMScoreAdjustment(ctx context.Context, s spec.UserGMScoreUpdate, tx ...*sql.Tx) error {
	if err := r.UserDAO.UpdateGMScoreAdjustment(ctx, s, tx...); err != nil {
		return err
	}

	r.mysteries.InvalidateLeaderboards(ctx)

	return nil
}

func (r *userRepository) DeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	blockCounterparts, err := r.blocks.GetBlockedIDs(ctx, userID, tx...)
	if err != nil {
		return err
	}

	if err := r.UserDAO.DeleteAccount(ctx, userID, tx...); err != nil {
		return err
	}

	r.invalidateAfterUserDelete(ctx, userID, blockCounterparts)

	return nil
}

func (r *userRepository) AdminDeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	blockCounterparts, err := r.blocks.GetBlockedIDs(ctx, userID, tx...)
	if err != nil {
		return err
	}

	if err := r.UserDAO.AdminDeleteAccount(ctx, userID, tx...); err != nil {
		return err
	}

	r.invalidateAfterUserDelete(ctx, userID, blockCounterparts)

	return nil
}

func (r *userRepository) invalidateAfterUserDelete(ctx context.Context, userID uuid.UUID, blockCounterparts []uuid.UUID) {
	r.blocks.InvalidateBlockedIDs(ctx, append([]uuid.UUID{userID}, blockCounterparts...)...)
	r.roles.InvalidateRole(ctx, userID)
	r.permissions.InvalidateVanityAssignments(ctx, userID.String())
	r.mysteries.InvalidateLeaderboards(ctx)
	r.posts.InvalidateCornerCounts(ctx)
	r.art.InvalidateCornerCounts(ctx)
	r.gameRooms.InvalidateTopWinners(ctx, leaderboardGameTypes...)

	secretIDs := make([]string, 0, len(secrets.All()))
	for _, secret := range secrets.All() {
		secretIDs = append(secretIDs, string(secret.ID))
	}
	r.userSecrets.InvalidateSecrets(ctx, secretIDs...)

	if err := r.cache.Del(ctx, cache.UserIP.Key(userID.String())); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("user_id", userID.String()).Msg("failed to invalidate the cached ip after deleting a user")
	}
}
