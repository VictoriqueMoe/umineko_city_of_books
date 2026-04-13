package model

import (
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
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

	ShipCommentMediaRow = CommentMediaRow
)

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
	mediaList := CommentMediaRowsToResponse(media)
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
