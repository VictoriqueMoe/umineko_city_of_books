package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	OCRow struct {
		ID                uuid.UUID
		UserID            uuid.UUID
		Name              string
		Description       string
		Series            string
		CustomSeriesName  string
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
		FavouriteCount    int
		UserFavourited    bool
		CommentCount      int
	}

	OCImageRow struct {
		ID           int64
		OCID         uuid.UUID
		ImageURL     string
		ThumbnailURL string
		Caption      string
		SortOrder    int
	}

	OCCommentRow struct {
		ID                uuid.UUID
		OCID              uuid.UUID
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

	OCCommentMediaRow = CommentMediaRow

	OCSummaryRow struct {
		ID               uuid.UUID
		Name             string
		Series           string
		CustomSeriesName string
		ThumbnailURL     string
	}
)

func (m *OCImageRow) ToResponse() dto.OCImage {
	return dto.OCImage{
		ID:           m.ID,
		ImageURL:     m.ImageURL,
		ThumbnailURL: m.ThumbnailURL,
		Caption:      m.Caption,
		SortOrder:    m.SortOrder,
	}
}

func OCImageRowsToResponse(rows []OCImageRow) []dto.OCImage {
	list := make([]dto.OCImage, len(rows))
	for i := range rows {
		list[i] = rows[i].ToResponse()
	}
	return list
}

func (r *OCRow) ToResponse(gallery []OCImageRow) dto.OCResponse {
	return dto.OCResponse{
		ID: r.ID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Name:             r.Name,
		Description:      r.Description,
		Series:           r.Series,
		CustomSeriesName: r.CustomSeriesName,
		ImageURL:         r.ImageURL,
		ThumbnailURL:     r.ThumbnailURL,
		Gallery:          OCImageRowsToResponse(gallery),
		VoteScore:        r.VoteScore,
		UserVote:         r.UserVote,
		FavouriteCount:   r.FavouriteCount,
		UserFavourited:   r.UserFavourited,
		CommentCount:     r.CommentCount,
		IsCrackOC:        r.VoteScore <= dto.CrackOCThreshold,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func (r *OCSummaryRow) ToResponse() dto.OCSummary {
	return dto.OCSummary{
		ID:               r.ID,
		Name:             r.Name,
		Series:           r.Series,
		CustomSeriesName: r.CustomSeriesName,
		ThumbnailURL:     r.ThumbnailURL,
	}
}

func (r *OCCommentRow) ToResponse(media []OCCommentMediaRow) dto.OCCommentResponse {
	return dto.OCCommentResponse{
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
		Media:     CommentMediaRowsToResponse(media),
		LikeCount: r.LikeCount,
		UserLiked: r.UserLiked,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
