package repository

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
	}

	NotificationRow struct {
		ID               int
		UserID           uuid.UUID
		Type             string
		ReferenceID      uuid.UUID
		ReferenceType    string
		ActorID          uuid.UUID
		Read             bool
		CreatedAt        string
		ActorUsername    string
		ActorDisplayName string
		ActorAvatarURL   string
	}
)

type (
	PostRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Corner            string
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		CommentCount      int
		UserLiked         bool
		ViewCount         int
	}

	EmbedRow struct {
		ID        int
		OwnerID   string
		URL       string
		EmbedType string
		Title     string
		Desc      string
		Image     string
		SiteName  string
		VideoID   string
		SortOrder int
	}

	PostMediaRow struct {
		ID           int
		PostID       uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}

	PostLikeUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
	}

	PostCommentRow struct {
		ID                uuid.UUID
		PostID            uuid.UUID
		ParentID          *uuid.UUID
		UserID            uuid.UUID
		Body              string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		UserLiked         bool
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

func (u *User) ToProfileResponse(stats *UserStats) *dto.UserProfileResponse {
	var dtoEmail = ""
	if u.EmailPublic {
		dtoEmail = u.Email
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
		Email:              dtoEmail,
		EmailPublic:        u.EmailPublic,
		EmailNotifications: u.EmailNotifications,
		HomePage:           u.HomePage,
		CreatedAt:          u.CreatedAt,
		Stats: dto.UserStatsDTO{
			TheoryCount:   stats.TheoryCount,
			ResponseCount: stats.ResponseCount,
			VotesReceived: stats.VotesReceived,
		},
	}
}
