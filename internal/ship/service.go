package ship

import (
	"context"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

type (
	Service interface {
		CreateShip(ctx context.Context, userID uuid.UUID, req dto.CreateShipRequest) (uuid.UUID, error)
		GetShip(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.ShipDetailResponse, error)
		UpdateShip(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateShipRequest) error
		DeleteShip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		ListShips(
			ctx context.Context,
			viewerID uuid.UUID,
			sort string,
			crackshipsOnly bool,
			series string,
			characterID string,
			page bounds.Page,
		) (*dto.ShipListResponse, error)
		ListShipsByUser(
			ctx context.Context,
			userID uuid.UUID,
			viewerID uuid.UUID,
			page bounds.Page,
		) (*dto.ShipListResponse, error)
		UploadShipImage(
			ctx context.Context,
			shipID uuid.UUID,
			userID uuid.UUID,
			contentType string,
			fileSize int64,
			reader io.Reader,
		) (string, error)

		Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error

		CreateComment(ctx context.Context, shipID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(
			ctx context.Context,
			commentID uuid.UUID,
			userID uuid.UUID,
			contentType string,
			fileSize int64,
			reader io.Reader,
		) (*dto.PostMediaResponse, error)

		ListCharacters(series quotefinder.Series) ([]dto.CharacterListEntry, error)
	}

	service struct {
		shipRepo      repository.ShipRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		uploadSvc     upload.Service
		mediaProc     *media.Processor
		uploader      *media.Uploader
		settingsSvc   settings.Service
		quoteClient   *quotefinder.Client
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	shipRepo repository.ShipRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	quoteClient *quotefinder.Client,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		shipRepo:      shipRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		uploadSvc:     uploadSvc,
		mediaProc:     mediaProc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		settingsSvc:   settingsSvc,
		quoteClient:   quoteClient,
		contentFilter: contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func validateCharacters(chars []dto.ShipCharacter) error {
	if len(chars) < 2 {
		return ErrTooFewCharacters
	}
	seen := make(map[string]bool)
	for _, c := range chars {
		key := strings.ToLower(c.Series + ":" + c.CharacterID + ":" + c.CharacterName)
		if seen[key] {
			return ErrDuplicateCharacters
		}
		seen[key] = true
	}
	return nil
}

func (s *service) CreateShip(ctx context.Context, userID uuid.UUID, req dto.CreateShipRequest) (uuid.UUID, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.filterTexts(ctx, title, req.Description); err != nil {
		return uuid.Nil, err
	}
	if err := validateCharacters(req.Characters); err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	description := strings.TrimSpace(req.Description)
	if err := s.shipRepo.CreateWithCharacters(ctx, id, userID, title, description, req.Characters); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (s *service) GetShip(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.ShipDetailResponse, error) {
	row, err := s.shipRepo.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	characters, _ := s.shipRepo.GetCharacters(ctx, id)
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	comments, _, _ := s.shipRepo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)

	commentIDs := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}
	commentMediaMap, _ := s.shipRepo.GetCommentMediaBatch(ctx, commentIDs)

	flatComments := make([]dto.ShipCommentResponse, len(comments))
	for i, c := range comments {
		flatComments[i] = shipCommentToResponse(c, commentMediaMap[c.ID])
	}
	threaded := utils.BuildTree(flatComments,
		func(c dto.ShipCommentResponse) uuid.UUID { return c.ID },
		func(c dto.ShipCommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.ShipCommentResponse, replies []dto.ShipCommentResponse) { c.Replies = replies },
	)

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, _ = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
	}

	return &dto.ShipDetailResponse{
		ShipResponse:  row.ToResponse(characters),
		Comments:      threaded,
		ViewerBlocked: viewerBlocked,
	}, nil
}

func (s *service) UpdateShip(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateShipRequest) error {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	if err := s.filterTexts(ctx, title, req.Description); err != nil {
		return err
	}
	if err := validateCharacters(req.Characters); err != nil {
		return err
	}

	description := strings.TrimSpace(req.Description)
	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyPost)
	return s.shipRepo.UpdateWithCharacters(ctx, id, userID, title, description, req.Characters, asAdmin)
}

func (s *service) DeleteShip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyPost) {
		return s.shipRepo.DeleteAsAdmin(ctx, id)
	}
	return s.shipRepo.Delete(ctx, id, userID)
}

func (s *service) ListShips(
	ctx context.Context,
	viewerID uuid.UUID,
	sort string,
	crackshipsOnly bool,
	series string,
	characterID string,
	page bounds.Page,
) (*dto.ShipListResponse, error) {
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)

	rows, total, err := s.shipRepo.List(ctx, viewerID, sort, crackshipsOnly, series, characterID, page.Limit(), page.Offset(), blockedIDs)
	if err != nil {
		return nil, err
	}

	shipIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		shipIDs[i] = r.ID
	}
	charactersMap, _ := s.shipRepo.GetCharactersBatch(ctx, shipIDs)

	ships := make([]dto.ShipResponse, len(rows))
	for i, r := range rows {
		ships[i] = r.ToResponse(charactersMap[r.ID])
	}

	return &dto.ShipListResponse{
		Ships:  ships,
		Total:  total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}, nil
}

func (s *service) ListShipsByUser(
	ctx context.Context,
	userID uuid.UUID,
	viewerID uuid.UUID,
	page bounds.Page,
) (*dto.ShipListResponse, error) {
	rows, total, err := s.shipRepo.ListByUser(ctx, userID, viewerID, page.Limit(), page.Offset())
	if err != nil {
		return nil, err
	}

	shipIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		shipIDs[i] = r.ID
	}
	charactersMap, _ := s.shipRepo.GetCharactersBatch(ctx, shipIDs)

	ships := make([]dto.ShipResponse, len(rows))
	for i, r := range rows {
		ships[i] = r.ToResponse(charactersMap[r.ID])
	}

	return &dto.ShipListResponse{
		Ships:  ships,
		Total:  total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}, nil
}

func (s *service) UploadShipImage(ctx context.Context, shipID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error) {
	authorID, err := s.shipRepo.GetAuthorID(ctx, shipID)
	if err != nil {
		return "", ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyPost) {
		return "", fmt.Errorf("not the ship author")
	}

	mediaID := uuid.New()
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	urlPath, err := s.uploadSvc.SaveImage(ctx, "ships", mediaID, fileSize, maxSize, reader)
	if err != nil {
		return "", err
	}

	if err := s.shipRepo.UpdateImage(ctx, shipID, urlPath, ""); err != nil {
		return "", err
	}

	return urlPath, nil
}

func (s *service) Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error {
	authorID, err := s.shipRepo.GetAuthorID(ctx, shipID)
	if err != nil {
		return ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return block.ErrUserBlocked
	}

	return s.shipRepo.Vote(ctx, userID, shipID, value)
}

func (s *service) CreateComment(ctx context.Context, shipID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return uuid.Nil, err
	}

	authorID, err := s.shipRepo.GetAuthorID(ctx, shipID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	id := uuid.New()
	if err := s.shipRepo.CreateComment(ctx, id, shipID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifShipCommented,
			ReferenceID:   shipID,
			ReferenceType: fmt.Sprintf("ship_comment:%s", id),
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "commented on your ship",
			EmailLink:     fmt.Sprintf("/ships/%s#comment-%s", shipID, id),
		})

		if req.ParentID != nil {
			parentAuthor, err := s.shipRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err == nil && parentAuthor != authorID {
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifShipCommentReply,
					ReferenceID:   shipID,
					ReferenceType: fmt.Sprintf("ship_comment:%s", id),
					ActorID:       userID,
					EmailActor:    actor.DisplayName,
					EmailAction:   "replied to your comment",
					EmailLink:     fmt.Sprintf("/ships/%s#comment-%s", shipID, id),
				})
			}
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.shipRepo.UpdateCommentAsAdmin(ctx, id, body)
	}
	return s.shipRepo.UpdateComment(ctx, id, userID, body)
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	isAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyComment)
	action := "ship_comment_delete"
	if isAdmin {
		if err := s.shipRepo.DeleteCommentAsAdmin(ctx, id); err != nil {
			return err
		}
		action = "ship_comment_delete_admin"
	} else {
		if err := s.shipRepo.DeleteComment(ctx, id, userID); err != nil {
			return err
		}
	}
	if err := s.auditRepo.Create(ctx, userID, action, "ship_comment", id.String(), ""); err != nil {
		return fmt.Errorf("audit comment delete: %w", err)
	}
	return nil
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	commentAuthorID, err := s.shipRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}
	if err := s.shipRepo.LikeComment(ctx, userID, commentID); err != nil {
		return err
	}

	go func() {
		if commentAuthorID == userID {
			return
		}
		bgCtx := context.Background()
		shipID, err := s.shipRepo.GetCommentEntityID(bgCtx, commentID)
		if err != nil {
			return
		}
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifShipCommentLiked,
			ReferenceID:   shipID,
			ReferenceType: fmt.Sprintf("ship_comment:%s", commentID),
			ActorID:       userID,
			EmailActor:    "Someone",
			EmailAction:   "liked your comment",
			EmailLink:     fmt.Sprintf("/ships/%s#comment-%s", shipID, commentID),
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.shipRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(
	ctx context.Context,
	commentID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	fileSize int64,
	reader io.Reader,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.shipRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	return s.uploader.SaveAndRecord(ctx, "ships", contentType, fileSize, reader,
		func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
			return s.shipRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, thumbURL, sortOrder)
		},
		s.shipRepo.UpdateCommentMediaURL,
		s.shipRepo.UpdateCommentMediaThumbnail,
	)
}

func (s *service) ListCharacters(series quotefinder.Series) ([]dto.CharacterListEntry, error) {
	chars, err := s.quoteClient.ListCharacters(series)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CharacterListEntry, len(chars))
	for i, c := range chars {
		result[i] = dto.CharacterListEntry{ID: c.ID, Name: c.Name, Group: c.Group}
	}
	return result, nil
}

func shipCommentToResponse(c repository.CommentRow, media []model.PostMediaRow) dto.ShipCommentResponse {
	return dto.ShipCommentResponse{
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
