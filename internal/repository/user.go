package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dto"

	"golang.org/x/crypto/bcrypt"
)

type (
	UserRepository interface {
		Create(ctx context.Context, username, password, displayName string) (*User, error)
		GetByID(ctx context.Context, id int) (*User, error)
		GetByUsername(ctx context.Context, username string) (*User, error)
		ValidatePassword(ctx context.Context, username, password string) (*User, error)
		UpdateProfile(ctx context.Context, userID int, req dto.UpdateProfileRequest) error
		UpdateAvatarURL(ctx context.Context, userID int, avatarURL string) error
		GetProfileByUsername(ctx context.Context, username string) (*User, *UserStats, error)
	}

	userRepository struct {
		db *sql.DB
	}
)

const (
	userColumns = `id, username, password_hash, display_name, created_at, bio, avatar_url, favourite_character, social_twitter, social_discord, social_waifulist, social_tumblr, social_github, website`
)

func scanUser(row interface{ Scan(dest ...any) error }) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
		&u.Bio, &u.AvatarURL, &u.FavouriteCharacter,
		&u.SocialTwitter, &u.SocialDiscord, &u.SocialWaifulist, &u.SocialTumblr, &u.SocialGithub, &u.Website)
	return &u, err
}

func (r *userRepository) Create(ctx context.Context, username, password, displayName string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, display_name) VALUES (?, ?, ?)`,
		username, string(hash), displayName,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	id, _ := result.LastInsertId()
	return &User{
		ID:          int(id),
		Username:    username,
		DisplayName: displayName,
	}, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int) (*User, error) {
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
		`SELECT `+userColumns+` FROM users WHERE username = ?`, username,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
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

func (r *userRepository) UpdateProfile(ctx context.Context, userID int, req dto.UpdateProfileRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, bio = ?, avatar_url = ?, favourite_character = ?,
		 social_twitter = ?, social_discord = ?, social_waifulist = ?, social_tumblr = ?, social_github = ?,
		 website = ?
		 WHERE id = ?`,
		req.DisplayName, req.Bio, req.AvatarURL, req.FavouriteCharacter,
		req.SocialTwitter, req.SocialDiscord, req.SocialWaifulist, req.SocialTumblr, req.SocialGithub, req.Website,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateAvatarURL(ctx context.Context, userID int, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET avatar_url = ? WHERE id = ?`, avatarURL, userID,
	)
	if err != nil {
		return fmt.Errorf("update avatar url: %w", err)
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
