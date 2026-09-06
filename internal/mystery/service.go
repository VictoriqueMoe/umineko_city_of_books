package mystery

import (
	"context"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/text"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		ListMysteries(ctx context.Context, sort string, solved *bool, viewerID uuid.UUID, page bounds.Page) (*dto.MysteryListResponse, error)
		GetMystery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.MysteryDetailResponse, error)
		CreateMystery(ctx context.Context, userID uuid.UUID, req dto.CreateMysteryRequest) (uuid.UUID, error)
		UpdateMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateMysteryRequest) error
		DeleteMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateAttemptRequest) (uuid.UUID, error)
		DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		VoteAttempt(ctx context.Context, attemptID uuid.UUID, userID uuid.UUID, value int) error
		MarkSolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, attemptID uuid.UUID) error
		MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID) error
		AddClue(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateClueRequest) error
		GetLeaderboard(ctx context.Context, page bounds.Page) (*dto.MysteryLeaderboardResponse, error)
		GetTopDetectiveIDs(ctx context.Context) ([]string, error)
		GetGMLeaderboard(ctx context.Context, page bounds.Page) (*dto.GMLeaderboardResponse, error)
		GetTopGMIDs(ctx context.Context) ([]string, error)
		ListByUser(ctx context.Context, userID uuid.UUID, page bounds.Page) (*dto.MysteryListResponse, error)
		CreateComment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)
		UploadAttachment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, fileName string, fileSize int64, reader io.Reader) (*dto.MysteryAttachment, error)
		DeleteAttachment(ctx context.Context, attachmentID int64, mysteryID uuid.UUID, userID uuid.UUID) error
		UploadMedia(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)
		DeleteMedia(ctx context.Context, mediaID int64, mysteryID uuid.UUID, userID uuid.UUID) error
		SetPaused(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, paused bool) error
		SetGmAway(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, away bool) error
		DeleteClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID) error
		UpdateClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID, body string) error
	}

	service struct {
		mysteryRepo   repository.MysteryRepository
		userRepo      repository.UserRepository
		followRepo    repository.FollowRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		mentionSvc    mention.Service
		settingsSvc   settings.Service
		uploadSvc     upload.Service
		uploader      *media.Uploader
		hub           *ws.Hub
		contentFilter *contentfilter.Manager
		ogCache       *og.Resolver
	}
)

func NewService(
	mysteryRepo repository.MysteryRepository,
	userRepo repository.UserRepository,
	followRepo repository.FollowRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	mentionSvc mention.Service,
	settingsSvc settings.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
	ogCache *og.Resolver,
) Service {
	return &service{
		mysteryRepo:   mysteryRepo,
		userRepo:      userRepo,
		followRepo:    followRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		mentionSvc:    mentionSvc,
		settingsSvc:   settingsSvc,
		uploadSvc:     uploadSvc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		hub:           hub,
		contentFilter: contentFilter,
		ogCache:       ogCache,
	}
}

func (s *service) audit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func clueBodies(clues []dto.CreateClueRequest) []string {
	out := make([]string, 0, len(clues))
	for _, c := range clues {
		out = append(out, c.Body)
	}
	return out
}

func newClues(clues []dto.CreateClueRequest) []repository.NewClue {
	out := make([]repository.NewClue, 0, len(clues))
	for i, clue := range clues {
		if strings.TrimSpace(clue.Body) == "" {
			continue
		}

		truthType := clue.TruthType
		if truthType == "" {
			truthType = "red"
		}

		out = append(out, repository.NewClue{
			Body:      clue.Body,
			TruthType: truthType,
			SortOrder: i,
		})
	}

	return out
}

func (s *service) ListMysteries(ctx context.Context, sort string, solved *bool, viewerID uuid.UUID, page bounds.Page) (*dto.MysteryListResponse, error) {
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	rows, total, err := s.mysteryRepo.List(ctx, sort, solved, page.Limit(), page.Offset(), blockedIDs)
	if err != nil {
		return nil, err
	}

	return s.buildMysteryList(rows, total, page.Limit(), page.Offset()), nil
}

func (s *service) buildMysteryList(rows []repository.MysteryRow, total, limit, offset int) *dto.MysteryListResponse {
	mysteries := make([]dto.MysteryResponse, len(rows))
	for i, r := range rows {
		resp := r.ToResponse()
		if clipped := text.ClampRunes(resp.Body, 200); len(clipped) != len(resp.Body) {
			resp.Body = clipped + "..."
		}
		mysteries[i] = resp
	}

	return &dto.MysteryListResponse{
		Mysteries: mysteries,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}
}

func (s *service) GetMystery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.MysteryDetailResponse, error) {
	row, err := s.mysteryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	allClues, _ := s.mysteryRepo.GetClues(ctx, id)
	if allClues == nil {
		allClues = []dto.MysteryClue{}
	}

	attemptRows, _ := s.mysteryRepo.GetAttempts(ctx, id, viewerID)
	flatAttempts := make([]dto.MysteryAttempt, len(attemptRows))
	for i, a := range attemptRows {
		flatAttempts[i] = dto.MysteryAttempt{
			ID:       a.ID,
			ParentID: a.ParentID,
			Author: dto.UserResponse{
				ID:          a.UserID,
				Username:    a.AuthorUsername,
				DisplayName: a.AuthorDisplayName,
				AvatarURL:   a.AuthorAvatarURL,
				Role:        role.Role(a.AuthorRole),
			},
			Body:      a.Body,
			IsWinner:  a.IsWinner,
			VoteScore: a.VoteScore,
			UserVote:  a.UserVote,
			CreatedAt: a.CreatedAt,
		}
	}

	attempts := utils.BuildTree(flatAttempts,
		func(a dto.MysteryAttempt) uuid.UUID { return a.ID },
		func(a dto.MysteryAttempt) *uuid.UUID { return a.ParentID },
		func(a *dto.MysteryAttempt, replies []dto.MysteryAttempt) { a.Replies = replies },
	)

	playerSet := make(map[uuid.UUID]struct{})
	for _, a := range attempts {
		if a.Author.ID != row.UserID {
			playerSet[a.Author.ID] = struct{}{}
		}
	}

	viewerRole, _ := s.authz.GetRole(ctx, viewerID)
	isGameMaster := viewerID == row.UserID || viewerRole == authz.RoleSuperAdmin
	if !isGameMaster && !row.Solved && !row.FreeForAll {
		filtered := make([]dto.MysteryAttempt, 0, len(attempts))
		for _, a := range attempts {
			if a.Author.ID == viewerID {
				filtered = append(filtered, a)
			}
		}
		attempts = filtered
	}

	clues := allClues
	if !isGameMaster && !row.Solved {
		clues = make([]dto.MysteryClue, 0, len(allClues))
		for _, c := range allClues {
			if c.PlayerID == nil || *c.PlayerID == viewerID {
				clues = append(clues, c)
			}
		}
	}

	var comments []dto.MysteryCommentResponse
	if row.Solved {
		blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
		commentRows, _, _ := s.mysteryRepo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)
		if len(commentRows) > 0 {
			commentIDs := make([]uuid.UUID, len(commentRows))
			for i, c := range commentRows {
				commentIDs[i] = c.ID
			}
			mediaBatch, _ := s.mysteryRepo.GetCommentMediaBatch(ctx, commentIDs)
			flat := make([]dto.MysteryCommentResponse, len(commentRows))
			for i, c := range commentRows {
				flat[i] = mysteryCommentToResponse(c, mediaBatch[c.ID])
			}
			comments = utils.BuildTree(flat,
				func(c dto.MysteryCommentResponse) uuid.UUID { return c.ID },
				func(c dto.MysteryCommentResponse) *uuid.UUID { return c.ParentID },
				func(c *dto.MysteryCommentResponse, replies []dto.MysteryCommentResponse) { c.Replies = replies },
			)
		}
	}
	if comments == nil {
		comments = []dto.MysteryCommentResponse{}
	}

	attachments, _ := s.mysteryRepo.GetAttachments(ctx, id)
	if attachments == nil {
		attachments = []dto.MysteryAttachment{}
	}

	mediaRows, _ := s.mysteryRepo.GetMedia(ctx, id)
	mediaList := model.MediaRowsToResponse(mediaRows)

	viewerHasSolved := false
	if viewerID != uuid.Nil && viewerID != row.UserID {
		viewerHasSolved, _ = s.mysteryRepo.UserHasWinningAttempt(ctx, id, viewerID)
	}

	resp := dto.MysteryDetailResponse{
		ID:                    row.ID,
		Title:                 row.Title,
		Body:                  row.Body,
		Difficulty:            row.Difficulty,
		Solved:                row.Solved,
		Paused:                row.Paused,
		GmAway:                row.GmAway,
		FreeForAll:            row.FreeForAll,
		KeepOpenAfterSolve:    row.KeepOpenAfterSolve,
		KnoxContract:          row.Knox,
		KnoxContractPublished: row.KnoxPublished,
		KnoxContractLocked:    row.KnoxPublished && row.AttemptCount > 0,
		SolverCount:           row.SolverCount,
		ViewerHasSolved:       viewerHasSolved,
		SolvedAt:              row.SolvedAt,
		PausedAt:              row.PausedAt,
		PausedDurationSeconds: row.PausedDurationSeconds,
		Author: dto.UserResponse{
			ID:          row.UserID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarURL,
			Role:        role.Role(row.AuthorRole),
		},
		Clues:       clues,
		Attempts:    attempts,
		Comments:    comments,
		Attachments: attachments,
		Media:       mediaList,
		PlayerCount: len(playerSet),
		CreatedAt:   row.CreatedAt,
	}
	if row.WinnerID != nil && row.WinnerUsername != nil {
		resp.Winner = &dto.UserResponse{
			ID:          *row.WinnerID,
			Username:    *row.WinnerUsername,
			DisplayName: *row.WinnerDisplayName,
			AvatarURL:   *row.WinnerAvatarURL,
			Role:        role.Role(*row.WinnerRole),
		}
	}

	return &resp, nil
}

func (s *service) CreateMystery(ctx context.Context, userID uuid.UUID, req dto.CreateMysteryRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	if err := s.contentFilter.Check(ctx, append([]string{req.Title, req.Body}, clueBodies(req.Clues)...)...); err != nil {
		return uuid.Nil, err
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	contract := dto.DefaultKnoxContract()
	if req.KnoxContract != nil {
		contract = *req.KnoxContract
	}

	created, err := s.mysteryRepo.CreateWithClues(ctx, repository.NewMystery{
		UserID:             userID,
		Title:              req.Title,
		Body:               req.Body,
		Difficulty:         req.Difficulty,
		FreeForAll:         req.FreeForAll,
		KeepOpenAfterSolve: req.KeepOpenAfterSolve,
		Knox:               contract,
		Clues:              newClues(req.Clues),
	})
	if err != nil {
		return uuid.Nil, err
	}

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindMystery, EntityID: created.ID}, userID, req.Body)

	go notification.SendFollowerNotification(context.Background(), s.followRepo, s.notifService, notification.FollowerNotifyParams{
		ActorID:       userID,
		Type:          dto.NotifMysteryCreated,
		ReferenceID:   created.ID,
		ReferenceType: "mystery",
		Action:        "posted a new mystery",
		Title:         req.Title,
	})

	return created.ID, nil
}

func (s *service) UpdateMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateMysteryRequest) error {
	if !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return fmt.Errorf("not authorised")
	}
	if err := s.contentFilter.Check(ctx, append([]string{req.Title, req.Body}, clueBodies(req.Clues)...)...); err != nil {
		return err
	}

	old, err := s.mysteryRepo.GetByID(ctx, id)
	if err != nil || old == nil {
		return ErrNotFound
	}
	oldClues, _ := s.mysteryRepo.GetClues(ctx, id)

	contract := old.Knox
	if req.KnoxContract != nil {
		contract = *req.KnoxContract
	}
	if old.KnoxPublished && contract != old.Knox && old.AttemptCount > 0 {
		return ErrContractLocked
	}

	if err := s.mysteryRepo.UpdateWithClues(ctx, repository.MysteryUpdate{
		ID:                 id,
		Title:              req.Title,
		Body:               req.Body,
		Difficulty:         req.Difficulty,
		FreeForAll:         req.FreeForAll,
		KeepOpenAfterSolve: req.KeepOpenAfterSolve,
		Knox:               contract,
		Clues:              newClues(req.Clues),
	}); err != nil {
		return err
	}

	if old.UserID != userID {
		var changes []string
		if old.Title != req.Title {
			changes = append(changes, "title")
		}
		if old.Body != req.Body {
			changes = append(changes, "description")
		}
		if old.Difficulty != req.Difficulty {
			changes = append(changes, "difficulty")
		}
		if cluesChanged(oldClues, req.Clues) {
			changes = append(changes, "truths")
		}
		if contract != old.Knox {
			changes = append(changes, "the Knox contract")
		}

		summary := "nothing"
		if len(changes) > 0 {
			summary = strings.Join(changes, ", ")
		}

		s.audit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionMysteryUpdateAdmin,
			TargetType: repository.AuditTargetMystery,
			TargetID:   id.String(),
			Details:    fmt.Sprintf("title=%q changed=%q clues=rewritten", old.Title, summary),
			SubjectID:  old.UserID,
		})

		if len(changes) > 0 {
			message := fmt.Sprintf("your mystery was edited (changed: %s)", strings.Join(changes, ", "))
			go func() {
				bgCtx := context.Background()
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   old.UserID,
					Type:          dto.NotifContentEdited,
					ReferenceID:   id,
					ReferenceType: "mystery",
					ActorID:       userID,
					Message:       message,
					EmailActor:    "A moderator",
					EmailAction:   "edited your mystery",
					EmailLink:     fmt.Sprintf("/mystery/%s", id),
				})
			}()
		}
	}

	return nil
}

func cluesChanged(old []dto.MysteryClue, new []dto.CreateClueRequest) bool {
	gmClues := make([]dto.MysteryClue, 0)
	for _, c := range old {
		if c.PlayerID == nil {
			gmClues = append(gmClues, c)
		}
	}
	if len(gmClues) != len(new) {
		return true
	}
	for i, c := range gmClues {
		if c.Body != new[i].Body || c.TruthType != new[i].TruthType {
			return true
		}
	}
	return false
}

func (s *service) DeleteMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	asAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyTheory)

	paths, err := s.mysteryRepo.DeleteWithFiles(ctx, repository.MysteryDelete{ID: id, UserID: userID, AsAdmin: asAdmin})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	if err := s.ogCache.ClearMetaCache(ctx, og.KindMystery, id.String()); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("mystery_id", id.String()).Msg("clear og meta cache failed")
	}

	return nil
}
