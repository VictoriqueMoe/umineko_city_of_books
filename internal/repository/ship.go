package repository

import (
	"context"

	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	ShipRepository interface {
		CreateWithCharacters(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter) error
		UpdateWithCharacters(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter, asAdmin bool) error
		UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ShipRow, error)
		GetAuthorID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error)
		List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ShipRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ShipRow, int, error)

		GetCharacters(ctx context.Context, shipID uuid.UUID) ([]model.ShipCharacterRow, error)
		GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID) (map[uuid.UUID][]model.ShipCharacterRow, error)

		Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error

		CreateComment(ctx context.Context, id uuid.UUID, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ShipCommentRow, int, error)
		GetCommentShipID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.ShipCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.ShipCommentMediaRow, error)
	}
)
