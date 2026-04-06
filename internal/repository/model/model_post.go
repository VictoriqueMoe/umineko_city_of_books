package model

import (
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
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

	PollRow struct {
		ID              string
		PostID          string
		DurationSeconds int
		ExpiresAt       string
	}

	PollOptionRow struct {
		ID        int
		PollID    string
		Label     string
		SortOrder int
		VoteCount int
	}
)

func (r *PostRow) ToResponse(media []PostMediaRow, embeds []EmbedRow) dto.PostResponse {
	return dto.PostResponse{
		ID: r.ID,
		Author: dto.UserResponse{
			ID:          r.UserID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
		Body:         r.Body,
		Media:        MediaRowsToResponse(media),
		Embeds:       EmbedRowsToResponse(embeds),
		LikeCount:    r.LikeCount,
		CommentCount: r.CommentCount,
		ViewCount:    r.ViewCount,
		UserLiked:    r.UserLiked,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (c *PostCommentRow) ToResponse(media []PostMediaRow, embeds []EmbedRow) dto.PostCommentResponse {
	return dto.PostCommentResponse{
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

func (p *PollRow) ToResponse(options []PollOptionRow, votedOption *int) *dto.PollResponse {
	expired := time.Now().UTC().After(ParseTime(p.ExpiresAt))
	showResults := votedOption != nil || expired

	totalVotes := 0
	for _, o := range options {
		totalVotes += o.VoteCount
	}

	dtoOptions := make([]dto.PollOptionResponse, len(options))
	for i, o := range options {
		opt := dto.PollOptionResponse{
			ID:    o.ID,
			Label: o.Label,
		}
		if showResults && totalVotes > 0 {
			opt.VoteCount = o.VoteCount
			opt.Percent = float64(o.VoteCount) / float64(totalVotes) * 100
		}
		dtoOptions[i] = opt
	}

	return &dto.PollResponse{
		ID:              p.ID,
		Options:         dtoOptions,
		TotalVotes:      totalVotes,
		UserVotedOption: votedOption,
		Expired:         expired,
		ExpiresAt:       p.ExpiresAt,
		DurationSeconds: p.DurationSeconds,
	}
}
