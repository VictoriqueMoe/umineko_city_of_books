package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
)

type (
	artDAO struct {
		db *sql.DB
		*ownedDAO
		*commentDAO[uuid.UUID]
		*likeDAO
		*viewDAO
	}
)

const artSelectBase = `
	SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
		a.gallery_id, a.created_at, a.updated_at,
		u.username, u.display_name, u.avatar_url,
		COALESCE(r.role, ''),
		(SELECT COUNT(*) FROM art_likes WHERE art_id = a.id),
		(SELECT COUNT(*) FROM art_comments WHERE art_id = a.id),
		a.view_count,
		EXISTS(SELECT 1 FROM art_likes WHERE art_id = a.id AND user_id = $1),
		a.is_spoiler
	FROM art a
	JOIN users u ON a.user_id = u.id
	LEFT JOIN user_roles r ON r.user_id = a.user_id`

func scanArtRow(row interface{ Scan(...any) error }, a *model.ArtRow) error {
	var createdAt time.Time
	var updatedAt *time.Time
	err := row.Scan(
		&a.ID, &a.UserID, &a.Corner, &a.ArtType, &a.Title, &a.Description, &a.ImageURL, &a.ThumbnailURL,
		&a.GalleryID, &createdAt, &updatedAt,
		&a.AuthorUsername, &a.AuthorDisplayName, &a.AuthorAvatarURL,
		&a.AuthorRole,
		&a.LikeCount, &a.CommentCount, &a.ViewCount, &a.UserLiked, &a.IsSpoiler,
	)
	if err != nil {
		return err
	}
	a.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	a.UpdatedAt = timePtrToString(updatedAt)
	return nil
}

func (r *artDAO) CreateArt(ctx context.Context, spec repository.NewArt, tx ...*sql.Tx) (*model.ArtRow, error) {
	var created model.ArtRow

	if err := scanArtRow(txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH a AS (
		     INSERT INTO art (user_id, corner, art_type, title, description, image_url, thumbnail_url, is_spoiler)
		     VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		     RETURNING *
		 )
		 SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
		        a.gallery_id, a.created_at, a.updated_at,
		        u.username, u.display_name, u.avatar_url,
		        COALESCE(r.role, ''),
		        0, 0, a.view_count, FALSE, a.is_spoiler
		 FROM a
		 JOIN users u ON a.user_id = u.id
		 LEFT JOIN user_roles r ON r.user_id = a.user_id`,
		spec.UserID, spec.Corner, spec.ArtType, spec.Title, spec.Description, spec.ImageURL, spec.ThumbnailURL, spec.IsSpoiler,
	), &created); err != nil {
		return nil, fmt.Errorf("create art: %w", err)
	}

	return &created, nil
}

func (r *artDAO) UpdateArt(ctx context.Context, spec repository.ArtUpdate, tx ...*sql.Tx) error {
	var res sql.Result
	var err error

	if spec.AsAdmin {
		res, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE art SET title = $1, description = $2, is_spoiler = $3, updated_at = NOW() WHERE id = $4`,
			spec.Title, spec.Description, spec.IsSpoiler, spec.ID,
		)
	} else {
		res, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE art SET title = $1, description = $2, is_spoiler = $3, updated_at = NOW() WHERE id = $4 AND user_id = $5`,
			spec.Title, spec.Description, spec.IsSpoiler, spec.ID, spec.UserID,
		)
	}
	if err != nil {
		return fmt.Errorf("update art: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("art not found or not owned")
	}

	return nil
}

func (r *artDAO) InsertTags(ctx context.Context, artID uuid.UUID, tags []string, tx ...*sql.Tx) error {
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}

		if _, err := txOrDB(r.db, tx).ExecContext(ctx,
			`INSERT INTO art_tags (art_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			artID, tag,
		); err != nil {
			return fmt.Errorf("add art tag: %w", err)
		}
	}

	return nil
}

func (r *artDAO) DeleteTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM art_tags WHERE art_id = $1`, artID); err != nil {
		return fmt.Errorf("delete art tags: %w", err)
	}

	return nil
}

func (r *artDAO) GetByID(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, tx ...*sql.Tx) (*model.ArtRow, error) {
	var a model.ArtRow
	err := scanArtRow(txOrDB(r.db, tx).QueryRowContext(ctx, artSelectBase+` WHERE a.id = $2`, viewerID, id), &a)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get art: %w", err)
	}
	return &a, nil
}

func artOrderClause(sort string) string {
	switch sort {
	case "popular":
		return ` ORDER BY (SELECT COUNT(*) FROM art_likes WHERE art_id = a.id) DESC, a.created_at DESC`
	case "views":
		return ` ORDER BY a.view_count DESC, a.created_at DESC`
	default:
		return ` ORDER BY a.created_at DESC`
	}
}

func (r *artDAO) ListAll(ctx context.Context, viewerID uuid.UUID, corner string, artType string, search string, tag string, sort string, limit, offset int, excludeUserIDs []uuid.UUID, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	var total int
	buildWhere := func(startIdx int) (string, []any, int) {
		idx := startIdx
		next := func() string {
			s := fmt.Sprintf("$%d", idx)
			idx++
			return s
		}
		parts := []string{"a.corner = " + next()}
		args := []any{corner}
		if artType != "" {
			parts = append(parts, "a.art_type = "+next())
			args = append(args, artType)
		}
		if search != "" {
			parts = append(parts, "(a.title LIKE "+next()+" OR a.description LIKE "+next()+" OR u.display_name LIKE "+next()+" OR u.username LIKE "+next()+")")
			like := "%" + search + "%"
			args = append(args, like, like, like, like)
		}
		if tag != "" {
			parts = append(parts, "EXISTS(SELECT 1 FROM art_tags WHERE art_id = a.id AND tag = "+next()+")")
			args = append(args, tag)
		}
		exclSQL, exclArgs := ExcludeClause("a.user_id", excludeUserIDs, idx)
		idx += len(exclArgs)
		args = append(args, exclArgs...)
		return " WHERE " + strings.Join(parts, " AND ") + exclSQL, args, idx
	}

	countWhere, countArgs, _ := buildWhere(1)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM art a JOIN users u ON a.user_id = u.id`+countWhere, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count art: %w", err)
	}

	listWhere, listArgs, nextIdx := buildWhere(2)
	limitPH := fmt.Sprintf("$%d", nextIdx)
	offsetPH := fmt.Sprintf("$%d", nextIdx+1)

	orderClause := artOrderClause(sort)
	query := artSelectBase + listWhere + orderClause + ` LIMIT ` + limitPH + ` OFFSET ` + offsetPH

	queryArgs := []any{viewerID}
	queryArgs = append(queryArgs, listArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list art: %w", err)
	}
	defer rows.Close()

	var arts []model.ArtRow
	for rows.Next() {
		var a model.ArtRow
		if err := scanArtRow(rows, &a); err != nil {
			return nil, 0, fmt.Errorf("scan art: %w", err)
		}
		arts = append(arts, a)
	}
	return arts, total, rows.Err()
}

func (r *artDAO) ListByUser(ctx context.Context, userID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM art WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user art: %w", err)
	}

	query := artSelectBase + ` WHERE a.user_id = $2 ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, viewerID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user art: %w", err)
	}
	defer rows.Close()

	var arts []model.ArtRow
	for rows.Next() {
		var a model.ArtRow
		if err := scanArtRow(rows, &a); err != nil {
			return nil, 0, fmt.Errorf("scan art: %w", err)
		}
		arts = append(arts, a)
	}
	return arts, total, rows.Err()
}

func (r *artDAO) GetArtAuthorID(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.ownedDAO.GetAuthorID(ctx, artID, tx...)
}

func (r *artDAO) GetImageURL(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) (string, error) {
	var url string
	err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT image_url FROM art WHERE id = $1`, artID).Scan(&url)
	if err != nil {
		return "", fmt.Errorf("get art image url: %w", err)
	}
	return url, nil
}

func (r *artDAO) GetArtImagePaths(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var (
		imageURL     string
		thumbnailURL string
	)

	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT image_url, thumbnail_url FROM art WHERE id = $1`, artID,
	).Scan(&imageURL, &thumbnailURL); err != nil {
		return nil, fmt.Errorf("get art image paths: %w", err)
	}

	var paths []string

	if imageURL != "" {
		paths = append(paths, imageURL)
	}

	if thumbnailURL != "" {
		paths = append(paths, thumbnailURL)
	}

	return paths, nil
}

func (r *artDAO) ListGalleryArtImages(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) ([]repository.ArtImageRef, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, image_url, thumbnail_url FROM art WHERE gallery_id = $1 AND user_id = $2`,
		galleryID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list gallery art images: %w", err)
	}
	defer rows.Close()

	var refs []repository.ArtImageRef
	for rows.Next() {
		var ref repository.ArtImageRef
		if err := rows.Scan(&ref.ArtID, &ref.ImageURL, &ref.ThumbnailURL); err != nil {
			return nil, fmt.Errorf("scan gallery art image: %w", err)
		}

		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

func (r *artDAO) GetTags(ctx context.Context, artID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT tag FROM art_tags WHERE art_id = $1 ORDER BY tag`, artID)
	if err != nil {
		return nil, fmt.Errorf("get art tags: %w", err)
	}

	return utils.ScanStrings(rows, "art tag")
}

func (r *artDAO) GetTagsBatch(ctx context.Context, artIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]string, error) {
	if len(artIDs) == 0 {
		return nil, nil
	}

	placeholders, args := utils.PlaceholderArgs(artIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT art_id, tag FROM art_tags WHERE art_id IN (`+strings.Join(placeholders, ", ")+`) ORDER BY tag`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get art tags: %w", err)
	}

	return utils.ScanGroups[uuid.UUID, string](rows, "art tag")
}

func (r *artDAO) GetPopularTags(ctx context.Context, corner string, limit int, tx ...*sql.Tx) ([]model.TagCount, error) {
	query := `SELECT t.tag, COUNT(*) as cnt FROM art_tags t JOIN art a ON t.art_id = a.id`
	var args []any

	if corner != "" {
		query += ` WHERE a.corner = $1`
		args = append(args, corner)
		query += ` GROUP BY t.tag ORDER BY cnt DESC LIMIT $2`
	} else {
		query += ` GROUP BY t.tag ORDER BY cnt DESC LIMIT $1`
	}
	args = append(args, limit)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get popular tags: %w", err)
	}
	defer rows.Close()

	var tags []model.TagCount
	for rows.Next() {
		var t model.TagCount
		if err := rows.Scan(&t.Tag, &t.Count); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *artDAO) GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT corner, COUNT(*) FROM art GROUP BY corner`)
	if err != nil {
		return nil, fmt.Errorf("art corner counts: %w", err)
	}

	return utils.ScanMap[string, int](rows, "art corner count")
}

func (r *artDAO) CountUserArtToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM art WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user art today: %w", err)
	}
	return count, nil
}

func (r *artDAO) AddCommentMedia(ctx context.Context, spec repository.NewArtCommentMedia, tx ...*sql.Tx) (int64, error) {
	return r.commentDAO.AddCommentMedia(ctx, spec.CommentID, spec.MediaURL, spec.MediaType, spec.ThumbnailURL, spec.SortOrder, tx...)
}

func (r *artDAO) SetGallery(ctx context.Context, artID uuid.UUID, userID uuid.UUID, galleryID *uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE art SET gallery_id = $1 WHERE id = $2 AND user_id = $3`,
		galleryID, artID, userID,
	)
	if err != nil {
		return fmt.Errorf("set art gallery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("art not found or not owned")
	}
	return nil
}

func (r *artDAO) CreateGallery(ctx context.Context, userID uuid.UUID, name string, description string, tx ...*sql.Tx) (*model.GalleryRow, error) {
	var g model.GalleryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH g AS (
		     INSERT INTO galleries (user_id, name, description) VALUES ($1, $2, $3)
		     RETURNING *
		 )
		 SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
		        ''::text, ''::text, 0,
		        g.created_at, g.updated_at,
		        u.username, u.display_name, u.avatar_url
		 FROM g
		 JOIN users u ON g.user_id = u.id`,
		userID, name, description,
	).Scan(
		&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
		&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
		&createdAt, &updatedAt,
		&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
	)
	if err != nil {
		return nil, fmt.Errorf("create gallery: %w", err)
	}

	g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	g.UpdatedAt = timePtrToString(updatedAt)

	return &g, nil
}

func (r *artDAO) UpdateGallery(ctx context.Context, id uuid.UUID, userID uuid.UUID, name string, description string, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE galleries SET name = $1, description = $2, updated_at = NOW() WHERE id = $3 AND user_id = $4`,
		name, description, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update gallery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}
	return nil
}

func (r *artDAO) SetGalleryCover(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, coverArtID *uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE galleries SET cover_art_id = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		coverArtID, galleryID, userID,
	)
	if err != nil {
		return fmt.Errorf("set gallery cover: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}
	return nil
}

func (r *artDAO) DeleteArtInGallery(ctx context.Context, galleryID uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	if _, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM art WHERE gallery_id = $1 AND user_id = $2`,
		galleryID, userID,
	); err != nil {
		return fmt.Errorf("delete art in gallery: %w", err)
	}

	return nil
}

func (r *artDAO) DeleteGalleryRow(ctx context.Context, id uuid.UUID, userID uuid.UUID, tx ...*sql.Tx) error {
	res, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM galleries WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete gallery: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("gallery not found or not owned")
	}

	return nil
}

func (r *artDAO) GetGalleryByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.GalleryRow, error) {
	var g model.GalleryRow
	var createdAt time.Time
	var updatedAt *time.Time
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
			COALESCE(a.image_url, ''), COALESCE(a.thumbnail_url, ''),
			(SELECT COUNT(*) FROM art WHERE gallery_id = g.id),
			g.created_at, g.updated_at,
			u.username, u.display_name, u.avatar_url
		FROM galleries g
		JOIN users u ON g.user_id = u.id
		LEFT JOIN art a ON g.cover_art_id = a.id
		WHERE g.id = $1`,
		id,
	).Scan(
		&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
		&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
		&createdAt, &updatedAt,
		&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get gallery: %w", err)
	}
	g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	g.UpdatedAt = timePtrToString(updatedAt)
	return &g, nil
}

func (r *artDAO) ListGalleriesByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.GalleryRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
			COALESCE(a.image_url, ''), COALESCE(a.thumbnail_url, ''),
			(SELECT COUNT(*) FROM art WHERE gallery_id = g.id),
			g.created_at, g.updated_at,
			u.username, u.display_name, u.avatar_url
		FROM galleries g
		JOIN users u ON g.user_id = u.id
		LEFT JOIN art a ON g.cover_art_id = a.id
		WHERE g.user_id = $1
		ORDER BY g.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list galleries: %w", err)
	}
	defer rows.Close()

	var galleries []model.GalleryRow
	for rows.Next() {
		var g model.GalleryRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
			&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
			&createdAt, &updatedAt,
			&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
		); err != nil {
			return nil, fmt.Errorf("scan gallery: %w", err)
		}
		g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		g.UpdatedAt = timePtrToString(updatedAt)
		galleries = append(galleries, g)
	}
	return galleries, rows.Err()
}

func (r *artDAO) ListAllGalleries(ctx context.Context, corner string, tx ...*sql.Tx) ([]model.GalleryRow, error) {
	query := `SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
			COALESCE(a.image_url, ''), COALESCE(a.thumbnail_url, ''),
			(SELECT COUNT(*) FROM art WHERE gallery_id = g.id),
			g.created_at, g.updated_at,
			u.username, u.display_name, u.avatar_url
		FROM galleries g
		JOIN users u ON g.user_id = u.id
		LEFT JOIN art a ON g.cover_art_id = a.id`
	args := []any{}

	if corner != "" {
		query += ` WHERE EXISTS(SELECT 1 FROM art WHERE gallery_id = g.id AND corner = $1)`
		args = append(args, corner)
	}

	query += ` ORDER BY g.created_at DESC`

	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all galleries: %w", err)
	}
	defer rows.Close()

	var galleries []model.GalleryRow
	for rows.Next() {
		var g model.GalleryRow
		var createdAt time.Time
		var updatedAt *time.Time
		if err := rows.Scan(
			&g.ID, &g.UserID, &g.Name, &g.Description, &g.CoverArtID,
			&g.CoverImageURL, &g.CoverThumbnailURL, &g.ArtCount,
			&createdAt, &updatedAt,
			&g.AuthorUsername, &g.AuthorDisplayName, &g.AuthorAvatarURL,
		); err != nil {
			return nil, fmt.Errorf("scan gallery: %w", err)
		}
		g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		g.UpdatedAt = timePtrToString(updatedAt)
		galleries = append(galleries, g)
	}
	return galleries, rows.Err()
}

func (r *artDAO) GetGalleryPreviewImages(ctx context.Context, galleryID uuid.UUID, limit int, tx ...*sql.Tx) ([]repository.PreviewImage, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT thumbnail_url, image_url FROM art WHERE gallery_id = $1 ORDER BY created_at DESC LIMIT $2`,
		galleryID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get gallery preview images: %w", err)
	}
	defer rows.Close()

	var imgs []repository.PreviewImage
	for rows.Next() {
		var p repository.PreviewImage
		if err := rows.Scan(&p.ThumbnailURL, &p.ImageURL); err != nil {
			return nil, fmt.Errorf("scan preview image: %w", err)
		}
		imgs = append(imgs, p)
	}
	return imgs, rows.Err()
}

func (r *artDAO) ListArtInGallery(ctx context.Context, galleryID uuid.UUID, viewerID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.ArtRow, int, error) {
	var total int
	if err := txOrDB(r.db, tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM art WHERE gallery_id = $1`, galleryID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count gallery art: %w", err)
	}

	query := artSelectBase + ` WHERE a.gallery_id = $2 ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, query, viewerID, galleryID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list gallery art: %w", err)
	}
	defer rows.Close()

	var arts []model.ArtRow
	for rows.Next() {
		var a model.ArtRow
		if err := scanArtRow(rows, &a); err != nil {
			return nil, 0, fmt.Errorf("scan gallery art: %w", err)
		}
		arts = append(arts, a)
	}
	return arts, total, rows.Err()
}
