package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	UserRepository interface {
		Create(ctx context.Context, username, password, displayName string) (*User, error)
		GetByID(ctx context.Context, id uuid.UUID) (*User, error)
		GetByUsername(ctx context.Context, username string) (*User, error)
		ExistsByUsername(ctx context.Context, username string) (bool, error)
		Count(ctx context.Context) (int, error)
		ValidatePassword(ctx context.Context, username, password string) (*User, error)
		UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) error
		UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error
		UpdateBannerURL(ctx context.Context, userID uuid.UUID, bannerURL string) error
		ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
		DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error
		GetProfileByUsername(ctx context.Context, username string) (*User, *UserStats, error)
		GetProfileByID(ctx context.Context, id uuid.UUID) (*User, *UserStats, error)
		ListAll(ctx context.Context, search string, limit, offset int) ([]User, int, error)
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
	userColumns = `id, username, password_hash, display_name, created_at, bio, avatar_url, banner_url, favourite_character, gender, pronoun_subject, pronoun_possessive, banned_at, banned_by, ban_reason, social_twitter, social_discord, social_waifulist, social_tumblr, social_github, website, banner_position`
)

func scanUser(row interface{ Scan(dest ...any) error }) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
		&u.Bio, &u.AvatarURL, &u.BannerURL, &u.FavouriteCharacter, &u.Gender,
		&u.PronounSubject, &u.PronounPossessive,
		&u.BannedAt, &u.BannedBy, &u.BanReason,
		&u.SocialTwitter, &u.SocialDiscord, &u.SocialWaifulist, &u.SocialTumblr, &u.SocialGithub, &u.Website,
		&u.BannerPosition)
	return &u, err
}

func (r *userRepository) Create(ctx context.Context, username, password, displayName string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id := uuid.New()

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, display_name) VALUES (?, ?, ?, ?)`,
		id, username, string(hash), displayName,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &User{
		ID:          id,
		Username:    username,
		DisplayName: displayName,
	}, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE LOWER(username) = LOWER(?)`, username,
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

func (r *userRepository) ValidatePassword(ctx context.Context, username, password string) (*User, error) {
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
		 website = ?
		 WHERE id = ?`,
		req.DisplayName, req.Bio, req.AvatarURL, req.BannerURL, req.BannerPosition, req.FavouriteCharacter, req.Gender,
		req.PronounSubject, req.PronounPossessive,
		req.SocialTwitter, req.SocialDiscord, req.SocialWaifulist, req.SocialTumblr, req.SocialGithub, req.Website,
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

func (r *userRepository) GetProfileByUsername(ctx context.Context, username string) (*User, *UserStats, error) {
	u, err := r.GetByUsername(ctx, username)
	if err != nil || u == nil {
		return u, nil, err
	}

	var stats UserStats
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

	return u, &stats, nil
}

func (r *userRepository) GetProfileByID(ctx context.Context, id uuid.UUID) (*User, *UserStats, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil || u == nil {
		return u, nil, err
	}

	var stats UserStats
	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM theories WHERE user_id = ?`, u.ID,
	).Scan(&stats.TheoryCount)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM responses WHERE user_id = ?`, u.ID,
	).Scan(&stats.ResponseCount)

	return u, &stats, nil
}

func (r *userRepository) ListAll(ctx context.Context, search string, limit, offset int) ([]User, int, error) {
	where := ""
	var args []interface{}
	if search != "" {
		where = " WHERE username LIKE ? OR display_name LIKE ?"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users"+where, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+userColumns+" FROM users"+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, total, rows.Err()
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
