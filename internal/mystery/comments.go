package mystery

import (
	"context"
	"fmt"
	"io"
	"strings"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

func (s *service) CreateComment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, req dto.CreateCommentRequest) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return uuid.Nil, ErrEmptyBody
	}
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return uuid.Nil, err
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

	id, err := s.mentionSvc.CreateComment(ctx, mention.CommentSpec{
		Kind:     mention.KindMysteryComment,
		EntityID: mysteryID,
		ParentID: req.ParentID,
		AuthorID: userID,
		Body:     body,
	})
	if err != nil {
		return uuid.Nil, err
	}

	go func() {
		bgCtx := context.Background()
		actor, err := s.userRepo.GetByID(bgCtx, userID)
		if err != nil || actor == nil {
			return
		}
		linkPath := fmt.Sprintf("/mystery/%s#comment-%s", mysteryID, id)

		if req.ParentID != nil {
			parentAuthor, err := s.mysteryRepo.GetCommentAuthorID(bgCtx, *req.ParentID)
			if err != nil || parentAuthor == userID {
				return
			}
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   parentAuthor,
				Type:          dto.NotifMysteryCommentReply,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "replied to your comment",
				EmailLink:     linkPath,
			})
		} else if authorID != userID {
			_ = s.notifService.Notify(bgCtx, dto.NotifyParams{
				RecipientID:   authorID,
				Type:          dto.NotifMysteryCommentReply,
				ReferenceID:   mysteryID,
				ReferenceType: fmt.Sprintf("mystery_comment:%s", id),
				ActorID:       userID,
				EmailActor:    actor.DisplayName,
				EmailAction:   "commented on your mystery",
				EmailLink:     linkPath,
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
	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}
	if !s.authz.Can(ctx, userID, authz.PermEditAnyComment) {
		return s.mysteryRepo.UpdateComment(ctx, id, userID, body)
	}

	authorID, err := s.mysteryRepo.GetCommentAuthorID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	if err := s.mysteryRepo.UpdateCommentAsAdmin(ctx, id, body); err != nil {
		return err
	}

	if authorID != userID {
		s.audit(ctx, repository.NewAuditEntry{
			ActorID:    userID,
			Action:     repository.AuditActionMysteryCommentUpdateAdmin,
			TargetType: repository.AuditTargetMysteryComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		})
	}

	return nil
}

func (s *service) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	isAdmin := s.authz.Can(ctx, userID, authz.PermDeleteAnyComment)

	paths, err := s.mysteryRepo.DeleteCommentWithAudit(ctx, repository.MysteryCommentDelete{
		ID:      id,
		UserID:  userID,
		AsAdmin: isAdmin,
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	return nil
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
	filename string,
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

	resp, err := s.uploader.SaveAndRecord(ctx, "mysteries", contentType, filename, fileSize, reader,
		func(mediaURL, mediaType, _, filename string, _ int) (int64, error) {
			return s.mysteryRepo.AddCommentMedia(ctx, repository.NewMysteryCommentMedia{
				CommentID: commentID,
				MediaURL:  mediaURL,
				MediaType: mediaType,
				Filename:  filename,
				SortOrder: sortOrder,
			})
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

func mysteryCommentToResponse(c repository.CommentRow, media []model.PostMediaRow) dto.MysteryCommentResponse {
	mediaList := model.MediaRowsToResponse(media)
	return dto.MysteryCommentResponse{
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
		Media:     mediaList,
		LikeCount: c.LikeCount,
		UserLiked: c.UserLiked,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
