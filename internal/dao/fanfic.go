package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"umineko_city_of_books/internal/dao/utils"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
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
)

func fanficNullTimePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	return new(t.Time.UTC().Format(time.RFC3339))
}

const fanficSelectBase = `
	SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
		f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
		f.word_count, f.favourite_count, f.view_count, f.comment_count,
		f.published_at, f.created_at, f.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM fanfic_chapters WHERE fanfic_id = f.id),
		EXISTS(SELECT 1 FROM fanfic_favourites WHERE fanfic_id = f.id AND user_id = ?),
		EXISTS(SELECT 1 FROM fanfic_characters WHERE fanfic_id = f.id AND is_pairing = TRUE)
	FROM fanfics f
	JOIN users u ON f.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = u.id`

func scanFanficRow(row interface{ Scan(...any) error }, f *model.FanficRow) error {
	var publishedAt, createdAt time.Time
	var updatedAt sql.NullTime
	err := row.Scan(
		&f.ID, &f.UserID, &f.Title, &f.Summary, &f.Series, &f.Rating, &f.Language, &f.Status,
		&f.IsOneshot, &f.ContainsLemons, &f.CoverImageURL, &f.CoverThumbnailURL,
		&f.WordCount, &f.FavouriteCount, &f.ViewCount, &f.CommentCount,
		&publishedAt, &createdAt, &updatedAt,
		&f.AuthorUsername, &f.AuthorDisplayName, &f.AuthorAvatarURL, &f.AuthorRole,
		&f.ChapterCount, &f.UserFavourited, &f.IsPairing,
	)
	if err != nil {
		return err
	}
	f.PublishedAt = publishedAt.UTC().Format(time.RFC3339)
	f.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	f.UpdatedAt = fanficNullTimePtr(updatedAt)
	return nil
}

func (r *fanficDAO) Create(ctx context.Context, s spec.NewFanfic, tx ...*sql.Tx) (*model.FanficRow, error) {
	var created model.FanficRow

	if err := scanFanficRow(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH f AS (
		     INSERT INTO fanfics (user_id, title, summary, series, rating, language, status, is_oneshot, contains_lemons)
		     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		     RETURNING *
		 )
		 SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
		        f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
		        f.word_count, f.favourite_count, f.view_count, f.comment_count,
		        f.published_at, f.created_at, f.updated_at,
		        u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		        0, FALSE, $10::boolean
		 FROM f
		 JOIN users u ON u.id = f.user_id
		 LEFT JOIN user_roles r ON r.user_id = u.id`,
		s.UserID, s.Title, s.Summary, s.Series, s.Rating, s.Language, s.Status, s.IsOneshot, s.ContainsLemons, s.IsPairing,
	), &created); err != nil {
		return nil, fmt.Errorf("create fanfic: %w", err)
	}

	return &created, nil
}

func (r *fanficDAO) Update(ctx context.Context, s spec.FanficUpdate, tx ...*sql.Tx) error {
	var (
		res sql.Result
		err error
	)

	if s.AsAdmin {
		res, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE fanfics SET title = $1, summary = $2, series = $3, rating = $4, language = $5, status = $6, is_oneshot = $7, contains_lemons = $8, updated_at = NOW() WHERE id = $9`,
			s.Title, s.Summary, s.Series, s.Rating, s.Language, s.Status, s.IsOneshot, s.ContainsLemons, s.ID,
		)
	} else {
		res, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE fanfics SET title = $1, summary = $2, series = $3, rating = $4, language = $5, status = $6, is_oneshot = $7, contains_lemons = $8, updated_at = NOW() WHERE id = $9 AND user_id = $10`,
			s.Title, s.Summary, s.Series, s.Rating, s.Language, s.Status, s.IsOneshot, s.ContainsLemons, s.ID, s.UserID,
		)
	}
	if err != nil {
		return fmt.Errorf("update fanfic: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("fanfic not found or not owned")
	}

	return nil
}

func (r *fanficDAO) AddGenres(ctx context.Context, s spec.FanficGenres, tx ...*sql.Tx) error {
	for _, g := range s.Genres {
		if _, err := txOrDB(r.db, tx).ExecContext(ctx,
			`INSERT INTO fanfic_genres (fanfic_id, genre) VALUES ($1, $2)`,
			s.FanficID, strings.TrimSpace(g),
		); err != nil {
			return fmt.Errorf("add fanfic genre: %w", err)
		}
	}

	return nil
}

func (r *fanficDAO) DeleteGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM fanfic_genres WHERE fanfic_id = $1`, fanficID); err != nil {
		return fmt.Errorf("delete fanfic genres: %w", err)
	}

	return nil
}

func (r *fanficDAO) AddTags(ctx context.Context, s spec.FanficTags, tx ...*sql.Tx) error {
	for _, t := range s.Tags {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}

		if _, err := txOrDB(r.db, tx).ExecContext(ctx,
			`INSERT INTO fanfic_tags (fanfic_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			s.FanficID, tag,
		); err != nil {
			return fmt.Errorf("add fanfic tag: %w", err)
		}
	}

	return nil
}

func (r *fanficDAO) DeleteTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM fanfic_tags WHERE fanfic_id = $1`, fanficID); err != nil {
		return fmt.Errorf("delete fanfic tags: %w", err)
	}

	return nil
}

func (r *fanficDAO) AddCharacters(ctx context.Context, s spec.FanficCharacters, tx ...*sql.Tx) error {
	for i, c := range s.Characters {
		if _, err := txOrDB(r.db, tx).ExecContext(ctx,
			`INSERT INTO fanfic_characters (fanfic_id, series, character_id, character_name, sort_order, is_pairing) VALUES ($1, $2, $3, $4, $5, $6)`,
			s.FanficID, c.Series, c.CharacterID, strings.TrimSpace(c.CharacterName), i, s.IsPairing,
		); err != nil {
			return fmt.Errorf("add fanfic character: %w", err)
		}
	}

	return nil
}

func (r *fanficDAO) DeleteCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM fanfic_characters WHERE fanfic_id = $1`, fanficID); err != nil {
		return fmt.Errorf("delete fanfic characters: %w", err)
	}

	return nil
}

func (r *fanficDAO) UpdateCoverImage(ctx context.Context, s spec.FanficCoverUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE fanfics SET cover_image_url = $1, cover_thumbnail_url = $2 WHERE id = $3`,
		s.ImageURL, s.ThumbnailURL, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic cover image: %w", err)
	}
	return nil
}

func (r *fanficDAO) UpdateWordCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE fanfics SET word_count = COALESCE((SELECT SUM(word_count) FROM fanfic_chapters WHERE fanfic_id = $1), 0), updated_at = NOW() WHERE id = $2`,
		fanficID, fanficID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic word count: %w", err)
	}
	return nil
}

func (r *fanficDAO) GetCoverImagePaths(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var (
		coverURL     string
		thumbnailURL string
	)

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT cover_image_url, cover_thumbnail_url FROM fanfics WHERE id = $1`,
		fanficID,
	).Scan(&coverURL, &thumbnailURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get fanfic cover paths: %w", err)
	}

	var paths []string
	if coverURL != "" {
		paths = append(paths, coverURL)
	}

	if thumbnailURL != "" {
		paths = append(paths, thumbnailURL)
	}

	return paths, nil
}

func (r *fanficDAO) GetByID(ctx context.Context, s spec.FanficLookup, tx ...*sql.Tx) (*model.FanficRow, error) {
	var f model.FanficRow
	err := scanFanficRow(txOrDB(r.db, tx).QueryRowContext(ctx, utils.Rebind(fanficSelectBase+` WHERE f.id = ?`), s.ViewerID, s.ID), &f)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get fanfic: %w", err)
	}
	return &f, nil
}

func fanficOrderClause(sort string) string {
	switch sort {
	case "published":
		return ` ORDER BY f.published_at DESC`
	case "favourites":
		return ` ORDER BY f.favourite_count DESC, f.updated_at DESC`
	default:
		return ` ORDER BY f.updated_at DESC`
	}
}

func (r *fanficDAO) List(ctx context.Context, q spec.FanficListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	whereParts := []string{"(f.status != 'draft' OR f.user_id = ?)"}
	args := []any{q.ViewerID}

	if !q.Params.ShowLemons {
		whereParts = append(whereParts, "f.contains_lemons = FALSE")
	}
	if q.Params.Series != "" {
		whereParts = append(whereParts, "f.series = ?")
		args = append(args, q.Params.Series)
	}
	if q.Params.Rating != "" {
		whereParts = append(whereParts, "f.rating = ?")
		args = append(args, q.Params.Rating)
	}
	if q.Params.Language != "" {
		whereParts = append(whereParts, "f.language = ?")
		args = append(args, q.Params.Language)
	}
	if q.Params.Status != "" {
		whereParts = append(whereParts, "f.status = ?")
		args = append(args, q.Params.Status)
	}
	if q.Params.GenreA != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM fanfic_genres WHERE fanfic_id = f.id AND genre = ?)")
		args = append(args, q.Params.GenreA)
	}
	if q.Params.GenreB != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM fanfic_genres WHERE fanfic_id = f.id AND genre = ?)")
		args = append(args, q.Params.GenreB)
	}
	if q.Params.Tag != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM fanfic_tags WHERE fanfic_id = f.id AND tag = ?)")
		args = append(args, q.Params.Tag)
	}

	characterFilter := func(name string) string {
		if q.Params.IsPairing {
			return "EXISTS(SELECT 1 FROM fanfic_characters WHERE fanfic_id = f.id AND character_name = ? AND is_pairing = TRUE)"
		}
		return "EXISTS(SELECT 1 FROM fanfic_characters WHERE fanfic_id = f.id AND character_name = ?)"
	}
	if q.Params.CharacterA != "" {
		whereParts = append(whereParts, characterFilter(q.Params.CharacterA))
		args = append(args, q.Params.CharacterA)
	}
	if q.Params.CharacterB != "" {
		whereParts = append(whereParts, characterFilter(q.Params.CharacterB))
		args = append(args, q.Params.CharacterB)
	}
	if q.Params.CharacterC != "" {
		whereParts = append(whereParts, characterFilter(q.Params.CharacterC))
		args = append(args, q.Params.CharacterC)
	}
	if q.Params.CharacterD != "" {
		whereParts = append(whereParts, characterFilter(q.Params.CharacterD))
		args = append(args, q.Params.CharacterD)
	}

	if q.Params.Search != "" {
		whereParts = append(whereParts, "(f.title ILIKE ? OR f.summary ILIKE ?)")
		search := "%" + q.Params.Search + "%"
		args = append(args, search, search)
	}

	exclSQL := ""
	var exclArgs []any
	if len(q.ExcludeUserIDs) > 0 {
		var marks []string
		marks, exclArgs = utils.QuestionArgs(q.ExcludeUserIDs)
		exclSQL = " AND f.user_id NOT IN (" + strings.Join(marks, ",") + ")"
	}
	whereClause := " WHERE " + strings.Join(whereParts, " AND ") + exclSQL

	var total int
	countArgs := append([]any{}, args...)
	countArgs = append(countArgs, exclArgs...)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		utils.Rebind(`SELECT COUNT(*) FROM fanfics f`+whereClause), countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fanfics: %w", err)
	}

	orderClause := fanficOrderClause(q.Params.Sort)
	query := utils.Rebind(fanficSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`)

	queryArgs := []any{q.ViewerID}
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, exclArgs...)
	queryArgs = append(queryArgs, q.Params.Limit, q.Params.Offset)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list fanfics: %w", err)
	}
	defer rows.Close()

	var fanfics []model.FanficRow
	for rows.Next() {
		var f model.FanficRow
		if err := scanFanficRow(rows, &f); err != nil {
			return nil, 0, fmt.Errorf("scan fanfic: %w", err)
		}
		fanfics = append(fanfics, f)
	}
	return fanfics, total, rows.Err()
}

func (r *fanficDAO) ListByUser(ctx context.Context, q spec.FanficUserListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM fanfics WHERE user_id = $1 AND (status != 'draft' OR user_id = $2)`, q.UserID, q.ViewerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user fanfics: %w", err)
	}

	query := utils.Rebind(fanficSelectBase + ` WHERE f.user_id = ? AND (f.status != 'draft' OR f.user_id = ?) ORDER BY f.updated_at DESC LIMIT ? OFFSET ?`)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, q.ViewerID, q.UserID, q.ViewerID, q.Limit, q.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user fanfics: %w", err)
	}
	defer rows.Close()

	var fanfics []model.FanficRow
	for rows.Next() {
		var f model.FanficRow
		if err := scanFanficRow(rows, &f); err != nil {
			return nil, 0, fmt.Errorf("scan fanfic: %w", err)
		}
		fanfics = append(fanfics, f)
	}
	return fanfics, total, rows.Err()
}

func (r *fanficDAO) CreateChapter(ctx context.Context, s spec.NewChapter, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	var c model.FanficChapterRow
	var createdAt time.Time
	var updatedAt sql.NullTime

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`INSERT INTO fanfic_chapters (fanfic_id, chapter_number, title, body, word_count)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, fanfic_id, chapter_number, title, body, word_count, created_at, updated_at`,
		s.FanficID, s.Number, s.Title, s.Body, s.WordCount,
	).Scan(&c.ID, &c.FanficID, &c.ChapterNum, &c.Title, &c.Body, &c.WordCount, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("create fanfic chapter: %w", err)
	}

	c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	c.UpdatedAt = fanficNullTimePtr(updatedAt)

	return &c, nil
}

func (r *fanficDAO) UpdateChapter(ctx context.Context, s spec.ChapterUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE fanfic_chapters SET title = $1, body = $2, word_count = $3, updated_at = NOW() WHERE id = $4`,
		s.Title, s.Body, s.WordCount, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic chapter: %w", err)
	}
	return nil
}

func (r *fanficDAO) DeleteChapter(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM fanfic_chapters WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete fanfic chapter: %w", err)
	}
	return nil
}

func (r *fanficDAO) GetChapter(ctx context.Context, s spec.FanficChapterLookup, tx ...*sql.Tx) (*model.FanficChapterRow, error) {
	var c model.FanficChapterRow
	var createdAt time.Time
	var updatedAt sql.NullTime
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id, fanfic_id, chapter_number, title, body, word_count, created_at, updated_at FROM fanfic_chapters WHERE fanfic_id = $1 AND chapter_number = $2`,
		s.FanficID, s.ChapterNumber,
	).Scan(&c.ID, &c.FanficID, &c.ChapterNum, &c.Title, &c.Body, &c.WordCount, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get fanfic chapter: %w", err)
	}
	c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	c.UpdatedAt = fanficNullTimePtr(updatedAt)
	return &c, nil
}

func (r *fanficDAO) ListChapters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficChapterSummaryRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, chapter_number, title, word_count FROM fanfic_chapters WHERE fanfic_id = $1 ORDER BY chapter_number ASC`,
		fanficID,
	)
	if err != nil {
		return nil, fmt.Errorf("list fanfic chapters: %w", err)
	}
	defer rows.Close()

	var chapters []model.FanficChapterSummaryRow
	for rows.Next() {
		var c model.FanficChapterSummaryRow
		if err := rows.Scan(&c.ID, &c.ChapterNum, &c.Title, &c.WordCount); err != nil {
			return nil, fmt.Errorf("scan fanfic chapter summary: %w", err)
		}
		chapters = append(chapters, c)
	}
	return chapters, rows.Err()
}

func (r *fanficDAO) GetChapterCount(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM fanfic_chapters WHERE fanfic_id = $1`, fanficID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get fanfic chapter count: %w", err)
	}
	return count, nil
}

func (r *fanficDAO) GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var next int
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COALESCE(MAX(chapter_number), 0) + 1 FROM fanfic_chapters WHERE fanfic_id = $1`, fanficID).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("get next chapter number: %w", err)
	}
	return next, nil
}

func (r *fanficDAO) GetChapterFanficID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var fanficID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT fanfic_id FROM fanfic_chapters WHERE id = $1`, chapterID).Scan(&fanficID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get chapter fanfic id: %w", err)
	}
	return fanficID, nil
}

func (r *fanficDAO) GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	var userID uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT f.user_id FROM fanfic_chapters c JOIN fanfics f ON c.fanfic_id = f.id WHERE c.id = $1`,
		chapterID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get chapter author: %w", err)
	}
	return userID, nil
}

func (r *fanficDAO) GetGenres(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT genre FROM fanfic_genres WHERE fanfic_id = $1 ORDER BY genre ASC`,
		fanficID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fanfic genres: %w", err)
	}

	return utils.ScanStrings(rows, "fanfic genre")
}

func (r *fanficDAO) GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	placeholders, args := utils.PlaceholderArgs(fanficIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT fanfic_id, genre FROM fanfic_genres WHERE fanfic_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY genre ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic genres: %w", err)
	}

	return utils.ScanGroups[uuid.UUID, string](rows, "fanfic genre")
}

func (r *fanficDAO) GetTags(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT tag FROM fanfic_tags WHERE fanfic_id = $1 ORDER BY tag ASC`,
		fanficID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fanfic tags: %w", err)
	}

	return utils.ScanStrings(rows, "fanfic tag")
}

func (r *fanficDAO) GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	placeholders, args := utils.PlaceholderArgs(fanficIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT fanfic_id, tag FROM fanfic_tags WHERE fanfic_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY tag ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic tags: %w", err)
	}

	return utils.ScanGroups[uuid.UUID, string](rows, "fanfic tag")
}

func (r *fanficDAO) GetCharacters(ctx context.Context, fanficID uuid.UUID, tx ...*sql.Tx) ([]model.FanficCharacterRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, fanfic_id, series, character_id, character_name, sort_order, is_pairing FROM fanfic_characters WHERE fanfic_id = $1 ORDER BY sort_order ASC`,
		fanficID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fanfic characters: %w", err)
	}
	defer rows.Close()

	var chars []model.FanficCharacterRow
	for rows.Next() {
		var c model.FanficCharacterRow
		if err := rows.Scan(&c.ID, &c.FanficID, &c.Series, &c.CharacterID, &c.CharacterName, &c.SortOrder, &c.IsPairing); err != nil {
			return nil, fmt.Errorf("scan fanfic character: %w", err)
		}
		chars = append(chars, c)
	}
	return chars, rows.Err()
}

func (r *fanficDAO) GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.FanficCharacterRow, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	placeholders, args := utils.PlaceholderArgs(fanficIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, fanfic_id, series, character_id, character_name, sort_order, is_pairing FROM fanfic_characters WHERE fanfic_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY sort_order ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic characters: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.FanficCharacterRow)
	for rows.Next() {
		var c model.FanficCharacterRow
		if err := rows.Scan(&c.ID, &c.FanficID, &c.Series, &c.CharacterID, &c.CharacterName, &c.SortOrder, &c.IsPairing); err != nil {
			return nil, fmt.Errorf("scan fanfic character: %w", err)
		}
		result[c.FanficID] = append(result[c.FanficID], c)
	}
	return result, rows.Err()
}

func (r *fanficDAO) RegisterOCCharacter(ctx context.Context, s spec.NewFanficOCCharacter, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO fanfic_oc_characters (name, created_by) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		strings.TrimSpace(s.Name), s.CreatorID,
	)
	if err != nil {
		return fmt.Errorf("register oc character: %w", err)
	}
	return nil
}

func (r *fanficDAO) SearchOCCharacters(ctx context.Context, query string, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT name FROM fanfic_oc_characters WHERE name LIKE $1 ORDER BY name ASC`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search oc characters: %w", err)
	}

	return utils.ScanStrings(rows, "oc character")
}

func (r *fanficDAO) GetLanguages(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT name FROM fanfic_languages ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("get languages: %w", err)
	}

	return utils.ScanStrings(rows, "language")
}

func (r *fanficDAO) RegisterLanguage(ctx context.Context, name string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO fanfic_languages (name) VALUES ($1) ON CONFLICT DO NOTHING`,
		strings.TrimSpace(name),
	)
	if err != nil {
		return fmt.Errorf("register language: %w", err)
	}
	return nil
}

func (r *fanficDAO) GetSeries(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT name FROM fanfic_series ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("get series: %w", err)
	}

	return utils.ScanStrings(rows, "series")
}

func (r *fanficDAO) RegisterSeries(ctx context.Context, name string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO fanfic_series (name) VALUES ($1) ON CONFLICT DO NOTHING`,
		strings.TrimSpace(name),
	)
	if err != nil {
		return fmt.Errorf("register series: %w", err)
	}
	return nil
}

func (r *fanficDAO) Favourite(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO fanfic_favourites (user_id, fanfic_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		s.UserID, s.FanficID,
	)
	if err != nil {
		return fmt.Errorf("favourite fanfic: %w", err)
	}
	_, err = txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE fanfics SET favourite_count = (SELECT COUNT(*) FROM fanfic_favourites WHERE fanfic_id = $1) WHERE id = $2`,
		s.FanficID, s.FanficID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic favourite count: %w", err)
	}
	return nil
}

func (r *fanficDAO) Unfavourite(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM fanfic_favourites WHERE user_id = $1 AND fanfic_id = $2`,
		s.UserID, s.FanficID,
	)
	if err != nil {
		return fmt.Errorf("unfavourite fanfic: %w", err)
	}
	_, err = txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE fanfics SET favourite_count = (SELECT COUNT(*) FROM fanfic_favourites WHERE fanfic_id = $1) WHERE id = $2`,
		s.FanficID, s.FanficID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic favourite count: %w", err)
	}
	return nil
}

func (r *fanficDAO) GetReadingProgress(ctx context.Context, s spec.FanficUserRef, tx ...*sql.Tx) (int, error) {
	var chapter int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT chapter_number FROM fanfic_reading_progress WHERE user_id = $1 AND fanfic_id = $2`,
		s.UserID, s.FanficID,
	).Scan(&chapter)
	if err != nil {
		return 0, nil
	}
	return chapter, nil
}

func (r *fanficDAO) SetReadingProgress(ctx context.Context, s spec.FanficReadingProgress, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO fanfic_reading_progress (user_id, fanfic_id, chapter_number, updated_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, fanfic_id) DO UPDATE SET chapter_number = $4, updated_at = NOW()`,
		s.UserID, s.FanficID, s.ChapterNumber, s.ChapterNumber,
	)
	if err != nil {
		return fmt.Errorf("set reading progress: %w", err)
	}
	return nil
}

func (r *fanficDAO) ListFavourites(ctx context.Context, q spec.FanficUserListFilter, tx ...*sql.Tx) ([]model.FanficRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fanfic_favourites WHERE user_id = $1`, q.UserID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count favourites: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		utils.Rebind(fanficSelectBase+` JOIN fanfic_favourites fav ON fav.fanfic_id = f.id WHERE fav.user_id = ? AND (f.status != 'draft' OR f.user_id = ?) ORDER BY fav.created_at DESC LIMIT ? OFFSET ?`),
		q.ViewerID, q.UserID, q.ViewerID, q.Limit, q.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list favourites: %w", err)
	}
	defer rows.Close()

	var result []model.FanficRow
	for rows.Next() {
		var f model.FanficRow
		if err := scanFanficRow(rows, &f); err != nil {
			return nil, 0, fmt.Errorf("scan favourite: %w", err)
		}
		result = append(result, f)
	}
	return result, total, rows.Err()
}
