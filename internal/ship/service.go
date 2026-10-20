package ship

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/upload"

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

		ListCharacters(ctx context.Context, series quotefinder.Series) ([]dto.CharacterListEntry, error)
	}

	service struct {
		shipRepo      repository.ShipRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		mentionSvc    mention.Service
		commentSvc    comment.Service
		uploadSvc     upload.Service
		mediaProc     *media.Processor
		settingsSvc   settings.Service
		quoteClient   *quotefinder.Client
		contentFilter *contentfilter.Manager
		ogCache       *og.Resolver
		activity      sidebar.ActivityBumper
	}
)

func NewService(
	shipRepo repository.ShipRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	mentionSvc mention.Service,
	commentSvc comment.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	settingsSvc settings.Service,
	quoteClient *quotefinder.Client,
	contentFilter *contentfilter.Manager,
	ogCache *og.Resolver,
	activity sidebar.ActivityBumper,
) Service {
	return &service{
		shipRepo:      shipRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		commentSvc:    commentSvc,
		uploadSvc:     uploadSvc,
		mediaProc:     mediaProc,
		settingsSvc:   settingsSvc,
		quoteClient:   quoteClient,
		contentFilter: contentFilter,
		ogCache:       ogCache,
		activity:      activity,
	}
}

func (s *service) writeAudit(ctx context.Context, entry audit.NewEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
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
	if err := s.contentFilter.Check(ctx, title, req.Description); err != nil {
		return uuid.Nil, err
	}
	if err := validateCharacters(req.Characters); err != nil {
		return uuid.Nil, err
	}

	description := strings.TrimSpace(req.Description)

	created, err := s.shipRepo.CreateWithCharacters(ctx, spec.NewShipWithCharacters{
		UserID:      userID,
		Title:       title,
		Description: description,
		Characters:  req.Characters,
	})
	if err != nil {
		return uuid.Nil, err
	}

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindShip, EntityID: created.ID}, userID, description)

	s.activity.BumpSidebarActivity(sidebar.KeyShips)

	return created.ID, nil
}

func (s *service) GetShip(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.ShipDetailResponse, error) {
	row, err := s.shipRepo.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	characters, err := s.shipRepo.GetCharacters(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ship characters: %w", err)
	}

	comments, err := s.commentSvc.Thread(ctx, spec.CommentTarget{Kind: spec.CommentKindShip, ID: id}, viewerID)
	if err != nil {
		return nil, fmt.Errorf("ship comments: %w", err)
	}

	viewerBlocked := false
	if viewerID != uuid.Nil {
		viewerBlocked, err = s.blockSvc.IsBlockedEither(ctx, viewerID, row.UserID)
		if err != nil {
			return nil, fmt.Errorf("block check: %w", err)
		}
	}

	return &dto.ShipDetailResponse{
		ShipResponse:  row.ToResponse(characters),
		Comments:      comments,
		ViewerBlocked: viewerBlocked,
	}, nil
}

func (s *service) UpdateShip(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateShipRequest) error {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	if err := s.contentFilter.Check(ctx, title, req.Description); err != nil {
		return err
	}
	if err := validateCharacters(req.Characters); err != nil {
		return err
	}

	authorID, err := s.shipRepo.GetAuthorID(ctx, id)
	if errors.Is(err, daoerr.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	description := strings.TrimSpace(req.Description)
	asAdmin := s.authz.Can(ctx, userID, authz.PermEditAnyPost)

	if err := s.shipRepo.UpdateWithCharacters(ctx, spec.ShipUpdate{
		ID:          id,
		UserID:      userID,
		Title:       title,
		Description: description,
		AsAdmin:     asAdmin,
		Characters:  req.Characters,
	}); err != nil {
		return err
	}

	if authorID != userID {
		s.writeAudit(ctx, audit.NewEntry{
			ActorID:    userID,
			Action:     audit.ActionShipUpdateAdmin,
			TargetType: audit.TargetShip,
			TargetID:   id.String(),
			Details:    fmt.Sprintf("title=%s", title),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteShip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	doomed, err := s.shipRepo.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: userID})
	if err != nil {
		return err
	}
	if doomed == nil {
		return ErrNotFound
	}

	asAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyPost)

	paths, err := s.shipRepo.DeleteShip(ctx, spec.ShipDeletion{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	action := audit.ActionShipDelete
	if doomed.UserID != userID {
		action = audit.ActionShipDeleteAdmin
	}

	s.writeAudit(ctx, audit.NewEntry{
		ActorID:    userID,
		Action:     action,
		TargetType: audit.TargetShip,
		TargetID:   id.String(),
		Details:    fmt.Sprintf("title=%s vote_score=%d comments=%d", doomed.Title, doomed.VoteScore, doomed.CommentCount),
		SubjectID:  doomed.UserID,
	})

	if err := s.ogCache.ClearMetaCache(ctx, og.KindShip, id.String()); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("ship_id", id.String()).Msg("clear og meta cache failed")
	}

	return nil
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
	blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	if err != nil {
		return nil, fmt.Errorf("blocked users: %w", err)
	}

	rows, total, err := s.shipRepo.List(ctx, spec.ShipListing{
		ViewerID:       viewerID,
		Sort:           sort,
		CrackshipsOnly: crackshipsOnly,
		Series:         series,
		CharacterID:    characterID,
		Limit:          page.Limit(),
		Offset:         page.Offset(),
		ExcludeUserIDs: blockedIDs,
	})
	if err != nil {
		return nil, err
	}

	return s.buildShipList(ctx, rows, total, page.Limit(), page.Offset())
}

func (s *service) ListShipsByUser(
	ctx context.Context,
	userID uuid.UUID,
	viewerID uuid.UUID,
	page bounds.Page,
) (*dto.ShipListResponse, error) {
	rows, total, err := s.shipRepo.ListByUser(ctx, spec.ShipUserListing{
		UserID:   userID,
		ViewerID: viewerID,
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	})
	if err != nil {
		return nil, err
	}

	return s.buildShipList(ctx, rows, total, page.Limit(), page.Offset())
}

func (s *service) buildShipList(ctx context.Context, rows []model.ShipRow, total, limit, offset int) (*dto.ShipListResponse, error) {
	shipIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		shipIDs[i] = r.ID
	}

	charactersMap, err := s.shipRepo.GetCharactersBatch(ctx, shipIDs)
	if err != nil {
		return nil, fmt.Errorf("ship characters: %w", err)
	}

	ships := make([]dto.ShipResponse, len(rows))
	for i, r := range rows {
		ships[i] = r.ToResponse(charactersMap[r.ID])
	}

	return &dto.ShipListResponse{
		Ships:  ships,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) UploadShipImage(ctx context.Context, shipID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (string, error) {
	authorID, err := s.shipRepo.GetAuthorID(ctx, shipID)
	if errors.Is(err, daoerr.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
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

	if err := s.shipRepo.UpdateImage(ctx, spec.ShipImageUpdate{
		ID:           shipID,
		ImageURL:     urlPath,
		ThumbnailURL: "",
	}); err != nil {
		return "", err
	}

	return urlPath, nil
}

func (s *service) Vote(ctx context.Context, userID uuid.UUID, shipID uuid.UUID, value int) error {
	authorID, err := s.shipRepo.GetAuthorID(ctx, shipID)
	if errors.Is(err, daoerr.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	blocked, err := s.blockSvc.IsBlockedEither(ctx, userID, authorID)
	if err != nil {
		return fmt.Errorf("block check: %w", err)
	}
	if blocked {
		return block.ErrUserBlocked
	}

	return s.shipRepo.Vote(ctx, spec.Vote{UserID: userID, TargetID: shipID, Value: value})
}

func (s *service) ListCharacters(ctx context.Context, series quotefinder.Series) ([]dto.CharacterListEntry, error) {
	chars, err := s.quoteClient.ListCharacters(ctx, series)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CharacterListEntry, len(chars))
	for i, c := range chars {
		result[i] = dto.CharacterListEntry{ID: c.ID, Name: c.Name, Group: c.Group}
	}
	return result, nil
}
