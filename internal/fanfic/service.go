package fanfic

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

type (
	Service interface {
		CreateFanfic(ctx context.Context, userID uuid.UUID, req dto.CreateFanficRequest) (uuid.UUID, error)
		GetFanfic(ctx context.Context, id, viewerID uuid.UUID, viewerHash string) (*dto.FanficDetailResponse, error)
		UpdateFanfic(ctx context.Context, id, userID uuid.UUID, req dto.UpdateFanficRequest) error
		DeleteFanfic(ctx context.Context, id, userID uuid.UUID) error
		ListFanfics(ctx context.Context, viewerID uuid.UUID, params repository.FanficListParams) (*dto.FanficListResponse, error)
		ListFanficsByUser(ctx context.Context, userID, viewerID uuid.UUID, limit, offset int) (*dto.FanficListResponse, error)
		UploadCoverImage(ctx context.Context, fanficID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error)
		RemoveCoverImage(ctx context.Context, fanficID, userID uuid.UUID) error

		CreateChapter(ctx context.Context, fanficID, userID uuid.UUID, req dto.CreateChapterRequest) (uuid.UUID, error)
		GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int, viewerID uuid.UUID) (*dto.FanficChapterResponse, error)
		UpdateChapter(ctx context.Context, chapterID, userID uuid.UUID, req dto.UpdateChapterRequest) error
		DeleteChapter(ctx context.Context, chapterID, userID uuid.UUID) error

		Favourite(ctx context.Context, userID, fanficID uuid.UUID) error
		Unfavourite(ctx context.Context, userID, fanficID uuid.UUID) error

		ListFavourites(ctx context.Context, userID, viewerID uuid.UUID, limit, offset int) (*dto.FanficListResponse, error)
		GetLanguages(ctx context.Context) ([]string, error)
		GetSeries(ctx context.Context) ([]string, error)
		SearchOCCharacters(ctx context.Context, query string) ([]string, error)

		CreateComment(ctx context.Context, fanficID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
	}

	service struct {
		fanficRepo    repository.FanficRepository
		userRepo      repository.UserRepository
		authz         authz.Service
		blockSvc      block.Service
		notifSvc      notification.Service
		uploadSvc     upload.Service
		mediaProc     *media.Processor
		uploader      *media.Uploader
		settingsSvc   settings.Service
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	fanficRepo repository.FanficRepository,
	userRepo repository.UserRepository,
	authzSvc authz.Service,
	blockSvc block.Service,
	notifSvc notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		fanficRepo:    fanficRepo,
		userRepo:      userRepo,
		authz:         authzSvc,
		blockSvc:      blockSvc,
		notifSvc:      notifSvc,
		uploadSvc:     uploadSvc,
		mediaProc:     mediaProc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		contentFilter: contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

var (
	validRatings = map[string]bool{
		"K": true, "K+": true, "T": true, "M": true,
	}

	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
)

func sanitiseTags(raw []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, t := range raw {
		t = strings.TrimSpace(t)
		lower := strings.ToLower(t)
		if t == "" || seen[lower] {
			continue
		}
		seen[lower] = true
		result = append(result, t)
	}
	return result
}

func countWords(html string) int {
	text := htmlTagRe.ReplaceAllString(html, " ")
	return len(strings.Fields(text))
}

func (s *service) CreateFanfic(ctx context.Context, userID uuid.UUID, req dto.CreateFanficRequest) (uuid.UUID, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.filterTexts(ctx, title, req.Summary, req.Body); err != nil {
		return uuid.Nil, err
	}
	if len(req.Genres) > 2 {
		return uuid.Nil, ErrTooManyGenres
	}

	tags := sanitiseTags(req.Tags)
	if len(tags) > 10 {
		return uuid.Nil, ErrTooManyTags
	}
	for _, t := range tags {
		if len(t) > 30 {
			return uuid.Nil, ErrTagTooLong
		}
	}

	rating := strings.TrimSpace(req.Rating)
	if rating == "" {
		rating = "K"
	}
	if !validRatings[rating] {
		return uuid.Nil, ErrInvalidRating
	}

	series := strings.TrimSpace(req.Series)
	if series != "" {
		_ = s.fanficRepo.RegisterSeries(ctx, series)
	}

	language := strings.TrimSpace(req.Language)
	if language != "" {
		_ = s.fanficRepo.RegisterLanguage(ctx, language)
	}

	status := strings.TrimSpace(req.Status)
	switch status {
	case "draft", "in_progress", "complete":
	default:
		status = "in_progress"
	}

	id := uuid.New()
	summary := strings.TrimSpace(req.Summary)
	if err := s.fanficRepo.CreateWithDetails(ctx, id, userID, title, summary, series, rating, language, status, req.IsOneshot, req.ContainsLemons, req.Genres, tags, req.Characters, req.IsPairing); err != nil {
		return uuid.Nil, err
	}

	if strings.TrimSpace(req.Body) != "" {
		body := strings.TrimSpace(req.Body)
		if err := s.fanficRepo.CreateChapter(ctx, uuid.New(), id, 1, "", body, countWords(body)); err != nil {
			return uuid.Nil, err
		}
		_ = s.fanficRepo.UpdateWordCount(ctx, id)
	}

	for _, c := range req.Characters {
		if c.CharacterID == "" && strings.TrimSpace(c.CharacterName) != "" {
			_ = s.fanficRepo.RegisterOCCharacter(ctx, strings.TrimSpace(c.CharacterName), userID)
		}
	}

	return id, nil
}

func (s *service) GetFanfic(ctx context.Context, id, viewerID uuid.UUID, viewerHash string) (*dto.FanficDetailResponse, error) {
	row, err := s.fanficRepo.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	if viewerHash != "" {
		isNew, _ := s.fanficRepo.RecordView(ctx, id, viewerHash)
		if isNew {
			row.ViewCount++
		}
	}

	if row.Status == "draft" && row.UserID != viewerID && !s.authz.Can(ctx, viewerID, authz.PermEditAnyTheory) {
		return nil, ErrNotFound
	}

	genres, _ := s.fanficRepo.GetGenres(ctx, id)
	tags, _ := s.fanficRepo.GetTags(ctx, id)
	characters, _ := s.fanficRepo.GetCharacters(ctx, id)

	chapterRows, _ := s.fanficRepo.ListChapters(ctx, id)
	chapters := make([]dto.FanficChapterSummary, len(chapterRows))
	for i, ch := range chapterRows {
		chapters[i] = dto.FanficChapterSummary{
			ID:         ch.ID,
			ChapterNum: ch.ChapterNum,
			Title:      ch.Title,
			WordCount:  ch.WordCount,
		}
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	comments, _ := s.fanficRepo.GetComments(ctx, id, viewerID, blockedIDs)

	var threaded []dto.FanficCommentResponse
	if len(comments) > 0 {
		commentIDs := make([]uuid.UUID, len(comments))
		for i, c := range comments {
			commentIDs[i] = c.ID
		}
		commentMediaMap, _ := s.fanficRepo.GetCommentMediaBatch(ctx, commentIDs)

		flatComments := make([]dto.FanficCommentResponse, len(comments))
		for i, c := range comments {
			flatComments[i] = c.ToResponse(commentMediaMap[c.ID])
		}
		threaded = utils.BuildTree(flatComments,
			func(c dto.FanficCommentResponse) uuid.UUID { return c.ID },
			func(c dto.FanficCommentResponse) *uuid.UUID { return c.ParentID },
			func(c *dto.FanficCommentResponse, replies []dto.FanficCommentResponse) { c.Replies = replies },
		)
	}

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, _ = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
	}

	readingProgress, _ := s.fanficRepo.GetReadingProgress(ctx, viewerID, id)

	return &dto.FanficDetailResponse{
		FanficResponse:  row.ToResponse(genres, tags, characters),
		Chapters:        chapters,
		Comments:        threaded,
		ReadingProgress: readingProgress,
		ViewerBlocked:   viewerBlocked,
	}, nil
}

func (s *service) UpdateFanfic(ctx context.Context, id, userID uuid.UUID, req dto.UpdateFanficRequest) error {
	authorID, err := s.fanficRepo.GetAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyTheory)
	if authorID != userID && !asAdmin {
		return ErrNotAuthor
	}
	if err := s.filterTexts(ctx, req.Title, req.Summary); err != nil {
		return err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	if len(req.Genres) > 2 {
		return ErrTooManyGenres
	}

	tags := sanitiseTags(req.Tags)
	if len(tags) > 10 {
		return ErrTooManyTags
	}
	for _, t := range tags {
		if len(t) > 30 {
			return ErrTagTooLong
		}
	}

	rating := strings.TrimSpace(req.Rating)
	if rating == "" {
		rating = "K"
	}
	if !validRatings[rating] {
		return ErrInvalidRating
	}

	series := strings.TrimSpace(req.Series)
	if series != "" {
		_ = s.fanficRepo.RegisterSeries(ctx, series)
	}

	language := strings.TrimSpace(req.Language)
	if language != "" {
		_ = s.fanficRepo.RegisterLanguage(ctx, language)
	}

	summary := strings.TrimSpace(req.Summary)
	status := strings.TrimSpace(req.Status)
	if err := s.fanficRepo.UpdateWithDetails(ctx, id, userID, title, summary, series, rating, language, status, req.IsOneshot, req.ContainsLemons, req.Genres, tags, req.Characters, req.IsPairing, asAdmin); err != nil {
		return err
	}

	for _, c := range req.Characters {
		if c.CharacterID == "" && strings.TrimSpace(c.CharacterName) != "" {
			_ = s.fanficRepo.RegisterOCCharacter(ctx, strings.TrimSpace(c.CharacterName), userID)
		}
	}

	return nil
}

func (s *service) DeleteFanfic(ctx context.Context, id, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		return s.fanficRepo.DeleteAsAdmin(ctx, id)
	}
	return s.fanficRepo.Delete(ctx, id, userID)
}

func (s *service) ListFanfics(ctx context.Context, viewerID uuid.UUID, params repository.FanficListParams) (*dto.FanficListResponse, error) {
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	rows, total, err := s.fanficRepo.List(ctx, viewerID, params, blockedIDs)
	if err != nil {
		return nil, err
	}
	return s.buildFanficList(ctx, rows, total, params.Limit, params.Offset)
}

func (s *service) buildFanficList(ctx context.Context, rows []model.FanficRow, total, limit, offset int) (*dto.FanficListResponse, error) {
	fanficIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		fanficIDs[i] = r.ID
	}
	genresMap, _ := s.fanficRepo.GetGenresBatch(ctx, fanficIDs)
	tagsMap, _ := s.fanficRepo.GetTagsBatch(ctx, fanficIDs)
	charactersMap, _ := s.fanficRepo.GetCharactersBatch(ctx, fanficIDs)

	fanfics := make([]dto.FanficResponse, len(rows))
	for i, r := range rows {
		resp := r.ToResponse(genresMap[r.ID], tagsMap[r.ID], charactersMap[r.ID])
		if len(resp.Summary) > 200 {
			resp.Summary = resp.Summary[:200] + "..."
		}
		fanfics[i] = resp
	}

	return &dto.FanficListResponse{
		Fanfics: fanfics,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *service) ListFanficsByUser(ctx context.Context, userID, viewerID uuid.UUID, limit, offset int) (*dto.FanficListResponse, error) {
	rows, total, err := s.fanficRepo.ListByUser(ctx, userID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.buildFanficList(ctx, rows, total, limit, offset)
}

func (s *service) ListFavourites(ctx context.Context, userID, viewerID uuid.UUID, limit, offset int) (*dto.FanficListResponse, error) {
	rows, total, err := s.fanficRepo.ListFavourites(ctx, userID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.buildFanficList(ctx, rows, total, limit, offset)
}

func (s *service) UploadCoverImage(ctx context.Context, fanficID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error) {
	authorID, err := s.fanficRepo.GetAuthorID(ctx, fanficID)
	if err != nil {
		return "", ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return "", fmt.Errorf("not the fanfic author")
	}

	mediaID := uuid.New()
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	urlPath, err := s.uploadSvc.SaveImage(ctx, "fanfics", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.fanficRepo.UpdateCoverImage(ctx, fanficID, urlPath, ""); err != nil {
		return "", err
	}

	return urlPath, nil
}

func (s *service) RemoveCoverImage(ctx context.Context, fanficID, userID uuid.UUID) error {
	authorID, err := s.fanficRepo.GetAuthorID(ctx, fanficID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return fmt.Errorf("not authorised")
	}
	return s.fanficRepo.UpdateCoverImage(ctx, fanficID, "", "")
}

func (s *service) CreateChapter(ctx context.Context, fanficID, userID uuid.UUID, req dto.CreateChapterRequest) (uuid.UUID, error) {
	authorID, err := s.fanficRepo.GetAuthorID(ctx, fanficID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if authorID != userID {
		return uuid.Nil, ErrNotAuthor
	}
	if err := s.filterTexts(ctx, req.Title, req.Body); err != nil {
		return uuid.Nil, err
	}

	body := strings.TrimSpace(sanitizeBody(req.Body))
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}

	chapterNum, err := s.fanficRepo.GetNextChapterNumber(ctx, fanficID)
	if err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	title := strings.TrimSpace(req.Title)
	if err := s.fanficRepo.CreateChapter(ctx, id, fanficID, chapterNum, title, body, countWords(body)); err != nil {
		return uuid.Nil, err
	}

	_ = s.fanficRepo.UpdateWordCount(ctx, fanficID)

	return id, nil
}

func (s *service) GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int, viewerID uuid.UUID) (*dto.FanficChapterResponse, error) {
	ch, err := s.fanficRepo.GetChapter(ctx, fanficID, chapterNumber)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrNotFound
	}

	totalChapters, err := s.fanficRepo.GetChapterCount(ctx, fanficID)
	if err != nil {
		return nil, err
	}

	if viewerID != uuid.Nil {
		_ = s.fanficRepo.SetReadingProgress(ctx, viewerID, fanficID, chapterNumber)
	}

	return &dto.FanficChapterResponse{
		ID:         ch.ID,
		ChapterNum: ch.ChapterNum,
		Title:      ch.Title,
		Body:       ch.Body,
		WordCount:  ch.WordCount,
		HasPrev:    chapterNumber > 1,
		HasNext:    chapterNumber < totalChapters,
		CreatedAt:  ch.CreatedAt,
		UpdatedAt:  ch.UpdatedAt,
	}, nil
}

func (s *service) UpdateChapter(ctx context.Context, chapterID, userID uuid.UUID, req dto.UpdateChapterRequest) error {
	authorID, err := s.fanficRepo.GetChapterAuthorID(ctx, chapterID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.filterTexts(ctx, req.Title, req.Body); err != nil {
		return err
	}

	body := strings.TrimSpace(sanitizeBody(req.Body))
	if body == "" {
		return ErrEmptyBody
	}

	title := strings.TrimSpace(req.Title)
	if err := s.fanficRepo.UpdateChapter(ctx, chapterID, title, body, countWords(body)); err != nil {
		return err
	}

	fanficID, err := s.fanficRepo.GetChapterFanficID(ctx, chapterID)
	if err != nil {
		return err
	}
	_ = s.fanficRepo.UpdateWordCount(ctx, fanficID)

	return nil
}

func (s *service) DeleteChapter(ctx context.Context, chapterID, userID uuid.UUID) error {
	authorID, err := s.fanficRepo.GetChapterAuthorID(ctx, chapterID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		return ErrNotAuthor
	}

	fanficID, err := s.fanficRepo.GetChapterFanficID(ctx, chapterID)
	if err != nil {
		return err
	}

	if err := s.fanficRepo.DeleteChapter(ctx, chapterID); err != nil {
		return err
	}

	_ = s.fanficRepo.UpdateWordCount(ctx, fanficID)

	return nil
}

func (s *service) Favourite(ctx context.Context, userID, fanficID uuid.UUID) error {
	authorID, err := s.fanficRepo.GetAuthorID(ctx, fanficID)
	if err != nil {
		return ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.fanficRepo.Favourite(ctx, userID, fanficID); err != nil {
		return err
	}

	go func() {
		if authorID == userID {
			return
		}
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/fanfics/%s", baseURL, fanficID)
		subject, emailBody := notification.NotifEmail(actor.DisplayName, "favourited your fanfic", "", linkURL)
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifFanficFavourited,
			ReferenceID:   fanficID,
			ReferenceType: "fanfic",
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}()

	return nil
}

func (s *service) Unfavourite(ctx context.Context, userID, fanficID uuid.UUID) error {
	return s.fanficRepo.Unfavourite(ctx, userID, fanficID)
}

func (s *service) GetLanguages(ctx context.Context) ([]string, error) {
	return s.fanficRepo.GetLanguages(ctx)
}

func (s *service) GetSeries(ctx context.Context) ([]string, error) {
	return s.fanficRepo.GetSeries(ctx)
}

func (s *service) SearchOCCharacters(ctx context.Context, query string) ([]string, error) {
	return s.fanficRepo.SearchOCCharacters(ctx, strings.TrimSpace(query))
}

func (s *service) CreateComment(ctx context.Context, fanficID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.fanficRepo.GetAuthorID(ctx, fanficID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	id := uuid.New()
	if err := s.fanficRepo.CreateComment(ctx, id, fanficID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/fanfics/%s#comment-%s", baseURL, fanficID, id)

		subject, emailBody := notification.NotifEmail(actor.DisplayName, "commented on your fanfic", "", linkURL)
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifFanficCommented,
			ReferenceID:   fanficID,
			ReferenceType: fmt.Sprintf("fanfic_comment:%s", id),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})

		if req.ParentID != nil {
			parentAuthor, err := s.fanficRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err == nil && parentAuthor != authorID {
				replySubject, replyBody := notification.NotifEmail(actor.DisplayName, "replied to your comment", "", linkURL)
				_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifFanficCommentReply,
					ReferenceID:   fanficID,
					ReferenceType: fmt.Sprintf("fanfic_comment:%s", id),
					ActorID:       userID,
					EmailSubject:  replySubject,
					EmailBody:     replyBody,
				})
			}
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.fanficRepo.UpdateCommentAsAdmin(ctx, id, body)
	}
	return s.fanficRepo.UpdateComment(ctx, id, userID, body)
}

func (s *service) DeleteComment(ctx context.Context, id, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.fanficRepo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.fanficRepo.DeleteComment(ctx, id, userID)
}

func (s *service) LikeComment(ctx context.Context, userID, commentID uuid.UUID) error {
	commentAuthorID, err := s.fanficRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}
	if err := s.fanficRepo.LikeComment(ctx, userID, commentID); err != nil {
		return err
	}

	go func() {
		if commentAuthorID == userID {
			return
		}
		bgCtx := context.Background()
		fanficID, err := s.fanficRepo.GetCommentFanficID(bgCtx, commentID)
		if err != nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/fanfics/%s#comment-%s", baseURL, fanficID, commentID)
		subject, emailBody := notification.NotifEmail("Someone", "liked your comment", "", linkURL)
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifFanficCommentLiked,
			ReferenceID:   fanficID,
			ReferenceType: fmt.Sprintf("fanfic_comment:%s", commentID),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     emailBody,
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error {
	return s.fanficRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(
	ctx context.Context,
	commentID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	fileSize int64,
	reader io.Reader,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.fanficRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	return s.uploader.SaveAndRecord(ctx, "fanfics", contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.fanficRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.fanficRepo.UpdateCommentMediaURL,
		s.fanficRepo.UpdateCommentMediaThumbnail,
	)
}
