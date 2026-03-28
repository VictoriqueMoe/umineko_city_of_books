package dto

type (
	UserResponse struct {
		ID          int    `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url,omitempty"`
	}

	UserProfileResponse struct {
		ID                 int          `json:"id"`
		Username           string       `json:"username"`
		DisplayName        string       `json:"display_name"`
		Bio                string       `json:"bio"`
		AvatarURL          string       `json:"avatar_url"`
		FavouriteCharacter string       `json:"favourite_character"`
		SocialTwitter      string       `json:"social_twitter"`
		SocialDiscord      string       `json:"social_discord"`
		SocialWaifulist    string       `json:"social_waifulist"`
		SocialTumblr       string       `json:"social_tumblr"`
		SocialGithub       string       `json:"social_github"`
		Website            string       `json:"website"`
		CreatedAt          string       `json:"created_at"`
		Stats              UserStatsDTO `json:"stats"`
	}

	UserStatsDTO struct {
		TheoryCount   int `json:"theory_count"`
		ResponseCount int `json:"response_count"`
		VotesReceived int `json:"votes_received"`
	}

	UpdateProfileRequest struct {
		DisplayName        string `json:"display_name"`
		Bio                string `json:"bio"`
		AvatarURL          string `json:"avatar_url"`
		FavouriteCharacter string `json:"favourite_character"`
		SocialTwitter      string `json:"social_twitter"`
		SocialDiscord      string `json:"social_discord"`
		SocialWaifulist    string `json:"social_waifulist"`
		SocialTumblr       string `json:"social_tumblr"`
		SocialGithub       string `json:"social_github"`
		Website            string `json:"website"`
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
	}
)

func (r LoginRequest) GetUsername() string { return r.Username }
func (r LoginRequest) GetPassword() string { return r.Password }
