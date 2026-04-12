package dto

import (
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	UserResponse struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		DisplayName string    `json:"display_name"`
		AvatarURL   string    `json:"avatar_url,omitempty"`
		Role        role.Role `json:"role,omitempty"`
	}

	UserProfileResponse struct {
		UserResponse
		Bio                string       `json:"bio"`
		EpisodeProgress    int          `json:"episode_progress"`
		BannerURL          string       `json:"banner_url"`
		BannerPosition     float64      `json:"banner_position"`
		FavouriteCharacter string       `json:"favourite_character"`
		Gender             string       `json:"gender"`
		PronounSubject     string       `json:"pronoun_subject"`
		PronounPossessive  string       `json:"pronoun_possessive"`
		Online             bool         `json:"online"`
		SocialTwitter      string       `json:"social_twitter"`
		SocialDiscord      string       `json:"social_discord"`
		SocialWaifulist    string       `json:"social_waifulist"`
		SocialTumblr       string       `json:"social_tumblr"`
		SocialGithub       string       `json:"social_github"`
		Website            string       `json:"website"`
		DmsEnabled         bool         `json:"dms_enabled"`
		Email              string       `json:"email,omitempty"`
		EmailPublic        bool         `json:"email_public"`
		EmailNotifications bool         `json:"email_notifications"`
		HomePage           string       `json:"home_page"`
		GameBoardSort      string       `json:"game_board_sort"`
		Theme              string       `json:"theme"`
		Font               string       `json:"font"`
		WideLayout         bool         `json:"wide_layout"`
		CreatedAt          string       `json:"created_at"`
		Stats              UserStatsDTO `json:"stats"`
	}

	UserStatsDTO struct {
		TheoryCount   int `json:"theory_count"`
		ResponseCount int `json:"response_count"`
		VotesReceived int `json:"votes_received"`
		ShipCount     int `json:"ship_count"`
		MysteryCount  int `json:"mystery_count"`
		FanficCount   int `json:"fanfic_count"`
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
		Email              string  `json:"email"`
		EmailPublic        bool    `json:"email_public"`
		EmailNotifications bool    `json:"email_notifications"`
		HomePage           string  `json:"home_page"`
		GameBoardSort      string  `json:"game_board_sort"`
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
		Username       string `json:"username"`
		Password       string `json:"password"`
		TurnstileToken string `json:"turnstile_token,omitempty"`
	}

	RegisterRequest struct {
		LoginRequest
		DisplayName string `json:"display_name"`
		InviteCode  string `json:"invite_code,omitempty"`
	}
)

func (r LoginRequest) GetUsername() string { return r.Username }
func (r LoginRequest) GetPassword() string { return r.Password }
