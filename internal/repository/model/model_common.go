package model

import (
	"time"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	PostMediaRow struct {
		ID           int
		PostID       uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		Filename     string
		SortOrder    int
	}

	PostLikeUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
	}
)

func (m *PostMediaRow) ToResponse() dto.PostMediaResponse {
	return dto.PostMediaResponse{
		ID:           m.ID,
		MediaURL:     m.MediaURL,
		MediaType:    m.MediaType,
		ThumbnailURL: m.ThumbnailURL,
		Filename:     m.Filename,
		SortOrder:    m.SortOrder,
	}
}

func MediaRowsToResponse(rows []PostMediaRow) []dto.PostMediaResponse {
	list := make([]dto.PostMediaResponse, len(rows))
	for i := range rows {
		list[i] = rows[i].ToResponse()
	}
	return list
}

func ParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}
