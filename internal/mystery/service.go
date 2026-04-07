package mystery

import (
	"context"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		ListMysteries(ctx context.Context, sort string, solved *bool, viewerID uuid.UUID, limit, offset int) (*dto.MysteryListResponse, error)
		GetMystery(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*dto.MysteryDetailResponse, error)
		CreateMystery(ctx context.Context, userID uuid.UUID, req dto.CreateMysteryRequest) (uuid.UUID, error)
		UpdateMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateMysteryRequest) error
		DeleteMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateAttemptRequest) (uuid.UUID, error)
		DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		VoteAttempt(ctx context.Context, attemptID uuid.UUID, userID uuid.UUID, value int) error
		MarkSolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, attemptID uuid.UUID) error
		AddClue(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateClueRequest) error
		GetLeaderboard(ctx context.Context, limit int) (*dto.MysteryLeaderboardResponse, error)
		GetTopDetectiveID(ctx context.Context) (string, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) (*dto.MysteryListResponse, error)
		CreateComment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
	}

	service struct {
		mysteryRepo  repository.MysteryRepository
		userRepo     repository.UserRepository
		authz        authz.Service
		blockSvc     block.Service
		notifService notification.Service
		settingsSvc  settings.Service
		uploadSvc    upload.Service
		hub          *ws.Hub
	}
)

func NewService(
	mysteryRepo repository.MysteryRepository,
	userRepo repository.UserRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
	uploadSvc upload.Service,
	hub *ws.Hub,
) Service {
	return &service{
		mysteryRepo:  mysteryRepo,
		userRepo:     userRepo,
		authz:        authzService,
		blockSvc:     blockSvc,
		notifService: notifService,
		settingsSvc:  settingsSvc,
		uploadSvc:    uploadSvc,
		hub:          hub,
	}
}

func (s *service) ListMysteries(ctx context.Context, sort string, solved *bool, viewerID uuid.UUID, limit, offset int) (*dto.MysteryListResponse, error) {
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	rows, total, err := s.mysteryRepo.List(ctx, sort, solved, limit, offset, blockedIDs)
	if err != nil {
		return nil, err
	}

	mysteries := make([]dto.MysteryResponse, len(rows))
	for i, r := range rows {
		resp := r.ToResponse()
		if len(resp.Body) > 200 {
			resp.Body = resp.Body[:200] + "..."
		}
		mysteries[i] = resp
	}

	return &dto.MysteryListResponse{
		Mysteries: mysteries,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
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
	if !isGameMaster && !row.Solved {
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
		commentRows, _ := s.mysteryRepo.GetComments(ctx, id, viewerID, blockedIDs)
		if len(commentRows) > 0 {
			commentIDs := make([]uuid.UUID, len(commentRows))
			for i, c := range commentRows {
				commentIDs[i] = c.ID
			}
			mediaBatch, _ := s.mysteryRepo.GetCommentMediaBatch(ctx, commentIDs)
			flat := make([]dto.MysteryCommentResponse, len(commentRows))
			for i, c := range commentRows {
				flat[i] = c.ToResponse(mediaBatch[c.ID])
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

	resp := dto.MysteryDetailResponse{
		ID:         row.ID,
		Title:      row.Title,
		Body:       row.Body,
		Difficulty: row.Difficulty,
		Solved:     row.Solved,
		SolvedAt:   row.SolvedAt,
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
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	id := uuid.New()
	if err := s.mysteryRepo.Create(ctx, id, userID, req.Title, req.Body, req.Difficulty); err != nil {
		return uuid.Nil, err
	}

	for i, clue := range req.Clues {
		if strings.TrimSpace(clue.Body) == "" {
			continue
		}
		truthType := clue.TruthType
		if truthType == "" {
			truthType = "red"
		}
		if err := s.mysteryRepo.AddClue(ctx, id, clue.Body, truthType, i, nil); err != nil {
			return uuid.Nil, err
		}
	}

	return id, nil
}

func (s *service) UpdateMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CreateMysteryRequest) error {
	if err := s.mysteryRepo.Update(ctx, id, userID, req.Title, req.Body, req.Difficulty); err != nil {
		return err
	}

	_ = s.mysteryRepo.DeleteClues(ctx, id)
	for i, clue := range req.Clues {
		if strings.TrimSpace(clue.Body) == "" {
			continue
		}
		truthType := clue.TruthType
		if truthType == "" {
			truthType = "red"
		}
		_ = s.mysteryRepo.AddClue(ctx, id, clue.Body, truthType, i, nil)
	}

	return nil
}

func (s *service) DeleteMystery(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyTheory) {
		return s.mysteryRepo.DeleteAsAdmin(ctx, id)
	}
	return s.mysteryRepo.Delete(ctx, id, userID)
}

func (s *service) CreateAttempt(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateAttemptRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, ErrEmptyBody
	}

	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if solved, err := s.mysteryRepo.IsSolved(ctx, mysteryID); err != nil {
		return uuid.Nil, err
	} else if solved {
		return uuid.Nil, ErrAlreadySolved
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	if req.ParentID != nil {
		parentAuthor, err := s.mysteryRepo.GetAttemptAuthorID(ctx, *req.ParentID)
		if err != nil {
			return uuid.Nil, ErrNotFound
		}
		if userID != authorID && userID != parentAuthor {
			return uuid.Nil, ErrCannotReply
		}
	}

	id := uuid.New()
	if err := s.mysteryRepo.CreateAttempt(ctx, id, mysteryID, userID, req.ParentID, strings.TrimSpace(req.Body)); err != nil {
		return uuid.Nil, err
	}

	actor, _ := s.userRepo.GetByID(ctx, userID)
	wsData := map[string]interface{}{
		"mystery_id": mysteryID,
		"attempt_id": id,
		"parent_id":  req.ParentID,
		"author_id":  userID,
	}
	if actor != nil {
		wsData["author_username"] = actor.Username
		wsData["author_display_name"] = actor.DisplayName
		wsData["author_avatar_url"] = actor.AvatarURL
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_attempt_created",
		Data: wsData,
	})

	go func() {
		bgCtx := context.Background()
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mystery/%s#attempt-%s", baseURL, mysteryID, id)
		attemptRef := fmt.Sprintf("mystery_attempt:%s", id)

		subject, body := notification.NotifEmail("Someone", "submitted an attempt on your mystery", "", linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   authorID,
			Type:          dto.NotifMysteryAttempt,
			ReferenceID:   mysteryID,
			ReferenceType: attemptRef,
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})

		if req.ParentID != nil {
			if parentAuthor, err := s.mysteryRepo.GetAttemptAuthorID(bgCtx, *req.ParentID); err == nil && parentAuthor != authorID {
				replySubject, replyBody := notification.NotifEmail("Someone", "replied to your attempt", "", linkURL)
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifMysteryReply,
					ReferenceID:   mysteryID,
					ReferenceType: attemptRef,
					ActorID:       userID,
					EmailSubject:  replySubject,
					EmailBody:     replyBody,
				})
			}
		}
	}()

	return id, nil
}

func (s *service) DeleteAttempt(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.mysteryRepo.DeleteAttemptAsAdmin(ctx, id)
	}
	return s.mysteryRepo.DeleteAttempt(ctx, id, userID)
}

func (s *service) VoteAttempt(ctx context.Context, attemptID uuid.UUID, userID uuid.UUID, value int) error {
	if value != 1 && value != -1 && value != 0 {
		return ErrInvalidVote
	}

	attemptAuthorID, err := s.mysteryRepo.GetAttemptAuthorID(ctx, attemptID)
	if err != nil {
		return ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, attemptAuthorID); blocked {
		return block.ErrUserBlocked
	}

	if err := s.mysteryRepo.VoteAttempt(ctx, userID, attemptID, value); err != nil {
		return err
	}

	if value != 0 {
		go func() {
			bgCtx := context.Background()
			mysteryID, err := s.mysteryRepo.GetAttemptMysteryID(bgCtx, attemptID)
			if err != nil {
				return
			}
			baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/mystery/%s#attempt-%s", baseURL, mysteryID, attemptID)
			subject, body := notification.NotifEmail("Someone", "voted on your attempt", "", linkURL)
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   attemptAuthorID,
				Type:          dto.NotifMysteryVote,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_attempt:%s", attemptID),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			})
		}()
	}

	return nil
}

func (s *service) MarkSolved(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, attemptID uuid.UUID) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}

	attemptAuthorID, err := s.mysteryRepo.GetAttemptAuthorID(ctx, attemptID)
	if err != nil {
		return ErrNotFound
	}
	attemptMysteryID, err := s.mysteryRepo.GetAttemptMysteryID(ctx, attemptID)
	if err != nil {
		return ErrNotFound
	}
	if attemptMysteryID != mysteryID {
		return fmt.Errorf("attempt does not belong to this mystery")
	}

	if err := s.mysteryRepo.MarkSolved(ctx, mysteryID, attemptID); err != nil {
		return err
	}

	s.hub.Broadcast(ws.Message{
		Type: "mystery_solved",
		Data: map[string]interface{}{
			"mystery_id": mysteryID,
			"attempt_id": attemptID,
			"winner_id":  attemptAuthorID,
		},
	})

	go func() {
		bgCtx := context.Background()
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mystery/%s#attempt-%s", baseURL, mysteryID, attemptID)
		subject, body := notification.NotifEmail("Someone", "chose your attempt as the winner!", "", linkURL)
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   attemptAuthorID,
			Type:          dto.NotifMysterySolved,
			ReferenceID:   mysteryID,
			ReferenceType: fmt.Sprintf("mystery_attempt:%s", attemptID),
			ActorID:       userID,
			EmailSubject:  subject,
			EmailBody:     body,
		})
	}()

	return nil
}

func (s *service) AddClue(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateClueRequest) error {
	if strings.TrimSpace(req.Body) == "" {
		return ErrEmptyBody
	}

	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID {
		return ErrNotAuthor
	}

	if req.TruthType == "" {
		req.TruthType = "red"
	}

	count, _ := s.mysteryRepo.CountAttempts(ctx, mysteryID)
	if err := s.mysteryRepo.AddClue(ctx, mysteryID, req.Body, req.TruthType, count, req.PlayerID); err != nil {
		return err
	}

	wsData := map[string]interface{}{
		"mystery_id": mysteryID,
		"truth_type": req.TruthType,
	}
	if req.PlayerID != nil {
		wsData["player_id"] = req.PlayerID
		s.hub.SendToUser(*req.PlayerID, ws.Message{
			Type: "mystery_clue_added",
			Data: wsData,
		})
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_clue_added",
		Data: wsData,
	})

	return nil
}

func (s *service) GetLeaderboard(ctx context.Context, limit int) (*dto.MysteryLeaderboardResponse, error) {
	rows, err := s.mysteryRepo.GetLeaderboard(ctx, limit)
	if err != nil {
		return nil, err
	}

	entries := make([]dto.MysteryLeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = dto.MysteryLeaderboardEntry{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			Score:           r.Score,
			EasySolved:      r.EasySolved,
			MediumSolved:    r.MediumSolved,
			HardSolved:      r.HardSolved,
			NightmareSolved: r.NightmareSolved,
		}
	}
	return &dto.MysteryLeaderboardResponse{Entries: entries}, nil
}

func (s *service) GetTopDetectiveID(ctx context.Context) (string, error) {
	return s.mysteryRepo.GetTopDetectiveID(ctx)
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) (*dto.MysteryListResponse, error) {
	rows, total, err := s.mysteryRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	mysteries := make([]dto.MysteryResponse, len(rows))
	for i, r := range rows {
		resp := r.ToResponse()
		if len(resp.Body) > 200 {
			resp.Body = resp.Body[:200] + "..."
		}
		mysteries[i] = resp
	}

	return &dto.MysteryListResponse{
		Mysteries: mysteries,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (s *service) CreateComment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}

	solved, err := s.mysteryRepo.IsSolved(ctx, mysteryID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if !solved {
		return uuid.Nil, ErrNotSolved
	}

	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, authorID); blocked {
		return uuid.Nil, block.ErrUserBlocked
	}

	id := uuid.New()
	if err := s.mysteryRepo.CreateComment(ctx, id, mysteryID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	if req.ParentID != nil {
		go func() {
			bgCtx := context.Background()
			parentAuthor, err := s.mysteryRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err != nil || parentAuthor == userID {
				return
			}
			actor, err := s.userRepo.GetByID(bgCtx, userID)
			if err != nil || actor == nil {
				return
			}
			baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/mystery/%s#comment-%s", baseURL, mysteryID, id)
			subject, emailBody := notification.NotifEmail(actor.DisplayName, "replied to your comment", "", linkURL)
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   parentAuthor,
				Type:          dto.NotifMysteryCommentReply,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_comment:%s", id),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     emailBody,
			})
		}()
	}

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.mysteryRepo.UpdateCommentAsAdmin(ctx, id, body)
	}
	return s.mysteryRepo.UpdateComment(ctx, id, userID, body)
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.authz.Can(ctx, userID, authz.PermDeleteAnyComment) {
		return s.mysteryRepo.DeleteCommentAsAdmin(ctx, id)
	}
	return s.mysteryRepo.DeleteComment(ctx, id, userID)
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	commentAuthorID, err := s.mysteryRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return block.ErrUserBlocked
	}
	return s.mysteryRepo.LikeComment(ctx, userID, commentID)
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.mysteryRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(
	ctx context.Context,
	commentID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	fileSize int64,
	reader io.Reader,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.mysteryRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, fmt.Errorf("not the comment author")
	}

	isVideo := strings.HasPrefix(contentType, "video/")
	mediaID := uuid.New()

	var urlPath string
	if isVideo {
		maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		urlPath, err = s.uploadSvc.SaveVideo(ctx, "mysteries", mediaID, contentType, fileSize, maxSize, reader)
	} else {
		maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		urlPath, err = s.uploadSvc.SaveImage(ctx, "mysteries", mediaID, contentType, fileSize, maxSize, reader)
	}
	if err != nil {
		return nil, err
	}

	existing, _ := s.mysteryRepo.GetCommentMedia(ctx, commentID)
	sortOrder := len(existing)

	mediaType := "image"
	if isVideo {
		mediaType = "video"
	}

	dbID, err := s.mysteryRepo.AddCommentMedia(ctx, commentID, urlPath, mediaType, "", sortOrder)
	if err != nil {
		return nil, err
	}

	return &dto.PostMediaResponse{
		ID:        int(dbID),
		MediaURL:  urlPath,
		MediaType: mediaType,
		SortOrder: sortOrder,
	}, nil
}
