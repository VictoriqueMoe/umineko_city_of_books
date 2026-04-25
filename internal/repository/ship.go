package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"umineko_city_of_books/internal/repository/model"

	"umineko_city_of_books/internal/db"
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

	shipRepository struct {
		db *sql.DB
	}
)

const shipSelectBase = `
	SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0),
		COALESCE((SELECT value FROM ship_votes WHERE ship_id = s.id AND user_id = $1), 0),
		(SELECT COUNT(*) FROM ship_comments WHERE ship_id = s.id)
	FROM ships s
	JOIN users u ON s.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = s.user_id`

func scanShipRow(row interface{ Scan(...interface{}) error }, s *model.ShipRow) error {
	var createdAt, updatedAt time.Time
	if err := row.Scan(
		&s.ID, &s.UserID, &s.Title, &s.Description, &s.ImageURL, &s.ThumbnailURL, &createdAt, &updatedAt,
		&s.AuthorUsername, &s.AuthorDisplayName, &s.AuthorAvatarURL, &s.AuthorRole,
		&s.VoteScore, &s.UserVote, &s.CommentCount,
	); err != nil {
		return err
	}
	s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	updated := updatedAt.UTC().Format(time.RFC3339)
	s.UpdatedAt = &updated
	return nil
}

func (r *shipRepository) CreateWithCharacters(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ships (id, user_id, title, description, image_url, thumbnail_url) VALUES ($1, $2, $3, $4, $5, $6)`,
			id, userID, title, description, "", "",
		); err != nil {
			return fmt.Errorf("create ship: %w", err)
		}
		return insertShipCharactersTx(ctx, tx, id, characters)
	})
}

func insertShipCharactersTx(ctx context.Context, tx *sql.Tx, shipID uuid.UUID, characters []dto.ShipCharacter) error {
	for i, c := range characters {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ship_characters (ship_id, series, character_id, character_name, sort_order) VALUES ($1, $2, $3, $4, $5)`,
			shipID, c.Series, c.CharacterID, strings.TrimSpace(c.CharacterName), i,
		); err != nil {
			return fmt.Errorf("add ship character: %w", err)
		}
	}
	return nil
}

func (r *shipRepository) UpdateWithCharacters(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, characters []dto.ShipCharacter, asAdmin bool) error {
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var res sql.Result
		var err error
		if asAdmin {
			res, err = tx.ExecContext(ctx,
				`UPDATE ships SET title = $1, description = $2, updated_at = NOW() WHERE id = $3`,
				title, description, id,
			)
		} else {
			res, err = tx.ExecContext(ctx,
				`UPDATE ships SET title = $1, description = $2, updated_at = NOW() WHERE id = $3 AND user_id = $4`,
				title, description, id, userID,
			)
		}
		if err != nil {
			return fmt.Errorf("update ship: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("ship not found or not owned")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM ship_characters WHERE ship_id = $1`, id); err != nil {
			return fmt.Errorf("delete ship characters: %w", err)
		}
		return insertShipCharactersTx(ctx, tx, id, characters)
	})
}

func (r *shipRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ships SET image_url = $1, thumbnail_url = $2 WHERE id = $3`,
		imageURL, thumbnailURL, id,
	)
	if err != nil {
		return fmt.Errorf("update ship image: %w", err)
	}
	return nil
}

func (r *shipRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ships WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete ship: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ship not found or not owned")
	}
	return nil
}

func (r *shipRepository) DeleteAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ships WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete ship: %w", err)
	}
	return nil
}

func (r *shipRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*model.ShipRow, error) {
	var s model.ShipRow
	err := scanShipRow(r.db.QueryRowContext(ctx, shipSelectBase+` WHERE s.id = $2`, viewerID, id), &s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ship: %w", err)
	}
	return &s, nil
}

func (r *shipRepository) GetAuthorID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM ships WHERE id = $1`, shipID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get ship author: %w", err)
	}
	return userID, nil
}

func (r *shipRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ShipRow, int, error) {
	buildWhere := func(startIdx int) (string, []interface{}, int) {
		idx := startIdx
		next := func() string {
			s := fmt.Sprintf("$%d", idx)
			idx++
			return s
		}
		parts := []string{"1=1"}
		var args []interface{}
		if series != "" {
			parts = append(parts, "EXISTS(SELECT 1 FROM ship_characters WHERE ship_id = s.id AND series = "+next()+")")
			args = append(args, series)
		}
		if characterID != "" {
			parts = append(parts, "EXISTS(SELECT 1 FROM ship_characters WHERE ship_id = s.id AND character_id = "+next()+")")
			args = append(args, characterID)
		}
		if crackshipsOnly {
			parts = append(parts, fmt.Sprintf("COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0) <= %d", dto.CrackshipThreshold))
		}
		exclSQL, exclArgs := ExcludeClause("s.user_id", excludeUserIDs, idx)
		idx += len(exclArgs)
		args = append(args, exclArgs...)
		return " WHERE " + strings.Join(parts, " AND ") + exclSQL, args, idx
	}

	countWhere, countArgs, _ := buildWhere(1)
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ships s`+countWhere, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ships: %w", err)
	}

	listWhere, listArgs, nextIdx := buildWhere(2)
	limitPH := fmt.Sprintf("$%d", nextIdx)
	offsetPH := fmt.Sprintf("$%d", nextIdx+1)
	orderClause := shipOrderClause(sort)
	query := shipSelectBase + listWhere + orderClause + ` LIMIT ` + limitPH + ` OFFSET ` + offsetPH

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, listArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ships: %w", err)
	}
	defer rows.Close()

	var ships []model.ShipRow
	for rows.Next() {
		var s model.ShipRow
		if err := scanShipRow(rows, &s); err != nil {
			return nil, 0, fmt.Errorf("scan ship: %w", err)
		}
		ships = append(ships, s)
	}
	return ships, total, rows.Err()
}

func shipOrderClause(sort string) string {
	voteScore := `COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0)`
	switch sort {
	case "top":
		return ` ORDER BY ` + voteScore + ` DESC, s.created_at DESC`
	case "crackship":
		return ` ORDER BY ` + voteScore + ` ASC, s.created_at DESC`
	case "controversial":
		return ` ORDER BY (
			(SELECT COUNT(*) FROM ship_votes WHERE ship_id = s.id AND value = 1) *
			(SELECT COUNT(*) FROM ship_votes WHERE ship_id = s.id AND value = -1)
		) DESC, s.created_at DESC`
	case "comments":
		return ` ORDER BY (SELECT COUNT(*) FROM ship_comments WHERE ship_id = s.id) DESC, s.created_at DESC`
	case "old":
		return ` ORDER BY s.created_at ASC`
	default:
		return ` ORDER BY s.created_at DESC`
	}
}

func (r *shipRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]model.ShipRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ships WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user ships: %w", err)
	}

	query := shipSelectBase + ` WHERE s.user_id = $2 ORDER BY s.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, viewerID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user ships: %w", err)
	}
	defer rows.Close()

	var ships []model.ShipRow
	for rows.Next() {
		var s model.ShipRow
		if err := scanShipRow(rows, &s); err != nil {
			return nil, 0, fmt.Errorf("scan ship: %w", err)
		}
		ships = append(ships, s)
	}
	return ships, total, rows.Err()
}

func (r *shipRepository) GetCharacters(ctx context.Context, shipID uuid.UUID) ([]model.ShipCharacterRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, ship_id, series, character_id, character_name, sort_order FROM ship_characters WHERE ship_id = $1 ORDER BY sort_order ASC`,
		shipID,
	)
	if err != nil {
		return nil, fmt.Errorf("get ship characters: %w", err)
	}
	defer rows.Close()

	var chars []model.ShipCharacterRow
	for rows.Next() {
		var c model.ShipCharacterRow
		if err := rows.Scan(&c.ID, &c.ShipID, &c.Series, &c.CharacterID, &c.CharacterName, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship character: %w", err)
		}
		chars = append(chars, c)
	}
	return chars, rows.Err()
}

func (r *shipRepository) GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID) (map[uuid.UUID][]model.ShipCharacterRow, error) {
	if len(shipIDs) == 0 {
		return nil, nil
	}

	placeholders := "$1"
	args := []interface{}{shipIDs[0]}
	for i, id := range shipIDs[1:] {
		placeholders += fmt.Sprintf(", $%d", i+2)
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, ship_id, series, character_id, character_name, sort_order FROM ship_characters WHERE ship_id IN (`+placeholders+`) ORDER BY sort_order ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get ship characters: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.ShipCharacterRow)
	for rows.Next() {
		var c model.ShipCharacterRow
		if err := rows.Scan(&c.ID, &c.ShipID, &c.Series, &c.CharacterID, &c.CharacterName, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship character: %w", err)
		}
		result[c.ShipID] = append(result[c.ShipID], c)
	}
	return result, rows.Err()
}

func (r *shipRepository) Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error {
	if value == 0 {
		_, err := r.db.ExecContext(ctx,
			`DELETE FROM ship_votes WHERE user_id = $1 AND ship_id = $2`,
			userID, shipID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_votes (user_id, ship_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, ship_id) DO UPDATE SET value = EXCLUDED.value`,
		userID, shipID, value,
	)
	if err != nil {
		return fmt.Errorf("vote ship: %w", err)
	}
	return nil
}

func (r *shipRepository) CreateComment(ctx context.Context, id uuid.UUID, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_comments (id, ship_id, parent_id, user_id, body) VALUES ($1, $2, $3, $4, $5)`,
		id, shipID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE ship_comments SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		body, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update ship comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *shipRepository) UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ship_comments SET body = $1, updated_at = NOW() WHERE id = $2`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ship_comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete ship comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found or not owned")
	}
	return nil
}

func (r *shipRepository) DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ship_comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("admin delete ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]model.ShipCommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs, 2)
	var total int
	countArgs := []interface{}{shipID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ship_comments WHERE ship_id = $1`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ship comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs, 3)
	limitPH := fmt.Sprintf("$%d", 3+len(exclArgs2))
	offsetPH := fmt.Sprintf("$%d", 4+len(exclArgs2))
	queryArgs := []interface{}{viewerID, shipID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.ship_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM ship_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM ship_comment_likes WHERE comment_id = c.id AND user_id = $1)
		FROM ship_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.ship_id = $2`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT `+limitPH+` OFFSET `+offsetPH,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get ship comments: %w", err)
	}
	defer rows.Close()

	var comments []model.ShipCommentRow
	for rows.Next() {
		var c model.ShipCommentRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.ShipID, &c.ParentID, &c.UserID, &c.Body, &createdAt, &updatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &c.UserLiked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan ship comment: %w", err)
		}
		c.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		c.UpdatedAt = timePtrToString(updatedAt)
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *shipRepository) GetCommentShipID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var shipID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT ship_id FROM ship_comments WHERE id = $1`, commentID).Scan(&shipID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get ship comment ship id: %w", err)
	}
	return shipID, nil
}

func (r *shipRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM ship_comments WHERE id = $1`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get ship comment author: %w", err)
	}
	return userID, nil
}

func (r *shipRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM ship_comment_likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO ship_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("add ship comment media: %w", err)
	}
	return id, nil
}

func (r *shipRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE ship_comment_media SET media_url = $1 WHERE id = $2`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update ship comment media url: %w", err)
	}
	return nil
}

func (r *shipRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE ship_comment_media SET thumbnail_url = $1 WHERE id = $2`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update ship comment media thumbnail: %w", err)
	}
	return nil
}

func (r *shipRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]model.ShipCommentMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM ship_comment_media WHERE comment_id = $1 ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get ship comment media: %w", err)
	}
	defer rows.Close()

	var media []model.ShipCommentMediaRow
	for rows.Next() {
		var m model.ShipCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *shipRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]model.ShipCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "$1"
	args := []interface{}{commentIDs[0]}
	for i, id := range commentIDs[1:] {
		placeholders += fmt.Sprintf(", $%d", i+2)
		args = append(args, id)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM ship_comment_media WHERE comment_id IN (`+placeholders+`) ORDER BY sort_order`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get ship comment media: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.ShipCommentMediaRow)
	for rows.Next() {
		var m model.ShipCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}
