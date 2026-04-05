package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	ShipRepository interface {
		Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, imageURL string, thumbnailURL string) error
		Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string) error
		UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, description string) error
		UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error
		Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*ShipRow, error)
		GetAuthorID(ctx context.Context, shipID uuid.UUID) (uuid.UUID, error)
		List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID) ([]ShipRow, int, error)
		ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]ShipRow, int, error)

		AddCharacter(ctx context.Context, shipID uuid.UUID, series string, characterID string, characterName string, sortOrder int) error
		DeleteCharacters(ctx context.Context, shipID uuid.UUID) error
		GetCharacters(ctx context.Context, shipID uuid.UUID) ([]ShipCharacterRow, error)
		GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID) (map[uuid.UUID][]ShipCharacterRow, error)

		Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error

		CreateComment(ctx context.Context, id uuid.UUID, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error
		UpdateCommentAsAdmin(ctx context.Context, id uuid.UUID, body string) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		DeleteCommentAsAdmin(ctx context.Context, id uuid.UUID) error
		GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]ShipCommentRow, int, error)
		GetCommentShipID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error

		AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error
		UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]ShipCommentMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]ShipCommentMediaRow, error)
	}

	shipRepository struct {
		db *sql.DB
	}
)

const shipSelectBase = `
	SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
		u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
		COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0),
		COALESCE((SELECT value FROM ship_votes WHERE ship_id = s.id AND user_id = ?), 0),
		(SELECT COUNT(*) FROM ship_comments WHERE ship_id = s.id)
	FROM ships s
	JOIN users u ON s.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = s.user_id`

func scanShipRow(row interface{ Scan(...interface{}) error }, s *ShipRow) error {
	return row.Scan(
		&s.ID, &s.UserID, &s.Title, &s.Description, &s.ImageURL, &s.ThumbnailURL, &s.CreatedAt, &s.UpdatedAt,
		&s.AuthorUsername, &s.AuthorDisplayName, &s.AuthorAvatarURL, &s.AuthorRole,
		&s.VoteScore, &s.UserVote, &s.CommentCount,
	)
}

func (r *shipRepository) Create(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string, imageURL string, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ships (id, user_id, title, description, image_url, thumbnail_url) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, title, description, imageURL, thumbnailURL,
	)
	if err != nil {
		return fmt.Errorf("create ship: %w", err)
	}
	return nil
}

func (r *shipRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title string, description string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE ships SET title = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		title, description, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update ship: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ship not found or not owned")
	}
	return nil
}

func (r *shipRepository) UpdateAsAdmin(ctx context.Context, id uuid.UUID, title string, description string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ships SET title = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, description, id,
	)
	if err != nil {
		return fmt.Errorf("admin update ship: %w", err)
	}
	return nil
}

func (r *shipRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ships SET image_url = ?, thumbnail_url = ? WHERE id = ?`,
		imageURL, thumbnailURL, id,
	)
	if err != nil {
		return fmt.Errorf("update ship image: %w", err)
	}
	return nil
}

func (r *shipRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ships WHERE id = ? AND user_id = ?`, id, userID)
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
	_, err := r.db.ExecContext(ctx, `DELETE FROM ships WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete ship: %w", err)
	}
	return nil
}

func (r *shipRepository) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*ShipRow, error) {
	var s ShipRow
	err := scanShipRow(r.db.QueryRowContext(ctx, shipSelectBase+` WHERE s.id = ?`, viewerID, id), &s)
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
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM ships WHERE id = ?`, shipID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get ship author: %w", err)
	}
	return userID, nil
}

func (r *shipRepository) List(ctx context.Context, viewerID uuid.UUID, sort string, crackshipsOnly bool, series string, characterID string, limit, offset int, excludeUserIDs []uuid.UUID) ([]ShipRow, int, error) {
	whereParts := []string{"1=1"}
	var args []interface{}

	if series != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM ship_characters WHERE ship_id = s.id AND series = ?)")
		args = append(args, series)
	}

	if characterID != "" {
		whereParts = append(whereParts, "EXISTS(SELECT 1 FROM ship_characters WHERE ship_id = s.id AND character_id = ?)")
		args = append(args, characterID)
	}

	if crackshipsOnly {
		whereParts = append(whereParts, fmt.Sprintf("COALESCE((SELECT SUM(value) FROM ship_votes WHERE ship_id = s.id), 0) <= %d", dto.CrackshipThreshold))
	}

	exclSQL, exclArgs := ExcludeClause("s.user_id", excludeUserIDs)
	whereClause := " WHERE " + strings.Join(whereParts, " AND ") + exclSQL

	var total int
	countArgs := append([]interface{}{}, args...)
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ships s`+whereClause, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ships: %w", err)
	}

	orderClause := shipOrderClause(sort)
	query := shipSelectBase + whereClause + orderClause + ` LIMIT ? OFFSET ?`

	queryArgs := []interface{}{viewerID}
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, exclArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ships: %w", err)
	}
	defer rows.Close()

	var ships []ShipRow
	for rows.Next() {
		var s ShipRow
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

func (r *shipRepository) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]ShipRow, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ships WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user ships: %w", err)
	}

	query := shipSelectBase + ` WHERE s.user_id = ? ORDER BY s.created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, viewerID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user ships: %w", err)
	}
	defer rows.Close()

	var ships []ShipRow
	for rows.Next() {
		var s ShipRow
		if err := scanShipRow(rows, &s); err != nil {
			return nil, 0, fmt.Errorf("scan ship: %w", err)
		}
		ships = append(ships, s)
	}
	return ships, total, rows.Err()
}

func (r *shipRepository) AddCharacter(ctx context.Context, shipID uuid.UUID, series string, characterID string, characterName string, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_characters (ship_id, series, character_id, character_name, sort_order) VALUES (?, ?, ?, ?, ?)`,
		shipID, series, characterID, characterName, sortOrder,
	)
	if err != nil {
		return fmt.Errorf("add ship character: %w", err)
	}
	return nil
}

func (r *shipRepository) DeleteCharacters(ctx context.Context, shipID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ship_characters WHERE ship_id = ?`, shipID)
	if err != nil {
		return fmt.Errorf("delete ship characters: %w", err)
	}
	return nil
}

func (r *shipRepository) GetCharacters(ctx context.Context, shipID uuid.UUID) ([]ShipCharacterRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, ship_id, series, character_id, character_name, sort_order FROM ship_characters WHERE ship_id = ? ORDER BY sort_order ASC`,
		shipID,
	)
	if err != nil {
		return nil, fmt.Errorf("get ship characters: %w", err)
	}
	defer rows.Close()

	var chars []ShipCharacterRow
	for rows.Next() {
		var c ShipCharacterRow
		if err := rows.Scan(&c.ID, &c.ShipID, &c.Series, &c.CharacterID, &c.CharacterName, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship character: %w", err)
		}
		chars = append(chars, c)
	}
	return chars, rows.Err()
}

func (r *shipRepository) GetCharactersBatch(ctx context.Context, shipIDs []uuid.UUID) (map[uuid.UUID][]ShipCharacterRow, error) {
	if len(shipIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{shipIDs[0]}
	for _, id := range shipIDs[1:] {
		placeholders += ", ?"
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

	result := make(map[uuid.UUID][]ShipCharacterRow)
	for rows.Next() {
		var c ShipCharacterRow
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
			`DELETE FROM ship_votes WHERE user_id = ? AND ship_id = ?`,
			userID, shipID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_votes (user_id, ship_id, value) VALUES (?, ?, ?)
		ON CONFLICT(user_id, ship_id) DO UPDATE SET value = ?`,
		userID, shipID, value, value,
	)
	if err != nil {
		return fmt.Errorf("vote ship: %w", err)
	}
	return nil
}

func (r *shipRepository) CreateComment(ctx context.Context, id uuid.UUID, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_comments (id, ship_id, parent_id, user_id, body) VALUES (?, ?, ?, ?, ?)`,
		id, shipID, parentID, userID, body,
	)
	if err != nil {
		return fmt.Errorf("create ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, body string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE ship_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
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
		`UPDATE ship_comments SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		body, id,
	)
	if err != nil {
		return fmt.Errorf("admin update ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ship_comments WHERE id = ? AND user_id = ?`, id, userID)
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
	_, err := r.db.ExecContext(ctx, `DELETE FROM ship_comments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("admin delete ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) GetComments(ctx context.Context, shipID uuid.UUID, viewerID uuid.UUID, limit, offset int, excludeUserIDs []uuid.UUID) ([]ShipCommentRow, int, error) {
	exclSQL, exclArgs := ExcludeClause("user_id", excludeUserIDs)
	var total int
	countArgs := []interface{}{shipID}
	countArgs = append(countArgs, exclArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ship_comments WHERE ship_id = ?`+exclSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ship comments: %w", err)
	}

	exclSQL2, exclArgs2 := ExcludeClause("c.user_id", excludeUserIDs)
	queryArgs := []interface{}{viewerID, shipID}
	queryArgs = append(queryArgs, exclArgs2...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.ship_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
			u.username, u.display_name, u.avatar_url, COALESCE(r.role, ''),
			(SELECT COUNT(*) FROM ship_comment_likes WHERE comment_id = c.id),
			EXISTS(SELECT 1 FROM ship_comment_likes WHERE comment_id = c.id AND user_id = ?)
		FROM ship_comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = c.user_id
		WHERE c.ship_id = ?`+exclSQL2+`
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get ship comments: %w", err)
	}
	defer rows.Close()

	var comments []ShipCommentRow
	for rows.Next() {
		var c ShipCommentRow
		var userLikedInt int
		if err := rows.Scan(
			&c.ID, &c.ShipID, &c.ParentID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&c.AuthorUsername, &c.AuthorDisplayName, &c.AuthorAvatarURL, &c.AuthorRole,
			&c.LikeCount, &userLikedInt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan ship comment: %w", err)
		}
		c.UserLiked = userLikedInt == 1
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *shipRepository) GetCommentShipID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var shipID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT ship_id FROM ship_comments WHERE id = ?`, commentID).Scan(&shipID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get ship comment ship id: %w", err)
	}
	return shipID, nil
}

func (r *shipRepository) GetCommentAuthorID(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM ship_comments WHERE id = ?`, commentID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get ship comment author: %w", err)
	}
	return userID, nil
}

func (r *shipRepository) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO ship_comment_likes (user_id, comment_id) VALUES (?, ?)`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("like ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM ship_comment_likes WHERE user_id = ? AND comment_id = ?`,
		userID, commentID,
	)
	if err != nil {
		return fmt.Errorf("unlike ship comment: %w", err)
	}
	return nil
}

func (r *shipRepository) AddCommentMedia(ctx context.Context, commentID uuid.UUID, mediaURL string, mediaType string, thumbnailURL string, sortOrder int) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ship_comment_media (comment_id, media_url, media_type, thumbnail_url, sort_order) VALUES (?, ?, ?, ?, ?)`,
		commentID, mediaURL, mediaType, thumbnailURL, sortOrder,
	)
	if err != nil {
		return 0, fmt.Errorf("add ship comment media: %w", err)
	}
	return res.LastInsertId()
}

func (r *shipRepository) UpdateCommentMediaURL(ctx context.Context, id int64, mediaURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE ship_comment_media SET media_url = ? WHERE id = ?`, mediaURL, id)
	if err != nil {
		return fmt.Errorf("update ship comment media url: %w", err)
	}
	return nil
}

func (r *shipRepository) UpdateCommentMediaThumbnail(ctx context.Context, id int64, thumbnailURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE ship_comment_media SET thumbnail_url = ? WHERE id = ?`, thumbnailURL, id)
	if err != nil {
		return fmt.Errorf("update ship comment media thumbnail: %w", err)
	}
	return nil
}

func (r *shipRepository) GetCommentMedia(ctx context.Context, commentID uuid.UUID) ([]ShipCommentMediaRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, comment_id, media_url, media_type, thumbnail_url, sort_order FROM ship_comment_media WHERE comment_id = ? ORDER BY sort_order`,
		commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get ship comment media: %w", err)
	}
	defer rows.Close()

	var media []ShipCommentMediaRow
	for rows.Next() {
		var m ShipCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship comment media: %w", err)
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *shipRepository) GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]ShipCommentMediaRow, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}

	placeholders := "?"
	args := []interface{}{commentIDs[0]}
	for _, id := range commentIDs[1:] {
		placeholders += ", ?"
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

	result := make(map[uuid.UUID][]ShipCommentMediaRow)
	for rows.Next() {
		var m ShipCommentMediaRow
		if err := rows.Scan(&m.ID, &m.CommentID, &m.MediaURL, &m.MediaType, &m.ThumbnailURL, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("scan ship comment media: %w", err)
		}
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, rows.Err()
}
