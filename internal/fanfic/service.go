package fanfic

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	fanficparams "umineko_city_of_books/internal/fanfic/params"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

const (
	defaultSeries   = "Umineko"
	defaultLanguage = "English"
)

type (
	Service interface {
		CreateFanfic(ctx context.Context, userID uuid.UUID, req dto.CreateFanficRequest) (uuid.UUID, error)
		GetFanfic(ctx context.Context, id, viewerID uuid.UUID, viewerHash string) (*dto.FanficDetailResponse, error)
		UpdateFanfic(ctx context.Context, id, userID uuid.UUID, req dto.UpdateFanficRequest) error
		DeleteFanfic(ctx context.Context, id, userID uuid.UUID) error
		ListFanfics(ctx context.Context, viewerID uuid.UUID, params fanficparams.ListParams) (*dto.FanficListResponse, error)
		ListFanficsByUser(ctx context.Context, userID, viewerID uuid.UUID, page bounds.Page) (*dto.FanficListResponse, error)
		UploadCoverImage(ctx context.Context, fanficID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error)
		RemoveCoverImage(ctx context.Context, fanficID, userID uuid.UUID) error

		CreateChapter(ctx context.Context, fanficID, userID uuid.UUID, req dto.CreateChapterRequest) (uuid.UUID, error)
		GetChapter(ctx context.Context, fanficID uuid.UUID, chapterNumber int, viewerID uuid.UUID) (*dto.FanficChapterResponse, error)
		UpdateChapter(ctx context.Context, chapterID, userID uuid.UUID, req dto.UpdateChapterRequest) error
		DeleteChapter(ctx context.Context, chapterID, userID uuid.UUID) error

		Favourite(ctx context.Context, userID, fanficID uuid.UUID) error
		Unfavourite(ctx context.Context, userID, fanficID uuid.UUID) error

		ListFavourites(ctx context.Context, userID, viewerID uuid.UUID, page bounds.Page) (*dto.FanficListResponse, error)
		GetLanguages(ctx context.Context) ([]string, error)
		GetSeries(ctx context.Context) ([]string, error)
		SearchOCCharacters(ctx context.Context, query string) ([]string, error)

		CreateComment(ctx context.Context, fanficID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
	}

	fanficFields struct {
		Tags     []string
		Rating   string
		Series   string
		Language string
	}

	service struct {
		fanficRepo    repository.FanficRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
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
	auditRepo repository.AuditLogRepository,
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
		auditRepo:     auditRepo,
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

func (s *service) writeAudit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Log.Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
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

func validateFanficFields(genres, rawTags []string, rawRating, rawSeries, rawLanguage string) (fanficFields, error) {
	if len(genres) > 2 {
		return fanficFields{}, ErrTooManyGenres
	}

	tags := sanitiseTags(rawTags)
	if len(tags) > 10 {
		return fanficFields{}, ErrTooManyTags
	}
	for _, t := range tags {
		if len(t) > 30 {
			return fanficFields{}, ErrTagTooLong
		}
	}

	rating := strings.TrimSpace(rawRating)
	if rating == "" {
		rating = "K"
	}
	if !validRatings[rating] {
		return fanficFields{}, ErrInvalidRating
	}

	series := strings.TrimSpace(rawSeries)
	if series == "" {
		series = defaultSeries
	}

	language := strings.TrimSpace(rawLanguage)
	if language == "" {
		language = defaultLanguage
	}

	return fanficFields{
		Tags:     tags,
		Rating:   rating,
		Series:   series,
		Language: language,
	}, nil
}

func fanficUpdateDetails(before *model.FanficRow, spec repository.FanficUpdate) string {
	if before == nil {
		return ""
	}

	var changed []string

	if before.Title != spec.Title {
		changed = append(changed, "title")
	}

	if before.Summary != spec.Summary {
		changed = append(changed, "summary")
	}

	if before.Series != spec.Series {
		changed = append(changed, "series")
	}

	if before.Rating != spec.Rating {
		changed = append(changed, "rating")
	}

	if before.Language != spec.Language {
		changed = append(changed, "language")
	}

	if before.Status != spec.Status {
		changed = append(changed, "status")
	}

	if before.IsOneshot != spec.IsOneshot {
		changed = append(changed, "is_oneshot")
	}

	if before.ContainsLemons != spec.ContainsLemons {
		changed = append(changed, "contains_lemons")
	}

	if len(changed) == 0 {
		return "changed=none"
	}

	return "changed=" + strings.Join(changed, ",")
}

func (s *service) CreateFanfic(ctx context.Context, userID uuid.UUID, req dto.CreateFanficRequest) (uuid.UUID, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.filterTexts(ctx, title, req.Summary, req.Body); err != nil {
		return uuid.Nil, err
	}

	fields, err := validateFanficFields(req.Genres, req.Tags, req.Rating, req.Series, req.Language)
	if err != nil {
		return uuid.Nil, err
	}

	status := strings.TrimSpace(req.Status)
	switch status {
	case "draft", "in_progress", "complete":
	default:
		status = "in_progress"
	}

	spec := repository.NewFanfic{
		UserID:         userID,
		Title:          title,
		Summary:        strings.TrimSpace(req.Summary),
		Series:         fields.Series,
		Rating:         fields.Rating,
		Language:       fields.Language,
		Status:         status,
		IsOneshot:      req.IsOneshot,
		ContainsLemons: req.ContainsLemons,
		IsPairing:      req.IsPairing,
		Genres:         req.Genres,
		Tags:           fields.Tags,
		Characters:     req.Characters,
	}

	if body := strings.TrimSpace(req.Body); body != "" {
		spec.FirstChapter = &repository.NewChapter{Number: 1, Body: body, WordCount: countWords(body)}
	}

	created, err := s.fanficRepo.CreateWithDetails(ctx, spec)
	if err != nil {
		return uuid.Nil, err
	}

	return created.ID, nil
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
	comments, _, _ := s.fanficRepo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)

	var threaded []dto.FanficCommentResponse
	if len(comments) > 0 {
		commentIDs := make([]uuid.UUID, len(comments))
		for i, c := range comments {
			commentIDs[i] = c.ID
		}
		commentMediaMap, _ := s.fanficRepo.GetCommentMediaBatch(ctx, commentIDs)

		flatComments := make([]dto.FanficCommentResponse, len(comments))
		for i, c := range comments {
			flatComments[i] = fanficCommentToResponse(c, commentMediaMap[c.ID])
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

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.filterTexts(ctx, req.Title, req.Summary); err != nil {
		return err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrEmptyTitle
	}

	fields, err := validateFanficFields(req.Genres, req.Tags, req.Rating, req.Series, req.Language)
	if err != nil {
		return err
	}

	spec := repository.FanficUpdate{
		ID:             id,
		UserID:         userID,
		Title:          title,
		Summary:        strings.TrimSpace(req.Summary),
		Series:         fields.Series,
		Rating:         fields.Rating,
		Language:       fields.Language,
		Status:         strings.TrimSpace(req.Status),
		IsOneshot:      req.IsOneshot,
		ContainsLemons: req.ContainsLemons,
		IsPairing:      req.IsPairing,
		AsAdmin:        asAdmin,
		Genres:         req.Genres,
		Tags:           fields.Tags,
		Characters:     req.Characters,
	}

	var before *model.FanficRow
	if asAdmin {
		before, _ = s.fanficRepo.GetByID(ctx, id, userID)
	}

	if err := s.fanficRepo.UpdateWithDetails(ctx, spec); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionFanficUpdateAdmin,
			TargetType: repository.AuditTargetFanfic,
			TargetID:   id.String(),
			Details:    fanficUpdateDetails(before, spec),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteFanfic(ctx context.Context, id, userID uuid.UUID) error {
	authorID, err := s.fanficRepo.GetAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermDeleteAnyPost)

	spec := repository.FanficDelete{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
	}

	paths, err := s.fanficRepo.DeleteFanfic(ctx, spec)
	if err != nil {
		return err
	}

	action := repository.AuditActionFanficDelete
	if asAdmin {
		action = repository.AuditActionFanficDeleteAdmin
	}

	s.writeAudit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: repository.AuditTargetFanfic,
		TargetID:   id.String(),
		SubjectID:  authorID,
	})

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) ListFanfics(ctx context.Context, viewerID uuid.UUID, params fanficparams.ListParams) (*dto.FanficListResponse, error) {
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

func (s *service) ListFanficsByUser(ctx context.Context, userID, viewerID uuid.UUID, page bounds.Page) (*dto.FanficListResponse, error) {
	rows, total, err := s.fanficRepo.ListByUser(ctx, userID, viewerID, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}
	return s.buildFanficList(ctx, rows, total, page.Limit(), page.Offset())
}

func (s *service) ListFavourites(ctx context.Context, userID, viewerID uuid.UUID, page bounds.Page) (*dto.FanficListResponse, error) {
	rows, total, err := s.fanficRepo.ListFavourites(ctx, userID, viewerID, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}
	return s.buildFanficList(ctx, rows, total, page.Limit(), page.Offset())
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

	spec := repository.NewChapter{
		Number:    chapterNum,
		Title:     strings.TrimSpace(req.Title),
		Body:      body,
		WordCount: countWords(body),
	}

	created, err := s.fanficRepo.CreateChapterWithCount(ctx, fanficID, spec)
	if err != nil {
		return uuid.Nil, err
	}

	return created.ID, nil
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

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.filterTexts(ctx, req.Title, req.Body); err != nil {
		return err
	}

	body := strings.TrimSpace(sanitizeBody(req.Body))
	if body == "" {
		return ErrEmptyBody
	}

	spec := repository.ChapterUpdate{
		ID:        chapterID,
		Title:     strings.TrimSpace(req.Title),
		Body:      body,
		WordCount: countWords(body),
	}

	if err := s.fanficRepo.UpdateChapterWithCount(ctx, spec); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionFanficChapterUpdateAdmin,
			TargetType: repository.AuditTargetFanficChapter,
			TargetID:   chapterID.String(),
			Details:    fmt.Sprintf("title=%s,word_count=%d", spec.Title, spec.WordCount),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteChapter(ctx context.Context, chapterID, userID uuid.UUID) error {
	authorID, err := s.fanficRepo.GetChapterAuthorID(ctx, chapterID)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID
	if asAdmin && !s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		return ErrNotAuthor
	}

	if err := s.fanficRepo.DeleteChapterWithCount(ctx, chapterID); err != nil {
		return err
	}

	action := repository.AuditActionFanficChapterDelete
	if asAdmin {
		action = repository.AuditActionFanficChapterDeleteAdmin
	}

	s.writeAudit(ctx, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: repository.AuditTargetFanficChapter,
		TargetID:   chapterID.String(),
		SubjectID:  authorID,
	})

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
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifFanficFavourited,
			ReferenceID:   fanficID,
			ReferenceType: "fanfic",
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "favourited your fanfic",
			EmailLink:     fmt.Sprintf("/fanfics/%s", fanficID),
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

	created, err := s.fanficRepo.CreateComment(ctx, fanficID, req.ParentID, userID, body)
	if err != nil {
		return uuid.Nil, err
	}

	id := created.ID

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifFanficCommented,
			ReferenceID:   fanficID,
			ReferenceType: fmt.Sprintf("fanfic_comment:%s", id),
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "commented on your fanfic",
			EmailLink:     fmt.Sprintf("/fanfics/%s#comment-%s", fanficID, id),
		})

		if req.ParentID != nil {
			parentAuthor, err := s.fanficRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err == nil && parentAuthor != authorID {
				_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifFanficCommentReply,
					ReferenceID:   fanficID,
					ReferenceType: fmt.Sprintf("fanfic_comment:%s", id),
					ActorID:       userID,
					EmailActor:    actor.DisplayName,
					EmailAction:   "replied to your comment",
					EmailLink:     fmt.Sprintf("/fanfics/%s#comment-%s", fanficID, id),
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

	authorID, err := s.fanficRepo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermEditAnyComment)

	spec := repository.FanficCommentUpdate{
		ID:      id,
		UserID:  userID,
		Body:    body,
		AsAdmin: asAdmin,
	}

	if err := s.fanficRepo.UpdateCommentBody(ctx, spec); err != nil {
		return err
	}

	if asAdmin {
		s.writeAudit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionFanficCommentUpdateAdmin,
			TargetType: repository.AuditTargetFanficComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id, userID uuid.UUID) error {
	authorID, err := s.fanficRepo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	asAdmin := authorID != userID && s.authz.Can(ctx, userID, authz.PermDeleteAnyComment)

	action := repository.AuditActionFanficCommentDelete
	if asAdmin {
		action = repository.AuditActionFanficCommentDeleteAdmin
	}

	spec := repository.FanficCommentDelete{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
		Audit: repository.NewAuditEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: repository.AuditTargetFanficComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		},
	}

	paths, err := s.fanficRepo.DeleteCommentWithAudit(ctx, spec)
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	return nil
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
		fanficID, err := s.fanficRepo.GetCommentEntityID(bgCtx, commentID)
		if err != nil {
			return
		}
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifFanficCommentLiked,
			ReferenceID:   fanficID,
			ReferenceType: fmt.Sprintf("fanfic_comment:%s", commentID),
			ActorID:       userID,
			EmailActor:    "Someone",
			EmailAction:   "liked your comment",
			EmailLink:     fmt.Sprintf("/fanfics/%s#comment-%s", fanficID, commentID),
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
	filename string,
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

	return s.uploader.SaveAndRecord(ctx, "fanfics", contentType, filename, fileSize, reader,
		func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error) {
			return s.fanficRepo.AddCommentMedia(ctx, repository.NewFanficCommentMedia{
				CommentID:    commentID,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
			})
		},
		s.fanficRepo.UpdateCommentMediaURL,
		s.fanficRepo.UpdateCommentMediaThumbnail,
	)
}

func fanficCommentToResponse(c repository.CommentRow, media []model.PostMediaRow) dto.FanficCommentResponse {
	return dto.FanficCommentResponse{
		ID:       c.ID,
		ParentID: c.ParentID,
		Author: dto.UserResponse{
			ID:          c.UserID,
			Username:    c.AuthorUsername,
			DisplayName: c.AuthorDisplayName,
			AvatarURL:   c.AuthorAvatarURL,
			Role:        role.Role(c.AuthorRole),
		},
		Body:      c.Body,
		Media:     model.MediaRowsToResponse(media),
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
