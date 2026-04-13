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
		SortOrder    int
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

	PostLikeUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
	}

	CommentMediaRow struct {
		ID           int
		CommentID    uuid.UUID
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		SortOrder    int
	}
)

func (m *CommentMediaRow) ToResponse() dto.PostMediaResponse {
	return dto.PostMediaResponse{
		ID:           m.ID,
		MediaURL:     m.MediaURL,
		MediaType:    m.MediaType,
		ThumbnailURL: m.ThumbnailURL,
		SortOrder:    m.SortOrder,
	}
}

func CommentMediaRowsToResponse(rows []CommentMediaRow) []dto.PostMediaResponse {
	list := make([]dto.PostMediaResponse, len(rows))
	for i := range rows {
		list[i] = rows[i].ToResponse()
	}
	return list
}

func (m *PostMediaRow) ToResponse() dto.PostMediaResponse {
	return dto.PostMediaResponse{
		ID:           m.ID,
		MediaURL:     m.MediaURL,
		MediaType:    m.MediaType,
		ThumbnailURL: m.ThumbnailURL,
		SortOrder:    m.SortOrder,
	}
}

func (e *EmbedRow) ToResponse() dto.EmbedResponse {
	return dto.EmbedResponse{
		URL:      e.URL,
		Type:     e.EmbedType,
		Title:    e.Title,
		Desc:     e.Desc,
		Image:    e.Image,
		SiteName: e.SiteName,
		VideoID:  e.VideoID,
	}
}

func MediaRowsToResponse(rows []PostMediaRow) []dto.PostMediaResponse {
	list := make([]dto.PostMediaResponse, len(rows))
	for i := range rows {
		list[i] = rows[i].ToResponse()
	}
	return list
}

func EmbedRowsToResponse(rows []EmbedRow) []dto.EmbedResponse {
	if len(rows) == 0 {
		return nil
	}
	list := make([]dto.EmbedResponse, len(rows))
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
