package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	FanficRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Title             string
		Summary           string
		Series            string
		Rating            string
		Language          string
		Status            string
		IsOneshot         bool
		ContainsLemons    bool
		CoverImageURL     string
		CoverThumbnailURL string
		WordCount         int
		ChapterCount      int
		FavouriteCount    int
		ViewCount         int
		CommentCount      int
		UserFavourited    bool
		IsPairing         bool
		PublishedAt       string
		CreatedAt         string
		UpdatedAt         *string
		AuthorUsername    string
		AuthorDisplayName string
		AuthorAvatarURL   string
		AuthorRole        string
	}

	FanficChapterRow struct {
		ID         uuid.UUID
		FanficID   uuid.UUID
		ChapterNum int
		Title      string
		Body       string
		WordCount  int
		CreatedAt  string
		UpdatedAt  *string
	}

	FanficChapterSummaryRow struct {
		ID         uuid.UUID
		ChapterNum int
		Title      string
		WordCount  int
	}

	FanficCharacterRow struct {
		ID            int
		FanficID      uuid.UUID
		Series        string
		CharacterID   string
		CharacterName string
		SortOrder     int
		IsPairing     bool
	}

	FanficCommentRow struct {
		ID                uuid.UUID
		FanficID          uuid.UUID
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

	FanficCommentMediaRow struct {
		ID           int
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}
)

func (r *FanficRow) ToResponse(genres []string, characters []FanficCharacterRow) dto.FanficResponse {
	chars := make([]dto.FanficCharacter, len(characters))
	for i, c := range characters {
		chars[i] = dto.FanficCharacter{
			Series:        c.Series,
			CharacterID:   c.CharacterID,
			CharacterName: c.CharacterName,
			SortOrder:     c.SortOrder,
		}
	}
	if genres == nil {
		genres = []string{}
	}
	return dto.FanficResponse{
		ID: r.ID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Title:             r.Title,
		Summary:           r.Summary,
		Series:            r.Series,
		Rating:            r.Rating,
		Language:          r.Language,
		Status:            r.Status,
		IsOneshot:         r.IsOneshot,
		ContainsLemons:    r.ContainsLemons,
		CoverImageURL:     r.CoverImageURL,
		CoverThumbnailURL: r.CoverThumbnailURL,
		Genres:            genres,
		Characters:        chars,
		IsPairing:         r.IsPairing,
		WordCount:         r.WordCount,
		ChapterCount:      r.ChapterCount,
		FavouriteCount:    r.FavouriteCount,
		ViewCount:         r.ViewCount,
		CommentCount:      r.CommentCount,
		UserFavourited:    r.UserFavourited,
		PublishedAt:       r.PublishedAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func (r *FanficCommentRow) ToResponse(media []FanficCommentMediaRow) dto.FanficCommentResponse {
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
	return dto.FanficCommentResponse{
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
