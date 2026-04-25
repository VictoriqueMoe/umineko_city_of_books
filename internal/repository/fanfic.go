package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	FanficRepository interface {
		CreateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool) error
		UpdateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool, asAdmin bool) error
		UpdateCoverImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error
		UpdateWordCount(ctx context.Context, fanficID uuid.UUID) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.FanficRow, error)
		GetAuthorID(ctx context.Context, fanficID uuid.UUID) (uuid.UUID, error)

		List(ctx context.Context, viewerID uuid.UUID, params FanficListParams, excludeUserIDs []uuid.UUID) ([]model.FanficRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit int, offset int) ([]model.FanficRow, int, error)

		CreateChapter(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, chapterNumber int, title string, body string, wordCount int) error
		UpdateChapter(ctx context.Context, id uuid.UUID, title string, body string, wordCount int) error
		DeleteChapter(ctx context.Context, id uuid.UUID) error
		GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int) (*model.FanficChapterRow, error)
		ListChapters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficChapterSummaryRow, error)
		GetChapterCount(ctx context.Context, fanficID uuid.UUID) (int, error)
		GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID) (int, error)
		GetChapterFanficID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error)
		GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error)

		GetGenres(ctx context.Context, fanficID uuid.UUID) ([]string, error)
		GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error)
		GetTags(ctx context.Context, fanficID uuid.UUID) ([]string, error)
		GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error)
		GetCharacters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficCharacterRow, error)
		GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCharacterRow, error)

		RegisterOCCharacter(ctx context.Context, name string, creatorID uuid.UUID) error
		SearchOCCharacters(ctx context.Context, query string) ([]string, error)
		GetLanguages(ctx context.Context) ([]string, error)
		RegisterLanguage(ctx context.Context, name string) error
		GetSeries(ctx context.Context) ([]string, error)
		RegisterSeries(ctx context.Context, name string) error

		Favourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error
		Unfavourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error
		RecordView(ctx context.Context, fanficID uuid.UUID, viewerHash string) (bool, error)
		GetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) (int, error)
		SetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, chapterNumber int) error
		ListFavourites(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.FanficRow, int, error)

		CreateComment(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.FanficCommentRow, error)
		GetCommentFanficID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.FanficCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCommentMediaRow, error)
	}

	FanficListParams struct {
		Sort       string
		Series     string
		Rating     string
		GenreA     string
		GenreB     string
		Language   string
		Status     string
		Tag        string
		CharacterA string
		CharacterB string
		CharacterC string
		CharacterD string
		IsPairing  bool
		ShowLemons bool
		Search     string
		Limit      int
		Offset     int
	}

	fanficRepository struct {
		db *sql.DB
	}
)

func fanficRenumber(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 16)
	idx := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteString(fmt.Sprintf("$%d", idx))
			idx++
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func fanficNullTimePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.UTC().Format(time.RFC3339)
	return &s
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

func scanFanficRow(row interface{ Scan(...interface{}) error }, f *model.FanficRow) error {
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

func insertFanficGenresTx(ctx context.Context, tx *sql.Tx, fanficID uuid.UUID, genres []string) error {
	for _, g := range genres {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO fanfic_genres (fanfic_id, genre) VALUES ($1, $2)`,
			fanficID, strings.TrimSpace(g),
		); err != nil {
			return fmt.Errorf("add fanfic genre: %w", err)
		}
	}
	return nil
}

func insertFanficTagsTx(ctx context.Context, tx *sql.Tx, fanficID uuid.UUID, tags []string) error {
	for _, t := range tags {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO fanfic_tags (fanfic_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			fanficID, tag,
		); err != nil {
			return fmt.Errorf("add fanfic tag: %w", err)
		}
	}
	return nil
}

func insertFanficCharactersTx(ctx context.Context, tx *sql.Tx, fanficID uuid.UUID, characters []dto.FanficCharacter, isPairing bool) error {
	for i, c := range characters {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO fanfic_characters (fanfic_id, series, character_id, character_name, sort_order, is_pairing) VALUES ($1, $2, $3, $4, $5, $6)`,
			fanficID, c.Series, c.CharacterID, strings.TrimSpace(c.CharacterName), i, isPairing,
		); err != nil {
			return fmt.Errorf("add fanfic character: %w", err)
		}
	}
	return nil
}

func (r *fanficRepository) CreateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO fanfics (id, user_id, title, summary, series, rating, language, status, is_oneshot, contains_lemons, cover_image_url, cover_thumbnail_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			id, userID, title, summary, series, rating, language, status, isOneshot, containsLemons, "", "",
		); err != nil {
			return fmt.Errorf("create fanfic: %w", err)
		}
		if err := insertFanficGenresTx(ctx, tx, id, genres); err != nil {
			return err
		}
		if err := insertFanficTagsTx(ctx, tx, id, tags); err != nil {
			return err
		}
		return insertFanficCharactersTx(ctx, tx, id, characters, isPairing)
	})
}

func (r *fanficRepository) UpdateWithDetails(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, summary string, series string, rating string, language string, status string, isOneshot bool, containsLemons bool, genres []string, tags []string, characters []dto.FanficCharacter, isPairing bool, asAdmin bool) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var res sql.Result
		var err error
		if asAdmin {
			res, err = tx.ExecContext(ctx,
				`UPDATE fanfics SET title = $1, summary = $2, series = $3, rating = $4, language = $5, status = $6, is_oneshot = $7, contains_lemons = $8, updated_at = NOW() WHERE id = $9`,
				title, summary, series, rating, language, status, isOneshot, containsLemons, id,
			)
		} else {
			res, err = tx.ExecContext(ctx,
				`UPDATE fanfics SET title = $1, summary = $2, series = $3, rating = $4, language = $5, status = $6, is_oneshot = $7, contains_lemons = $8, updated_at = NOW() WHERE id = $9 AND user_id = $10`,
				title, summary, series, rating, language, status, isOneshot, containsLemons, id, userID,
			)
		}
		if err != nil {
			return fmt.Errorf("update fanfic: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("fanfic not found or not owned")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM fanfic_genres WHERE fanfic_id = $1`, id); err != nil {
			return fmt.Errorf("delete fanfic genres: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM fanfic_tags WHERE fanfic_id = $1`, id); err != nil {
			return fmt.Errorf("delete fanfic tags: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM fanfic_characters WHERE fanfic_id = $1`, id); err != nil {
			return fmt.Errorf("delete fanfic characters: %w", err)
		}
		if err := insertFanficGenresTx(ctx, tx, id, genres); err != nil {
			return err
		}
		if err := insertFanficTagsTx(ctx, tx, id, tags); err != nil {
			return err
		}
		return insertFanficCharactersTx(ctx, tx, id, characters, isPairing)
	})
}

func (r *fanficRepository) UpdateCoverImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fanfics SET cover_image_url = $1, cover_thumbnail_url = $2 WHERE id = $3`,
		imageURL, thumbnailURL, id,
	)
	if err != nil {
		return fmt.Errorf("update fanfic cover image: %w", err)
	}
	return nil
}

func (r *fanficRepository) UpdateWordCount(ctx context.Context, fanficID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fanfics SET word_count = COALESCE((SELECT SUM(word_count) FROM fanfic_chapters WHERE fanfic_id = $1), 0), updated_at = NOW() WHERE id = $2`,
		fanficID, fanficID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic word count: %w", err)
	}
	return nil
}

func (r *fanficRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM fanfics WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete fanfic: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("fanfic not found or not owned")
	}
	return nil
}

func (r *fanficRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fanfics WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete fanfic: %w", err)
	}
	return nil
}

func (r *fanficRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.FanficRow, error) {
	var f model.FanficRow
	err := scanFanficRow(r.db.QueryRowContext(ctx, fanficRenumber(fanficSelectBase+` WHERE f.id = ?`), viewerID, id), &f)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get fanfic: %w", err)
	}
	return &f, nil
}

func (r *fanficRepository) GetAuthorID(ctx context.Context, fanficID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM fanfics WHERE id = $1`, fanficID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get fanfic author: %w", err)
	}
	return userID, nil
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

func (r *fanficRepository) List(ctx context.Context, viewerID uuid.UUID, params FanficListParams, excludeUserIDs []uuid.UUID) ([]model.FanficRow, int, error) {
	whereParts := []string{"(f.status != 'draft' OR f.user_id = ?)"}
	args := []interface{}{viewerID}

	if !params.ShowLemons {
		whereParts = append(whereParts, "f.contains_lemons = FALSE")
	}
	if params.Series != "" {
		whereParts = append(whereParts, "f.series = ?")
		args = append(args, params.Series)
	}
	if params.Rating != "" {
		whereParts = append(whereParts, "f.rating = ?")
		args = append(args, params.Rating)
	}
	if params.Language != "" {
		whereParts = append(whereParts, "f.language = ?")
		args = append(args, params.Language)
	}
	if params.Status != "" {
		whereParts = append(whereParts, "f.status = ?")
		args = append(args, params.Status)
	}
	if params.GenreA != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM fanfic_genres WHERE fanfic_id = f.id AND genre = ?)")
		args = append(args, params.GenreA)
	}
	if params.GenreB != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM fanfic_genres WHERE fanfic_id = f.id AND genre = ?)")
		args = append(args, params.GenreB)
	}
	if params.Tag != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM fanfic_tags WHERE fanfic_id = f.id AND tag = ?)")
		args = append(args, params.Tag)
	}

	characterFilter := func(name string) string {
		if params.IsPairing {
			return "EXISTS(SELECT 1 FROM fanfic_characters WHERE fanfic_id = f.id AND character_name = ? AND is_pairing = TRUE)"
		}
		return "EXISTS(SELECT 1 FROM fanfic_characters WHERE fanfic_id = f.id AND character_name = ?)"
	}
	if params.CharacterA != "" {
		whereParts = append(whereParts, characterFilter(params.CharacterA))
		args = append(args, params.CharacterA)
	}
	if params.CharacterB != "" {
		whereParts = append(whereParts, characterFilter(params.CharacterB))
		args = append(args, params.CharacterB)
	}
	if params.CharacterC != "" {
		whereParts = append(whereParts, characterFilter(params.CharacterC))
		args = append(args, params.CharacterC)
	}
	if params.CharacterD != "" {
		whereParts = append(whereParts, characterFilter(params.CharacterD))
		args = append(args, params.CharacterD)
	}

	if params.Search != "" {
		whereParts = append(whereParts, "(f.title ILIKE ? OR f.summary ILIKE ?)")
		search := "%" + params.Search + "%"
		args = append(args, search, search)
	}

	exclSQL := ""
	var exclArgs []interface{}
	if len(excludeUserIDs) > 0 {
		marks := make([]string, len(excludeUserIDs))
		exclArgs = make([]interface{}, len(excludeUserIDs))
		for i, id := range excludeUserIDs {
			marks[i] = "?"
			exclArgs[i] = id
		}
		exclSQL = " AND f.user_id NOT IN (" + strings.Join(marks, ",") + ")"
	}
	whereClause := " WHERE " + strings.Join(whereParts, " AND ") + exclSQL

	var total int
	countArgs := append([]interface{}{}, args...)
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx,
		fanficRenumber(`SELECT COUNT(*) FROM fanfics f`+whereClause), countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fanfics: %w", err)
	}

	orderClause := fanficOrderClause(params.Sort)
	query := fanficRenumber(fanficSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`)

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, exclArgs...)
	queryArgs = append(queryArgs, params.Limit, params.Offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
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

func (r *fanficRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit int, offset int) ([]model.FanficRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fanfics WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user fanfics: %w", err)
	}

	query := fanficRenumber(fanficSelectBase + ` WHERE f.user_id = ? ORDER BY f.updated_at DESC LIMIT ? OFFSET ?`)
	rows, err := r.db.QueryContext(ctx, query, viewerID, userID, limit, offset)
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

func (r *fanficRepository) CreateChapter(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, chapterNumber int, title string, body string, wordCount int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_chapters (id, fanfic_id, chapter_number, title, body, word_count) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, fanficID, chapterNumber, title, body, wordCount,
	)
	if err != nil {
		return fmt.Errorf("create fanfic chapter: %w", err)
	}
	return nil
}

func (r *fanficRepository) UpdateChapter(ctx context.Context, id uuid.UUID, title string, body string, wordCount int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fanfic_chapters SET title = $1, body = $2, word_count = $3, updated_at = NOW() WHERE id = $4`,
		title, body, wordCount, id,
	)
	if err != nil {
		return fmt.Errorf("update fanfic chapter: %w", err)
	}
	return nil
}

func (r *fanficRepository) DeleteChapter(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fanfic_chapters WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete fanfic chapter: %w", err)
	}
	return nil
}

func (r *fanficRepository) GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int) (*model.FanficChapterRow, error) {
	var c model.FanficChapterRow
	var createdAt time.Time
	var updatedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, fanfic_id, chapter_number, title, body, word_count, created_at, updated_at FROM fanfic_chapters WHERE fanfic_id = $1 AND chapter_number = $2`,
		fanficID, chapterNumber,
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

func (r *fanficRepository) ListChapters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficChapterSummaryRow, error) {
	rows, err := r.db.QueryContext(ctx,
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

func (r *fanficRepository) GetChapterCount(ctx context.Context, fanficID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fanfic_chapters WHERE fanfic_id = $1`, fanficID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get fanfic chapter count: %w", err)
	}
	return count, nil
}

func (r *fanficRepository) GetNextChapterNumber(ctx context.Context, fanficID uuid.UUID) (int, error) {
	var next int
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(chapter_number), 0) + 1 FROM fanfic_chapters WHERE fanfic_id = $1`, fanficID).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("get next chapter number: %w", err)
	}
	return next, nil
}

func (r *fanficRepository) GetChapterFanficID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error) {
	var fanficID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT fanfic_id FROM fanfic_chapters WHERE id = $1`, chapterID).Scan(&fanficID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get chapter fanfic id: %w", err)
	}
	return fanficID, nil
}

func (r *fanficRepository) GetChapterAuthorID(ctx context.Context, chapterID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT f.user_id FROM fanfic_chapters c JOIN fanfics f ON c.fanfic_id = f.id WHERE c.id = $1`,
		chapterID,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get chapter author: %w", err)
	}
	return userID, nil
}

func (r *fanficRepository) GetGenres(ctx context.Context, fanficID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT genre FROM fanfic_genres WHERE fanfic_id = $1 ORDER BY genre ASC`,
		fanficID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fanfic genres: %w", err)
	}
	defer rows.Close()

	var genres []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, fmt.Errorf("scan fanfic genre: %w", err)
		}
		genres = append(genres, g)
	}
	return genres, rows.Err()
}

func (r *fanficRepository) GetGenresBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(fanficIDs))
	args := make([]interface{}, len(fanficIDs))
	for i, id := range fanficIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT fanfic_id, genre FROM fanfic_genres WHERE fanfic_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY genre ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic genres: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]string)
	for rows.Next() {
		var fanficID uuid.UUID
		var genre string
		if err := rows.Scan(&fanficID, &genre); err != nil {
			return nil, fmt.Errorf("scan fanfic genre: %w", err)
		}
		result[fanficID] = append(result[fanficID], genre)
	}
	return result, rows.Err()
}

func (r *fanficRepository) GetTags(ctx context.Context, fanficID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT tag FROM fanfic_tags WHERE fanfic_id = $1 ORDER BY tag ASC`,
		fanficID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fanfic tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan fanfic tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *fanficRepository) GetTagsBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(fanficIDs))
	args := make([]interface{}, len(fanficIDs))
	for i, id := range fanficIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT fanfic_id, tag FROM fanfic_tags WHERE fanfic_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY tag ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic tags: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]string)
	for rows.Next() {
		var fanficID uuid.UUID
		var tag string
		if err := rows.Scan(&fanficID, &tag); err != nil {
			return nil, fmt.Errorf("scan fanfic tag: %w", err)
		}
		result[fanficID] = append(result[fanficID], tag)
	}
	return result, rows.Err()
}

func (r *fanficRepository) GetCharacters(ctx context.Context, fanficID uuid.UUID) ([]model.FanficCharacterRow, error) {
	rows, err := r.db.QueryContext(ctx,
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

func (r *fanficRepository) GetCharactersBatch(ctx context.Context, fanficIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCharacterRow, error) {
	if len(fanficIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(fanficIDs))
	args := make([]interface{}, len(fanficIDs))
	for i, id := range fanficIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
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

func (r *fanficRepository) RegisterOCCharacter(ctx context.Context, name string, creatorID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_oc_characters (name, created_by) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		strings.TrimSpace(name), creatorID,
	)
	if err != nil {
		return fmt.Errorf("register oc character: %w", err)
	}
	return nil
}

func (r *fanficRepository) SearchOCCharacters(ctx context.Context, query string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT name FROM fanfic_oc_characters WHERE name LIKE $1 ORDER BY name ASC`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search oc characters: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan oc character: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (r *fanficRepository) GetLanguages(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT name FROM fanfic_languages ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("get languages: %w", err)
	}
	defer rows.Close()

	var langs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan language: %w", err)
		}
		langs = append(langs, name)
	}
	return langs, rows.Err()
}

func (r *fanficRepository) RegisterLanguage(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_languages (name) VALUES ($1) ON CONFLICT DO NOTHING`,
		strings.TrimSpace(name),
	)
	if err != nil {
		return fmt.Errorf("register language: %w", err)
	}
	return nil
}

func (r *fanficRepository) GetSeries(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT name FROM fanfic_series ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("get series: %w", err)
	}
	defer rows.Close()

	var series []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan series: %w", err)
		}
		series = append(series, name)
	}
	return series, rows.Err()
}

func (r *fanficRepository) RegisterSeries(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_series (name) VALUES ($1) ON CONFLICT DO NOTHING`,
		strings.TrimSpace(name),
	)
	if err != nil {
		return fmt.Errorf("register series: %w", err)
	}
	return nil
}

func (r *fanficRepository) Favourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_favourites (user_id, fanfic_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, fanficID,
	)
	if err != nil {
		return fmt.Errorf("favourite fanfic: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE fanfics SET favourite_count = (SELECT COUNT(*) FROM fanfic_favourites WHERE fanfic_id = $1) WHERE id = $2`,
		fanficID, fanficID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic favourite count: %w", err)
	}
	return nil
}

func (r *fanficRepository) Unfavourite(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fanfic_favourites WHERE user_id = $1 AND fanfic_id = $2`,
		userID, fanficID,
	)
	if err != nil {
		return fmt.Errorf("unfavourite fanfic: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE fanfics SET favourite_count = (SELECT COUNT(*) FROM fanfic_favourites WHERE fanfic_id = $1) WHERE id = $2`,
		fanficID, fanficID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic favourite count: %w", err)
	}
	return nil
}

func (r *fanficRepository) RecordView(ctx context.Context, fanficID uuid.UUID, viewerHash string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_views (fanfic_id, viewer_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		fanficID, viewerHash,
	)
	if err != nil {
		return false, fmt.Errorf("record fanfic view: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		_, err = r.db.ExecContext(ctx,
			`UPDATE fanfics SET view_count = (SELECT COUNT(*) FROM fanfic_views WHERE fanfic_id = $1) WHERE id = $2`,
			fanficID, fanficID,
		)
		if err != nil {
			return false, fmt.Errorf("update fanfic view count: %w", err)
		}
	}
	return n > 0, nil
}

func (r *fanficRepository) GetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID) (int, error) {
	var chapter int
	err := r.db.QueryRowContext(ctx,
		`SELECT chapter_number FROM fanfic_reading_progress WHERE user_id = $1 AND fanfic_id = $2`,
		userID, fanficID,
	).Scan(&chapter)
	if err != nil {
		return 0, nil
	}
	return chapter, nil
}

func (r *fanficRepository) SetReadingProgress(ctx context.Context, userID uuid.UUID, fanficID uuid.UUID, chapterNumber int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_reading_progress (user_id, fanfic_id, chapter_number, updated_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, fanfic_id) DO UPDATE SET chapter_number = $4, updated_at = NOW()`,
		userID, fanficID, chapterNumber, chapterNumber,
	)
	if err != nil {
		return fmt.Errorf("set reading progress: %w", err)
	}
	return nil
}

func (r *fanficRepository) ListFavourites(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.FanficRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fanfic_favourites WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count favourites: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		fanficRenumber(fanficSelectBase+` JOIN fanfic_favourites fav ON fav.fanfic_id = f.id WHERE fav.user_id = ? AND (f.status != 'draft' OR f.user_id = ?) ORDER BY fav.created_at DESC LIMIT ? OFFSET ?`),
		viewerID, userID, viewerID, limit, offset,
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

func (r *fanficRepository) CreateComment(ctx context.Context, id uuid.UUID, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_comments (id, fanfic_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, fanficID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create fanfic comment: %w", err)
	}
	return nil
}

func (r *fanficRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE fanfic_comments SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update fanfic comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *fanficRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fanfic_comments SET body = $1, updated_at = NOW() WHERE id = $2`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update fanfic comment: %w", err)
	}
	return nil
}

func (r *fanficRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM fanfic_comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete fanfic comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *fanficRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM fanfic_comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete fanfic comment: %w", err)
	}
	return nil
}

func (r *fanficRepository) GetComments(ctx context.Context, fanficID uuid.UUID, viewerID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.FanficCommentRow, error) {
	args := []interface{}{viewerID, fanficID}
	exclSQL := ""
	if len(excludeUserIDs) > 0 {
		marks := make([]string, len(excludeUserIDs))
		for i, id := range excludeUserIDs {
			marks[i] = "?"
			args = append(args, id)
		}
		exclSQL = " AND c.user_id NOT IN (" + strings.Join(marks, ",") + ")"
	}

	query := fanficRenumber(`SELECT c.id, c.fanfic_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM fanfic_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM fanfic_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM fanfic_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE c.fanfic_id = ?` + exclSQL + `
		ORDER BY c.created_at ASC`)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get fanfic comments: %w", err)
	}
	defer rows.Close()

	var comments []model.FanficCommentRow
	for rows.Next() {
		var c model.FanficCommentRow
		var createdAt time.Time
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&c.ID, &c.FanficID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, fmt.Errorf("scan fanfic comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		c.UpdatedAt = fanficNullTimePtr(updatedAt)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *fanficRepository) GetCommentFanficID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var fanficID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT fanfic_id FROM fanfic_comments WHERE id = $1`, commentID).Scan(&fanficID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get fanfic comment fanfic id: %w", err)
	}
	return fanficID, nil
}

func (r *fanficRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM fanfic_comments WHERE id = $1`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get fanfic comment author: %w", err)
	}
	return userID, nil
}

func (r *fanficRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fanfic_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like fanfic comment: %w", err)
	}
	return nil
}

func (r *fanficRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fanfic_comment_likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike fanfic comment: %w", err)
	}
	return nil
}

func (r *fanficRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO fanfic_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add fanfic comment media: %w", err)
	}
	return id, nil
}

func (r *fanficRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE fanfic_comment_media SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update fanfic comment media url: %w", err)
	}
	return nil
}

func (r *fanficRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE fanfic_comment_media SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update fanfic comment media thumbnail: %w", err)
	}
	return nil
}

func (r *fanficRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.FanficCommentMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM fanfic_comment_media WHERE comment_id = $1 ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fanfic comment media: %w", err)
	}
	defer rows.Close()

	var media []model.FanficCommentMediaRow
	for rows.Next() {
		var m model.FanficCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan fanfic comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *fanficRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.FanficCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(commentIDs))
	args := make([]interface{}, len(commentIDs))
	for i, id := range commentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM fanfic_comment_media WHERE comment_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get fanfic comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.FanficCommentMediaRow)
	for rows.Next() {
		var m model.FanficCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan fanfic comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}
