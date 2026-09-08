package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	OCDAO interface {
		Create(ctx context.Context, s spec.NewOC, tx ...*sql.Tx) (*model.OCRow, error)
		Update(ctx context.Context, s spec.OCUpdate, tx ...*sql.Tx) error
		UpdateImage(ctx context.Context, s spec.OCImageUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, s spec.OCByID, tx ...*sql.Tx) (*model.OCRow, error)
		GetAuthorID(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetImagePaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetGalleryPaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		List(ctx context.Context, s spec.OCListFilter, tx ...*sql.Tx) ([]model.OCRow, int, error)
		ListByUser(ctx context.Context, s spec.OCUserListFilter, tx ...*sql.Tx) ([]model.OCRow, int, error)
		ListSummariesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.OCSummaryRow, error)
		HasOC(ctx context.Context, s spec.OCNameLookup, tx ...*sql.Tx) (bool, error)

		AddGalleryImage(ctx context.Context, s spec.NewOCGalleryImage, tx ...*sql.Tx) (int64, error)
		UpdateGalleryImageURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateGalleryImageThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateGalleryImage(ctx context.Context, s spec.OCGalleryImageUpdate, tx ...*sql.Tx) error
		DeleteGalleryImage(ctx context.Context, s spec.MediaDeletion, tx ...*sql.Tx) error
		GetGallery(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]model.OCImageRow, error)
		GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.OCImageRow, error)

		Vote(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error
		Favourite(ctx context.Context, s spec.Like, tx ...*sql.Tx) error
		Unfavourite(ctx context.Context, s spec.Like, tx ...*sql.Tx) error

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, s spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error

		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
	}

	ocDAO struct {
		db *sql.DB
		*ownedDAO
		*voteDAO
		*commentDAO[uuid.UUID]
	}

	ocJoinRow = sqlcgen.GetOCByIDRow
)

func toOCRow(row ocJoinRow) model.OCRow {
	return model.OCRow{
		ID:                row.ID,
		UserID:            row.UserID,
		Name:              row.Name,
		Description:       row.Description,
		Series:            row.Series,
		CustomSeriesName:  row.CustomSeriesName,
		ImageURL:          row.ImageUrl,
		ThumbnailURL:      row.ThumbnailUrl,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         new(row.UpdatedAt.UTC().Format(time.RFC3339)),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
		AuthorRole:        row.AuthorRole,
		VoteScore:         int(row.VoteScore),
		UserVote:          int(row.UserVote),
		FavouriteCount:    int(row.FavouriteCount),
		UserFavourited:    row.UserFavourited,
		CommentCount:      int(row.CommentCount),
	}
}

func toOCImageRow(row sqlcgen.OcImage) model.OCImageRow {
	return model.OCImageRow{
		ID:           row.ID,
		OCID:         row.OcID,
		ImageURL:     row.ImageUrl,
		ThumbnailURL: row.ThumbnailUrl,
		Caption:      row.Caption,
		SortOrder:    int(row.SortOrder),
	}
}

func (r *ocDAO) Create(ctx context.Context, s spec.NewOC, tx ...*sql.Tx) (*model.OCRow, error) {
	created, err := genQueries(r.db, tx).CreateOC(ctx, sqlcgen.CreateOCParams{
		UserID:           s.UserID,
		Name:             s.Name,
		Description:      s.Description,
		Series:           s.Series,
		CustomSeriesName: s.CustomSeriesName,
	})
	if err != nil {
		return nil, fmt.Errorf("create oc: %w", err)
	}

	return new(toOCRow(ocJoinRow(created))), nil
}

func (r *ocDAO) Update(ctx context.Context, s spec.OCUpdate, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	var (
		affected int64
		err      error
	)

	if s.AsAdmin {
		affected, err = queries.UpdateOCAsAdmin(ctx, sqlcgen.UpdateOCAsAdminParams{
			Name:             s.Name,
			Description:      s.Description,
			Series:           s.Series,
			CustomSeriesName: s.CustomSeriesName,
			ID:               s.ID,
		})
	} else {
		affected, err = queries.UpdateOC(ctx, sqlcgen.UpdateOCParams{
			Name:             s.Name,
			Description:      s.Description,
			Series:           s.Series,
			CustomSeriesName: s.CustomSeriesName,
			ID:               s.ID,
			UserID:           s.UserID,
		})
	}
	if err != nil {
		return fmt.Errorf("update oc: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("oc not found or not owned")
	}

	return nil
}

func (r *ocDAO) UpdateImage(ctx context.Context, s spec.OCImageUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateOCImage(ctx, sqlcgen.UpdateOCImageParams{
		ImageUrl:     s.ImageURL,
		ThumbnailUrl: s.ThumbnailURL,
		ID:           s.ID,
	})
	if err != nil {
		return fmt.Errorf("update oc image: %w", err)
	}

	return nil
}

func appendOCPaths(paths []string, values ...string) []string {
	for _, value := range values {
		if value == "" {
			continue
		}

		paths = append(paths, value)
	}

	return paths
}

func (r *ocDAO) GetImagePaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	row, err := genQueries(r.db, tx).GetOCImagePaths(ctx, ocID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get oc image paths: %w", err)
	}

	return appendOCPaths(nil, row.ImageUrl, row.ThumbnailUrl), nil
}

func (r *ocDAO) GetGalleryPaths(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).GetOCGalleryPaths(ctx, ocID)
	if err != nil {
		return nil, fmt.Errorf("get oc gallery paths: %w", err)
	}

	var paths []string
	for _, row := range rows {
		paths = appendOCPaths(paths, row.ImageUrl, row.ThumbnailUrl)
	}

	return paths, nil
}

func (r *ocDAO) GetByID(ctx context.Context, s spec.OCByID, tx ...*sql.Tx) (*model.OCRow, error) {
	row, err := genQueries(r.db, tx).GetOCByID(ctx, sqlcgen.GetOCByIDParams{
		UserID: s.ViewerID,
		ID:     s.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get oc: %w", err)
	}

	return new(toOCRow(row)), nil
}

func (r *ocDAO) HasOC(ctx context.Context, s spec.OCNameLookup, tx ...*sql.Tx) (bool, error) {
	exists, err := genQueries(r.db, tx).OCNameExists(ctx, sqlcgen.OCNameExistsParams{
		UserID:  s.UserID,
		Column2: s.Name,
	})
	if err != nil {
		return false, fmt.Errorf("check oc exists: %w", err)
	}

	return exists, nil
}

func (r *ocDAO) List(ctx context.Context, s spec.OCListFilter, tx ...*sql.Tx) ([]model.OCRow, int, error) {
	queries := genQueries(r.db, tx)
	excluded := joinUUIDs(s.ExcludeUserIDs)

	total, err := queries.CountOCs(ctx, sqlcgen.CountOCsParams{
		Series:           s.Series,
		CustomSeriesName: s.CustomSeriesName,
		OwnerID:          s.OwnerID,
		CrackOnly:        s.CrackOCsOnly,
		ExcludeUserIds:   excluded,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count ocs: %w", err)
	}

	params := sqlcgen.ListOCsByNewParams{
		ViewerID:         s.ViewerID,
		Series:           s.Series,
		CustomSeriesName: s.CustomSeriesName,
		OwnerID:          s.OwnerID,
		CrackOnly:        s.CrackOCsOnly,
		ExcludeUserIds:   excluded,
		PageOffset:       int32(s.Offset),
		PageLimit:        int32(s.Limit),
	}

	var rows []ocJoinRow

	switch s.Sort {
	case "top":
		found, listErr := queries.ListOCsByTop(ctx, sqlcgen.ListOCsByTopParams(params))
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	case "crack":
		found, listErr := queries.ListOCsByCrack(ctx, sqlcgen.ListOCsByCrackParams(params))
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	case "favourites":
		found, listErr := queries.ListOCsByFavourites(ctx, sqlcgen.ListOCsByFavouritesParams(params))
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	case "comments":
		found, listErr := queries.ListOCsByComments(ctx, sqlcgen.ListOCsByCommentsParams(params))
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	case "name":
		found, listErr := queries.ListOCsByName(ctx, sqlcgen.ListOCsByNameParams(params))
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	case "old":
		found, listErr := queries.ListOCsByOld(ctx, sqlcgen.ListOCsByOldParams(params))
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	default:
		found, listErr := queries.ListOCsByNew(ctx, params)
		if listErr != nil {
			return nil, 0, fmt.Errorf("list ocs: %w", listErr)
		}

		for _, row := range found {
			rows = append(rows, ocJoinRow(row))
		}
	}

	var ocs []model.OCRow
	for _, row := range rows {
		ocs = append(ocs, toOCRow(row))
	}

	return ocs, int(total), nil
}

func (r *ocDAO) ListByUser(ctx context.Context, s spec.OCUserListFilter, tx ...*sql.Tx) ([]model.OCRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountOCsByUser(ctx, s.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count user ocs: %w", err)
	}

	rows, err := queries.ListOCsByUser(ctx, sqlcgen.ListOCsByUserParams{
		ViewerID:   s.ViewerID,
		OwnerID:    s.UserID,
		PageLimit:  int32(s.Limit),
		PageOffset: int32(s.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user ocs: %w", err)
	}

	var ocs []model.OCRow
	for _, row := range rows {
		ocs = append(ocs, toOCRow(ocJoinRow(row)))
	}

	return ocs, int(total), nil
}

func (r *ocDAO) ListSummariesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.OCSummaryRow, error) {
	rows, err := genQueries(r.db, tx).ListOCSummariesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list oc summaries: %w", err)
	}

	var summaries []model.OCSummaryRow
	for _, row := range rows {
		summaries = append(summaries, model.OCSummaryRow{
			ID:               row.ID,
			Name:             row.Name,
			Series:           row.Series,
			CustomSeriesName: row.CustomSeriesName,
			ThumbnailURL:     row.ThumbnailUrl,
		})
	}

	return summaries, nil
}

func (r *ocDAO) AddGalleryImage(ctx context.Context, s spec.NewOCGalleryImage, tx ...*sql.Tx) (int64, error) {
	id, err := genQueries(r.db, tx).AddOCGalleryImage(ctx, sqlcgen.AddOCGalleryImageParams{
		OcID:         s.OCID,
		ImageUrl:     s.ImageURL,
		ThumbnailUrl: s.ThumbnailURL,
		Caption:      s.Caption,
		SortOrder:    int32(s.SortOrder),
	})
	if err != nil {
		return 0, fmt.Errorf("add oc gallery image: %w", err)
	}

	return id, nil
}

func (r *ocDAO) UpdateGalleryImageURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateOCGalleryImageURL(ctx, sqlcgen.UpdateOCGalleryImageURLParams{
		ImageUrl: s.URL,
		ID:       s.ID,
	})
	if err != nil {
		return fmt.Errorf("update oc gallery image url: %w", err)
	}

	return nil
}

func (r *ocDAO) UpdateGalleryImageThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateOCGalleryImageThumbnail(ctx, sqlcgen.UpdateOCGalleryImageThumbnailParams{
		ThumbnailUrl: s.URL,
		ID:           s.ID,
	})
	if err != nil {
		return fmt.Errorf("update oc gallery image thumbnail: %w", err)
	}

	return nil
}

func (r *ocDAO) UpdateGalleryImage(ctx context.Context, s spec.OCGalleryImageUpdate, tx ...*sql.Tx) error {
	if s.Caption == nil && s.SortOrder == nil {
		return nil
	}

	queries := genQueries(r.db, tx)

	var (
		affected int64
		err      error
	)

	switch {
	case s.Caption != nil && s.SortOrder != nil:
		affected, err = queries.UpdateOCGalleryImageDetails(ctx, sqlcgen.UpdateOCGalleryImageDetailsParams{
			Caption:   *s.Caption,
			SortOrder: int32(*s.SortOrder),
			ID:        s.ID,
			OcID:      s.OCID,
		})
	case s.Caption != nil:
		affected, err = queries.UpdateOCGalleryImageCaption(ctx, sqlcgen.UpdateOCGalleryImageCaptionParams{
			Caption: *s.Caption,
			ID:      s.ID,
			OcID:    s.OCID,
		})
	default:
		affected, err = queries.UpdateOCGalleryImageSortOrder(ctx, sqlcgen.UpdateOCGalleryImageSortOrderParams{
			SortOrder: int32(*s.SortOrder),
			ID:        s.ID,
			OcID:      s.OCID,
		})
	}
	if err != nil {
		return fmt.Errorf("update oc gallery image: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("gallery image not found or not in oc")
	}

	return nil
}

func (r *ocDAO) DeleteGalleryImage(ctx context.Context, s spec.MediaDeletion, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteOCGalleryImage(ctx, sqlcgen.DeleteOCGalleryImageParams{
		ID:   s.ID,
		OcID: s.TargetID,
	})
	if err != nil {
		return fmt.Errorf("delete oc gallery image: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("gallery image not found or not in oc")
	}

	return nil
}

func (r *ocDAO) GetGallery(ctx context.Context, ocID uuid.UUID, tx ...*sql.Tx) ([]model.OCImageRow, error) {
	rows, err := genQueries(r.db, tx).GetOCGallery(ctx, ocID)
	if err != nil {
		return nil, fmt.Errorf("get oc gallery: %w", err)
	}

	var images []model.OCImageRow
	for _, row := range rows {
		images = append(images, toOCImageRow(row))
	}

	return images, nil
}

func (r *ocDAO) GetGalleryBatch(ctx context.Context, ocIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.OCImageRow, error) {
	if len(ocIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetOCGalleryBatch(ctx, joinUUIDs(ocIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get oc gallery: %w", err)
	}

	result := make(map[uuid.UUID][]model.OCImageRow)
	for _, row := range rows {
		image := toOCImageRow(row)
		result[image.OCID] = append(result[image.OCID], image)
	}

	return result, nil
}

func (r *ocDAO) Favourite(ctx context.Context, s spec.Like, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).FavouriteOC(ctx, sqlcgen.FavouriteOCParams{
		UserID: s.UserID,
		OcID:   s.TargetID,
	})
	if err != nil {
		return fmt.Errorf("favourite oc: %w", err)
	}

	return nil
}

func (r *ocDAO) Unfavourite(ctx context.Context, s spec.Like, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UnfavouriteOC(ctx, sqlcgen.UnfavouriteOCParams{
		UserID: s.UserID,
		OcID:   s.TargetID,
	})
	if err != nil {
		return fmt.Errorf("unfavourite oc: %w", err)
	}

	return nil
}
