package model

import (
	"strings"

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
		LockedAt               *string
		LockedBy               *uuid.UUID
		LockReason             string
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
		EmailVerified          bool
		VerifyGraceUntil       string
		DOB                    string
		DOBPublic              bool
		EmailNotifications     bool
		PlayMessageSound       bool
		PlayNotificationSound  bool
		HomePage               string
		GameBoardSort          string
		DefaultProfileTab      string
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

func (u *User) DisplayLabel() string {
	if u == nil {
		return "A user"
	}
	if name := strings.TrimSpace(u.DisplayName); name != "" {
		return name
	}
	if name := strings.TrimSpace(u.Username); name != "" {
		return name
	}
	return "A user"
}

func (u *User) ToResponse() *dto.UserResponse {
	return &dto.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Role:        role.Role(u.Role),
		Banned:      u.BannedAt != nil,
		BanReason:   u.BanReason,
		Locked:      u.LockedAt != nil,
		LockReason:  u.LockReason,
	}
}

func (u *User) ToProfileResponse(stats *UserStats, isSelf bool) *dto.UserProfileResponse {
	resp := &dto.UserProfileResponse{
		UserResponse:           *u.ToResponse(),
		Bio:                    u.Bio,
		EpisodeProgress:        u.EpisodeProgress,
		HigurashiArcProgress:   u.HigurashiArcProgress,
		CiconiaChapterProgress: u.CiconiaChapterProgress,
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
		DOBPublic:              u.DOBPublic,
		EmailPublic:            u.EmailPublic,
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

	if u.EmailPublic || isSelf {
		resp.Email = u.Email
	}
	if u.DOBPublic || isSelf {
		resp.DOB = u.DOB
	}
	if isSelf {
		resp.Private = &dto.UserPrivateFields{
			EmailVerified:         u.EmailVerified,
			VerifyGraceUntil:      u.VerifyGraceUntil,
			EmailNotifications:    u.EmailNotifications,
			PlayMessageSound:      u.PlayMessageSound,
			PlayNotificationSound: u.PlayNotificationSound,
			HomePage:              u.HomePage,
			GameBoardSort:         u.GameBoardSort,
			DefaultProfileTab:     u.DefaultProfileTab,
			Theme:                 u.Theme,
			Font:                  u.Font,
			WideLayout:            u.WideLayout,
		}
	}

	return resp
}
