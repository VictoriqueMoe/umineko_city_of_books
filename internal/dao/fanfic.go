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
	FanficDAO interface {
		Create(ctx context.Context, s spec.NewFanfic, tx ...*sql.Tx) (*model.FanficRow, error)
		Update(ctx context.Context, s spec.FanficUpdate, tx ...*sql.Tx) error
		UpdateCoverImage(ctx context.Context, s spec.FanficCoverUpdate, tx ...*sql.Tx) error
		UpdateWordCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, s spec.FanficLookup, tx ...*sql.Tx) (*model.FanficRow, error)
		GetAuthorID(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCoverImagePaths(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		List(ctx context.Context, q spec.FanficListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error)
		ListByUser(ctx context.Context, q spec.FanficUserListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error)

		CreateChapter(ctx context.Context, s spec.NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error)
		UpdateChapter(ctx context.Context, s spec.ChapterUpdate, tx ...*sql.Tx) error
		DeleteChapter(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetChapter(ctx context.Context, s spec.FanficChapterLookup, tx ...*sql.Tx) (*model.FanficChapterRow, error)
		ListChapters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficChapterSummaryRow, error)
		GetChapterCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetChapterFanficID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		GetGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)
		AddGenres(ctx context.Context, s spec.FanficGenres, tx ...*sql.Tx) error
		DeleteGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error
		GetTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error)
		AddTags(ctx context.Context, s spec.FanficTags, tx ...*sql.Tx) error
		DeleteTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error
		GetCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficCharacterRow, error)
		GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.FanficCharacterRow, error)
		AddCharacters(ctx context.Context, s spec.FanficCharacters, tx ...*sql.Tx) error
		DeleteCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error

		RegisterOCCharacter(ctx context.Context, s spec.NewFanficOCCharacter, tx ...*sql.Tx) error
		SearchOCCharacters(ctx context.Context, query string, tx ...*sql.Tx) ([]string, error)
		GetLanguages(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		RegisterLanguage(ctx context.Context, name string, tx ...*sql.Tx) error
		GetSeries(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		RegisterSeries(ctx context.Context, name string, tx ...*sql.Tx) error

		Favourite(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) error
		Unfavourite(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) error
		RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error)
		IncrementViewCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetReadingProgress(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) (int, error)
		SetReadingProgress(ctx context.Context, s spec.FanficReadingProgress, tx ...*sql.Tx) error
		ListFavourites(ctx context.Context, q spec.FanficUserListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error)

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
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	fanficDAO struct {
		db *sql.DB
		*ownedDAO
		*commentDAO[uuid.UUID]
		*viewDAO
	}

	fanficJoinRow = sqlcgen.GetFanficByIDRow
)

func toFanficRow(row fanficJoinRow) model.FanficRow {
	return model.FanficRow{
		ID:                row.ID,
		UserID:            row.UserID,
		Title:             row.Title,
		Summary:           row.Summary,
		Series:            row.Series,
		Rating:            row.Rating,
		Language:          row.Language,
		Status:            row.Status,
		IsOneshot:         row.IsOneshot,
		ContainsLemons:    row.ContainsLemons,
		CoverImageURL:     row.CoverImageUrl,
		CoverThumbnailURL: row.CoverThumbnailUrl,
		WordCount:         int(row.WordCount),
		ChapterCount:      int(row.ChapterCount),
		FavouriteCount:    int(row.FavouriteCount),
		ViewCount:         int(row.ViewCount),
		CommentCount:      int(row.CommentCount),
		UserFavourited:    row.UserFavourited,
		IsPairing:         row.IsPairing,
		PublishedAt:       row.PublishedAt.UTC().Format(time.RFC3339),
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         new(row.UpdatedAt.UTC().Format(time.RFC3339)),
		AuthorUsername:    row.AuthorUsername,
		AuthorDisplayName: row.AuthorDisplayName,
		AuthorAvatarURL:   row.AuthorAvatarUrl,
		AuthorRole:        row.AuthorRole,
	}
}

func toFanficChapterRow(row sqlcgen.FanficChapter) model.FanficChapterRow {
	return model.FanficChapterRow{
		ID:         row.ID,
		FanficID:   row.FanficID,
		ChapterNum: int(row.ChapterNumber),
		Title:      row.Title,
		Body:       row.Body,
		WordCount:  int(row.WordCount),
		CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  new(row.UpdatedAt.UTC().Format(time.RFC3339)),
	}
}

func toFanficCharacterRow(row sqlcgen.FanficCharacter) model.FanficCharacterRow {
	return model.FanficCharacterRow{
		ID:            int(row.ID),
		FanficID:      row.FanficID,
		Series:        row.Series,
		CharacterID:   row.CharacterID,
		CharacterName: row.CharacterName,
		SortOrder:     int(row.SortOrder),
		IsPairing:     row.IsPairing,
	}
}

func (r *fanficDAO) Create(ctx context.Context, s spec.NewFanfic, tx ...*sql.Tx) (*model.FanficRow, error) {
	created, err := genQueries(r.db, tx).CreateFanfic(ctx, sqlcgen.CreateFanficParams{
		UserID:         s.UserID,
		Title:          s.Title,
		Summary:        s.Summary,
		Series:         s.Series,
		Rating:         s.Rating,
		Language:       s.Language,
		Status:         s.Status,
		IsOneshot:      s.IsOneshot,
		ContainsLemons: s.ContainsLemons,
		Column10:       s.IsPairing,
	})
	if err != nil {
		return nil, fmt.Errorf("create fanfic: %w", err)
	}

	return new(toFanficRow(fanficJoinRow(created))), nil
}

func (r *fanficDAO) Update(ctx context.Context, s spec.FanficUpdate, tx ...*sql.Tx) error {
	var (
		affected int64
		err      error
	)

	queries := genQueries(r.db, tx)
	if s.AsAdmin {
		affected, err = queries.UpdateFanficAsAdmin(ctx, sqlcgen.UpdateFanficAsAdminParams{
			Title:          s.Title,
			Summary:        s.Summary,
			Series:         s.Series,
			Rating:         s.Rating,
			Language:       s.Language,
			Status:         s.Status,
			IsOneshot:      s.IsOneshot,
			ContainsLemons: s.ContainsLemons,
			ID:             s.ID,
		})
	} else {
		affected, err = queries.UpdateFanficAsOwner(ctx, sqlcgen.UpdateFanficAsOwnerParams{
			Title:          s.Title,
			Summary:        s.Summary,
			Series:         s.Series,
			Rating:         s.Rating,
			Language:       s.Language,
			Status:         s.Status,
			IsOneshot:      s.IsOneshot,
			ContainsLemons: s.ContainsLemons,
			ID:             s.ID,
			UserID:         s.UserID,
		})
	}
	if err != nil {
		return fmt.Errorf("update fanfic: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("fanfic not found or not owned")
	}

	return nil
}

func (r *fanficDAO) AddGenres(ctx context.Context, s spec.FanficGenres, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	for _, g := range s.Genres {
		err := queries.AddFanficGenre(ctx, sqlcgen.AddFanficGenreParams{
			FanficID: s.FanficID,
			Genre:    strings.TrimSpace(g),
		})
		if err != nil {
			return fmt.Errorf("add fanfic genre: %w", err)
		}
	}

	return nil
}

func (r *fanficDAO) DeleteGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteFanficGenres(ctx, fanficID); err != nil {
		return fmt.Errorf("delete fanfic genres: %w", err)
	}

	return nil
}

func (r *fanficDAO) AddTags(ctx context.Context, s spec.FanficTags, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	for _, t := range s.Tags {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}

		err := queries.AddFanficTag(ctx, sqlcgen.AddFanficTagParams{
			FanficID: s.FanficID,
			Tag:      tag,
		})
		if err != nil {
			return fmt.Errorf("add fanfic tag: %w", err)
		}
	}

	return nil
}

func (r *fanficDAO) DeleteTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteFanficTags(ctx, fanficID); err != nil {
		return fmt.Errorf("delete fanfic tags: %w", err)
	}

	return nil
}

func (r *fanficDAO) AddCharacters(ctx context.Context, s spec.FanficCharacters, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	for i, c := range s.Characters {
		err := queries.AddFanficCharacter(ctx, sqlcgen.AddFanficCharacterParams{
			FanficID:      s.FanficID,
			Series:        c.Series,
			CharacterID:   c.CharacterID,
			CharacterName: strings.TrimSpace(c.CharacterName),
			SortOrder:     int32(i),
			IsPairing:     s.IsPairing,
		})
		if err != nil {
			return fmt.Errorf("add fanfic character: %w", err)
		}
	}

	return nil
}

func (r *fanficDAO) DeleteCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteFanficCharacters(ctx, fanficID); err != nil {
		return fmt.Errorf("delete fanfic characters: %w", err)
	}

	return nil
}

func (r *fanficDAO) UpdateCoverImage(ctx context.Context, s spec.FanficCoverUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateFanficCoverImage(ctx, sqlcgen.UpdateFanficCoverImageParams{
		CoverImageUrl:     s.ImageURL,
		CoverThumbnailUrl: s.ThumbnailURL,
		ID:                s.ID,
	})
	if err != nil {
		return fmt.Errorf("update fanfic cover image: %w", err)
	}

	return nil
}

func (r *fanficDAO) UpdateWordCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UpdateFanficWordCount(ctx, fanficID); err != nil {
		return fmt.Errorf("update fanfic word count: %w", err)
	}

	return nil
}

func (r *fanficDAO) GetCoverImagePaths(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	row, err := genQueries(r.db, tx).GetFanficCoverPaths(ctx, fanficID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get fanfic cover paths: %w", err)
	}

	var paths []string
	if row.CoverImageUrl != "" {
		paths = append(paths, row.CoverImageUrl)
	}

	if row.CoverThumbnailUrl != "" {
		paths = append(paths, row.CoverThumbnailUrl)
	}

	return paths, nil
}

func (r *fanficDAO) GetByID(ctx context.Context, s spec.FanficLookup, tx ...*sql.Tx) (*model.FanficRow, error) {
	row, err := genQueries(r.db, tx).GetFanficByID(ctx, sqlcgen.GetFanficByIDParams{
		UserID: s.ViewerID,
		ID:     s.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get fanfic: %w", err)
	}

	return new(toFanficRow(row)), nil
}

func (r *fanficDAO) List(ctx context.Context, q spec.FanficListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	queries := genQueries(r.db, tx)

	search := ""
	if q.Params.Search != "" {
		search = "%" + q.Params.Search + "%"
	}

	total, err := queries.CountFanfics(ctx, sqlcgen.CountFanficsParams{
		ViewerID:       q.ViewerID,
		ShowLemons:     q.Params.ShowLemons,
		Series:         q.Params.Series,
		Rating:         q.Params.Rating,
		Language:       q.Params.Language,
		Status:         q.Params.Status,
		GenreA:         q.Params.GenreA,
		GenreB:         q.Params.GenreB,
		Tag:            q.Params.Tag,
		CharacterA:     q.Params.CharacterA,
		IsPairing:      q.Params.IsPairing,
		CharacterB:     q.Params.CharacterB,
		CharacterC:     q.Params.CharacterC,
		CharacterD:     q.Params.CharacterD,
		Search:         search,
		ExcludeUserIds: joinUUIDs(q.ExcludeUserIDs),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count fanfics: %w", err)
	}

	params := sqlcgen.ListFanficsByUpdatedParams{
		ViewerID:       q.ViewerID,
		ShowLemons:     q.Params.ShowLemons,
		Series:         q.Params.Series,
		Rating:         q.Params.Rating,
		Language:       q.Params.Language,
		Status:         q.Params.Status,
		GenreA:         q.Params.GenreA,
		GenreB:         q.Params.GenreB,
		Tag:            q.Params.Tag,
		CharacterA:     q.Params.CharacterA,
		IsPairing:      q.Params.IsPairing,
		CharacterB:     q.Params.CharacterB,
		CharacterC:     q.Params.CharacterC,
		CharacterD:     q.Params.CharacterD,
		Search:         search,
		ExcludeUserIds: joinUUIDs(q.ExcludeUserIDs),
		RowOffset:      int32(q.Params.Offset),
		RowLimit:       int32(q.Params.Limit),
	}

	var rows []fanficJoinRow

	switch q.Params.Sort {
	case "published":
		listed, err := queries.ListFanficsByPublished(ctx, sqlcgen.ListFanficsByPublishedParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list fanfics: %w", err)
		}

		for _, row := range listed {
			rows = append(rows, fanficJoinRow(row))
		}
	case "favourites":
		listed, err := queries.ListFanficsByFavourites(ctx, sqlcgen.ListFanficsByFavouritesParams(params))
		if err != nil {
			return nil, 0, fmt.Errorf("list fanfics: %w", err)
		}

		for _, row := range listed {
			rows = append(rows, fanficJoinRow(row))
		}
	default:
		listed, err := queries.ListFanficsByUpdated(ctx, params)
		if err != nil {
			return nil, 0, fmt.Errorf("list fanfics: %w", err)
		}

		for _, row := range listed {
			rows = append(rows, fanficJoinRow(row))
		}
	}

	var fanfics []model.FanficRow
	for _, row := range rows {
		fanfics = append(fanfics, toFanficRow(row))
	}

	return fanfics, int(total), nil
}

func (r *fanficDAO) ListByUser(ctx context.Context, q spec.FanficUserListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountFanficsByUser(ctx, sqlcgen.CountFanficsByUserParams{
		UserID:   q.UserID,
		UserID_2: q.ViewerID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count user fanfics: %w", err)
	}

	rows, err := queries.ListFanficsByUser(ctx, sqlcgen.ListFanficsByUserParams{
		UserID:   q.ViewerID,
		UserID_2: q.UserID,
		Limit:    int32(q.Limit),
		Offset:   int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user fanfics: %w", err)
	}

	var fanfics []model.FanficRow
	for _, row := range rows {
		fanfics = append(fanfics, toFanficRow(fanficJoinRow(row)))
	}

	return fanfics, int(total), nil
}

func (r *fanficDAO) CreateChapter(ctx context.Context, s spec.NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	created, err := genQueries(r.db, tx).CreateFanficChapter(ctx, sqlcgen.CreateFanficChapterParams{
		FanficID:      s.FanficID,
		ChapterNumber: int32(s.Number),
		Title:         s.Title,
		Body:          s.Body,
		WordCount:     int32(s.WordCount),
	})
	if err != nil {
		return nil, fmt.Errorf("create fanfic chapter: %w", err)
	}

	return new(toFanficChapterRow(created)), nil
}

func (r *fanficDAO) UpdateChapter(ctx context.Context, s spec.ChapterUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateFanficChapter(ctx, sqlcgen.UpdateFanficChapterParams{
		Title:     s.Title,
		Body:      s.Body,
		WordCount: int32(s.WordCount),
		ID:        s.ID,
	})
	if err != nil {
		return fmt.Errorf("update fanfic chapter: %w", err)
	}

	return nil
}

func (r *fanficDAO) DeleteChapter(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteFanficChapter(ctx, id); err != nil {
		return fmt.Errorf("delete fanfic chapter: %w", err)
	}

	return nil
}

func (r *fanficDAO) GetChapter(ctx context.Context, s spec.FanficChapterLookup, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	row, err := genQueries(r.db, tx).GetFanficChapter(ctx, sqlcgen.GetFanficChapterParams{
		FanficID:      s.FanficID,
		ChapterNumber: int32(s.ChapterNumber),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get fanfic chapter: %w", err)
	}

	return new(toFanficChapterRow(row)), nil
}

func (r *fanficDAO) ListChapters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficChapterSummaryRow, error) {
	rows, err := genQueries(r.db, tx).ListFanficChapters(ctx, fanficID)
	if err != nil {
		return nil, fmt.Errorf("list fanfic chapters: %w", err)
	}

	var chapters []model.FanficChapterSummaryRow
	for _, row := range rows {
		chapters = append(chapters, model.FanficChapterSummaryRow{
			ID:         row.ID,
			ChapterNum: int(row.ChapterNumber),
			Title:      row.Title,
			WordCount:  int(row.WordCount),
		})
	}

	return chapters, nil
}

func (r *fanficDAO) GetChapterCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountFanficChapters(ctx, fanficID)
	if err != nil {
		return 0, fmt.Errorf("get fanfic chapter count: %w", err)
	}

	return int(count), nil
}

func (r *fanficDAO) GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	next, err := genQueries(r.db, tx).GetNextFanficChapterNumber(ctx, fanficID)
	if err != nil {
		return 0, fmt.Errorf("get next chapter number: %w", err)
	}

	return int(next), nil
}

func (r *fanficDAO) GetChapterFanficID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	fanficID, err := genQueries(r.db, tx).GetFanficChapterFanficID(ctx, chapterID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get chapter fanfic id: %w", err)
	}

	return fanficID, nil
}

func (r *fanficDAO) GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	userID, err := genQueries(r.db, tx).GetFanficChapterAuthorID(ctx, chapterID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get chapter author: %w", err)
	}

	return userID, nil
}

func (r *fanficDAO) GetGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	genres, err := genQueries(r.db, tx).GetFanficGenres(ctx, fanficID)
	if err != nil {
		return nil, fmt.Errorf("get fanfic genres: %w", err)
	}

	if len(genres) == 0 {
		return nil, nil
	}

	return genres, nil
}

func (r *fanficDAO) GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetFanficGenresBatch(ctx, joinUUIDs(fanficIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic genres: %w", err)
	}

	result := make(map[uuid.UUID][]string)
	for _, row := range rows {
		result[row.FanficID] = append(result[row.FanficID], row.Genre)
	}

	return result, nil
}

func (r *fanficDAO) GetTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	tags, err := genQueries(r.db, tx).GetFanficTags(ctx, fanficID)
	if err != nil {
		return nil, fmt.Errorf("get fanfic tags: %w", err)
	}

	if len(tags) == 0 {
		return nil, nil
	}

	return tags, nil
}

func (r *fanficDAO) GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetFanficTagsBatch(ctx, joinUUIDs(fanficIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic tags: %w", err)
	}

	result := make(map[uuid.UUID][]string)
	for _, row := range rows {
		result[row.FanficID] = append(result[row.FanficID], row.Tag)
	}

	return result, nil
}

func (r *fanficDAO) GetCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficCharacterRow, error) {
	rows, err := genQueries(r.db, tx).GetFanficCharacters(ctx, fanficID)
	if err != nil {
		return nil, fmt.Errorf("get fanfic characters: %w", err)
	}

	var chars []model.FanficCharacterRow
	for _, row := range rows {
		chars = append(chars, toFanficCharacterRow(row))
	}

	return chars, nil
}

func (r *fanficDAO) GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.FanficCharacterRow, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetFanficCharactersBatch(ctx, joinUUIDs(fanficIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic characters: %w", err)
	}

	result := make(map[uuid.UUID][]model.FanficCharacterRow)
	for _, row := range rows {
		result[row.FanficID] = append(result[row.FanficID], toFanficCharacterRow(row))
	}

	return result, nil
}

func (r *fanficDAO) RegisterOCCharacter(ctx context.Context, s spec.NewFanficOCCharacter, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).RegisterFanficOCCharacter(ctx, sqlcgen.RegisterFanficOCCharacterParams{
		Name:      strings.TrimSpace(s.Name),
		CreatedBy: s.CreatorID,
	})
	if err != nil {
		return fmt.Errorf("register oc character: %w", err)
	}

	return nil
}

func (r *fanficDAO) SearchOCCharacters(ctx context.Context, query string, tx ...*sql.Tx) ([]string, error) {
	names, err := genQueries(r.db, tx).SearchFanficOCCharacters(ctx, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("search oc characters: %w", err)
	}

	if len(names) == 0 {
		return nil, nil
	}

	return names, nil
}

func (r *fanficDAO) GetLanguages(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	languages, err := genQueries(r.db, tx).GetFanficLanguages(ctx)
	if err != nil {
		return nil, fmt.Errorf("get languages: %w", err)
	}

	if len(languages) == 0 {
		return nil, nil
	}

	return languages, nil
}

func (r *fanficDAO) RegisterLanguage(ctx context.Context, name string, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).RegisterFanficLanguage(ctx, strings.TrimSpace(name)); err != nil {
		return fmt.Errorf("register language: %w", err)
	}

	return nil
}

func (r *fanficDAO) GetSeries(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	series, err := genQueries(r.db, tx).GetFanficSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("get series: %w", err)
	}

	if len(series) == 0 {
		return nil, nil
	}

	return series, nil
}

func (r *fanficDAO) RegisterSeries(ctx context.Context, name string, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).RegisterFanficSeries(ctx, strings.TrimSpace(name)); err != nil {
		return fmt.Errorf("register series: %w", err)
	}

	return nil
}

func (r *fanficDAO) Favourite(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	err := queries.FavouriteFanfic(ctx, sqlcgen.FavouriteFanficParams{
		UserID:   s.UserID,
		FanficID: s.FanficID,
	})
	if err != nil {
		return fmt.Errorf("favourite fanfic: %w", err)
	}

	if err := queries.SyncFanficFavouriteCount(ctx, s.FanficID); err != nil {
		return fmt.Errorf("update fanfic favourite count: %w", err)
	}

	return nil
}

func (r *fanficDAO) Unfavourite(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	err := queries.UnfavouriteFanfic(ctx, sqlcgen.UnfavouriteFanficParams{
		UserID:   s.UserID,
		FanficID: s.FanficID,
	})
	if err != nil {
		return fmt.Errorf("unfavourite fanfic: %w", err)
	}

	if err := queries.SyncFanficFavouriteCount(ctx, s.FanficID); err != nil {
		return fmt.Errorf("update fanfic favourite count: %w", err)
	}

	return nil
}

func (r *fanficDAO) GetReadingProgress(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) (int, error) {
	chapter, err := genQueries(r.db, tx).GetFanficReadingProgress(ctx, sqlcgen.GetFanficReadingProgressParams{
		UserID:   s.UserID,
		FanficID: s.FanficID,
	})
	if err != nil {
		return 0, nil
	}

	return int(chapter), nil
}

func (r *fanficDAO) SetReadingProgress(ctx context.Context, s spec.FanficReadingProgress, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetFanficReadingProgress(ctx, sqlcgen.SetFanficReadingProgressParams{
		UserID:        s.UserID,
		FanficID:      s.FanficID,
		ChapterNumber: int32(s.ChapterNumber),
	})
	if err != nil {
		return fmt.Errorf("set reading progress: %w", err)
	}

	return nil
}

func (r *fanficDAO) ListFavourites(ctx context.Context, q spec.FanficUserListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountFanficFavourites(ctx, q.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count favourites: %w", err)
	}

	rows, err := queries.ListFanficFavourites(ctx, sqlcgen.ListFanficFavouritesParams{
		UserID:   q.ViewerID,
		UserID_2: q.UserID,
		Limit:    int32(q.Limit),
		Offset:   int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list favourites: %w", err)
	}

	var result []model.FanficRow
	for _, row := range rows {
		result = append(result, toFanficRow(fanficJoinRow(row)))
	}

	return result, int(total), nil
}
