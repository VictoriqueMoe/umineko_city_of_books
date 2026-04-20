package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	UserRepository interface {
		Create(ctx context.Context, username, password, displayName string) (*model.User, error)
		GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
		GetByUsername(ctx context.Context, username string) (*model.User, error)
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
		DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error
		GetProfileByUsername(ctx context.Context, username string) (*model.User, *model.UserStats, error)
		GetProfileByID(ctx context.Context, id uuid.UUID) (*model.User, *model.UserStats, error)
		ListAll(ctx context.Context, search string, limit, offset int) ([]model.User, int, error)
		ListPublic(ctx context.Context) ([]model.User, error)
		SearchByName(ctx context.Context, query string, limit int) ([]model.User, error)
		BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string) error
		UnbanUser(ctx context.Context, userID uuid.UUID) error
		IsBanned(ctx context.Context, userID uuid.UUID) (bool, error)
		AdminDeleteAccount(ctx context.Context, userID uuid.UUID) error
	}

	userRepository struct {
		db *sql.DB
	}
)

const (
	userColumns = `u.id, u.username, u.password_hash, u.display_name, u.created_at, u.bio, u.avatar_url, u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive, u.banned_at, u.banned_by, u.ban_reason, u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress, u.ciconia_chapter_progress, u.email, u.email_public, u.dob, u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page, u.game_board_sort, u.theme, u.font, u.wide_layout, u.ip, u.mystery_score_adjustment, u.gm_score_adjustment, COALESCE(r.role, '')`
)

func scanUser(row interface{ Scan(dest ...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
		&u.Bio, &u.AvatarURL, &u.BannerURL, &u.FavouriteCharacter, &u.Gender,
		&u.PronounSubject, &u.PronounPossessive,
		&u.BannedAt, &u.BannedBy, &u.BanReason,
		&u.SocialTwitter, &u.SocialDiscord, &u.SocialWaifulist, &u.SocialTumblr, &u.SocialGithub, &u.Website,
		&u.BannerPosition, &u.DmsEnabled, &u.EpisodeProgress, &u.HigurashiArcProgress, &u.CiconiaChapterProgress, &u.Email, &u.EmailPublic, &u.DOB, &u.DOBPublic, &u.EmailNotifications, &u.PlayMessageSound, &u.PlayNotificationSound, &u.HomePage, &u.GameBoardSort, &u.Theme, &u.Font, &u.WideLayout, &u.IP, &u.MysteryScoreAdjustment, &u.GMScoreAdjustment, &u.Role)
	return &u, err
}

func (r *userRepository) Create(ctx context.Context, username, password, displayName string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id := uuid.New()

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, display_name, home_page) VALUES (?, ?, ?, ?, ?)`,
		id, username, string(hash), displayName, "landing",
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &model.User{
		ID:          id,
		Username:    username,
		DisplayName: displayName,
	}, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE LOWER(u.username) = LOWER(?)`, username,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE LOWER(username) = LOWER(?)`, username,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}
	return count > 0, nil
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *userRepository) ValidatePassword(ctx context.Context, username, password string) (*model.User, error) {
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

func (r *userRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, bio = ?, avatar_url = ?, banner_url = ?, banner_position = ?, favourite_character = ?, gender = ?,
		 pronoun_subject = ?, pronoun_possessive = ?,
		 social_twitter = ?, social_discord = ?, social_waifulist = ?, social_tumblr = ?, social_github = ?,
		 website = ?, dms_enabled = ?, episode_progress = ?, higurashi_arc_progress = ?, ciconia_chapter_progress = ?, email = ?, email_public = ?, dob = ?, dob_public = ?, email_notifications = ?, play_message_sound = ?, play_notification_sound = ?, home_page = ?, game_board_sort = ?
		 WHERE id = ?`,
		req.DisplayName, req.Bio, req.AvatarURL, req.BannerURL, req.BannerPosition, req.FavouriteCharacter, req.Gender,
		req.PronounSubject, req.PronounPossessive,
		req.SocialTwitter, req.SocialDiscord, req.SocialWaifulist, req.SocialTumblr, req.SocialGithub, req.Website,
		req.DmsEnabled, req.EpisodeProgress, req.HigurashiArcProgress, req.CiconiaChapterProgress, req.Email, req.EmailPublic, req.DOB, req.DOBPublic, req.EmailNotifications, req.PlayMessageSound, req.PlayNotificationSound, req.HomePage, req.GameBoardSort,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET avatar_url = ? WHERE id = ?`, avatarURL, userID,
	)
	if err != nil {
		return fmt.Errorf("update avatar url: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banner_url = ? WHERE id = ?`, bannerURL, userID,
	)
	if err != nil {
		return fmt.Errorf("update banner url: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateIP(ctx context.Context, userID uuid.UUID, ip string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET ip = ? WHERE id = ?`, ip, userID,
	)
	if err != nil {
		return fmt.Errorf("update ip: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateGameBoardSort(ctx context.Context, userID uuid.UUID, sort string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET game_board_sort = ? WHERE id = ?`, sort, userID,
	)
	if err != nil {
		return fmt.Errorf("update game board sort: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateAppearance(ctx context.Context, userID uuid.UUID, theme, font string, wideLayout bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET theme = ?, font = ?, wide_layout = ? WHERE id = ?`, theme, font, wideLayout, userID,
	)
	if err != nil {
		return fmt.Errorf("update appearance: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateMysteryScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mystery_score_adjustment = ? WHERE id = ?`, adjustment, userID,
	)
	if err != nil {
		return fmt.Errorf("update mystery score adjustment: %w", err)
	}
	return nil
}

func (r *userRepository) GetDetectiveRawScore(ctx context.Context, userID uuid.UUID) (int, error) {
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
		FROM mysteries m WHERE m.winner_id = ? AND m.solved = 1`, userID,
	).Scan(&score)
	return score, err
}

func (r *userRepository) GetGMRawScore(ctx context.Context, userID uuid.UUID) (int, error) {
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
			+ MIN((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
		), 0)
		FROM mysteries m WHERE m.user_id = ? AND m.solved = 1`, userID,
	).Scan(&score)
	return score, err
}

func (r *userRepository) UpdateGMScoreAdjustment(ctx context.Context, userID uuid.UUID, adjustment int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET gm_score_adjustment = ? WHERE id = ?`, adjustment, userID,
	)
	if err != nil {
		return fmt.Errorf("update gm score adjustment: %w", err)
	}
	return nil
}

func (r *userRepository) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
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
		`UPDATE users SET password_hash = ? WHERE id = ?`, string(hash), userID,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *userRepository) DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error {
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
		`DELETE FROM users WHERE id = ?`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func (r *userRepository) GetProfileByUsername(ctx context.Context, username string) (*model.User, *model.UserStats, error) {
	u, err := r.GetByUsername(ctx, username)
	if err != nil || u == nil {
		return u, nil, err
	}

	var stats model.UserStats
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = ?`, u.ID,
	).Scan(&stats.TheoryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = ?`, u.ID,
	).Scan(&stats.ResponseCount)

	var theoryVotes, responseVotes int
	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tv.value), 0) FROM theory_votes tv JOIN theories t ON tv.theory_id = t.id WHERE t.user_id = ?`, u.ID,
	).Scan(&theoryVotes)

	r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(rv.value), 0) FROM response_votes rv JOIN responses r ON rv.response_id = r.id WHERE r.user_id = ?`, u.ID,
	).Scan(&responseVotes)

	stats.VotesReceived = theoryVotes + responseVotes

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ships WHERE user_id = ?`, u.ID,
	).Scan(&stats.ShipCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM mysteries WHERE user_id = ?`, u.ID,
	).Scan(&stats.MysteryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fanfics WHERE user_id = ?`, u.ID,
	).Scan(&stats.FanficCount)

	return u, &stats, nil
}

func (r *userRepository) GetProfileByID(ctx context.Context, id uuid.UUID) (*model.User, *model.UserStats, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil || u == nil {
		return u, nil, err
	}

	var stats model.UserStats
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = ?`, u.ID,
	).Scan(&stats.TheoryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = ?`, u.ID,
	).Scan(&stats.ResponseCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ships WHERE user_id = ?`, u.ID,
	).Scan(&stats.ShipCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM mysteries WHERE user_id = ?`, u.ID,
	).Scan(&stats.MysteryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fanfics WHERE user_id = ?`, u.ID,
	).Scan(&stats.FanficCount)

	return u, &stats, nil
}

func (r *userRepository) ListAll(ctx context.Context, search string, limit, offset int) ([]model.User, int, error) {
	where := ""
	var args []interface{}
	if search != "" {
		where = " WHERE u.username LIKE ? OR u.display_name LIKE ?"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
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

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+userColumns+" FROM users u LEFT JOIN user_roles r ON r.user_id = u.id"+where+" ORDER BY u.created_at DESC LIMIT ? OFFSET ?", args...,
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

func (r *userRepository) ListPublic(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.banned_at IS NULL ORDER BY u.display_name COLLATE NOCASE`,
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

func (r *userRepository) SearchByName(ctx context.Context, query string, limit int) ([]model.User, error) {
	like := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users u LEFT JOIN user_roles r ON r.user_id = u.id WHERE u.banned_at IS NULL AND (u.username LIKE ? OR u.display_name LIKE ?) ORDER BY CASE WHEN u.username LIKE ? THEN 0 ELSE 1 END, u.display_name COLLATE NOCASE LIMIT ?`,
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

func (r *userRepository) BanUser(ctx context.Context, userID uuid.UUID, bannedBy uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banned_at = CURRENT_TIMESTAMP, banned_by = ?, ban_reason = ? WHERE id = ?`,
		bannedBy, reason, userID,
	)
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	return nil
}

func (r *userRepository) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banned_at = NULL, banned_by = NULL, ban_reason = '' WHERE id = ?`, userID,
	)
	if err != nil {
		return fmt.Errorf("unban user: %w", err)
	}
	return nil
}

func (r *userRepository) IsBanned(ctx context.Context, userID uuid.UUID) (bool, error) {
	var bannedAt *string
	err := r.db.QueryRowContext(ctx,
		`SELECT banned_at FROM users WHERE id = ?`, userID,
	).Scan(&bannedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check ban: %w", err)
	}
	return bannedAt != nil, nil
}

func (r *userRepository) AdminDeleteAccount(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("admin delete account: %w", err)
	}
	return nil
}
