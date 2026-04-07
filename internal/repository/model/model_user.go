package model

import (
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	User struct {
		ID                 uuid.UUID
		Username           string
		PasswordHash       string
		DisplayName        string
		CreatedAt          string
		Bio                string
		AvatarURL          string
		BannerURL          string
		FavouriteCharacter string
		Gender             string
		PronounSubject     string
		PronounPossessive  string
		BannedAt           *string
		BannedBy           *uuid.UUID
		BanReason          string
		SocialTwitter      string
		SocialDiscord      string
		SocialWaifulist    string
		SocialTumblr       string
		SocialGithub       string
		Website            string
		BannerPosition     float64
		DmsEnabled         bool
		EpisodeProgress    int
		Email              string
		EmailPublic        bool
		EmailNotifications bool
		HomePage           string
	}

	UserStats struct {
		TheoryCount   int
		ResponseCount int
		VotesReceived int
		ShipCount     int
		MysteryCount  int
		FanficCount   int
	}
)

func (u *User) ToResponse() *dto.UserResponse {
	return &dto.UserResponse{
		ID:              u.ID,
		Username:        u.Username,
		DisplayName:     u.DisplayName,
		AvatarURL:       u.AvatarURL,
		EpisodeProgress: u.EpisodeProgress,
		HomePage:        u.HomePage,
	}
}

func (u *User) ToProfileResponse(stats *UserStats, isSelf bool) *dto.UserProfileResponse {
	email := ""
	emailPublic := u.EmailPublic
	emailNotifications := false
	homePage := ""
	if u.EmailPublic || isSelf {
		email = u.Email
	}
	if isSelf {
		emailNotifications = u.EmailNotifications
		homePage = u.HomePage
	}
	return &dto.UserProfileResponse{
		ID:                 u.ID,
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Bio:                u.Bio,
		AvatarURL:          u.AvatarURL,
		BannerURL:          u.BannerURL,
		BannerPosition:     u.BannerPosition,
		FavouriteCharacter: u.FavouriteCharacter,
		Gender:             u.Gender,
		PronounSubject:     u.PronounSubject,
		PronounPossessive:  u.PronounPossessive,
		SocialTwitter:      u.SocialTwitter,
		SocialDiscord:      u.SocialDiscord,
		SocialWaifulist:    u.SocialWaifulist,
		SocialTumblr:       u.SocialTumblr,
		SocialGithub:       u.SocialGithub,
		Website:            u.Website,
		DmsEnabled:         u.DmsEnabled,
		EpisodeProgress:    u.EpisodeProgress,
		Email:              email,
		EmailPublic:        emailPublic,
		EmailNotifications: emailNotifications,
		HomePage:           homePage,
		CreatedAt:          u.CreatedAt,
		Stats: dto.UserStatsDTO{
			TheoryCount:   stats.TheoryCount,
			ResponseCount: stats.ResponseCount,
			VotesReceived: stats.VotesReceived,
			ShipCount:     stats.ShipCount,
			MysteryCount:  stats.MysteryCount,
			FanficCount:   stats.FanficCount,
		},
	}
}
