package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	UserDAO interface {
		Create(ctx context.Context, s spec.NewUser, tx ...*sql.Tx) (*model.User, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID, tx ...*sql.Tx) ([]model.User, error)
		GetByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, error)
		GetByUsernames(ctx context.Context, usernames []string, tx ...*sql.Tx) ([]model.User, error)
		ExistsByUsername(ctx context.Context, username string, tx ...*sql.Tx) (bool, error)
		Count(ctx context.Context, tx ...*sql.Tx) (int, error)
		UpdateProfile(ctx context.Context, s spec.UserProfileUpdate, tx ...*sql.Tx) error
		UpdateAvatarURL(ctx context.Context, s spec.UserAvatarUpdate, tx ...*sql.Tx) error
		UpdateBannerURL(ctx context.Context, s spec.UserBannerUpdate, tx ...*sql.Tx) error
		UpdateIP(ctx context.Context, s spec.UserIPUpdate, tx ...*sql.Tx) error
		UpdateGameBoardSort(ctx context.Context, s spec.UserGameBoardSortUpdate, tx ...*sql.Tx) error
		UpdateAppearance(ctx context.Context, s spec.UserAppearanceUpdate, tx ...*sql.Tx) error
		UpdateMysteryScoreAdjustment(ctx context.Context, s spec.UserMysteryScoreUpdate, tx ...*sql.Tx) error
		UpdateGMScoreAdjustment(ctx context.Context, s spec.UserGMScoreUpdate, tx ...*sql.Tx) error
		GetDetectiveRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetGMRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetPasswordHash(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		SetPasswordHash(ctx context.Context, s spec.UserPasswordHashUpdate, tx ...*sql.Tx) error
		SetEmail(ctx context.Context, s spec.UserEmailUpdate, tx ...*sql.Tx) error
		SetDisplayName(ctx context.Context, s spec.UserDisplayNameUpdate, tx ...*sql.Tx) error
		SetDisplayNameLocked(ctx context.Context, s spec.UserDisplayNameLockUpdate, tx ...*sql.Tx) error
		ListByIP(ctx context.Context, s spec.UserIPFilter, tx ...*sql.Tx) ([]model.User, error)
		MarkEmailVerified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		MarkEmailUnverified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		EmailInUse(ctx context.Context, s spec.UserEmailFilter, tx ...*sql.Tx) (bool, error)
		RequiresEmailVerification(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		DeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		GetProfileByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, *model.UserStats, error)
		GetProfileByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, *model.UserStats, error)
		ListAll(ctx context.Context, s spec.UserListFilter, tx ...*sql.Tx) ([]model.User, int, error)
		ListPublic(ctx context.Context, tx ...*sql.Tx) ([]model.User, error)
		SearchByName(ctx context.Context, s spec.UserSearchFilter, tx ...*sql.Tx) ([]model.User, error)
		BanUser(ctx context.Context, s spec.UserBan, tx ...*sql.Tx) error
		UnbanUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		IsBanned(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		LockUser(ctx context.Context, s spec.UserLock, tx ...*sql.Tx) error
		UnlockUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		ApproveUser(ctx context.Context, s spec.UserApproval, tx ...*sql.Tx) error
		UnapproveUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		IsLocked(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error)
		AdminDeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	userDAO struct {
		db *sql.DB
	}

	userJoinRow = sqlcgen.GetUserByIDRow
)

func toUserModel(row userJoinRow) model.User {
	u := model.User{
		ID:                     row.ID,
		Username:               row.Username,
		PasswordHash:           row.PasswordHash,
		DisplayName:            row.DisplayName,
		DisplayNameLocked:      row.DisplayNameLocked,
		CreatedAt:              row.CreatedAt.UTC().Format(time.RFC3339),
		Bio:                    row.Bio,
		AvatarURL:              row.AvatarUrl,
		BannerURL:              row.BannerUrl,
		FavouriteCharacter:     row.FavouriteCharacter,
		Gender:                 row.Gender,
		PronounSubject:         row.PronounSubject,
		PronounPossessive:      row.PronounPossessive,
		BannedAt:               nullTimeToStringPtr(row.BannedAt),
		BannedBy:               row.BannedBy,
		BanReason:              row.BanReason,
		LockedAt:               nullTimeToStringPtr(row.LockedAt),
		LockedBy:               row.LockedBy,
		LockReason:             row.LockReason,
		ApprovedAt:             nullTimeToStringPtr(row.ApprovedAt),
		ApprovedBy:             row.ApprovedBy,
		SocialTwitter:          row.SocialTwitter,
		SocialDiscord:          row.SocialDiscord,
		SocialWaifulist:        row.SocialWaifulist,
		SocialTumblr:           row.SocialTumblr,
		SocialGithub:           row.SocialGithub,
		SocialBluesky:          row.SocialBluesky,
		Website:                row.Website,
		BannerPosition:         float64(row.BannerPosition),
		DmsEnabled:             row.DmsEnabled,
		EpisodeProgress:        int(row.EpisodeProgress),
		HigurashiArcProgress:   int(row.HigurashiArcProgress),
		CiconiaChapterProgress: int(row.CiconiaChapterProgress),
		Email:                  row.Email,
		EmailPublic:            row.EmailPublic,
		EmailVerified:          row.EmailVerified,
		VerifyGraceUntil:       row.VerifyGraceUntil.UTC().Format(time.RFC3339),
		DOB:                    row.Dob,
		DOBPublic:              row.DobPublic,
		EmailNotifications:     row.EmailNotifications,
		PlayMessageSound:       row.PlayMessageSound,
		PlayNotificationSound:  row.PlayNotificationSound,
		HomePage:               row.HomePage,
		GameBoardSort:          row.GameBoardSort,
		DefaultProfileTab:      row.DefaultProfileTab,
		Theme:                  row.Theme,
		Font:                   row.Font,
		WideLayout:             row.WideLayout,
		MysteryScoreAdjustment: int(row.MysteryScoreAdjustment),
		GMScoreAdjustment:      int(row.GmScoreAdjustment),
		Role:                   row.Role,
		IsBot:                  row.IsBot,
		FollowActivity:         row.FollowActivityNotifications,
		EchoesEnabled:          row.EchoesEnabled,
	}

	if row.Ip.Valid {
		u.IP = new(row.Ip.String)
	}

	return u
}

func (r *userDAO) Create(ctx context.Context, s spec.NewUser, tx ...*sql.Tx) (*model.User, error) {
	created, err := genQueries(r.db, tx).CreateUser(ctx, sqlcgen.CreateUserParams{
		Username:      s.Username,
		Email:         s.Email,
		PasswordHash:  s.PasswordHash,
		DisplayName:   s.DisplayName,
		AvatarUrl:     s.AvatarURL,
		HomePage:      s.HomePage,
		IsBot:         s.IsBot,
		DmsEnabled:    s.DMsEnabled,
		EmailVerified: s.EmailVerified,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return new(toUserModel(userJoinRow(created))), nil
}

func (r *userDAO) SetEmail(ctx context.Context, s spec.UserEmailUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetUserEmail(ctx, sqlcgen.SetUserEmailParams{
		Email: s.Email,
		ID:    s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set email: %w", err)
	}

	return nil
}

func (r *userDAO) SetDisplayName(ctx context.Context, s spec.UserDisplayNameUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetUserDisplayName(ctx, sqlcgen.SetUserDisplayNameParams{
		DisplayName: s.DisplayName,
		ID:          s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set display name: %w", err)
	}

	return nil
}

func (r *userDAO) SetDisplayNameLocked(ctx context.Context, s spec.UserDisplayNameLockUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetUserDisplayNameLocked(ctx, sqlcgen.SetUserDisplayNameLockedParams{
		DisplayNameLocked: s.Locked,
		ID:                s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set display name locked: %w", err)
	}

	return nil
}

func (r *userDAO) ListByIP(ctx context.Context, s spec.UserIPFilter, tx ...*sql.Tx) ([]model.User, error) {
	rows, err := genQueries(r.db, tx).ListUsersByIP(ctx, sqlcgen.ListUsersByIPParams{
		Column1: s.IP,
		ID:      s.ExcludeUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("list users by ip: %w", err)
	}

	var users []model.User
	for _, row := range rows {
		users = append(users, toUserModel(userJoinRow(row)))
	}

	return users, nil
}

func (r *userDAO) MarkEmailVerified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkUserEmailVerified(ctx, userID); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}

	return nil
}

func (r *userDAO) MarkEmailUnverified(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkUserEmailUnverified(ctx, userID); err != nil {
		return fmt.Errorf("mark email unverified: %w", err)
	}

	return nil
}

func (r *userDAO) EmailInUse(ctx context.Context, s spec.UserEmailFilter, tx ...*sql.Tx) (bool, error) {
	exists, err := genQueries(r.db, tx).UserEmailInUse(ctx, sqlcgen.UserEmailInUseParams{
		Lower: s.Email,
		ID:    s.ExcludeUserID,
	})
	if err != nil {
		return false, fmt.Errorf("check email in use: %w", err)
	}

	return exists, nil
}

func (r *userDAO) RequiresEmailVerification(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	blocked, err := genQueries(r.db, tx).UserRequiresEmailVerification(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check email verification: %w", err)
	}

	return blocked, nil
}

func (r *userDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, error) {
	row, err := genQueries(r.db, tx).GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return new(toUserModel(row)), nil
}

func (r *userDAO) GetByIDs(ctx context.Context, ids []uuid.UUID, tx ...*sql.Tx) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetUsersByIDs(ctx, joinUUIDs(ids))
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}

	var users []model.User
	for _, row := range rows {
		users = append(users, toUserModel(userJoinRow(row)))
	}

	return users, nil
}

func (r *userDAO) GetByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, error) {
	row, err := genQueries(r.db, tx).GetUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return new(toUserModel(userJoinRow(row))), nil
}

func (r *userDAO) GetByUsernames(ctx context.Context, usernames []string, tx ...*sql.Tx) ([]model.User, error) {
	if len(usernames) == 0 {
		return nil, nil
	}

	lowered := make([]string, 0, len(usernames))
	for _, username := range usernames {
		lowered = append(lowered, strings.ToLower(username))
	}

	rows, err := genQueries(r.db, tx).GetUsersByUsernames(ctx, strings.Join(lowered, ","))
	if err != nil {
		return nil, fmt.Errorf("get users by usernames: %w", err)
	}

	var users []model.User
	for _, row := range rows {
		users = append(users, toUserModel(userJoinRow(row)))
	}

	return users, nil
}

func (r *userDAO) ExistsByUsername(ctx context.Context, username string, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountUsersByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}

	return count > 0, nil
}

func (r *userDAO) Count(ctx context.Context, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return int(count), nil
}

func (r *userDAO) GetPasswordHash(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	hash, err := genQueries(r.db, tx).GetUserPasswordHash(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get password hash: %w", err)
	}

	return hash, nil
}

func (r *userDAO) UpdateProfile(ctx context.Context, s spec.UserProfileUpdate, tx ...*sql.Tx) error {
	req := s.Profile

	err := genQueries(r.db, tx).UpdateUserProfile(ctx, sqlcgen.UpdateUserProfileParams{
		DisplayName:                 req.DisplayName,
		Bio:                         req.Bio,
		BannerPosition:              float32(req.BannerPosition),
		FavouriteCharacter:          req.FavouriteCharacter,
		Gender:                      req.Gender,
		PronounSubject:              req.PronounSubject,
		PronounPossessive:           req.PronounPossessive,
		SocialTwitter:               req.SocialTwitter,
		SocialDiscord:               req.SocialDiscord,
		SocialWaifulist:             req.SocialWaifulist,
		SocialTumblr:                req.SocialTumblr,
		SocialGithub:                req.SocialGithub,
		SocialBluesky:               req.SocialBluesky,
		Website:                     req.Website,
		DmsEnabled:                  req.DmsEnabled,
		EpisodeProgress:             int32(req.EpisodeProgress),
		HigurashiArcProgress:        int32(req.HigurashiArcProgress),
		CiconiaChapterProgress:      int32(req.CiconiaChapterProgress),
		Email:                       req.Email,
		EmailPublic:                 req.EmailPublic,
		Dob:                         req.DOB,
		DobPublic:                   req.DOBPublic,
		EmailNotifications:          req.EmailNotifications,
		PlayMessageSound:            req.PlayMessageSound,
		PlayNotificationSound:       req.PlayNotificationSound,
		HomePage:                    req.HomePage,
		GameBoardSort:               req.GameBoardSort,
		DefaultProfileTab:           req.DefaultProfileTab,
		FollowActivityNotifications: req.FollowActivity,
		EchoesEnabled:               req.EchoesEnabled,
		ID:                          s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}

	return nil
}

func (r *userDAO) UpdateAvatarURL(ctx context.Context, s spec.UserAvatarUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserAvatarURL(ctx, sqlcgen.UpdateUserAvatarURLParams{
		AvatarUrl: s.AvatarURL,
		ID:        s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update avatar url: %w", err)
	}

	return nil
}

func (r *userDAO) UpdateBannerURL(ctx context.Context, s spec.UserBannerUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserBannerURL(ctx, sqlcgen.UpdateUserBannerURLParams{
		BannerUrl: s.BannerURL,
		ID:        s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update banner url: %w", err)
	}

	return nil
}

func (r *userDAO) UpdateIP(ctx context.Context, s spec.UserIPUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserIP(ctx, sqlcgen.UpdateUserIPParams{
		Column1: s.IP,
		ID:      s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update ip: %w", err)
	}

	return nil
}

func (r *userDAO) UpdateGameBoardSort(ctx context.Context, s spec.UserGameBoardSortUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserGameBoardSort(ctx, sqlcgen.UpdateUserGameBoardSortParams{
		GameBoardSort: s.Sort,
		ID:            s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update game board sort: %w", err)
	}

	return nil
}

func (r *userDAO) UpdateAppearance(ctx context.Context, s spec.UserAppearanceUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserAppearance(ctx, sqlcgen.UpdateUserAppearanceParams{
		Theme:      s.Theme,
		Font:       s.Font,
		WideLayout: s.WideLayout,
		ID:         s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update appearance: %w", err)
	}

	return nil
}

func (r *userDAO) UpdateMysteryScoreAdjustment(ctx context.Context, s spec.UserMysteryScoreUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserMysteryScoreAdjustment(ctx, sqlcgen.UpdateUserMysteryScoreAdjustmentParams{
		MysteryScoreAdjustment: int32(s.Adjustment),
		ID:                     s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update mystery score adjustment: %w", err)
	}

	return nil
}

func (r *userDAO) GetDetectiveRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	score, err := genQueries(r.db, tx).GetUserDetectiveRawScore(ctx, &userID)

	return int(score), err
}

func (r *userDAO) GetGMRawScore(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	score, err := genQueries(r.db, tx).GetUserGMRawScore(ctx, userID)

	return int(score), err
}

func (r *userDAO) UpdateGMScoreAdjustment(ctx context.Context, s spec.UserGMScoreUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateUserGMScoreAdjustment(ctx, sqlcgen.UpdateUserGMScoreAdjustmentParams{
		GmScoreAdjustment: int32(s.Adjustment),
		ID:                s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update gm score adjustment: %w", err)
	}

	return nil
}

func (r *userDAO) SetPasswordHash(ctx context.Context, s spec.UserPasswordHashUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetUserPasswordHash(ctx, sqlcgen.SetUserPasswordHashParams{
		PasswordHash: s.PasswordHash,
		ID:           s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

func (r *userDAO) DeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	return nil
}

func (r *userDAO) GetProfileByUsername(ctx context.Context, username string, tx ...*sql.Tx) (*model.User, *model.UserStats, error) {
	u, err := r.GetByUsername(ctx, username, tx...)
	if err != nil || u == nil {
		return u, nil, err
	}

	queries := genQueries(r.db, tx)

	theoryCount, err := queries.CountProfileTheories(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile theories: %w", err)
	}

	responseCount, err := queries.CountProfileResponses(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile responses: %w", err)
	}

	theoryVotes, err := queries.SumProfileTheoryVotes(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("sum profile theory votes: %w", err)
	}

	responseVotes, err := queries.SumProfileResponseVotes(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("sum profile response votes: %w", err)
	}

	shipCount, err := queries.CountProfileShips(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile ships: %w", err)
	}

	mysteryCount, err := queries.CountProfileMysteries(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile mysteries: %w", err)
	}

	fanficCount, err := queries.CountProfileFanfics(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile fanfics: %w", err)
	}

	stats := model.UserStats{
		TheoryCount:   int(theoryCount),
		ResponseCount: int(responseCount),
		VotesReceived: int(theoryVotes + responseVotes),
		ShipCount:     int(shipCount),
		MysteryCount:  int(mysteryCount),
		FanficCount:   int(fanficCount),
	}

	return u, &stats, nil
}

func (r *userDAO) GetProfileByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.User, *model.UserStats, error) {
	u, err := r.GetByID(ctx, id, tx...)
	if err != nil || u == nil {
		return u, nil, err
	}

	queries := genQueries(r.db, tx)

	theoryCount, err := queries.CountProfileTheories(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile theories: %w", err)
	}

	responseCount, err := queries.CountProfileResponses(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile responses: %w", err)
	}

	shipCount, err := queries.CountProfileShips(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile ships: %w", err)
	}

	mysteryCount, err := queries.CountProfileMysteries(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile mysteries: %w", err)
	}

	fanficCount, err := queries.CountProfileFanfics(ctx, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("count profile fanfics: %w", err)
	}

	stats := model.UserStats{
		TheoryCount:   int(theoryCount),
		ResponseCount: int(responseCount),
		ShipCount:     int(shipCount),
		MysteryCount:  int(mysteryCount),
		FanficCount:   int(fanficCount),
	}

	return u, &stats, nil
}

func (r *userDAO) ListAll(ctx context.Context, s spec.UserListFilter, tx ...*sql.Tx) ([]model.User, int, error) {
	pattern := ""
	if s.Search != "" {
		pattern = "%" + s.Search + "%"
	}

	queries := genQueries(r.db, tx)

	total, err := queries.CountUsersFiltered(ctx, sqlcgen.CountUsersFilteredParams{
		Column1: s.Search,
		Column2: pattern,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := queries.ListUsers(ctx, sqlcgen.ListUsersParams{
		Column1: s.Search,
		Column2: pattern,
		Limit:   int32(s.Limit),
		Offset:  int32(s.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	var users []model.User
	for _, row := range rows {
		users = append(users, toUserModel(userJoinRow(row)))
	}

	return users, int(total), nil
}

func (r *userDAO) ListPublic(ctx context.Context, tx ...*sql.Tx) ([]model.User, error) {
	rows, err := genQueries(r.db, tx).ListPublicUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list public users: %w", err)
	}

	var users []model.User
	for _, row := range rows {
		users = append(users, toUserModel(userJoinRow(row)))
	}

	return users, nil
}

func (r *userDAO) SearchByName(ctx context.Context, s spec.UserSearchFilter, tx ...*sql.Tx) ([]model.User, error) {
	rows, err := genQueries(r.db, tx).SearchUsersByName(ctx, sqlcgen.SearchUsersByNameParams{
		Column1: "%" + s.Query + "%",
		Column2: s.Query + "%",
		Limit:   int32(s.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	var users []model.User
	for _, row := range rows {
		users = append(users, toUserModel(userJoinRow(row)))
	}

	return users, nil
}

func (r *userDAO) BanUser(ctx context.Context, s spec.UserBan, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).BanUser(ctx, sqlcgen.BanUserParams{
		BannedBy:  &s.BannedBy,
		BanReason: s.Reason,
		ID:        s.UserID,
	})
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}

	return nil
}

func (r *userDAO) UnbanUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UnbanUser(ctx, userID); err != nil {
		return fmt.Errorf("unban user: %w", err)
	}

	return nil
}

func (r *userDAO) IsBanned(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	bannedAt, err := genQueries(r.db, tx).GetUserBannedAt(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check ban: %w", err)
	}

	return bannedAt.Valid, nil
}

func (r *userDAO) LockUser(ctx context.Context, s spec.UserLock, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).LockUser(ctx, sqlcgen.LockUserParams{
		LockedBy:   &s.LockedBy,
		LockReason: s.Reason,
		ID:         s.UserID,
	})
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}

	return nil
}

func (r *userDAO) UnlockUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UnlockUser(ctx, userID); err != nil {
		return fmt.Errorf("unlock user: %w", err)
	}

	return nil
}

func (r *userDAO) ApproveUser(ctx context.Context, s spec.UserApproval, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).ApproveUser(ctx, sqlcgen.ApproveUserParams{
		ApprovedBy: &s.ApprovedBy,
		ID:         s.UserID,
	})
	if err != nil {
		return fmt.Errorf("approve user: %w", err)
	}

	return nil
}

func (r *userDAO) UnapproveUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UnapproveUser(ctx, userID); err != nil {
		return fmt.Errorf("unapprove user: %w", err)
	}

	return nil
}

func (r *userDAO) IsLocked(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	lockedAt, err := genQueries(r.db, tx).GetUserLockedAt(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check lock: %w", err)
	}

	return lockedAt.Valid, nil
}

func (r *userDAO) AdminDeleteAccount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("admin delete account: %w", err)
	}

	return nil
}
