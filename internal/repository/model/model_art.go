package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
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

func (c *ArtCommentRow) ToResponse(media []PostMediaRow, embeds []EmbedRow) dto.ArtCommentResponse {
	return dto.ArtCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     MediaRowsToResponse(media),
		Embeds:    EmbedRowsToResponse(embeds),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
