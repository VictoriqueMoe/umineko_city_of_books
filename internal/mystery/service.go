package mystery

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
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
		GetTopDetectiveIDs(ctx context.Context) ([]string, error)
		GetGMLeaderboard(ctx context.Context, limit int) (*dto.GMLeaderboardResponse, error)
		GetTopGMIDs(ctx context.Context) ([]string, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) (*dto.MysteryListResponse, error)
		CreateComment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)
		UploadAttachment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, fileName string, fileSize int64, reader io.Reader) (*dto.MysteryAttachment, error)
		DeleteAttachment(ctx context.Context, attachmentID int64, mysteryID uuid.UUID, userID uuid.UUID) error
		SetPaused(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, paused bool) error
		SetGmAway(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, away bool) error
		DeleteClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID) error
		UpdateClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID, body string) error
	}

	service struct {
		mysteryRepo  repository.MysteryRepository
		userRepo     repository.UserRepository
		authz        authz.Service
		blockSvc     block.Service
		notifService notification.Service
		settingsSvc  settings.Service
		uploadSvc    upload.Service
		uploader     *media.Uploader
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
	mediaProc *media.Processor,
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
		uploader:     media.NewUploader(uploadSvc, settingsSvc, mediaProc),
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

	attachments, _ := s.mysteryRepo.GetAttachments(ctx, id)
	if attachments == nil {
		attachments = []dto.MysteryAttachment{}
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
	if err := s.mysteryRepo.Create(ctx, id, userID, req.Title, req.Body, req.Difficulty, req.FreeForAll); err != nil {
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
	if !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return fmt.Errorf("not authorised")
	}

	old, err := s.mysteryRepo.GetByID(ctx, id)
	if err != nil || old == nil {
		return ErrNotFound
	}
	oldClues, _ := s.mysteryRepo.GetClues(ctx, id)

	if err := s.mysteryRepo.UpdateAsAdmin(ctx, id, req.Title, req.Body, req.Difficulty, req.FreeForAll); err != nil {
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
		if len(changes) > 0 {
			message := fmt.Sprintf("your mystery was edited (changed: %s)", strings.Join(changes, ", "))
			go func() {
				bgCtx := context.Background()
				baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
				linkURL := fmt.Sprintf("%s/mystery/%s", baseURL, id)
				subject, body := notification.NotifEmail("A moderator", "edited your mystery", "", linkURL)
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   old.UserID,
					Type:          dto.NotifContentEdited,
					ReferenceID:   id,
					ReferenceType: "mystery",
					ActorID:       userID,
					Message:       message,
					EmailSubject:  subject,
					EmailBody:     body,
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
	if paused, _ := s.mysteryRepo.IsPaused(ctx, mysteryID); paused && authorID != userID {
		return uuid.Nil, ErrMysteryPaused
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
	if attemptAuthorID == authorID {
		return fmt.Errorf("cannot select your own attempt as the winner")
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

		playerIDs, _ := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		solvedLink := fmt.Sprintf("%s/mystery/%s", baseURL, mysteryID)
		solvedSubject, solvedBody := notification.NotifEmail("The Game Master", "solved a mystery you were playing", "", solvedLink)
		for _, pid := range playerIDs {
			if pid == attemptAuthorID {
				continue
			}
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   pid,
				Type:          dto.NotifMysterySolvedAll,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				Message:       "a mystery you were playing has been solved",
				EmailSubject:  solvedSubject,
				EmailBody:     solvedBody,
			})
		}

		topIDs, err := s.mysteryRepo.GetTopDetectiveIDs(bgCtx)
		if err == nil {
			s.hub.Broadcast(ws.Message{
				Type: "top_detective_changed",
				Data: map[string]interface{}{
					"user_ids": topIDs,
				},
			})
		}
		topGMIDs, err := s.mysteryRepo.GetTopGMIDs(bgCtx)
		if err == nil {
			s.hub.Broadcast(ws.Message{
				Type: "top_gm_changed",
				Data: map[string]interface{}{
					"user_ids": topGMIDs,
				},
			})
		}
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

	count, _ := s.mysteryRepo.CountClues(ctx, mysteryID)
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

		recipient := *req.PlayerID
		go func() {
			bgCtx := context.Background()
			baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
			linkURL := fmt.Sprintf("%s/mystery/%s", baseURL, mysteryID)
			subject, body := notification.NotifEmail("The Game Master", "revealed a private red truth to you", "", linkURL)
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   recipient,
				Type:          dto.NotifMysteryPrivateClue,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     body,
			})
		}()
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
			ScoreAdjustment: r.ScoreAdjustment,
		}
	}
	return &dto.MysteryLeaderboardResponse{Entries: entries}, nil
}

func (s *service) GetTopDetectiveIDs(ctx context.Context) ([]string, error) {
	return s.mysteryRepo.GetTopDetectiveIDs(ctx)
}

func (s *service) GetGMLeaderboard(ctx context.Context, limit int) (*dto.GMLeaderboardResponse, error) {
	rows, err := s.mysteryRepo.GetGMLeaderboard(ctx, limit)
	if err != nil {
		return nil, err
	}
	entries := make([]dto.GMLeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = dto.GMLeaderboardEntry{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			Score:        r.Score,
			MysteryCount: r.MysteryCount,
			PlayerCount:  r.PlayerCount,
		}
	}
	return &dto.GMLeaderboardResponse{Entries: entries}, nil
}

func (s *service) GetTopGMIDs(ctx context.Context) ([]string, error) {
	return s.mysteryRepo.GetTopGMIDs(ctx)
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

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mystery/%s#comment-%s", baseURL, mysteryID, id)

		if req.ParentID != nil {
			parentAuthor, err := s.mysteryRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err != nil || parentAuthor == userID {
				return
			}
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
		} else if authorID != userID {
			subject, emailBody := notification.NotifEmail(actor.DisplayName, "commented on your mystery", "", linkURL)
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifMysteryCommentReply,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_comment:%s", id),
				ActorID:       userID,
				EmailSubject:  subject,
				EmailBody:     emailBody,
			})
		}
	}()

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

	existing, _ := s.mysteryRepo.GetCommentMedia(ctx, commentID)
	sortOrder := len(existing)

	resp, err := s.uploader.SaveAndRecord(ctx, "mysteries", contentType, fileSize, reader,
		func(mediaURL, mediaType, _ string, _ int) (int64, error) {
			return s.mysteryRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, "", sortOrder)
		},
		s.mysteryRepo.UpdateCommentMediaURL,
		s.mysteryRepo.UpdateCommentMediaThumbnail,
	)
	if err != nil {
		return nil, err
	}
	resp.SortOrder = sortOrder
	return resp, nil
}

func (s *service) UploadAttachment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, fileName string, fileSize int64, reader io.Reader) (*dto.MysteryAttachment, error) {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return nil, ErrNotAuthor
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxGeneralSize))
	if fileSize > maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed (%dMB)", maxSize/(1024*1024))
	}

	existing, _ := s.mysteryRepo.GetAttachments(ctx, mysteryID)
	for _, a := range existing {
		if a.FileName == fileName {
			return nil, fmt.Errorf("a file named %q is already attached", fileName)
		}
	}

	subDir := "mystery-attachments/" + mysteryID.String()
	urlPath, err := s.uploadSvc.SaveFile(subDir, fileName, reader)
	if err != nil {
		return nil, err
	}

	dbID, err := s.mysteryRepo.AddAttachment(ctx, mysteryID, urlPath, fileName, int(fileSize))
	if err != nil {
		return nil, err
	}

	return &dto.MysteryAttachment{
		ID:       int(dbID),
		FileURL:  urlPath,
		FileName: fileName,
		FileSize: int(fileSize),
	}, nil
}

func (s *service) DeleteAttachment(ctx context.Context, attachmentID int64, mysteryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}

	attachments, _ := s.mysteryRepo.GetAttachments(ctx, mysteryID)
	var fileURL string
	for _, a := range attachments {
		if a.ID == int(attachmentID) {
			fileURL = a.FileURL
			break
		}
	}

	if err := s.mysteryRepo.DeleteAttachment(ctx, attachmentID, mysteryID); err != nil {
		return err
	}

	if fileURL != "" {
		diskPath := s.uploadSvc.GetUploadDir() + strings.TrimPrefix(fileURL, "/uploads")
		os.Remove(diskPath)
	}

	return nil
}

func (s *service) SetPaused(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, paused bool) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.mysteryRepo.SetPaused(ctx, mysteryID, paused); err != nil {
		return err
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_paused",
		Data: map[string]interface{}{
			"mystery_id": mysteryID,
			"paused":     paused,
		},
	})

	go func() {
		bgCtx := context.Background()
		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil || len(playerIDs) == 0 {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mystery/%s", baseURL, mysteryID)

		notifType := dto.NotifMysteryPaused
		message := "paused a mystery you are playing"
		if !paused {
			notifType = dto.NotifMysteryUnpaused
			message = "resumed a mystery you are playing"
		}
		subject, body := notification.NotifEmail("The Game Master", message, "", linkURL)
		for _, pid := range playerIDs {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   pid,
				Type:          notifType,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				Message:       message,
				EmailSubject:  subject,
				EmailBody:     body,
			})
		}
	}()

	return nil
}

func (s *service) SetGmAway(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, away bool) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return ErrNotAuthor
	}
	if err := s.mysteryRepo.SetGmAway(ctx, mysteryID, away); err != nil {
		return err
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_gm_away",
		Data: map[string]interface{}{
			"mystery_id": mysteryID,
			"gm_away":    away,
		},
	})

	go func() {
		bgCtx := context.Background()
		playerIDs, err := s.mysteryRepo.GetPlayerIDs(bgCtx, mysteryID)
		if err != nil || len(playerIDs) == 0 {
			return
		}
		baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
		linkURL := fmt.Sprintf("%s/mystery/%s", baseURL, mysteryID)

		notifType := dto.NotifMysteryGmAway
		message := "marked themselves as away on a mystery you are playing"
		if !away {
			notifType = dto.NotifMysteryGmBack
			message = "is back on a mystery you are playing"
		}
		subject, body := notification.NotifEmail("The Game Master", message, "", linkURL)
		for _, pid := range playerIDs {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   pid,
				Type:          notifType,
				ReferenceID:   mysteryID,
				ReferenceType: "mystery",
				ActorID:       userID,
				Message:       message,
				EmailSubject:  subject,
				EmailBody:     body,
			})
		}
	}()

	return nil
}

func (s *service) DeleteClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID) error {
	if err := s.mysteryRepo.DeleteClue(ctx, clueID); err != nil {
		return err
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_clue_updated",
		Data: map[string]interface{}{"mystery_id": mysteryID},
	})
	return nil
}

func (s *service) UpdateClue(ctx context.Context, mysteryID uuid.UUID, clueID int, userID uuid.UUID, body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrEmptyBody
	}
	if err := s.mysteryRepo.UpdateClue(ctx, clueID, strings.TrimSpace(body)); err != nil {
		return err
	}
	s.hub.Broadcast(ws.Message{
		Type: "mystery_clue_updated",
		Data: map[string]interface{}{"mystery_id": mysteryID},
	})
	return nil
}
