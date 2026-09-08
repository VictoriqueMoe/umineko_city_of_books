package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	ArtDAO interface {
		CreateArt(ctx context.Context, s spec.NewArt, tx ...*sql.Tx) (*model.ArtRow, error)
		UpdateArt(ctx context.Context, s spec.ArtUpdate, tx ...*sql.Tx) error
		GetByID(ctx context.Context, s spec.ArtLookup, tx ...*sql.Tx) (*model.ArtRow, error)
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		ListAll(ctx context.Context, q spec.ArtFilter, tx ...*sql.Tx) ([]model.ArtRow, int, error)
		ListByUser(ctx context.Context, q spec.ArtUserFilter, tx ...*sql.Tx) ([]model.ArtRow, int, error)
		GetArtAuthorID(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetImageURL(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetArtImagePaths(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		ListGalleryArtImages(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) ([]model.ArtImageRef, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		Like(ctx context.Context, s spec.Like, tx ...*sql.Tx) error
		Unlike(ctx context.Context, s spec.Like, tx ...*sql.Tx) error
		GetLikedBy(ctx context.Context, q spec.LikedByQuery, tx ...*sql.Tx) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error)
		IncrementViewCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error

		InsertTags(ctx context.Context, s spec.ArtTagInsert, tx ...*sql.Tx) error
		DeleteTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) error
		GetTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetTagsBatch(ctx context.Context, artIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)
		GetPopularTags(ctx context.Context, q spec.PopularTagFilter, tx ...*sql.Tx) ([]model.TagCount, error)

		GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error)
		CountUserArtToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error

		SetGallery(ctx context.Context, s spec.ArtGalleryAssignment, tx ...*sql.Tx) error

		CreateGallery(ctx context.Context, s spec.NewGallery, tx ...*sql.Tx) (*model.GalleryRow, error)
		UpdateGallery(ctx context.Context, s spec.GalleryUpdate, tx ...*sql.Tx) error
		SetGalleryCover(ctx context.Context, s spec.GalleryCoverUpdate, tx ...*sql.Tx) error
		DeleteArtInGallery(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) error
		DeleteGalleryRow(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) error
		GetGalleryByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GalleryRow, error)
		ListGalleriesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.GalleryRow, error)
		ListAllGalleries(ctx context.Context, corner string, tx ...*sql.Tx) ([]model.GalleryRow, error)
		GetGalleryPreviewImages(ctx context.Context, q spec.GalleryPreviewFilter, tx ...*sql.Tx) ([]model.PreviewImage, error)
		ListArtInGallery(ctx context.Context, q spec.GalleryArtFilter, tx ...*sql.Tx) ([]model.ArtRow, int, error)
	}

	artDAO struct {
		db *sql.DB
		*ownedDAO
		*commentDAO[uuid.UUID]
		*likeDAO
		*viewDAO
	}

	artJoinRow = sqlcgen.GetArtByIDRow

	galleryJoinRow = sqlcgen.GetGalleryByIDRow
)

var (
	ErrArtNotOwned = errors.New("art or gallery not found or not owned")
)

func toArtRow(row artJoinRow) model.ArtRow {
	return model.ArtRow{
		ID:                row.ID,
		UserID:            row.UserID,
		Corner:            row.Corner,
		ArtType:           row.ArtType,
		Title:             row.Title,
		Description:       row.Description,
		ImageURL:          row.ImageUrl,
		ThumbnailURL:      row.ThumbnailUrl,
		GalleryID:         row.GalleryID,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         nullTimeToStringPtr(row.UpdatedAt),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
		AuthorRole:        row.AuthorRole,
		LikeCount:         int(row.LikeCount),
		CommentCount:      int(row.CommentCount),
		ViewCount:         int(row.ViewCount),
		UserLiked:         row.UserLiked,
		IsSpoiler:         row.IsSpoiler,
	}
}

func toGalleryRow(row galleryJoinRow) model.GalleryRow {
	return model.GalleryRow{
		ID:                row.ID,
		UserID:            row.UserID,
		Name:              row.Name,
		Description:       row.Description,
		CoverArtID:        row.CoverArtID,
		CoverImageURL:     row.CoverImageUrl,
		CoverThumbnailURL: row.CoverThumbnailUrl,
		ArtCount:          int(row.ArtCount),
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         nullTimeToStringPtr(row.UpdatedAt),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
	}
}

func (r *artDAO) CreateArt(ctx context.Context, s spec.NewArt, tx ...*sql.Tx) (*model.ArtRow, error) {
	created, err := genQueries(r.db, tx).CreateArt(ctx, sqlcgen.CreateArtParams{
		UserID:       s.UserID,
		Corner:       s.Corner,
		ArtType:      s.ArtType,
		Title:        s.Title,
		Description:  s.Description,
		ImageUrl:     s.ImageURL,
		ThumbnailUrl: s.ThumbnailURL,
		IsSpoiler:    s.IsSpoiler,
	})
	if err != nil {
		return nil, fmt.Errorf("create art: %w", err)
	}

	return new(toArtRow(artJoinRow(created))), nil
}

func (r *artDAO) UpdateArt(ctx context.Context, s spec.ArtUpdate, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	var (
		affected int64
		err      error
	)

	if s.AsAdmin {
		affected, err = queries.UpdateArtAsAdmin(ctx, sqlcgen.UpdateArtAsAdminParams{
			Title:       s.Title,
			Description: s.Description,
			IsSpoiler:   s.IsSpoiler,
			ID:          s.ID,
		})
	} else {
		affected, err = queries.UpdateArt(ctx, sqlcgen.UpdateArtParams{
			Title:       s.Title,
			Description: s.Description,
			IsSpoiler:   s.IsSpoiler,
			ID:          s.ID,
			UserID:      s.UserID,
		})
	}
	if err != nil {
		return fmt.Errorf("update art: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("art not found or not owned")
	}

	return nil
}

func (r *artDAO) InsertTags(ctx context.Context, s spec.ArtTagInsert, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	for _, tag := range s.Tags {
		normalised := strings.TrimSpace(strings.ToLower(tag))
		if normalised == "" {
			continue
		}

		err := queries.InsertArtTag(ctx, sqlcgen.InsertArtTagParams{
			ArtID: s.ArtID,
			Tag:   normalised,
		})
		if err != nil {
			return fmt.Errorf("add art tag: %w", err)
		}
	}

	return nil
}

func (r *artDAO) DeleteTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteArtTags(ctx, artID); err != nil {
		return fmt.Errorf("delete art tags: %w", err)
	}

	return nil
}

func (r *artDAO) GetByID(ctx context.Context, s spec.ArtLookup, tx ...*sql.Tx) (*model.ArtRow, error) {
	row, err := genQueries(r.db, tx).GetArtByID(ctx, sqlcgen.GetArtByIDParams{
		UserID: s.ViewerID,
		ID:     s.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get art: %w", err)
	}

	return new(toArtRow(row)), nil
}

func (r *artDAO) ListAll(ctx context.Context, q spec.ArtFilter, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	queries := genQueries(r.db, tx)

	search := ""
	if q.Search != "" {
		search = "%" + q.Search + "%"
	}

	excluded := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountArtFiltered(ctx, sqlcgen.CountArtFilteredParams{
		Corner:         q.Corner,
		ArtType:        q.ArtType,
		Search:         search,
		Tag:            q.Tag,
		ExcludeUserIds: excluded,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count art: %w", err)
	}

	params := sqlcgen.ListArtByNewParams{
		ViewerID:       q.ViewerID,
		Corner:         q.Corner,
		ArtType:        q.ArtType,
		Search:         search,
		Tag:            q.Tag,
		ExcludeUserIds: excluded,
		RowOffset:      int32(q.Offset),
		RowLimit:       int32(q.Limit),
	}

	var arts []model.ArtRow

	switch q.Sort {
	case "popular":
		rows, err := queries.ListArtByPopular(ctx, sqlcgen.ListArtByPopularParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list art: %w", err)
		}

		for _, row := range rows {
			arts = append(arts, toArtRow(artJoinRow(row)))
		}
	case "views":
		rows, err := queries.ListArtByViews(ctx, sqlcgen.ListArtByViewsParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list art: %w", err)
		}

		for _, row := range rows {
			arts = append(arts, toArtRow(artJoinRow(row)))
		}
	default:
		rows, err := queries.ListArtByNew(ctx, params)
		if err != nil {
			return nil, 0, fmt.Errorf("list art: %w", err)
		}

		for _, row := range rows {
			arts = append(arts, toArtRow(artJoinRow(row)))
		}
	}

	return arts, int(total), nil
}

func (r *artDAO) ListByUser(ctx context.Context, q spec.ArtUserFilter, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountArtByUser(ctx, q.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count user art: %w", err)
	}

	rows, err := queries.ListArtByUser(ctx, sqlcgen.ListArtByUserParams{
		ViewerID:  q.ViewerID,
		UserID:    q.UserID,
		RowLimit:  int32(q.Limit),
		RowOffset: int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user art: %w", err)
	}

	var arts []model.ArtRow
	for _, row := range rows {
		arts = append(arts, toArtRow(artJoinRow(row)))
	}

	return arts, int(total), nil
}

func (r *artDAO) GetArtAuthorID(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.ownedDAO.GetAuthorID(ctx, artID, tx...)
}

func (r *artDAO) GetImageURL(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (string, error) {
	url, err := genQueries(r.db, tx).GetArtImageURL(ctx, artID)
	if err != nil {
		return "", fmt.Errorf("get art image url: %w", err)
	}

	return url, nil
}

func (r *artDAO) GetArtImagePaths(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	row, err := genQueries(r.db, tx).GetArtImagePaths(ctx, artID)
	if err != nil {
		return nil, fmt.Errorf("get art image paths: %w", err)
	}

	var paths []string

	if row.ImageUrl != "" {
		paths = append(paths, row.ImageUrl)
	}

	if row.ThumbnailUrl != "" {
		paths = append(paths, row.ThumbnailUrl)
	}

	return paths, nil
}

func (r *artDAO) ListGalleryArtImages(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) ([]model.ArtImageRef, error) {
	rows, err := genQueries(r.db, tx).ListGalleryArtImages(ctx, sqlcgen.ListGalleryArtImagesParams{
		GalleryID: s.GalleryID,
		UserID:    s.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("list gallery art images: %w", err)
	}

	var refs []model.ArtImageRef
	for _, row := range rows {
		refs = append(refs, model.ArtImageRef{
			ArtID:        row.ID,
			ImageURL:     row.ImageUrl,
			ThumbnailURL: row.ThumbnailUrl,
		})
	}

	return refs, nil
}

func (r *artDAO) GetTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	tags, err := genQueries(r.db, tx).GetArtTags(ctx, artID)
	if err != nil {
		return nil, fmt.Errorf("get art tags: %w", err)
	}

	return tags, nil
}

func (r *artDAO) GetTagsBatch(ctx context.Context, artIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(artIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetArtTagsBatch(ctx, joinUUIDs(artIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get art tags: %w", err)
	}

	groups := make(map[uuid.UUID][]string)
	for _, row := range rows {
		groups[row.ArtID] = append(groups[row.ArtID], row.Tag)
	}

	return groups, nil
}

func (r *artDAO) GetPopularTags(ctx context.Context, q spec.PopularTagFilter, tx ...*sql.Tx) ([]model.TagCount, error) {
	rows, err := genQueries(r.db, tx).GetPopularArtTags(ctx, sqlcgen.GetPopularArtTagsParams{
		Column1: q.Corner,
		Limit:   int32(q.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get popular tags: %w", err)
	}

	var tags []model.TagCount
	for _, row := range rows {
		tags = append(tags, model.TagCount{
			Tag:   row.Tag,
			Count: int(row.Cnt),
		})
	}

	return tags, nil
}

func (r *artDAO) GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error) {
	rows, err := genQueries(r.db, tx).GetArtCornerCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("art corner counts: %w", err)
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Corner] = int(row.Count)
	}

	return counts, nil
}

func (r *artDAO) CountUserArtToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUserArtToday(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count user art today: %w", err)
	}

	return int(count), nil
}

func (r *artDAO) SetGallery(ctx context.Context, s spec.ArtGalleryAssignment, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).SetArtGallery(ctx, sqlcgen.SetArtGalleryParams{
		GalleryID: s.GalleryID,
		ArtID:     s.ArtID,
		UserID:    s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set art gallery: %w", err)
	}

	if affected == 0 {
		return ErrArtNotOwned
	}

	return nil
}

func (r *artDAO) CreateGallery(ctx context.Context, s spec.NewGallery, tx ...*sql.Tx) (*model.GalleryRow, error) {
	created, err := genQueries(r.db, tx).CreateGallery(ctx, sqlcgen.CreateGalleryParams{
		UserID:      s.UserID,
		Name:        s.Name,
		Description: s.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create gallery: %w", err)
	}

	return new(toGalleryRow(galleryJoinRow(created))), nil
}

func (r *artDAO) UpdateGallery(ctx context.Context, s spec.GalleryUpdate, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).UpdateGallery(ctx, sqlcgen.UpdateGalleryParams{
		Name:        s.Name,
		Description: s.Description,
		ID:          s.ID,
		UserID:      s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update gallery: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}

	return nil
}

func (r *artDAO) SetGalleryCover(ctx context.Context, s spec.GalleryCoverUpdate, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).SetGalleryCover(ctx, sqlcgen.SetGalleryCoverParams{
		CoverArtID: s.CoverArtID,
		GalleryID:  s.GalleryID,
		UserID:     s.UserID,
	})
	if err != nil {
		return fmt.Errorf("set gallery cover: %w", err)
	}

	if affected == 0 {
		return ErrArtNotOwned
	}

	return nil
}

func (r *artDAO) DeleteArtInGallery(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).DeleteArtInGallery(ctx, sqlcgen.DeleteArtInGalleryParams{
		GalleryID: s.GalleryID,
		UserID:    s.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete art in gallery: %w", err)
	}

	return nil
}

func (r *artDAO) DeleteGalleryRow(ctx context.Context, s spec.GalleryRef, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteGalleryRow(ctx, sqlcgen.DeleteGalleryRowParams{
		ID:     s.GalleryID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete gallery: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}

	return nil
}

func (r *artDAO) GetGalleryByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GalleryRow, error) {
	row, err := genQueries(r.db, tx).GetGalleryByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get gallery: %w", err)
	}

	return new(toGalleryRow(row)), nil
}

func (r *artDAO) ListGalleriesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.GalleryRow, error) {
	rows, err := genQueries(r.db, tx).ListGalleriesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list galleries: %w", err)
	}

	var galleries []model.GalleryRow
	for _, row := range rows {
		galleries = append(galleries, toGalleryRow(galleryJoinRow(row)))
	}

	return galleries, nil
}

func (r *artDAO) ListAllGalleries(ctx context.Context, corner string, tx ...*sql.Tx) ([]model.GalleryRow, error) {
	rows, err := genQueries(r.db, tx).ListAllGalleries(ctx, corner)
	if err != nil {
		return nil, fmt.Errorf("list all galleries: %w", err)
	}

	var galleries []model.GalleryRow
	for _, row := range rows {
		galleries = append(galleries, toGalleryRow(galleryJoinRow(row)))
	}

	return galleries, nil
}

func (r *artDAO) GetGalleryPreviewImages(ctx context.Context, q spec.GalleryPreviewFilter, tx ...*sql.Tx) ([]model.PreviewImage, error) {
	rows, err := genQueries(r.db, tx).GetGalleryPreviewImages(ctx, sqlcgen.GetGalleryPreviewImagesParams{
		GalleryID: q.GalleryID,
		RowLimit:  int32(q.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get gallery preview images: %w", err)
	}

	var imgs []model.PreviewImage
	for _, row := range rows {
		imgs = append(imgs, model.PreviewImage{
			ThumbnailURL: row.ThumbnailUrl,
			ImageURL:     row.ImageUrl,
		})
	}

	return imgs, nil
}

func (r *artDAO) ListArtInGallery(ctx context.Context, q spec.GalleryArtFilter, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountArtInGallery(ctx, q.GalleryID)
	if err != nil {
		return nil, 0, fmt.Errorf("count gallery art: %w", err)
	}

	rows, err := queries.ListArtInGallery(ctx, sqlcgen.ListArtInGalleryParams{
		ViewerID:  q.ViewerID,
		GalleryID: q.GalleryID,
		RowLimit:  int32(q.Limit),
		RowOffset: int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list gallery art: %w", err)
	}

	var arts []model.ArtRow
	for _, row := range rows {
		arts = append(arts, toArtRow(artJoinRow(row)))
	}

	return arts, int(total), nil
}
