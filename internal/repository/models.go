package repository

import "umineko_city_of_books/internal/dto"

type (
	User struct {
		ID                 int
		Username           string
		PasswordHash       string
		DisplayName        string
		CreatedAt          string
		Bio                string
		AvatarURL          string
		FavouriteCharacter string
		SocialTwitter      string
		SocialDiscord      string
		SocialWaifulist    string
		SocialTumblr       string
		SocialGithub       string
		Website            string
	}

	UserStats struct {
		TheoryCount   int
		ResponseCount int
		VotesReceived int
	}
)

func (u *User) ToResponse() *dto.UserResponse {
	return &dto.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
	}
}

func (u *User) ToProfileResponse(stats *UserStats) *dto.UserProfileResponse {
	return &dto.UserProfileResponse{
		ID:                 u.ID,
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Bio:                u.Bio,
		AvatarURL:          u.AvatarURL,
		FavouriteCharacter: u.FavouriteCharacter,
		SocialTwitter:      u.SocialTwitter,
		SocialDiscord:      u.SocialDiscord,
		SocialWaifulist:    u.SocialWaifulist,
		SocialTumblr:       u.SocialTumblr,
		SocialGithub:       u.SocialGithub,
		Website:            u.Website,
		CreatedAt:          u.CreatedAt,
		Stats: dto.UserStatsDTO{
			TheoryCount:   stats.TheoryCount,
			ResponseCount: stats.ResponseCount,
			VotesReceived: stats.VotesReceived,
		},
	}
}
