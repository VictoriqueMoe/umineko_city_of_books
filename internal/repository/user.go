package repository

import (
	"context"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	UserRepository interface {
		Create(ctx context.Context, username, email, password, displayName string) (*model.User, error)
		GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID) ([]model.User, error)
		GetByUsername(ctx context.Context, username string) (*model.User, error)
		GetByUsernames(ctx context.Context, usernames []string) ([]model.User, error)
		ExistsByUsername(ctx context.Context, username string) (bool, error)
		Count(ctx context.Context) (int, error)
		ValidatePassword(ctx context.Context, username, password string) (*model.User, error)
		UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error
		UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error
		UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string) error
		UpdateIP(ctx context.Context, userID uuid.UUID, ip string) error
		UpdateGameBoardSort(ctx context.Context, userID uuid.UUID, sort string) error
		UpdateAppearance(ctx context.Context, userID uuid.UUID, theme, font string, wideLayout bool) error
		UpdateMysteryScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error
		UpdateGMScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error
		GetDetectiveRawScore(ctx context.Context, userID uuid.UUID) (int, error)
		GetGMRawScore(ctx context.Context, userID uuid.UUID) (int, error)
		ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
		SetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error
		SetEmail(ctx context.Context, userID uuid.UUID, email string) error
		MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
		EmailInUse(ctx context.Context, email string, excludeUserID uuid.UUID) (bool, error)
		RequiresEmailVerification(ctx context.Context, userID uuid.UUID) (bool, error)
		DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error
		GetProfileByUsername(ctx context.Context, username string) (*model.User, *model.UserStats, error)
		GetProfileByID(ctx context.Context, id uuid.UUID) (*model.User, *model.UserStats, error)
		ListAll(ctx context.Context, search string, limit, offset int) ([]model.User, int, error)
		ListPublic(ctx context.Context) ([]model.User, error)
		SearchByName(ctx context.Context, query string, limit int) ([]model.User, error)
		BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string) error
		UnbanUser(ctx context.Context, userID uuid.UUID) error
		IsBanned(ctx context.Context, userID uuid.UUID) (bool, error)
		LockUser(ctx context.Context, userID uuid.UUID, lockedBy uuid.UUID, reason string) error
		UnlockUser(ctx context.Context, userID uuid.UUID) error
		IsLocked(ctx context.Context, userID uuid.UUID) (bool, error)
		AdminDeleteAccount(ctx context.Context, userID uuid.UUID) error
	}
)
