package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	User struct {
		ID                     uuid.UUID
		Username               string
		PasswordHash           string
		DisplayName            string
		CreatedAt              string
		Bio                    string
		AvatarURL              string
		BannerURL              string
		FavouriteCharacter     string
		Gender                 string
		PronounSubject         string
		PronounPossessive      string
		BannedAt               *string
		BannedBy               *uuid.UUID
		BanReason              string
		SocialTwitter          string
		SocialDiscord          string
		SocialWaifulist        string
		SocialTumblr           string
		SocialGithub           string
		Website                string
		BannerPosition         float64
		DmsEnabled             bool
		EpisodeProgress        int
		HigurashiArcProgress   int
		CiconiaChapterProgress int
		Email                  string
		EmailPublic            bool
		DOB                    string
		DOBPublic              bool
		EmailNotifications     bool
		PlayMessageSound       bool
		PlayNotificationSound  bool
		HomePage               string
		GameBoardSort          string
		Theme                  string
		Font                   string
		WideLayout             bool
		IP                     *string
		MysteryScoreAdjustment int
		GMScoreAdjustment      int
		Role                   string
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
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Role:        role.Role(u.Role),
	}
}

func (u *User) ToProfileResponse(stats *UserStats, isSelf bool) *dto.UserProfileResponse {
	email := ""
	emailPublic := u.EmailPublic
	emailNotifications := false
	playMessageSound := false
	playNotificationSound := false
	homePage := ""
	gameBoardSort := ""
	theme := ""
	font := ""
	wideLayout := false
	dob := ""
	dobPublic := u.DOBPublic
	if u.EmailPublic || isSelf {
		email = u.Email
	}
	if u.DOBPublic || isSelf {
		dob = u.DOB
	}
	if isSelf {
		emailNotifications = u.EmailNotifications
		playMessageSound = u.PlayMessageSound
		playNotificationSound = u.PlayNotificationSound
		homePage = u.HomePage
		gameBoardSort = u.GameBoardSort
		theme = u.Theme
		font = u.Font
		wideLayout = u.WideLayout
	}
	return &dto.UserProfileResponse{
		UserResponse:           *u.ToResponse(),
		EpisodeProgress:        u.EpisodeProgress,
		HigurashiArcProgress:   u.HigurashiArcProgress,
		CiconiaChapterProgress: u.CiconiaChapterProgress,
		Bio:                    u.Bio,
		BannerURL:              u.BannerURL,
		BannerPosition:         u.BannerPosition,
		FavouriteCharacter:     u.FavouriteCharacter,
		Gender:                 u.Gender,
		PronounSubject:         u.PronounSubject,
		PronounPossessive:      u.PronounPossessive,
		SocialTwitter:          u.SocialTwitter,
		SocialDiscord:          u.SocialDiscord,
		SocialWaifulist:        u.SocialWaifulist,
		SocialTumblr:           u.SocialTumblr,
		SocialGithub:           u.SocialGithub,
		Website:                u.Website,
		DmsEnabled:             u.DmsEnabled,
		DOB:                    dob,
		DOBPublic:              dobPublic,
		Email:                  email,
		EmailPublic:            emailPublic,
		EmailNotifications:     emailNotifications,
		PlayMessageSound:       playMessageSound,
		PlayNotificationSound:  playNotificationSound,
		HomePage:               homePage,
		GameBoardSort:          gameBoardSort,
		Theme:                  theme,
		Font:                   font,
		WideLayout:             wideLayout,
		CreatedAt:              u.CreatedAt,
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
