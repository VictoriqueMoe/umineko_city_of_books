package dto

import (
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	UserResponse struct {
		ID              uuid.UUID `json:"id"`
		Username        string    `json:"username"`
		DisplayName     string    `json:"display_name"`
		AvatarURL       string    `json:"avatar_url,omitempty"`
		Role            role.Role `json:"role,omitempty"`
		EpisodeProgress int       `json:"episode_progress"`
	}

	UserProfileResponse struct {
		ID                 uuid.UUID    `json:"id"`
		Username           string       `json:"username"`
		DisplayName        string       `json:"display_name"`
		Bio                string       `json:"bio"`
		AvatarURL          string       `json:"avatar_url"`
		BannerURL          string       `json:"banner_url"`
		BannerPosition     float64      `json:"banner_position"`
		FavouriteCharacter string       `json:"favourite_character"`
		Gender             string       `json:"gender"`
		PronounSubject     string       `json:"pronoun_subject"`
		PronounPossessive  string       `json:"pronoun_possessive"`
		Role               role.Role    `json:"role,omitempty"`
		Online             bool         `json:"online"`
		SocialTwitter      string       `json:"social_twitter"`
		SocialDiscord      string       `json:"social_discord"`
		SocialWaifulist    string       `json:"social_waifulist"`
		SocialTumblr       string       `json:"social_tumblr"`
		SocialGithub       string       `json:"social_github"`
		Website            string       `json:"website"`
		DmsEnabled         bool         `json:"dms_enabled"`
		EpisodeProgress    int          `json:"episode_progress"`
		CreatedAt          string       `json:"created_at"`
		Stats              UserStatsDTO `json:"stats"`
	}

	UserStatsDTO struct {
		TheoryCount   int `json:"theory_count"`
		ResponseCount int `json:"response_count"`
		VotesReceived int `json:"votes_received"`
	}

	UpdateProfileRequest struct {
		DisplayName        string  `json:"display_name"`
		Bio                string  `json:"bio"`
		AvatarURL          string  `json:"avatar_url"`
		BannerURL          string  `json:"banner_url"`
		BannerPosition     float64 `json:"banner_position"`
		FavouriteCharacter string  `json:"favourite_character"`
		Gender             string  `json:"gender"`
		PronounSubject     string  `json:"pronoun_subject"`
		PronounPossessive  string  `json:"pronoun_possessive"`
		SocialTwitter      string  `json:"social_twitter"`
		SocialDiscord      string  `json:"social_discord"`
		SocialWaifulist    string  `json:"social_waifulist"`
		SocialTumblr       string  `json:"social_tumblr"`
		SocialGithub       string  `json:"social_github"`
		Website            string  `json:"website"`
		DmsEnabled         bool    `json:"dms_enabled"`
		EpisodeProgress    int     `json:"episode_progress"`
	}

	ChangePasswordRequest struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	DeleteAccountRequest struct {
		Password string `json:"password"`
	}

	Credentials interface {
		GetUsername() string
		GetPassword() string
	}

	LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	RegisterRequest struct {
		LoginRequest
		DisplayName string `json:"display_name"`
		InviteCode  string `json:"invite_code,omitempty"`
	}
)

func (r LoginRequest) GetUsername() string { return r.Username }
func (r LoginRequest) GetPassword() string { return r.Password }
