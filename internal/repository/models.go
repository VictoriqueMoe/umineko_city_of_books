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
		ShipCount     int
		MysteryCount  int
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

	ShipRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Title             string
		Description       string
		ImageURL          string
		ThumbnailURL      string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
		VoteScore         int
		UserVote          int
		CommentCount      int
	}

	ShipCharacterRow struct {
		ID            int
		ShipID        uuid.UUID
		Series        string
		CharacterID   string
		CharacterName string
		SortOrder     int
	}

	ShipCommentRow struct {
		ID                uuid.UUID
		ShipID            uuid.UUID
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

	ShipCommentMediaRow struct {
		ID           int
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
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

func (r *ShipRow) ToResponse(characters []ShipCharacterRow) dto.ShipResponse {
	chars := make([]dto.ShipCharacter, len(characters))
	for i, c := range characters {
		chars[i] = dto.ShipCharacter{
			Series:        c.Series,
			CharacterID:   c.CharacterID,
			CharacterName: c.CharacterName,
			SortOrder:     c.SortOrder,
		}
	}
	return dto.ShipResponse{
		ID: r.ID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Title:        r.Title,
		Description:  r.Description,
		ImageURL:     r.ImageURL,
		ThumbnailURL: r.ThumbnailURL,
		Characters:   chars,
		VoteScore:    r.VoteScore,
		UserVote:     r.UserVote,
		CommentCount: r.CommentCount,
		IsCrackship:  r.VoteScore <= dto.CrackshipThreshold,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (r *ShipCommentRow) ToResponse(media []ShipCommentMediaRow) dto.ShipCommentResponse {
	mediaList := make([]dto.PostMediaResponse, len(media))
	for i, m := range media {
		mediaList[i] = dto.PostMediaResponse{
			ID:           m.ID,
			MediaURL:     m.MediaURL,
			MediaType:    m.MediaType,
			ThumbnailURL: m.ThumbnailURL,
			SortOrder:    m.SortOrder,
		}
	}
	return dto.ShipCommentResponse{
		ID:       r.ID,
		ParentID: r.ParentID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Body:      r.Body,
		Media:     mediaList,
		LikeCount: r.LikeCount,
		UserLiked: r.UserLiked,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (r *MysteryRow) ToResponse() dto.MysteryResponse {
	resp := dto.MysteryResponse{
		ID:         r.ID,
		Title:      r.Title,
		Body:       r.Body,
		Difficulty: r.Difficulty,
		Solved:     r.Solved,
		SolvedAt:   r.SolvedAt,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		AttemptCount: r.AttemptCount,
		ClueCount:    r.ClueCount,
		CreatedAt:    r.CreatedAt,
	}
	if r.WinnerID != nil && r.WinnerUsername != nil {
		resp.Winner = &dto.UserResponse{
			ID:          *r.WinnerID,
			Username:    *r.WinnerUsername,
			DisplayName: *r.WinnerDisplayName,
			AvatarURL:   *r.WinnerAvatarURL,
			Role:        role.Role(*r.WinnerRole),
		}
	}
	return resp
}
