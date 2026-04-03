package repository

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

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
		Message          string
		Read             bool
		CreatedAt        string
		ActorUsername    string
		ActorDisplayName string
		ActorAvatarURL   string
		ActorRole        string
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

type (
	ArtRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Corner            string
		ArtType           string
		Title             string
		Description       string
		ImageURL          string
		ThumbnailURL      string
		GalleryID         *uuid.UUID
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		LikeCount         int
		CommentCount      int
		ViewCount         int
		UserLiked         bool
	}

	GalleryRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Name              string
		Description       string
		CoverArtID        *uuid.UUID
		CoverImageURL     string
		CoverThumbnailURL string
		ArtCount          int
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
	}

	ArtCommentRow struct {
		ID                uuid.UUID
		ArtID             uuid.UUID
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

	TagCount struct {
		Tag   string
		Count int
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
		},
	}
}

func (r *ArtRow) ToResponse(tags []string) dto.ArtResponse {
	if tags == nil {
		tags = []string{}
	}
	return dto.ArtResponse{
		ID: r.ID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Corner:       r.Corner,
		ArtType:      r.ArtType,
		Title:        r.Title,
		Description:  r.Description,
		ImageURL:     r.ImageURL,
		ThumbnailURL: r.ThumbnailURL,
		GalleryID:    r.GalleryID,
		Tags:         tags,
		LikeCount:    r.LikeCount,
		CommentCount: r.CommentCount,
		ViewCount:    r.ViewCount,
		UserLiked:    r.UserLiked,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (g *GalleryRow) ToResponse() dto.GalleryResponse {
	return dto.GalleryResponse{
		ID: g.ID,
		Author: dto.UserResponse{
			ID:          g.UserID,
			Username:    g.AuthorUsername,
			DisplayName: g.AuthorDisplayName,
			AvatarURL:   g.AuthorAvatarURL,
		},
		Name:              g.Name,
		Description:       g.Description,
		CoverImageURL:     g.CoverImageURL,
		CoverThumbnailURL: g.CoverThumbnailURL,
		ArtCount:          g.ArtCount,
		CreatedAt:         g.CreatedAt,
		UpdatedAt:         g.UpdatedAt,
	}
}
