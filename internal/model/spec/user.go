package spec

import (
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
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

	UserProfileUpdate struct {
		UserID  uuid.UUID
		Profile dto.UpdateProfileRequest
	}

	UserAvatarUpdate struct {
		UserID    uuid.UUID
		AvatarURL string
	}

	UserBannerUpdate struct {
		UserID    uuid.UUID
		BannerURL string
	}

	UserIPUpdate struct {
		UserID uuid.UUID
		IP     string
	}

	UserGameBoardSortUpdate struct {
		UserID uuid.UUID
		Sort   string
	}

	UserAppearanceUpdate struct {
		UserID     uuid.UUID
		Theme      string
		Font       string
		WideLayout bool
	}

	UserMysteryScoreUpdate struct {
		UserID     uuid.UUID
		Adjustment int
	}

	UserGMScoreUpdate struct {
		UserID     uuid.UUID
		Adjustment int
	}

	UserPasswordHashUpdate struct {
		UserID       uuid.UUID
		PasswordHash string
	}

	UserEmailUpdate struct {
		UserID uuid.UUID
		Email  string
	}

	UserDisplayNameUpdate struct {
		UserID      uuid.UUID
		DisplayName string
	}

	UserDisplayNameLockUpdate struct {
		UserID uuid.UUID
		Locked bool
	}

	UserIPFilter struct {
		IP            string
		ExcludeUserID uuid.UUID
	}

	UserEmailFilter struct {
		Email         string
		ExcludeUserID uuid.UUID
	}

	UserListFilter struct {
		Search string
		Limit  int
		Offset int
	}

	UserSearchFilter struct {
		Query string
		Limit int
	}

	UserBan struct {
		UserID   uuid.UUID
		BannedBy uuid.UUID
		Reason   string
	}

	UserLock struct {
		UserID   uuid.UUID
		LockedBy uuid.UUID
		Reason   string
	}

	UserApproval struct {
		UserID     uuid.UUID
		ApprovedBy uuid.UUID
	}

	UserEmailVerification struct {
		UserID   uuid.UUID
		Verified bool
	}

	UserEmailConfirmation struct {
		UserID    uuid.UUID
		TokenHash string
	}
)
