package secret

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

var (
	ErrNotFound    = errors.New("secret not found")
	ErrEmptyBody   = errors.New("comment body cannot be empty")
	ErrNotOwner    = errors.New("not the comment author")
	ErrForbidden   = errors.New("forbidden")
	ErrUserBlocked = block.ErrUserBlocked
)

type (
	Service interface {
		List(ctx context.Context, viewerID uuid.UUID) (*dto.SecretListResponse, error)
		Get(ctx context.Context, id string, viewerID uuid.UUID) (*dto.SecretDetailResponse, error)

		CreateComment(ctx context.Context, secretID string, userID uuid.UUID, req dto.CreateSecretCommentRequest) (uuid.UUID, error)
		UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateSecretCommentRequest) error
		DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
		LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error
		UploadCommentMedia(ctx context.Context, commentID uuid.UUID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.PostMediaResponse, error)

		BroadcastProgress(ctx context.Context, parentID string, actor uuid.UUID)
		BroadcastSolved(ctx context.Context, parentID string, actor uuid.UUID, solvedAt string)
	}

	service struct {
		secretRepo    repository.SecretRepository
		userSecretSvc repository.UserSecretRepository
		userRepo      repository.UserRepository
		auditRepo     repository.AuditLogRepository
		authz         authz.Service
		blockSvc      block.Service
		notifService  notification.Service
		settingsSvc   settings.Service
		uploader      *media.Uploader
		hub           *ws.Hub
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	secretRepo repository.SecretRepository,
	userSecretRepo repository.UserSecretRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	blockSvc block.Service,
	notifService notification.Service,
	settingsSvc settings.Service,
	uploadSvc upload.Service,
	mediaProc *media.Processor,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		secretRepo:    secretRepo,
		userSecretSvc: userSecretRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		authz:         authzService,
		blockSvc:      blockSvc,
		notifService:  notifService,
		settingsSvc:   settingsSvc,
		uploader:      media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		hub:           hub,
		contentFilter: contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func secretRoomID(id string) string {
	return "secret:" + id
}

func (s *service) List(ctx context.Context, viewerID uuid.UUID) (*dto.SecretListResponse, error) {
	listed := secrets.Listed()
	ids := make([]string, len(listed))
	for i := range listed {
		ids[i] = string(listed[i].ID)
	}

	commentCounts, _ := s.secretRepo.CountCommentsBySecret(ctx, ids)

	result := make([]dto.SecretSummary, 0, len(listed))
	for i := range listed {
		summary, err := s.buildSummary(ctx, listed[i], viewerID, commentCounts[string(listed[i].ID)])
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
	}

	solverRows, _ := s.secretRepo.GetSolversLeaderboard(ctx, ids)
	solvers := make([]dto.SecretSolverEntry, len(solverRows))
	for i := range solverRows {
		r := solverRows[i]
		solvers[i] = dto.SecretSolverEntry{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			Solved:     r.SolvedCount,
			LastSolved: r.LastSolvedAt,
		}
	}

	return &dto.SecretListResponse{Secrets: result, SolversLeaderboard: solvers}, nil
}

func (s *service) buildSummary(ctx context.Context, spec secrets.Spec, viewerID uuid.UUID, commentCount int) (dto.SecretSummary, error) {
	pieceIDs := secrets.PieceIDStrings(spec)
	summary := dto.SecretSummary{
		ID:           string(spec.ID),
		Title:        spec.Title,
		Description:  spec.Description,
		TotalPieces:  len(pieceIDs),
		CommentCount: commentCount,
	}

	solver, err := s.secretRepo.GetFirstSolver(ctx, string(spec.ID))
	if err != nil {
		return summary, err
	}
	if solver != nil {
		summary.Solved = true
		summary.SolvedAt = solver.UnlockedAt
		summary.Solver = &dto.UserResponse{
			ID:          solver.UserID,
			Username:    solver.Username,
			DisplayName: solver.DisplayName,
			AvatarURL:   solver.AvatarURL,
			Role:        role.Role(solver.Role),
		}
	}

	if viewerID != uuid.Nil && len(pieceIDs) > 0 {
		count, _ := s.secretRepo.GetPieceCountForUser(ctx, viewerID, pieceIDs)
		summary.ViewerProgress = count
	}
	return summary, nil
}

func (s *service) Get(ctx context.Context, id string, viewerID uuid.UUID) (*dto.SecretDetailResponse, error) {
	spec, ok := secrets.Lookup(id)
	if !ok || spec.Title == "" {
		return nil, ErrNotFound
	}

	commentCounts, _ := s.secretRepo.CountCommentsBySecret(ctx, []string{id})
	summary, err := s.buildSummary(ctx, spec, viewerID, commentCounts[id])
	if err != nil {
		return nil, err
	}

	pieceIDs := secrets.PieceIDStrings(spec)
	var leaderboard []dto.SecretLeaderboardEntry
	if len(pieceIDs) > 0 {
		rows, err := s.secretRepo.GetProgressLeaderboard(ctx, pieceIDs)
		if err != nil {
			return nil, err
		}
		solvedUsers, _ := s.solvedUserSet(ctx, id)
		for _, r := range rows {
			leaderboard = append(leaderboard, dto.SecretLeaderboardEntry{
				User: dto.UserResponse{
					ID:          r.UserID,
					Username:    r.Username,
					DisplayName: r.DisplayName,
					AvatarURL:   r.AvatarURL,
					Role:        role.Role(r.Role),
				},
				Pieces: r.Pieces,
				Solved: solvedUsers[r.UserID],
			})
		}
	}
	if leaderboard == nil {
		leaderboard = []dto.SecretLeaderboardEntry{}
	}

	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	commentRows, _, _ := s.secretRepo.GetComments(ctx, id, viewerID, 500, 0, blockedIDs)
	var comments []dto.SecretCommentResponse
	if len(commentRows) > 0 {
		commentIDs := make([]uuid.UUID, len(commentRows))
		for i := range commentRows {
			commentIDs[i] = commentRows[i].ID
		}
		mediaBatch, _ := s.secretRepo.GetCommentMediaBatch(ctx, commentIDs)
		flat := make([]dto.SecretCommentResponse, len(commentRows))
		for i := range commentRows {
			flat[i] = secretCommentToResponse(commentRows[i], mediaBatch[commentRows[i].ID])
		}
		comments = utils.BuildTree(flat,
			func(c dto.SecretCommentResponse) uuid.UUID { return c.ID },
			func(c dto.SecretCommentResponse) *uuid.UUID { return c.ParentID },
			func(c *dto.SecretCommentResponse, replies []dto.SecretCommentResponse) { c.Replies = replies },
		)
	}
	if comments == nil {
		comments = []dto.SecretCommentResponse{}
	}

	return &dto.SecretDetailResponse{
		SecretSummary: summary,
		Riddle:        spec.Riddle,
		Leaderboard:   leaderboard,
		Comments:      comments,
	}, nil
}

func (s *service) solvedUserSet(ctx context.Context, parentID string) (map[uuid.UUID]bool, error) {
	ids, err := s.userSecretSvc.GetUserIDsWithSecret(ctx, parentID)
	if err != nil {
		return nil, err
	}
	set := make(map[uuid.UUID]bool, len(ids))
	for i := range ids {
		set[ids[i]] = true
	}
	return set, nil
}

func (s *service) CreateComment(ctx context.Context, secretID string, userID uuid.UUID, req dto.CreateSecretCommentRequest) (uuid.UUID, error) {
	spec, ok := secrets.Lookup(secretID)
	if !ok || spec.Title == "" {
		return uuid.Nil, ErrNotFound
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	if err := s.secretRepo.CreateComment(ctx, id, secretID, req.ParentID, userID, body); err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		var parentAuthor uuid.UUID
		if req.ParentID != nil {
			if parent, err := s.secretRepo.GetCommentAuthorID(bgCtx, *req.ParentID); err == nil && parent != userID {
				parentAuthor = parent
				_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
					RecipientID:   parentAuthor,
					Type:          dto.NotifSecretCommentReply,
					ReferenceType: fmt.Sprintf("secret_comment:%s:%s", secretID, id),
					ActorID:       userID,
					EmailActor:    actor.DisplayName,
					EmailAction:   "replied to your comment",
					EmailLink:     fmt.Sprintf("/secrets/%s#comment-%s", secretID, id),
				})
			}
		}

		commenters, err := s.secretRepo.GetCommenterIDs(bgCtx, secretID)
		if err != nil {
			return
		}
		for _, rid := range commenters {
			if rid == userID || rid == parentAuthor {
				continue
			}
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   rid,
				Type:          dto.NotifSecretCommented,
				ReferenceType: fmt.Sprintf("secret_comment:%s:%s", secretID, id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "posted a new comment on a hunt you're following",
				EmailLink:     fmt.Sprintf("/secrets/%s#comment-%s", secretID, id),
			})
		}
	}()

	return id, nil
}

func (s *service) UpdateComment(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.UpdateSecretCommentRequest) error {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return ErrEmptyBody
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return err
	}
	if s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.secretRepo.UpdateCommentAsAdmin(ctx, id, body)
	}
	return s.secretRepo.UpdateComment(ctx, id, userID, body)
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	isAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyComment)
	action := "secret_comment_delete"
	if isAdmin {
		if err := s.secretRepo.DeleteCommentAsAdmin(ctx, id); err != nil {
			return err
		}
		action = "secret_comment_delete_admin"
	} else {
		if err := s.secretRepo.DeleteComment(ctx, id, userID); err != nil {
			return err
		}
	}
	if err := s.auditRepo.Create(ctx, userID, action, "secret_comment", id.String(), ""); err != nil {
		return fmt.Errorf("audit comment delete: %w", err)
	}
	return nil
}

func (s *service) LikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	commentAuthorID, err := s.secretRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, commentAuthorID); blocked {
		return ErrUserBlocked
	}
	if err := s.secretRepo.LikeComment(ctx, userID, commentID); err != nil {
		return err
	}

	go func() {
		bgCtx := context.Background()
		if commentAuthorID == userID {
			return
		}
		secretID, err := s.secretRepo.GetCommentEntityID(bgCtx, commentID)
		if err != nil || secretID == "" {
			return
		}
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   commentAuthorID,
			Type:          dto.NotifSecretCommentLiked,
			ReferenceType: fmt.Sprintf("secret_comment:%s:%s", secretID, commentID),
			ActorID:       userID,
			EmailActor:    actor.DisplayName,
			EmailAction:   "liked your comment",
			EmailLink:     fmt.Sprintf("/secrets/%s#comment-%s", secretID, commentID),
		})
	}()

	return nil
}

func (s *service) UnlikeComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	return s.secretRepo.UnlikeComment(ctx, userID, commentID)
}

func (s *service) UploadCommentMedia(
	ctx context.Context,
	commentID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	fileSize int64,
	reader io.Reader,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.secretRepo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID {
		return nil, ErrNotOwner
	}

	existing, _ := s.secretRepo.GetCommentMedia(ctx, commentID)
	sortOrder := len(existing)

	resp, err := s.uploader.SaveAndRecord(ctx, "secrets", contentType, fileSize, reader,
		func(mediaURL, mediaType, _ string, _ int) (int64, error) {
			return s.secretRepo.AddCommentMedia(ctx, commentID, mediaURL, mediaType, "", sortOrder)
		},
		s.secretRepo.UpdateCommentMediaURL,
		s.secretRepo.UpdateCommentMediaThumbnail,
	)
	if err != nil {
		return nil, err
	}
	resp.SortOrder = sortOrder
	return resp, nil
}

func (s *service) BroadcastProgress(ctx context.Context, parentID string, actor uuid.UUID) {
	spec, ok := secrets.Lookup(parentID)
	if !ok || spec.Title == "" {
		return
	}
	pieceIDs := secrets.PieceIDStrings(spec)
	summary, err := s.secretRepo.GetUserProgressSummary(ctx, actor, pieceIDs)
	if err != nil || summary == nil {
		return
	}
	event := dto.SecretProgressEvent{
		SecretID: parentID,
		User: dto.UserResponse{
			ID:          summary.UserID,
			Username:    summary.Username,
			DisplayName: summary.DisplayName,
			AvatarURL:   summary.AvatarURL,
			Role:        role.Role(summary.Role),
		},
		Pieces:      summary.Pieces,
		TotalPieces: len(pieceIDs),
	}
	s.hub.BroadcastToTopic(secretRoomID(parentID), ws.Message{Type: "secret_progress", Data: event})
}

func (s *service) BroadcastSolved(ctx context.Context, parentID string, actor uuid.UUID, solvedAt string) {
	user, err := s.userRepo.GetByID(ctx, actor)
	if err != nil || user == nil {
		return
	}
	spec, specOK := secrets.Lookup(parentID)
	solverName := user.DisplayName
	if solverName == "" {
		solverName = user.Username
	}
	event := dto.SecretSolvedEvent{
		SecretID: parentID,
		Solver: dto.UserResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			Role:        role.Role(user.Role),
		},
		SolvedAt: solvedAt,
	}
	s.hub.BroadcastToTopic(secretRoomID(parentID), ws.Message{Type: "secret_solved", Data: event})

	if specOK && spec.VanityRoleID != "" {
		s.hub.Broadcast(ws.Message{Type: "vanity_roles_changed", Data: map[string]interface{}{}})
	}

	if specOK && spec.Title != "" {
		pieceIDs := secrets.PieceIDStrings(spec)
		participants, err := s.userSecretSvc.GetUserIDsWithAnyPiece(ctx, pieceIDs)
		if err == nil {
			closedData := map[string]interface{}{
				"secret_id":    parentID,
				"secret_title": spec.Title,
				"solver":       event.Solver,
			}
			message := fmt.Sprintf("solved %s before you could. Uu~ try again next time.", spec.Title)
			emailLink := fmt.Sprintf("/secrets/%s", parentID)
			go func(participants []uuid.UUID, actorName, action, link, msg string) {
				bgCtx := context.Background()
				for _, pid := range participants {
					if pid == actor {
						continue
					}
					_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
						RecipientID:   pid,
						Type:          dto.NotifSecretSolvedByOther,
						ReferenceType: fmt.Sprintf("secret:%s", parentID),
						ActorID:       actor,
						Message:       msg,
						EmailActor:    actorName,
						EmailAction:   action,
						EmailLink:     link,
					})
					s.hub.SendToUser(pid, ws.Message{Type: "secret_closed", Data: closedData})
				}
			}(participants, solverName, message, emailLink, message)
		}
	}
}

func secretCommentToResponse(c repository.CommentRow, media []model.PostMediaRow) dto.SecretCommentResponse {
	return dto.SecretCommentResponse{
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
