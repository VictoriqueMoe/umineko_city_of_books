package repository

import (
	"context"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/cache"
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

type userRepository struct {
	dao   UserRepository
	cache *cache.Manager
}

func NewUserRepo(dao UserRepository, c *cache.Manager) UserRepository {
	return &userRepository{dao: dao, cache: c}
}

func (r *userRepository) Create(ctx context.Context, username, email, password, displayName string) (*model.User, error) {
	return r.dao.Create(ctx, username, email, password, displayName)
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *userRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]model.User, error) {
	return r.dao.GetByIDs(ctx, ids)
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.dao.GetByUsername(ctx, username)
}

func (r *userRepository) GetByUsernames(ctx context.Context, usernames []string) ([]model.User, error) {
	return r.dao.GetByUsernames(ctx, usernames)
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return r.dao.ExistsByUsername(ctx, username)
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	return r.dao.Count(ctx)
}

func (r *userRepository) ValidatePassword(ctx context.Context, username, password string) (*model.User, error) {
	return r.dao.ValidatePassword(ctx, username, password)
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error {
	return r.dao.UpdateProfile(ctx, userID, req)
}

func (r *userRepository) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	return r.dao.UpdateAvatarURL(ctx, userID, avatarURL)
}

func (r *userRepository) UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string) error {
	return r.dao.UpdateBannerURL(ctx, userID, bannerURL)
}

func (r *userRepository) UpdateIP(ctx context.Context, userID uuid.UUID, ip string) error {
	return r.dao.UpdateIP(ctx, userID, ip)
}

func (r *userRepository) UpdateGameBoardSort(ctx context.Context, userID uuid.UUID, sort string) error {
	return r.dao.UpdateGameBoardSort(ctx, userID, sort)
}

func (r *userRepository) UpdateAppearance(ctx context.Context, userID uuid.UUID, theme, font string, wideLayout bool) error {
	return r.dao.UpdateAppearance(ctx, userID, theme, font, wideLayout)
}

func (r *userRepository) UpdateMysteryScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error {
	if err := r.dao.UpdateMysteryScoreAdjustment(ctx, userID, adjustment); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopDetectives.Key())
}

func (r *userRepository) UpdateGMScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error {
	if err := r.dao.UpdateGMScoreAdjustment(ctx, userID, adjustment); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.MysteryTopGMs.Key())
}

func (r *userRepository) GetDetectiveRawScore(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.GetDetectiveRawScore(ctx, userID)
}

func (r *userRepository) GetGMRawScore(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.GetGMRawScore(ctx, userID)
}

func (r *userRepository) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	return r.dao.ChangePassword(ctx, userID, oldPassword, newPassword)
}

func (r *userRepository) SetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	return r.dao.SetPassword(ctx, userID, newPassword)
}

func (r *userRepository) SetEmail(ctx context.Context, userID uuid.UUID, email string) error {
	return r.dao.SetEmail(ctx, userID, email)
}

func (r *userRepository) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return r.dao.MarkEmailVerified(ctx, userID)
}

func (r *userRepository) EmailInUse(ctx context.Context, email string, excludeUserID uuid.UUID) (bool, error) {
	return r.dao.EmailInUse(ctx, email, excludeUserID)
}

func (r *userRepository) RequiresEmailVerification(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.dao.RequiresEmailVerification(ctx, userID)
}

func (r *userRepository) DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error {
	return r.dao.DeleteAccount(ctx, userID, password)
}

func (r *userRepository) GetProfileByUsername(ctx context.Context, username string) (*model.User, *model.UserStats, error) {
	return r.dao.GetProfileByUsername(ctx, username)
}

func (r *userRepository) GetProfileByID(ctx context.Context, id uuid.UUID) (*model.User, *model.UserStats, error) {
	return r.dao.GetProfileByID(ctx, id)
}

func (r *userRepository) ListAll(ctx context.Context, search string, limit, offset int) ([]model.User, int, error) {
	return r.dao.ListAll(ctx, search, limit, offset)
}

func (r *userRepository) ListPublic(ctx context.Context) ([]model.User, error) {
	return r.dao.ListPublic(ctx)
}

func (r *userRepository) SearchByName(ctx context.Context, query string, limit int) ([]model.User, error) {
	return r.dao.SearchByName(ctx, query, limit)
}

func (r *userRepository) BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string) error {
	return r.dao.BanUser(ctx, userID, bannedBy, reason)
}

func (r *userRepository) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	return r.dao.UnbanUser(ctx, userID)
}

func (r *userRepository) IsBanned(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.dao.IsBanned(ctx, userID)
}

func (r *userRepository) LockUser(ctx context.Context, userID uuid.UUID, lockedBy uuid.UUID, reason string) error {
	return r.dao.LockUser(ctx, userID, lockedBy, reason)
}

func (r *userRepository) UnlockUser(ctx context.Context, userID uuid.UUID) error {
	return r.dao.UnlockUser(ctx, userID)
}

func (r *userRepository) IsLocked(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.dao.IsLocked(ctx, userID)
}

func (r *userRepository) AdminDeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return r.dao.AdminDeleteAccount(ctx, userID)
}
