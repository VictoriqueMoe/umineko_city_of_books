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
		GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error)
	}
)

type shipRepository struct {
	dao ShipRepository
}

func NewShipRepo(dao ShipRepository) ShipRepository {
	return &shipRepository{dao: dao}
}

func (r *shipRepository) CreateWithCharacters(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter) error {
	return r.dao.CreateWithCharacters(ctx, id, userID, title, description, characters)
}

func (r *shipRepository) UpdateWithCharacters(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter, asAdmin bool) error {
	return r.dao.UpdateWithCharacters(ctx, id, userID, title, description, characters, asAdmin)
}

func (r *shipRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	return r.dao.UpdateImage(ctx, id, imageURL, thumbnailURL)
}

func (r *shipRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.Delete(ctx, id, userID)
}

func (r *shipRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteAsAdmin(ctx, id)
}

func (r *shipRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ShipRow, error) {
	return r.dao.GetByID(ctx, id, viewerID)
}

func (r *shipRepository) GetAuthorID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetAuthorID(ctx, shipID)
}

func (r *shipRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ShipRow, int, error) {
	return r.dao.List(ctx, viewerID, sort, crackshipsOnly, series, characterID, limit, offset, excludeUserIDs)
}

func (r *shipRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ShipRow, int, error) {
	return r.dao.ListByUser(ctx, userID, viewerID, limit, offset)
}

func (r *shipRepository) GetCharacters(ctx context.Context, shipID uuid.UUID) ([]model.ShipCharacterRow, error) {
	return r.dao.GetCharacters(ctx, shipID)
}

func (r *shipRepository) GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID) (map[uuid.UUID][]model.ShipCharacterRow, error) {
	return r.dao.GetCharactersBatch(ctx, shipIDs)
}

func (r *shipRepository) Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error {
	return r.dao.Vote(ctx, userID, shipID, value)
}

func (r *shipRepository) CreateComment(ctx context.Context, id uuid.UUID, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.CreateComment(ctx, id, shipID, parentID, userID, body)
}

func (r *shipRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	return r.dao.UpdateComment(ctx, id, userID, body)
}

func (r *shipRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	return r.dao.UpdateCommentAsAdmin(ctx, id, body)
}

func (r *shipRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.dao.DeleteComment(ctx, id, userID)
}

func (r *shipRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	return r.dao.DeleteCommentAsAdmin(ctx, id)
}

func (r *shipRepository) GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]CommentRow, int, error) {
	return r.dao.GetComments(ctx, shipID, viewerID, limit, offset, excludeUserIDs)
}

func (r *shipRepository) GetCommentEntityID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentEntityID(ctx, commentID)
}

func (r *shipRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	return r.dao.GetCommentAuthorID(ctx, commentID)
}

func (r *shipRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.LikeComment(ctx, userID, commentID)
}

func (r *shipRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return r.dao.UnlikeComment(ctx, userID, commentID)
}

func (r *shipRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	return r.dao.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbnailURL, sortOrder)
}

func (r *shipRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	return r.dao.UpdateCommentMediaURL(ctx, id, mediaURL)
}

func (r *shipRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	return r.dao.UpdateCommentMediaThumbnail(ctx, id, thumbnailURL)
}

func (r *shipRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.PostMediaRow, error) {
	return r.dao.GetCommentMedia(ctx, commentID)
}

func (r *shipRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.PostMediaRow, error) {
	return r.dao.GetCommentMediaBatch(ctx, commentIDs)
}
