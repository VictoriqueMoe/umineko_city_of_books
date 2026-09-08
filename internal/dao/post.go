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
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/text"

	"github.com/google/uuid"
)

type (
	PostDAO interface {
		Create(ctx context.Context, s spec.NewPost, tx ...*sql.Tx) (*model.PostRow, error)
		UpdatePost(ctx context.Context, s spec.PostUpdate, tx ...*sql.Tx) error
		GetByID(ctx context.Context, s spec.PostLookup, tx ...*sql.Tx) (*model.PostRow, error)
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		IncrementViewCount(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		ListAll(ctx context.Context, q spec.PostFeedQuery, tx ...*sql.Tx) ([]model.PostRow, int, error)
		ListByFollowing(ctx context.Context, q spec.PostFollowingFeedQuery, tx ...*sql.Tx) ([]model.PostRow, int, error)
		ListByUser(ctx context.Context, q spec.PostUserPage, tx ...*sql.Tx) ([]model.PostRow, int, error)

		AddMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		DeleteMedia(ctx context.Context, s spec.MediaDeletion, tx ...*sql.Tx) (string, error)
		UpdateMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetMedia(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetMediaBatch(ctx context.Context, postIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)

		Like(ctx context.Context, s spec.Like, tx ...*sql.Tx) error
		Unlike(ctx context.Context, s spec.Like, tx ...*sql.Tx) error
		GetLikedBy(ctx context.Context, q spec.LikedByQuery, tx ...*sql.Tx) ([]model.PostLikeUser, error)
		RecordView(ctx context.Context, s spec.ViewRecord, tx ...*sql.Tx) (bool, error)
		GetPostAuthorID(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetSharedContentAuthor(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) (uuid.UUID, error)

		ResolveSuggestion(ctx context.Context, s spec.SuggestionResolution, tx ...*sql.Tx) error
		UnresolveSuggestion(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) error

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentByID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (*model.CommentRow, error)
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

		CountUserPostsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error)

		GetShareCount(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) (int, error)
		GetShareCountsBatch(ctx context.Context, q spec.SharedContentBatchRef, tx ...*sql.Tx) (map[string]int, error)
		IncrementShareCount(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) error
		DecrementShareCount(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) error
		GetSharedContentFields(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (*string, *string, error)
		GetSharedContentPreviews(refs []model.SharedContentRef, tx ...*sql.Tx) map[string]*dto.SharedContentPreview

		CreatePoll(ctx context.Context, s spec.NewPostPoll, tx ...*sql.Tx) (*model.PollRow, error)
		AddPollOption(ctx context.Context, s spec.NewPostPollOption, tx ...*sql.Tx) error
		GetPollByPostID(ctx context.Context, q spec.PostPollQuery, tx ...*sql.Tx) (*model.PollRow, []model.PollOptionRow, *int, error)
		GetPollsByPostIDs(ctx context.Context, q spec.PostPollBatchQuery, tx ...*sql.Tx) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error)
		VotePoll(ctx context.Context, s spec.PostPollVote, tx ...*sql.Tx) error
	}

	postDAO struct {
		db *sql.DB
		*ownedDAO
		*commentDAO[uuid.UUID]
		*likeDAO
		*mediaDAO
		*viewDAO
	}

	postJoinRow       = sqlcgen.GetPostByIDRow
	pollJoinRow       = sqlcgen.GetPostPollRow
	pollOptionJoinRow = sqlcgen.ListPostPollOptionsRow

	postFeedRow interface {
		sqlcgen.ListPostsFeedByNewRow |
			sqlcgen.ListPostsFeedByLikesRow |
			sqlcgen.ListPostsFeedByCommentsRow |
			sqlcgen.ListPostsFeedByViewsRow |
			sqlcgen.ListPostsFeedByRelevanceRow |
			sqlcgen.ListFollowingPostsByNewRow |
			sqlcgen.ListFollowingPostsByLikesRow |
			sqlcgen.ListFollowingPostsByCommentsRow |
			sqlcgen.ListFollowingPostsByViewsRow |
			sqlcgen.ListFollowingPostsByRelevanceRow
	}
)

func mapPostFeedRows[T postFeedRow](rows []T, err error) ([]model.PostRow, error) {
	if err != nil {
		return nil, err
	}

	var posts []model.PostRow
	for _, row := range rows {
		posts = append(posts, toPostRow(postJoinRow(row)))
	}

	return posts, nil
}

func postSearchPattern(search string) string {
	if search == "" {
		return ""
	}

	return "%" + search + "%"
}

func toPostRow(row postJoinRow) model.PostRow {
	out := model.PostRow{
		ID:                row.ID,
		UserID:            row.UserID,
		Corner:            row.Corner,
		Body:              row.Body,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         nullTimeToStringPtr(row.UpdatedAt),
		AuthorUsername:    row.Username,
		AuthorDisplayName: row.DisplayName,
		AuthorAvatarURL:   row.AvatarUrl,
		AuthorRole:        row.AuthorRole,
		LikeCount:         int(row.LikeCount),
		CommentCount:      int(row.CommentCount),
		UserLiked:         row.UserLiked,
		ViewCount:         int(row.ViewCount),
		ResolvedStatus:    row.ResolvedStatus,
	}

	if row.SharedContentID.Valid {
		out.SharedContentID = &row.SharedContentID.String
	}

	if row.SharedContentType.Valid {
		out.SharedContentType = &row.SharedContentType.String
	}

	return out
}

func toPollRow(row pollJoinRow) model.PollRow {
	return model.PollRow{
		ID:              row.ID.String(),
		PostID:          row.PostID.String(),
		DurationSeconds: int(row.DurationSeconds),
		ExpiresAt:       row.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

func toPollOptionRow(row pollOptionJoinRow) model.PollOptionRow {
	return model.PollOptionRow{
		ID:        int(row.ID),
		PollID:    row.PollID.String(),
		Label:     row.Label,
		SortOrder: int(row.SortOrder),
		VoteCount: int(row.VoteCount),
	}
}

func (r *postDAO) Create(ctx context.Context, s spec.NewPost, tx ...*sql.Tx) (*model.PostRow, error) {
	var (
		sharedContentID   sql.NullString
		sharedContentType sql.NullString
	)

	if s.SharedContent != nil {
		sharedContentID = sql.NullString{String: s.SharedContent.ID, Valid: true}
		sharedContentType = sql.NullString{String: s.SharedContent.Type, Valid: true}
	}

	created, err := genQueries(r.db, tx).CreatePost(ctx, sqlcgen.CreatePostParams{
		UserID:            s.UserID,
		Corner:            s.Corner,
		Body:              s.Body,
		SharedContentID:   sharedContentID,
		SharedContentType: sharedContentType,
	})
	if err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}

	return new(toPostRow(postJoinRow(created))), nil
}

func (r *postDAO) UpdatePost(ctx context.Context, s spec.PostUpdate, tx ...*sql.Tx) error {
	var (
		affected int64
		err      error
	)

	queries := genQueries(r.db, tx)

	if s.AsAdmin {
		affected, err = queries.UpdatePostAsAdmin(ctx, sqlcgen.UpdatePostAsAdminParams{
			Body: s.Body,
			ID:   s.ID,
		})
	} else {
		affected, err = queries.UpdatePostAsOwner(ctx, sqlcgen.UpdatePostAsOwnerParams{
			Body:   s.Body,
			ID:     s.ID,
			UserID: s.UserID,
		})
	}
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("post not found or not owned")
	}

	return nil
}

func (r *postDAO) GetByID(ctx context.Context, s spec.PostLookup, tx ...*sql.Tx) (*model.PostRow, error) {
	row, err := genQueries(r.db, tx).GetPostByID(ctx, sqlcgen.GetPostByIDParams{
		UserID: s.ViewerID,
		ID:     s.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get post: %w", err)
	}

	return new(toPostRow(row)), nil
}

func (r *postDAO) ListAll(ctx context.Context, q spec.PostFeedQuery, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	queries := genQueries(r.db, tx)

	search := postSearchPattern(q.Search)
	excluded := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountPostsFeed(ctx, sqlcgen.CountPostsFeedParams{
		Corner:         q.Corner,
		Search:         search,
		ResolvedFilter: q.ResolvedFilter,
		ExcludeUserIds: excluded,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	params := sqlcgen.ListPostsFeedByNewParams{
		ViewerID:       q.ViewerID,
		Corner:         q.Corner,
		Search:         search,
		ResolvedFilter: q.ResolvedFilter,
		ExcludeUserIds: excluded,
		RowOffset:      int32(q.Offset),
		RowLimit:       int32(q.Limit),
	}

	var posts []model.PostRow

	switch q.Sort {
	case "new":
		posts, err = mapPostFeedRows(queries.ListPostsFeedByNew(ctx, params))
	case "likes":
		posts, err = mapPostFeedRows(queries.ListPostsFeedByLikes(ctx, sqlcgen.ListPostsFeedByLikesParams(params)))
	case "comments":
		posts, err = mapPostFeedRows(queries.ListPostsFeedByComments(ctx, sqlcgen.ListPostsFeedByCommentsParams(params)))
	case "views":
		posts, err = mapPostFeedRows(queries.ListPostsFeedByViews(ctx, sqlcgen.ListPostsFeedByViewsParams(params)))
	default:
		posts, err = mapPostFeedRows(queries.ListPostsFeedByRelevance(ctx, sqlcgen.ListPostsFeedByRelevanceParams{
			ViewerID:       params.ViewerID,
			Corner:         params.Corner,
			Search:         params.Search,
			ResolvedFilter: params.ResolvedFilter,
			ExcludeUserIds: params.ExcludeUserIds,
			Seed:           int64(q.Seed),
			RowOffset:      params.RowOffset,
			RowLimit:       params.RowLimit,
		}))
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}

	return posts, int(total), nil
}

func (r *postDAO) ListByFollowing(ctx context.Context, q spec.PostFollowingFeedQuery, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	queries := genQueries(r.db, tx)

	excluded := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountFollowingPostsFeed(ctx, sqlcgen.CountFollowingPostsFeedParams{
		Corner:         q.Corner,
		ViewerID:       q.UserID,
		ExcludeUserIds: excluded,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count following posts: %w", err)
	}

	params := sqlcgen.ListFollowingPostsByNewParams{
		ViewerID:       q.UserID,
		Corner:         q.Corner,
		ExcludeUserIds: excluded,
		RowOffset:      int32(q.Offset),
		RowLimit:       int32(q.Limit),
	}

	var posts []model.PostRow

	switch q.Sort {
	case "new":
		posts, err = mapPostFeedRows(queries.ListFollowingPostsByNew(ctx, params))
	case "likes":
		posts, err = mapPostFeedRows(queries.ListFollowingPostsByLikes(ctx, sqlcgen.ListFollowingPostsByLikesParams(params)))
	case "comments":
		posts, err = mapPostFeedRows(queries.ListFollowingPostsByComments(ctx, sqlcgen.ListFollowingPostsByCommentsParams(params)))
	case "views":
		posts, err = mapPostFeedRows(queries.ListFollowingPostsByViews(ctx, sqlcgen.ListFollowingPostsByViewsParams(params)))
	default:
		posts, err = mapPostFeedRows(queries.ListFollowingPostsByRelevance(ctx, sqlcgen.ListFollowingPostsByRelevanceParams{
			ViewerID:       params.ViewerID,
			Corner:         params.Corner,
			ExcludeUserIds: params.ExcludeUserIds,
			Seed:           int64(q.Seed),
			RowOffset:      params.RowOffset,
			RowLimit:       params.RowLimit,
		}))
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list following posts: %w", err)
	}

	return posts, int(total), nil
}

func (r *postDAO) ListByUser(ctx context.Context, q spec.PostUserPage, tx ...*sql.Tx) ([]model.PostRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountPostsByUser(ctx, q.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count user posts: %w", err)
	}

	rows, err := queries.ListPostsByUser(ctx, sqlcgen.ListPostsByUserParams{
		UserID:   q.ViewerID,
		UserID_2: q.UserID,
		Limit:    int32(q.Limit),
		Offset:   int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user posts: %w", err)
	}

	var posts []model.PostRow
	for _, row := range rows {
		posts = append(posts, toPostRow(postJoinRow(row)))
	}

	return posts, int(total), nil
}

func (r *postDAO) GetPostAuthorID(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.ownedDAO.GetAuthorID(ctx, postID, tx...)
}

func (r *postDAO) GetSharedContentAuthor(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) (uuid.UUID, error) {
	id, err := uuid.Parse(ref.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get shared content author: %w", err)
	}

	queries := genQueries(r.db, tx)

	var userID uuid.UUID
	switch ref.Type {
	case "post":
		userID, err = queries.GetSharedPostAuthor(ctx, id)
	case "art":
		userID, err = queries.GetSharedArtAuthor(ctx, id)
	case "ship":
		userID, err = queries.GetSharedShipAuthor(ctx, id)
	case "mystery":
		userID, err = queries.GetSharedMysteryAuthor(ctx, id)
	case "theory":
		userID, err = queries.GetSharedTheoryAuthor(ctx, id)
	case "fanfic":
		userID, err = queries.GetSharedFanficAuthor(ctx, id)
	default:
		return uuid.Nil, fmt.Errorf("unknown shared content type: %s", ref.Type)
	}

	if err != nil {
		return uuid.Nil, fmt.Errorf("get shared content author: %w", err)
	}

	return userID, nil
}

func (r *postDAO) ResolveSuggestion(ctx context.Context, s spec.SuggestionResolution, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).ResolvePostSuggestion(ctx, sqlcgen.ResolvePostSuggestionParams{
		PostID:     s.PostID,
		ResolvedBy: &s.ResolvedBy,
		Status:     s.Status,
	})
	if err != nil {
		return fmt.Errorf("resolve suggestion: %w", err)
	}

	return nil
}

func (r *postDAO) UnresolveSuggestion(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).UnresolvePostSuggestion(ctx, postID); err != nil {
		return fmt.Errorf("unresolve suggestion: %w", err)
	}

	return nil
}

func (r *postDAO) CountUserPostsToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUserPostsToday(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count user posts today: %w", err)
	}

	return int(count), nil
}

func (r *postDAO) GetCornerCounts(ctx context.Context, tx ...*sql.Tx) (map[string]int, error) {
	rows, err := genQueries(r.db, tx).GetPostCornerCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("corner counts: %w", err)
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Corner] = int(row.PostCount)
	}

	return counts, nil
}

func (r *postDAO) GetShareCount(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).GetShareCount(ctx, sqlcgen.GetShareCountParams{
		ContentID:   ref.ID,
		ContentType: ref.Type,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get share count: %w", err)
	}

	return int(count), nil
}

func (r *postDAO) GetShareCountsBatch(ctx context.Context, q spec.SharedContentBatchRef, tx ...*sql.Tx) (map[string]int, error) {
	if len(q.ContentIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListShareCounts(ctx, sqlcgen.ListShareCountsParams{
		Column1:     strings.Join(q.ContentIDs, ","),
		ContentType: q.ContentType,
	})
	if err != nil {
		return nil, fmt.Errorf("batch get share counts: %w", err)
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.ContentID] = int(row.ShareCount)
	}

	return counts, nil
}

func (r *postDAO) IncrementShareCount(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).IncrementShareCount(ctx, sqlcgen.IncrementShareCountParams{
		ContentID:   ref.ID,
		ContentType: ref.Type,
	})
	if err != nil {
		return fmt.Errorf("increment share count: %w", err)
	}

	return nil
}

func (r *postDAO) DecrementShareCount(ctx context.Context, ref model.SharedContentRef, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).DecrementShareCount(ctx, sqlcgen.DecrementShareCountParams{
		ContentID:   ref.ID,
		ContentType: ref.Type,
	})
	if err != nil {
		return fmt.Errorf("decrement share count: %w", err)
	}

	return nil
}

func (r *postDAO) GetSharedContentFields(ctx context.Context, postID uuid.UUID, tx ...*sql.Tx) (*string, *string, error) {
	row, err := genQueries(r.db, tx).GetPostSharedContentFields(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get shared content fields: %w", err)
	}

	var contentID, contentType *string

	if row.SharedContentID.Valid {
		contentID = &row.SharedContentID.String
	}

	if row.SharedContentType.Valid {
		contentType = &row.SharedContentType.String
	}

	return contentID, contentType, nil
}

func (r *postDAO) GetSharedContentPreviews(refs []model.SharedContentRef, tx ...*sql.Tx) map[string]*dto.SharedContentPreview {
	result := make(map[string]*dto.SharedContentPreview)
	if len(refs) == 0 {
		return result
	}

	ctx := context.Background()

	grouped := make(map[string][]string)
	for _, ref := range refs {
		grouped[ref.Type] = append(grouped[ref.Type], ref.ID)
	}

	for contentType, ids := range grouped {
		switch contentType {
		case "post":
			r.fetchPostPreviews(ctx, ids, result, tx...)
		case "art":
			r.fetchArtPreviews(ctx, ids, result, tx...)
		case "ship":
			r.fetchShipPreviews(ctx, ids, result, tx...)
		case "mystery":
			r.fetchMysteryPreviews(ctx, ids, result, tx...)
		case "theory":
			r.fetchTheoryPreviews(ctx, ids, result, tx...)
		case "fanfic":
			r.fetchFanficPreviews(ctx, ids, result, tx...)
		}
	}

	for _, ref := range refs {
		key := ref.Type + ":" + ref.ID
		if _, ok := result[key]; !ok {
			result[key] = &dto.SharedContentPreview{
				ID:          ref.ID,
				ContentType: ref.Type,
				Deleted:     true,
				URL:         contentURL(ref.Type, ref.ID),
			}
		}
	}

	return result
}

func contentURL(contentType, id string) string {
	switch contentType {
	case "post":
		return "/game-board/" + id
	case "art":
		return "/gallery/art/" + id
	case "ship":
		return "/ships/" + id
	case "mystery":
		return "/mystery/" + id
	case "theory":
		return "/theory/" + id
	case "fanfic":
		return "/fanfiction/" + id
	default:
		return "/"
	}
}

func truncateBody(body string, maxLen int) string {
	clipped := text.ClampRunes(body, maxLen)
	if len(clipped) == len(body) {
		return body
	}

	return clipped + "..."
}

func (r *postDAO) fetchPostPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	queries := genQueries(r.db, tx)
	joined := strings.Join(ids, ",")

	rows, err := queries.ListSharedPostPreviews(ctx, joined)
	if err != nil {
		return
	}

	for _, row := range rows {
		id := row.ID.String()
		result["post:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "post",
			Body:        truncateBody(row.Body, 200),
			Author: &dto.UserResponse{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarUrl,
				Role:        role.Role(row.AuthorRole),
			},
			URL:          "/game-board/" + id,
			Corner:       row.Corner,
			LikeCount:    int(row.LikeCount),
			CommentCount: int(row.CommentCount),
		}
	}

	mediaRows, err := queries.ListSharedPostPreviewMedia(ctx, joined)
	if err != nil {
		return
	}

	for _, row := range mediaRows {
		preview, ok := result["post:"+row.PostID.String()]
		if !ok || len(preview.Media) >= 4 {
			continue
		}

		preview.Media = append(preview.Media, dto.PostMediaResponse{
			MediaURL:     row.MediaUrl,
			MediaType:    row.MediaType,
			ThumbnailURL: row.ThumbnailUrl,
			SortOrder:    int(row.SortOrder),
			IsSpoiler:    row.IsSpoiler,
		})
	}
}

func (r *postDAO) fetchArtPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	rows, err := genQueries(r.db, tx).ListSharedArtPreviews(ctx, strings.Join(ids, ","))
	if err != nil {
		return
	}

	for _, row := range rows {
		img := row.ThumbnailUrl
		if img == "" {
			img = row.ImageUrl
		}

		id := row.ID.String()
		result["art:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "art",
			Title:       row.Title,
			Body:        truncateBody(row.Description, 200),
			ImageURL:    img,
			Author: &dto.UserResponse{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarUrl,
				Role:        role.Role(row.AuthorRole),
			},
			URL:    "/gallery/art/" + id,
			Corner: row.Corner,
		}
	}
}

func (r *postDAO) fetchShipPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	rows, err := genQueries(r.db, tx).ListSharedShipPreviews(ctx, strings.Join(ids, ","))
	if err != nil {
		return
	}

	for _, row := range rows {
		img := row.ThumbnailUrl
		if img == "" {
			img = row.ImageUrl
		}

		id := row.ID.String()
		result["ship:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "ship",
			Title:       row.Title,
			Body:        truncateBody(row.Description, 200),
			ImageURL:    img,
			Author: &dto.UserResponse{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarUrl,
				Role:        role.Role(row.AuthorRole),
			},
			URL:       "/ships/" + id,
			VoteScore: int(row.VoteScore),
		}
	}
}

func (r *postDAO) fetchMysteryPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	rows, err := genQueries(r.db, tx).ListSharedMysteryPreviews(ctx, strings.Join(ids, ","))
	if err != nil {
		return
	}

	for _, row := range rows {
		id := row.ID.String()
		result["mystery:"+id] = &dto.SharedContentPreview{
			ID:          id,
			ContentType: "mystery",
			Title:       row.Title,
			Body:        truncateBody(row.Body, 200),
			Difficulty:  row.Difficulty,
			Solved:      row.Solved,
			Author: &dto.UserResponse{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarUrl,
				Role:        role.Role(row.AuthorRole),
			},
			URL: "/mystery/" + id,
		}
	}
}

func (r *postDAO) fetchTheoryPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	rows, err := genQueries(r.db, tx).ListSharedTheoryPreviews(ctx, strings.Join(ids, ","))
	if err != nil {
		return
	}

	for _, row := range rows {
		id := row.ID.String()
		result["theory:"+id] = &dto.SharedContentPreview{
			ID:               id,
			ContentType:      "theory",
			Title:            row.Title,
			Body:             truncateBody(row.Body, 200),
			Series:           row.Series,
			CredibilityScore: float64(row.CredibilityScore),
			Author: &dto.UserResponse{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarUrl,
				Role:        role.Role(row.AuthorRole),
			},
			URL: "/theory/" + id,
		}
	}
}

func (r *postDAO) fetchFanficPreviews(ctx context.Context, ids []string, result map[string]*dto.SharedContentPreview, tx ...*sql.Tx) {
	rows, err := genQueries(r.db, tx).ListSharedFanficPreviews(ctx, strings.Join(ids, ","))
	if err != nil {
		return
	}

	for _, row := range rows {
		img := row.CoverThumbnailUrl
		if img == "" {
			img = row.CoverImageUrl
		}

		id := row.ID.String()
		result["fanfic:"+id] = &dto.SharedContentPreview{
			ID:           id,
			ContentType:  "fanfic",
			Title:        row.Title,
			Body:         truncateBody(row.Summary, 200),
			ImageURL:     img,
			Series:       row.Series,
			Rating:       row.Rating,
			WordCount:    int(row.WordCount),
			ChapterCount: int(row.ChapterCount),
			Author: &dto.UserResponse{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarUrl,
				Role:        role.Role(row.AuthorRole),
			},
			URL: "/fanfiction/" + id,
		}
	}
}

func (r *postDAO) CreatePoll(ctx context.Context, s spec.NewPostPoll, tx ...*sql.Tx) (*model.PollRow, error) {
	created, err := genQueries(r.db, tx).CreatePostPoll(ctx, sqlcgen.CreatePostPollParams{
		PostID:          s.PostID,
		DurationSeconds: int32(s.DurationSeconds),
		Column3:         s.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create poll: %w", err)
	}

	return new(toPollRow(pollJoinRow(created))), nil
}

func (r *postDAO) AddPollOption(ctx context.Context, s spec.NewPostPollOption, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AddPostPollOption(ctx, sqlcgen.AddPostPollOptionParams{
		Column1:   s.PollID,
		Label:     s.Label,
		SortOrder: int32(s.SortOrder),
	})
	if err != nil {
		return fmt.Errorf("add poll option: %w", err)
	}

	return nil
}

func (r *postDAO) GetPollByPostID(ctx context.Context, q spec.PostPollQuery, tx ...*sql.Tx) (*model.PollRow, []model.PollOptionRow, *int, error) {
	queries := genQueries(r.db, tx)

	pollRow, err := queries.GetPostPoll(ctx, q.PostID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get poll: %w", err)
	}

	poll := toPollRow(pollRow)

	optionRows, err := queries.ListPostPollOptions(ctx, pollRow.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get poll options: %w", err)
	}

	var options []model.PollOptionRow
	for _, row := range optionRows {
		options = append(options, toPollOptionRow(row))
	}

	var votedOption *int
	if q.ViewerID != uuid.Nil {
		optionID, err := queries.GetPostPollVote(ctx, sqlcgen.GetPostPollVoteParams{
			PollID: pollRow.ID,
			UserID: q.ViewerID,
		})
		if err == nil {
			votedOption = new(int(optionID))
		}
	}

	return &poll, options, votedOption, nil
}

func (r *postDAO) GetPollsByPostIDs(ctx context.Context, q spec.PostPollBatchQuery, tx ...*sql.Tx) (map[uuid.UUID]*model.PollRow, map[uuid.UUID][]model.PollOptionRow, map[uuid.UUID]*int, error) {
	if len(q.PostIDs) == 0 {
		return nil, nil, nil, nil
	}

	queries := genQueries(r.db, tx)

	pollRows, err := queries.ListPostPolls(ctx, joinUUIDs(q.PostIDs))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch get polls: %w", err)
	}

	polls := make(map[uuid.UUID]*model.PollRow, len(pollRows))
	pollToPost := make(map[uuid.UUID]uuid.UUID, len(pollRows))
	pollIDs := make([]uuid.UUID, 0, len(pollRows))
	for _, row := range pollRows {
		polls[row.PostID] = new(toPollRow(pollJoinRow(row)))
		pollToPost[row.ID] = row.PostID
		pollIDs = append(pollIDs, row.ID)
	}

	if len(pollIDs) == 0 {
		return polls, nil, nil, nil
	}

	joinedPollIDs := joinUUIDs(pollIDs)

	optionRows, err := queries.ListPostPollOptionsByPollIDs(ctx, joinedPollIDs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("batch get poll options: %w", err)
	}

	optionsByPost := make(map[uuid.UUID][]model.PollOptionRow)
	for _, row := range optionRows {
		postID := pollToPost[row.PollID]
		optionsByPost[postID] = append(optionsByPost[postID], toPollOptionRow(pollOptionJoinRow(row)))
	}

	votes := make(map[uuid.UUID]*int)
	if q.ViewerID != uuid.Nil {
		voteRows, err := queries.ListPostPollVotesByPollIDs(ctx, sqlcgen.ListPostPollVotesByPollIDsParams{
			Column1: joinedPollIDs,
			UserID:  q.ViewerID,
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("batch get poll votes: %w", err)
		}

		for _, row := range voteRows {
			postID := pollToPost[row.PollID]
			votes[postID] = new(int(row.OptionID))
		}
	}

	return polls, optionsByPost, votes, nil
}

func (r *postDAO) VotePoll(ctx context.Context, s spec.PostPollVote, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).VotePostPoll(ctx, sqlcgen.VotePostPollParams{
		PollID:   s.PollID,
		UserID:   s.UserID,
		OptionID: int64(s.OptionID),
	})
	if err != nil {
		return fmt.Errorf("vote poll: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("already voted")
	}

	return nil
}
