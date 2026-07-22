package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	userDAO struct {
		db *sql.DB
	}
)

const (
	userColumns = `u.id, u.username, u.password_hash, u.display_name, u.created_at, u.bio, u.avatar_url, u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive, u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress, u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob, u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page, u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip, u.mystery_score_adjustment, u.gm_score_adjustment, COALESCE(r.role, '')`
)

func scanUser(row interface{ Scan(dest ...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
		&u.Bio, &u.AvatarURL, &u.BannerURL, &u.FavouriteCharacter, &u.Gender,
		&u.PronounSubject, &u.PronounPossessive,
		&u.BannedAt, &u.BannedBy, &u.BanReason,
		&u.LockedAt, &u.LockedBy, &u.LockReason,
		&u.SocialTwitter, &u.SocialDiscord, &u.SocialWaifulist, &u.SocialTumblr, &u.SocialGithub, &u.Website,
		&u.BannerPosition, &u.DmsEnabled, &u.EpisodeProgress, &u.HigurashiArcProgress, &u.CiconiaChapterProgress, &u.Email, &u.EmailPublic, &u.EmailVerified, &u.VerifyGraceUntil, &u.DOB, &u.DOBPublic, &u.EmailNotifications, &u.PlayMessageSound, &u.PlayNotificationSound, &u.HomePage, &u.GameBoardSort, &u.DefaultProfileTab, &u.Theme, &u.Font, &u.WideLayout, &u.IP, &u.MysteryScoreAdjustment, &u.GMScoreAdjustment, &u.Role)
	return &u, err
}

func (r *userDAO) Create(ctx context.Context, username, email, password, displayName string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id := uuid.New()

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password_hash, display_name, home_page) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, username, email, string(hash), displayName, "landing",
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &model.User{
		ID:          id,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	}, nil
}

func (r *userDAO) SetEmail(ctx context.Context, userID uuid.UUID, email string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET email = $1, email_verified = FALSE WHERE id = $2`, email, userID,
	)
	if err != nil {
		return fmt.Errorf("set email: %w", err)
	}
	return nil
}

func (r *userDAO) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET email_verified = TRUE WHERE id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return nil
}

func (r *userDAO) EmailInUse(ctx context.Context, email string, excludeUserID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1) AND email <> '' AND id <> $2)`,
		email, excludeUserID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check email in use: %w", err)
	}
	return exists, nil
}

func (r *userDAO) RequiresEmailVerification(ctx context.Context, userID uuid.UUID) (bool, error) {
	var blocked bool
	err := r.db.QueryRowContext(ctx,
		`SELECT NOT email_verified AND NOW() >= verify_grace_until FROM users WHERE id = $1`, userID,
	).Scan(&blocked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check email verification: %w", err)
	}
	return blocked, nil
}

func (r *userDAO) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.id = $1`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *userDAO) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, len(ids))
	placeholders := make([]string, len(ids))
	for i := range ids {
		args[i] = ids[i]
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *userDAO) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE LOWER(u.username) = LOWER($1)`, username,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

func (r *userDAO) GetByUsernames(ctx context.Context, usernames []string) ([]model.User, error) {
	if len(usernames) == 0 {
		return nil, nil
	}
	args := make([]any, len(usernames))
	placeholders := make([]string, len(usernames))
	for i := range usernames {
		args[i] = strings.ToLower(usernames[i])
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE LOWER(u.username) IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get users by usernames: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *userDAO) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE LOWER(username) = LOWER($1)`, username,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}
	return count > 0, nil
}

func (r *userDAO) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *userDAO) ValidatePassword(ctx context.Context, username, password string) (*model.User, error) {
	u, err := r.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil
	}

	return u, nil
}

func (r *userDAO) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = $1, bio = $2, banner_position = $3, favourite_character = $4, gender = $5,
		 pronoun_subject = $6, pronoun_possessive = $7,
		 social_twitter = $8, social_discord = $9, social_waifulist = $10, social_tumblr = $11, social_github = $12,
		 website = $13, dms_enabled = $14, episode_progress = $15, higurashi_arc_progress = $16, ciconia_chapter_progress = $17, email = $18, email_public = $19, dob = $20, dob_public = $21, email_notifications = $22, play_message_sound = $23, play_notification_sound = $24, home_page = $25, game_board_sort = $26, default_profile_tab = $27
		 WHERE id = $28`,
		req.DisplayName, req.Bio, req.BannerPosition, req.FavouriteCharacter, req.Gender,
		req.PronounSubject, req.PronounPossessive,
		req.SocialTwitter, req.SocialDiscord, req.SocialWaifulist, req.SocialTumblr, req.SocialGithub, req.Website,
		req.DmsEnabled, req.EpisodeProgress, req.HigurashiArcProgress, req.CiconiaChapterProgress, req.Email, req.EmailPublic, req.DOB, req.DOBPublic, req.EmailNotifications, req.PlayMessageSound, req.PlayNotificationSound, req.HomePage, req.GameBoardSort, req.DefaultProfileTab,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (r *userDAO) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET avatar_url = $1 WHERE id = $2`, avatarURL, userID,
	)
	if err != nil {
		return fmt.Errorf("update avatar url: %w", err)
	}
	return nil
}

func (r *userDAO) UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banner_url = $1 WHERE id = $2`, bannerURL, userID,
	)
	if err != nil {
		return fmt.Errorf("update banner url: %w", err)
	}
	return nil
}

func (r *userDAO) UpdateIP(ctx context.Context, userID uuid.UUID, ip string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET ip = $1 WHERE id = $2`, ip, userID,
	)
	if err != nil {
		return fmt.Errorf("update ip: %w", err)
	}
	return nil
}

func (r *userDAO) UpdateGameBoardSort(ctx context.Context, userID uuid.UUID, sort string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET game_board_sort = $1 WHERE id = $2`, sort, userID,
	)
	if err != nil {
		return fmt.Errorf("update game board sort: %w", err)
	}
	return nil
}

func (r *userDAO) UpdateAppearance(ctx context.Context, userID uuid.UUID, theme, font string, wideLayout bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET theme = $1, font = $2, wide_layout = $3 WHERE id = $4`, theme, font, wideLayout, userID,
	)
	if err != nil {
		return fmt.Errorf("update appearance: %w", err)
	}
	return nil
}

func (r *userDAO) UpdateMysteryScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mystery_score_adjustment = $1 WHERE id = $2`, adjustment, userID,
	)
	if err != nil {
		return fmt.Errorf("update mystery score adjustment: %w", err)
	}
	return nil
}

func (r *userDAO) GetDetectiveRawScore(ctx context.Context, userID uuid.UUID) (int, error) {
	var score int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(
			CASE m.difficulty
				WHEN 'easy' THEN 2
				WHEN 'medium' THEN 4
				WHEN 'hard' THEN 6
				WHEN 'nightmare' THEN 8
				ELSE 4
			END
		), 0)
		FROM mysteries m WHERE m.winner_id = $1 AND m.solved = TRUE`, userID,
	).Scan(&score)
	return score, err
}

func (r *userDAO) GetGMRawScore(ctx context.Context, userID uuid.UUID) (int, error) {
	var score int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(
			CASE m.difficulty
				WHEN 'easy' THEN 2
				WHEN 'medium' THEN 4
				WHEN 'hard' THEN 6
				WHEN 'nightmare' THEN 8
				ELSE 4
			END
			+ LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
		), 0)
		FROM mysteries m WHERE m.user_id = $1 AND m.solved = TRUE`, userID,
	).Scan(&score)
	return score, err
}

func (r *userDAO) UpdateGMScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET gm_score_adjustment = $1 WHERE id = $2`, adjustment, userID,
	)
	if err != nil {
		return fmt.Errorf("update gm score adjustment: %w", err)
	}
	return nil
}

func (r *userDAO) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	u, err := r.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("incorrect password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), userID,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *userDAO) SetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), userID,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *userDAO) DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error {
	u, err := r.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return fmt.Errorf("incorrect password")
	}

	_, err = r.db.ExecContext(ctx,
		`DELETE FROM users WHERE id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func (r *userDAO) GetProfileByUsername(ctx context.Context, username string) (*model.User, *model.UserStats, error) {
	u, err := r.GetByUsername(ctx, username)
	if err != nil || u == nil {
		return u, nil, err
	}

	var stats model.UserStats
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = $1`, u.ID,
	).Scan(&stats.TheoryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = $1`, u.ID,
	).Scan(&stats.ResponseCount)

	var theoryVotes, responseVotes int
	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tv.value), 0) FROM theory_votes tv JOIN theories t ON tv.theory_id = t.id WHERE t.user_id = $1`, u.ID,
	).Scan(&theoryVotes)

	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(rv.value), 0) FROM response_votes rv JOIN responses r ON rv.response_id = r.id WHERE r.user_id = $1`, u.ID,
	).Scan(&responseVotes)

	stats.VotesReceived = theoryVotes + responseVotes

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ships WHERE user_id = $1`, u.ID,
	).Scan(&stats.ShipCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM mysteries WHERE user_id = $1`, u.ID,
	).Scan(&stats.MysteryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fanfics WHERE user_id = $1`, u.ID,
	).Scan(&stats.FanficCount)

	return u, &stats, nil
}

func (r *userDAO) GetProfileByID(ctx context.Context, id uuid.UUID) (*model.User, *model.UserStats, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil || u == nil {
		return u, nil, err
	}

	var stats model.UserStats
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = $1`, u.ID,
	).Scan(&stats.TheoryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = $1`, u.ID,
	).Scan(&stats.ResponseCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ships WHERE user_id = $1`, u.ID,
	).Scan(&stats.ShipCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM mysteries WHERE user_id = $1`, u.ID,
	).Scan(&stats.MysteryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fanfics WHERE user_id = $1`, u.ID,
	).Scan(&stats.FanficCount)

	return u, &stats, nil
}

func (r *userDAO) ListAll(ctx context.Context, search string, limit, offset int) ([]model.User, int, error) {
	where := ""
	var args []interface{}
	if search != "" {
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
		where = " WHERE u.username ILIKE $1 OR u.display_name ILIKE $2"
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users u"+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf("SELECT "+userColumns+" FROM users u LEFT JOIN user_roles r ON r.user_id = u.id"+where+" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", limitIdx, offsetIdx), args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, total, rows.Err()
}

func (r *userDAO) ListPublic(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.banned_at IS NULL ORDER BY LOWER(u.display_name)`,
	)
	if err != nil {
		return nil, fmt.Errorf("list public users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *userDAO) SearchByName(ctx context.Context, query string, limit int) ([]model.User, error) {
	like := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.banned_at IS NULL AND (u.username ILIKE $1 OR u.display_name ILIKE $2) ORDER BY CASE WHEN u.username ILIKE $3 THEN 0 ELSE 1 END, LOWER(u.display_name) LIMIT $4`,
		like, like, query+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *userDAO) BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banned_at = NOW(), banned_by = $1, ban_reason = $2 WHERE id = $3`,
		bannedBy, reason, userID,
	)
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	return nil
}

func (r *userDAO) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banned_at = NULL, banned_by = NULL, ban_reason = '' WHERE id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("unban user: %w", err)
	}
	return nil
}

func (r *userDAO) IsBanned(ctx context.Context, userID uuid.UUID) (bool, error) {
	var bannedAt *string
	err := r.db.QueryRowContext(ctx,
		`SELECT banned_at FROM users WHERE id = $1`, userID,
	).Scan(&bannedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check ban: %w", err)
	}
	return bannedAt != nil, nil
}

func (r *userDAO) LockUser(ctx context.Context, userID uuid.UUID, lockedBy uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET locked_at = NOW(), locked_by = $1, lock_reason = $2 WHERE id = $3`,
		lockedBy, reason, userID,
	)
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	return nil
}

func (r *userDAO) UnlockUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET locked_at = NULL, locked_by = NULL, lock_reason = '' WHERE id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("unlock user: %w", err)
	}
	return nil
}

func (r *userDAO) IsLocked(ctx context.Context, userID uuid.UUID) (bool, error) {
	var lockedAt *string
	err := r.db.QueryRowContext(ctx,
		`SELECT locked_at FROM users WHERE id = $1`, userID,
	).Scan(&lockedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check lock: %w", err)
	}
	return lockedAt != nil, nil
}

func (r *userDAO) AdminDeleteAccount(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("admin delete account: %w", err)
	}
	return nil
}
