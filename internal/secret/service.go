package secret

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		List(ctx context.Context, viewerID uuid.UUID) (*dto.SecretListResponse, error)
		Get(ctx context.Context, id string, viewerID uuid.UUID) (*dto.SecretDetailResponse, error)

		CreateComment(ctx context.Context, secretID string, userID uuid.UUID, req dto.CreateCommentRequest, files []upload.FileUpload) (uuid.UUID, error)

		BroadcastProgress(ctx context.Context, parentID string, actor uuid.UUID)
		BroadcastSolved(ctx context.Context, parentID string, actor uuid.UUID, solvedAt string)
	}

	service struct {
		secretRepo    repository.SecretRepository
		userSecretSvc repository.UserSecretRepository
		userRepo      repository.UserRepository
		commentRepo   repository.CommentRepository
		notifService  notification.Service
		commentSvc    comment.Service
		hub           *ws.Hub
		activity      sidebar.ActivityBumper
	}
)

func NewService(
	secretRepo repository.SecretRepository,
	userSecretRepo repository.UserSecretRepository,
	userRepo repository.UserRepository,
	commentRepo repository.CommentRepository,
	notifService notification.Service,
	commentSvc comment.Service,
	hub *ws.Hub,
	activity sidebar.ActivityBumper,
) Service {
	return &service{
		secretRepo:    secretRepo,
		userSecretSvc: userSecretRepo,
		userRepo:      userRepo,
		commentRepo:   commentRepo,
		notifService:  notifService,
		commentSvc:    commentSvc,
		hub:           hub,
		activity:      activity,
	}
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

	commentCounts, err := s.secretRepo.CountCommentsBySecret(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("secret comment counts: %w", err)
	}

	result := make([]dto.SecretSummary, 0, len(listed))
	for i := range listed {
		summary, err := s.buildSummary(ctx, listed[i], viewerID, commentCounts[string(listed[i].ID)])
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
	}

	solverRows, err := s.secretRepo.GetSolversLeaderboard(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("secret solvers leaderboard: %w", err)
	}
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

func (s *service) buildSummary(ctx context.Context, secretSpec secrets.Spec, viewerID uuid.UUID, commentCount int) (dto.SecretSummary, error) {
	pieceIDs := secrets.PieceIDStrings(secretSpec)
	summary := dto.SecretSummary{
		ID:           string(secretSpec.ID),
		Title:        secretSpec.Title,
		Description:  secretSpec.Description,
		TotalPieces:  len(pieceIDs),
		CommentCount: commentCount,
	}

	solver, err := s.secretRepo.GetFirstSolver(ctx, string(secretSpec.ID))
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
		count, err := s.secretRepo.GetPieceCountForUser(ctx, spec.SecretPieceCount{UserID: viewerID, PieceIDs: pieceIDs})
		if err != nil {
			return summary, fmt.Errorf("viewer progress: %w", err)
		}
		summary.ViewerProgress = count
	}
	return summary, nil
}

func (s *service) Get(ctx context.Context, id string, viewerID uuid.UUID) (*dto.SecretDetailResponse, error) {
	secretSpec, ok := secrets.Lookup(id)
	if !ok || secretSpec.Title == "" {
		return nil, ErrNotFound
	}

	commentCounts, err := s.secretRepo.CountCommentsBySecret(ctx, []string{id})
	if err != nil {
		return nil, fmt.Errorf("secret comment counts: %w", err)
	}

	summary, err := s.buildSummary(ctx, secretSpec, viewerID, commentCounts[id])
	if err != nil {
		return nil, err
	}

	pieceIDs := secrets.PieceIDStrings(secretSpec)
	var leaderboard []dto.SecretLeaderboardEntry
	if len(pieceIDs) > 0 {
		rows, err := s.secretRepo.GetProgressLeaderboard(ctx, pieceIDs)
		if err != nil {
			return nil, err
		}
		solvedUsers, err := s.solvedUserSet(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("secret solvers: %w", err)
		}
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

	comments, err := s.commentSvc.Thread(ctx, spec.CommentTarget{Kind: spec.CommentKindSecret, Key: id}, viewerID)
	if err != nil {
		return nil, fmt.Errorf("secret comments: %w", err)
	}

	return &dto.SecretDetailResponse{
		SecretSummary: summary,
		Riddle:        secretSpec.Riddle,
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

func (s *service) CreateComment(ctx context.Context, secretID string, userID uuid.UUID, req dto.CreateCommentRequest, files []upload.FileUpload) (uuid.UUID, error) {
	secretSpec, ok := secrets.Lookup(secretID)
	if !ok || secretSpec.Title == "" {
		return uuid.Nil, ErrNotFound
	}

	id, err := s.commentSvc.Create(ctx, spec.CommentTarget{Kind: spec.CommentKindSecret, Key: secretID}, userID, req, files)
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("user_id", userID.String()).Msg("secret comment notifications skipped, actor lookup failed")

			return
		}
		if actor == nil {
			return
		}

		var parentAuthor uuid.UUID
		if req.ParentID != nil {
			parentAuthor, err = s.commentRepo.GetCommentAuthorID(bgCtx, spec.CommentRef{Kind: spec.CommentKindSecret, CommentID: *req.ParentID})
			if err != nil {
				logger.Ctx(bgCtx).Warn().Err(err).Str("comment_id", req.ParentID.String()).Msg("secret comment fan-out skipped, parent lookup failed")

				return
			}
		}

		commenters, err := s.secretRepo.GetCommenterIDs(bgCtx, secretID)
		if err != nil {
			logger.Ctx(bgCtx).Warn().Err(err).Str("secret_id", secretID).Msg("secret comment fan-out skipped, commenter lookup failed")

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

	s.activity.BumpSidebarActivity(sidebar.KeySecrets)

	return id, nil
}

func (s *service) BroadcastProgress(ctx context.Context, parentID string, actor uuid.UUID) {
	secretSpec, ok := secrets.Lookup(parentID)
	if !ok || secretSpec.Title == "" {
		return
	}
	pieceIDs := secrets.PieceIDStrings(secretSpec)
	summary, err := s.secretRepo.GetUserProgressSummary(ctx, spec.SecretPieceCount{UserID: actor, PieceIDs: pieceIDs})
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("secret_id", parentID).Msg("secret progress broadcast skipped, summary lookup failed")

		return
	}
	if summary == nil {
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
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("secret_id", parentID).Msg("secret solved broadcast skipped, solver lookup failed")

		return
	}
	if user == nil {
		return
	}
	secretSpec, specOK := secrets.Lookup(parentID)
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

	if specOK && secretSpec.VanityRoleID != "" {
		s.hub.Broadcast(ws.Message{Type: "vanity_roles_changed", Data: map[string]any{}})
	}

	if specOK && secretSpec.Title != "" {
		pieceIDs := secrets.PieceIDStrings(secretSpec)
		participants, err := s.userSecretSvc.GetUserIDsWithAnyPiece(ctx, pieceIDs)
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("secret_id", parentID).Msg("secret closed notices skipped, participant lookup failed")
		}
		if err == nil {
			closedData := map[string]any{
				"secret_id":    parentID,
				"secret_title": secretSpec.Title,
				"solver":       event.Solver,
			}
			message := fmt.Sprintf("solved %s before you could. Uu~ try again next time.", secretSpec.Title)
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
