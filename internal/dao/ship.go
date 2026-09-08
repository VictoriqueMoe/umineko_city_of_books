package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ShipDAO interface {
		Create(ctx context.Context, s spec.NewShip, tx ...*sql.Tx) (*model.ShipRow, error)
		UpdateDetails(ctx context.Context, s spec.ShipDetailsUpdate, tx ...*sql.Tx) error
		UpdateImage(ctx context.Context, s spec.ShipImageUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, q spec.ShipLookup, tx ...*sql.Tx) (*model.ShipRow, error)
		GetAuthorID(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetImagePaths(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		List(ctx context.Context, q spec.ShipListing, tx ...*sql.Tx) ([]model.ShipRow, int, error)
		ListByUser(ctx context.Context, q spec.ShipUserListing, tx ...*sql.Tx) ([]model.ShipRow, int, error)

		InsertCharacters(ctx context.Context, s spec.NewShipCharacters, tx ...*sql.Tx) error
		DeleteCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) error
		GetCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]model.ShipCharacterRow, error)
		GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.ShipCharacterRow, error)

		Vote(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
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

	shipDAO struct {
		db *sql.DB
		*ownedDAO
		*voteDAO
		*commentDAO[uuid.UUID]
	}

	shipJoinRow = sqlcgen.GetShipByIDRow

	shipListRow interface {
		sqlcgen.ListShipsByNewRow | sqlcgen.ListShipsByTopRow | sqlcgen.ListShipsByCrackshipRow |
			sqlcgen.ListShipsByControversialRow | sqlcgen.ListShipsByCommentsRow | sqlcgen.ListShipsByOldRow
	}
)

func toShipRow(row shipJoinRow) model.ShipRow {
	return model.ShipRow{
		ID:                row.ID,
		UserID:            row.UserID,
		Title:             row.Title,
		Description:       row.Description,
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
		CommentCount:      int(row.CommentCount),
	}
}

func toShipRows[T shipListRow](rows []T) []model.ShipRow {
	var ships []model.ShipRow
	for _, row := range rows {
		ships = append(ships, toShipRow(shipJoinRow(row)))
	}

	return ships
}

func toShipCharacterRow(row sqlcgen.ShipCharacter) model.ShipCharacterRow {
	return model.ShipCharacterRow{
		ID:            int(row.ID),
		ShipID:        row.ShipID,
		Series:        row.Series,
		CharacterID:   row.CharacterID,
		CharacterName: row.CharacterName,
		SortOrder:     int(row.SortOrder),
	}
}

func (r *shipDAO) Create(ctx context.Context, s spec.NewShip, tx ...*sql.Tx) (*model.ShipRow, error) {
	created, err := genQueries(r.db, tx).CreateShip(ctx, sqlcgen.CreateShipParams{
		UserID:      s.UserID,
		Title:       s.Title,
		Description: s.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create ship: %w", err)
	}

	return new(toShipRow(shipJoinRow(created))), nil
}

func (r *shipDAO) UpdateDetails(ctx context.Context, s spec.ShipDetailsUpdate, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	var (
		affected int64
		err      error
	)

	if s.AsAdmin {
		affected, err = queries.UpdateShipDetailsAsAdmin(ctx, sqlcgen.UpdateShipDetailsAsAdminParams{
			Title:       s.Title,
			Description: s.Description,
			ID:          s.ID,
		})
	} else {
		affected, err = queries.UpdateShipDetails(ctx, sqlcgen.UpdateShipDetailsParams{
			Title:       s.Title,
			Description: s.Description,
			ID:          s.ID,
			UserID:      s.UserID,
		})
	}
	if err != nil {
		return fmt.Errorf("update ship: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("ship not found or not owned")
	}

	return nil
}

func (r *shipDAO) UpdateImage(ctx context.Context, s spec.ShipImageUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateShipImage(ctx, sqlcgen.UpdateShipImageParams{
		ImageUrl:     s.ImageURL,
		ThumbnailUrl: s.ThumbnailURL,
		ID:           s.ID,
	})
	if err != nil {
		return fmt.Errorf("update ship image: %w", err)
	}

	return nil
}

func appendShipPaths(paths []string, values ...string) []string {
	for _, value := range values {
		if value == "" {
			continue
		}

		paths = append(paths, value)
	}

	return paths
}

func (r *shipDAO) GetImagePaths(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	row, err := genQueries(r.db, tx).GetShipImagePaths(ctx, shipID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get ship image paths: %w", err)
	}

	return appendShipPaths(nil, row.ImageUrl, row.ThumbnailUrl), nil
}

func (r *shipDAO) GetByID(ctx context.Context, q spec.ShipLookup, tx ...*sql.Tx) (*model.ShipRow, error) {
	row, err := genQueries(r.db, tx).GetShipByID(ctx, sqlcgen.GetShipByIDParams{
		UserID: q.ViewerID,
		ID:     q.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get ship: %w", err)
	}

	return new(toShipRow(row)), nil
}

func (r *shipDAO) List(ctx context.Context, q spec.ShipListing, tx ...*sql.Tx) ([]model.ShipRow, int, error) {
	queries := genQueries(r.db, tx)

	excluded := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountShips(ctx, sqlcgen.CountShipsParams{
		Column1: q.Series,
		Column2: q.CharacterID,
		Column3: q.CrackshipsOnly,
		Column4: dto.CrackshipThreshold,
		Column5: excluded,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count ships: %w", err)
	}

	params := sqlcgen.ListShipsByNewParams{
		UserID:  q.ViewerID,
		Column2: q.Series,
		Column3: q.CharacterID,
		Column4: q.CrackshipsOnly,
		Column5: dto.CrackshipThreshold,
		Column6: excluded,
		Limit:   int32(q.Limit),
		Offset:  int32(q.Offset),
	}

	var ships []model.ShipRow

	switch q.Sort {
	case "top":
		rows, err := queries.ListShipsByTop(ctx, sqlcgen.ListShipsByTopParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list ships: %w", err)
		}

		ships = toShipRows(rows)
	case "crackship":
		rows, err := queries.ListShipsByCrackship(ctx, sqlcgen.ListShipsByCrackshipParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list ships: %w", err)
		}

		ships = toShipRows(rows)
	case "controversial":
		rows, err := queries.ListShipsByControversial(ctx, sqlcgen.ListShipsByControversialParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list ships: %w", err)
		}

		ships = toShipRows(rows)
	case "comments":
		rows, err := queries.ListShipsByComments(ctx, sqlcgen.ListShipsByCommentsParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list ships: %w", err)
		}

		ships = toShipRows(rows)
	case "old":
		rows, err := queries.ListShipsByOld(ctx, sqlcgen.ListShipsByOldParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list ships: %w", err)
		}

		ships = toShipRows(rows)
	default:
		rows, err := queries.ListShipsByNew(ctx, params)
		if err != nil {
			return nil, 0, fmt.Errorf("list ships: %w", err)
		}

		ships = toShipRows(rows)
	}

	return ships, int(total), nil
}

func (r *shipDAO) ListByUser(ctx context.Context, q spec.ShipUserListing, tx ...*sql.Tx) ([]model.ShipRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountShipsByUser(ctx, q.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count user ships: %w", err)
	}

	rows, err := queries.ListShipsByUser(ctx, sqlcgen.ListShipsByUserParams{
		UserID:   q.ViewerID,
		UserID_2: q.UserID,
		Limit:    int32(q.Limit),
		Offset:   int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user ships: %w", err)
	}

	var ships []model.ShipRow
	for _, row := range rows {
		ships = append(ships, toShipRow(shipJoinRow(row)))
	}

	return ships, int(total), nil
}

func (r *shipDAO) InsertCharacters(ctx context.Context, s spec.NewShipCharacters, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	for i, c := range s.Characters {
		err := queries.InsertShipCharacter(ctx, sqlcgen.InsertShipCharacterParams{
			ShipID:        s.ShipID,
			Series:        c.Series,
			CharacterID:   c.CharacterID,
			CharacterName: strings.TrimSpace(c.CharacterName),
			SortOrder:     int32(i),
		})
		if err != nil {
			return fmt.Errorf("add ship character: %w", err)
		}
	}

	return nil
}

func (r *shipDAO) DeleteCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteShipCharacters(ctx, shipID); err != nil {
		return fmt.Errorf("delete ship characters: %w", err)
	}

	return nil
}

func (r *shipDAO) GetCharacters(ctx context.Context, shipID uuid.UUID, tx ...*sql.Tx) ([]model.ShipCharacterRow, error) {
	rows, err := genQueries(r.db, tx).GetShipCharacters(ctx, shipID)
	if err != nil {
		return nil, fmt.Errorf("get ship characters: %w", err)
	}

	var chars []model.ShipCharacterRow
	for _, row := range rows {
		chars = append(chars, toShipCharacterRow(row))
	}

	return chars, nil
}

func (r *shipDAO) GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.ShipCharacterRow, error) {
	if len(shipIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetShipCharactersBatch(ctx, joinUUIDs(shipIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get ship characters: %w", err)
	}

	result := make(map[uuid.UUID][]model.ShipCharacterRow)
	for _, row := range rows {
		c := toShipCharacterRow(row)
		result[c.ShipID] = append(result[c.ShipID], c)
	}

	return result, nil
}
