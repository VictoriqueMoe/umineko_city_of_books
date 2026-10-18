package comment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

const (
	threadLimit       = 500
	replyAction       = "replied to your comment"
	likeAction        = "liked your comment"
	commentedAction   = "commented on your "
	editedContentType = "comment"
)

type (
	Service interface {
		Create(ctx context.Context, target spec.CommentTarget, actorID uuid.UUID, req dto.CreateCommentRequest, files []upload.FileUpload) (uuid.UUID, error)
		Update(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID, body string) error
		Delete(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID) error
		Like(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID) error
		Unlike(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID) error
		UploadMedia(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error)
		Thread(ctx context.Context, target spec.CommentTarget, viewerID uuid.UUID) ([]dto.CommentResponse, error)
	}

	service struct {
		repo          repository.CommentRepository
		userRepo      repository.UserRepository
		authz         authz.Service
		blockSvc      block.Service
		notifSvc      notification.Service
		mentionSvc    mention.Service
		uploadSvc     upload.Service
		uploader      upload.MediaRecorder
		settingsSvc   settings.Service
		contentFilter *contentfilter.Manager
	}
)

func NewService(
	repo repository.CommentRepository,
	userRepo repository.UserRepository,
	authzSvc authz.Service,
	blockSvc block.Service,
	notifSvc notification.Service,
	mentionSvc mention.Service,
	uploadSvc upload.Service,
	uploader upload.MediaRecorder,
	settingsSvc settings.Service,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		repo:          repo,
		userRepo:      userRepo,
		authz:         authzSvc,
		blockSvc:      blockSvc,
		notifSvc:      notifSvc,
		mentionSvc:    mentionSvc,
		uploadSvc:     uploadSvc,
		uploader:      uploader,
		settingsSvc:   settingsSvc,
		contentFilter: contentFilter,
	}
}

func (s *service) Create(ctx context.Context, target spec.CommentTarget, actorID uuid.UUID, req dto.CreateCommentRequest, files []upload.FileUpload) (uuid.UUID, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" && len(files) == 0 {
		return uuid.Nil, ErrEmptyBody
	}

	if body != "" {
		if err := s.contentFilter.Check(ctx, body); err != nil {
			return uuid.Nil, err
		}
	}

	validated, err := upload.ValidateMedia(ctx, s.settingsSvc, files)
	if err != nil {
		return uuid.Nil, err
	}

	ownerID, err := s.repo.GetTargetOwnerID(ctx, target)
	if errors.Is(err, dao.ErrNotFound) {
		return uuid.Nil, ErrTargetNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	if ownerID != uuid.Nil {
		blocked, err := s.blockSvc.IsBlockedEither(ctx, actorID, ownerID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("block check: %w", err)
		}
		if blocked {
			return uuid.Nil, block.ErrUserBlocked
		}
	}

	mediaDir, err := s.repo.GetCommentMediaDir(target.Kind)
	if err != nil {
		return uuid.Nil, err
	}

	created, err := s.repo.CreateComment(ctx, spec.NewComment{Target: target, ParentID: req.ParentID, UserID: actorID, Body: body})
	if err != nil {
		return uuid.Nil, err
	}

	ref := spec.CommentRef{Kind: target.Kind, CommentID: created.ID}
	if _, err := validated.Save(ctx, s.uploader, s.uploadSvc, s.mediaTarget(ctx, ref, mediaDir)); err != nil {
		rollback := spec.CommentDeletion{CommentRef: ref, UserID: actorID, AsAdmin: true}
		if deleteErr := s.repo.DeleteComment(context.WithoutCancel(ctx), rollback); deleteErr != nil {
			return uuid.Nil, errors.Join(err, fmt.Errorf("roll back comment %s: %w", created.ID, deleteErr))
		}

		return uuid.Nil, err
	}

	reference := toReference(target, created.ID)
	s.mentionSvc.NotifyAsync(ctx, reference, actorID, body)

	go s.notifyCreated(context.WithoutCancel(ctx), reference, actorID, ownerID, req.ParentID)

	return created.ID, nil
}

func (s *service) Update(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return ErrEmptyBody
	}

	if err := s.contentFilter.Check(ctx, body); err != nil {
		return err
	}

	authorID, err := s.authorID(ctx, ref)
	if err != nil {
		return err
	}

	update := spec.CommentUpdate{CommentRef: ref, UserID: actorID, Body: body}
	if authorID == actorID {
		return s.repo.UpdateComment(ctx, update)
	}

	if !s.authz.Can(ctx, actorID, authz.PermEditAnyComment) {
		return ErrForbidden
	}

	update.AsAdmin = true
	audited := spec.AuditedCommentUpdate{CommentUpdate: update, Audit: auditEntry(ref, "update_admin", actorID, authorID)}
	if err := s.repo.UpdateCommentWithAudit(ctx, audited); err != nil {
		return err
	}

	go s.notifyEdited(context.WithoutCancel(ctx), ref, authorID, actorID)

	return nil
}

func (s *service) Delete(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID) error {
	authorID, err := s.authorID(ctx, ref)
	if err != nil {
		return err
	}

	deletion := spec.CommentDeletion{CommentRef: ref, UserID: actorID}
	action := "delete"
	if authorID != actorID {
		if !s.authz.Can(ctx, actorID, authz.PermDeleteAnyComment) {
			return ErrForbidden
		}

		deletion.AsAdmin = true
		action = "delete_admin"
	}

	paths, err := s.repo.DeleteCommentWithAudit(ctx, spec.AuditedCommentDeletion{
		CommentDeletion: deletion,
		Audit:           auditEntry(ref, action, actorID, authorID),
	})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	return nil
}

func (s *service) Like(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID) error {
	authorID, err := s.authorID(ctx, ref)
	if err != nil {
		return err
	}

	blocked, err := s.blockSvc.IsBlockedEither(ctx, actorID, authorID)
	if err != nil {
		return fmt.Errorf("block check: %w", err)
	}
	if blocked {
		return block.ErrUserBlocked
	}

	if err := s.repo.LikeComment(ctx, spec.CommentLike{CommentRef: ref, UserID: actorID}); err != nil {
		return err
	}

	if authorID != actorID {
		go s.notifyLiked(context.WithoutCancel(ctx), ref, authorID, actorID)
	}

	return nil
}

func (s *service) Unlike(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID) error {
	return s.repo.UnlikeComment(ctx, spec.CommentLike{CommentRef: ref, UserID: actorID})
}

func (s *service) UploadMedia(ctx context.Context, ref spec.CommentRef, actorID uuid.UUID, contentType string, filename string, fileSize int64, reader io.Reader, isSpoiler bool) (*dto.PostMediaResponse, error) {
	authorID, err := s.authorID(ctx, ref)
	if err != nil {
		return nil, err
	}
	if authorID != actorID {
		return nil, authz.ErrNotCommentAuthor
	}

	mediaDir, err := s.repo.GetCommentMediaDir(ref.Kind)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetCommentMedia(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("existing comment media: %w", err)
	}

	sortOrder := len(existing)
	target := s.mediaTarget(ctx, ref, mediaDir)

	resp, err := s.uploader.SaveAndRecord(ctx, mediaDir, contentType, filename, fileSize, reader, isSpoiler,
		func(mediaURL, mediaType, thumbURL, savedName string, _ int) (int64, error) {
			return s.repo.AddCommentMedia(ctx, spec.NewCommentMedia{
				Kind: ref.Kind,
				NewMedia: spec.NewMedia{
					TargetID:     ref.CommentID,
					MediaURL:     mediaURL,
					MediaType:    mediaType,
					ThumbnailURL: thumbURL,
					Filename:     savedName,
					SortOrder:    sortOrder,
					IsSpoiler:    isSpoiler,
				},
			})
		},
		target.UpdateURL,
		target.UpdateThumb,
	)
	if err != nil {
		return nil, err
	}

	resp.SortOrder = sortOrder

	return resp, nil
}

func (s *service) Thread(ctx context.Context, target spec.CommentTarget, viewerID uuid.UUID) ([]dto.CommentResponse, error) {
	blockedIDs, err := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	if err != nil {
		return nil, fmt.Errorf("blocked users: %w", err)
	}

	ownerID, err := s.repo.GetTargetOwnerID(ctx, target)
	if err != nil {
		return nil, err
	}

	rows, _, err := s.repo.GetComments(ctx, spec.CommentQuery{
		Target:         target,
		ViewerID:       viewerID,
		Limit:          threadLimit,
		ExcludeUserIDs: blockedIDs,
	})
	if err != nil {
		return nil, err
	}

	commentIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		commentIDs = append(commentIDs, row.ID)
	}

	media, err := s.repo.GetCommentMediaBatch(ctx, spec.CommentMediaBatch{Kind: target.Kind, CommentIDs: commentIDs})
	if err != nil {
		return nil, err
	}

	flat := make([]dto.CommentResponse, len(rows))
	for i, row := range rows {
		flat[i] = toResponse(row, media[row.ID], ownerID)
	}

	return utils.BuildTree(flat,
		func(c dto.CommentResponse) uuid.UUID { return c.ID },
		func(c dto.CommentResponse) *uuid.UUID { return c.ParentID },
		func(c *dto.CommentResponse, replies []dto.CommentResponse) { c.Replies = replies },
	), nil
}

func (s *service) authorID(ctx context.Context, ref spec.CommentRef) (uuid.UUID, error) {
	authorID, err := s.repo.GetCommentAuthorID(ctx, ref)
	if errors.Is(err, dao.ErrNotFound) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	return authorID, nil
}

func (s *service) mediaTarget(ctx context.Context, ref spec.CommentRef, mediaDir string) upload.MediaTarget {
	return upload.MediaTarget{
		SubDir: mediaDir,
		Record: func(r upload.MediaRecord) (int64, error) {
			return s.repo.AddCommentMedia(ctx, spec.NewCommentMedia{Kind: ref.Kind, NewMedia: r.NewMedia(ref.CommentID)})
		},
		UpdateURL: func(ctx context.Context, u spec.MediaURLUpdate, tx ...*sql.Tx) error {
			return s.repo.UpdateCommentMediaURL(ctx, spec.CommentMediaURLUpdate{Kind: ref.Kind, MediaURLUpdate: u}, tx...)
		},
		UpdateThumb: func(ctx context.Context, u spec.MediaURLUpdate, tx ...*sql.Tx) error {
			return s.repo.UpdateCommentMediaThumbnail(ctx, spec.CommentMediaURLUpdate{Kind: ref.Kind, MediaURLUpdate: u}, tx...)
		},
	}
}

func (s *service) notifyCreated(ctx context.Context, reference mention.Reference, actorID uuid.UUID, ownerID uuid.UUID, parentID *uuid.UUID) {
	base, ok := s.notice(ctx, reference, actorID)
	if !ok {
		return
	}

	kind := spec.CommentKind(reference.Kind)
	family := familyOf(kind)
	notified := map[uuid.UUID]struct{}{actorID: {}}

	if parentID != nil {
		parentAuthorID, err := s.repo.GetCommentAuthorID(ctx, spec.CommentRef{Kind: kind, CommentID: *parentID})
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("comment_id", reference.ChildID.String()).Msg("load parent comment author failed")
		}

		if _, done := notified[parentAuthorID]; err == nil && !done {
			notified[parentAuthorID] = struct{}{}
			s.send(ctx, base, parentAuthorID, dto.NotificationType(family+"_comment_reply"), replyAction)
		}
	}

	if _, done := notified[ownerID]; ownerID != uuid.Nil && !done {
		s.send(ctx, base, ownerID, dto.NotificationType(family+"_commented"), commentedAction+nounOf(family))
	}
}

func (s *service) notifyLiked(ctx context.Context, ref spec.CommentRef, authorID uuid.UUID, actorID uuid.UUID) {
	reference, err := s.locate(ctx, ref)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("comment_id", ref.CommentID.String()).Msg("locate liked comment failed")
		return
	}

	base, ok := s.notice(ctx, reference, actorID)
	if !ok {
		return
	}

	s.send(ctx, base, authorID, dto.NotifCommentLiked, likeAction)
}

func (s *service) notifyEdited(ctx context.Context, ref spec.CommentRef, authorID uuid.UUID, editorID uuid.UUID) {
	reference, err := s.locate(ctx, ref)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("comment_id", ref.CommentID.String()).Msg("locate edited comment failed")
		return
	}

	target, err := reference.Resolve()
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("comment_id", ref.CommentID.String()).Msg("resolve edited comment failed")
		return
	}

	notification.SendEditNotification(ctx, s.userRepo, s.notifSvc, notification.EditNotifyParams{
		AuthorID:      authorID,
		EditorID:      editorID,
		ContentType:   editedContentType,
		ReferenceID:   target.ReferenceID(reference),
		ReferenceType: target.ReferenceType(reference),
		LinkPath:      target.LinkURL(reference),
	})
}

func (s *service) locate(ctx context.Context, ref spec.CommentRef) (mention.Reference, error) {
	target, err := s.repo.GetCommentTarget(ctx, ref)
	if err != nil {
		return mention.Reference{}, err
	}

	return toReference(target, ref.CommentID), nil
}

func (s *service) notice(ctx context.Context, reference mention.Reference, actorID uuid.UUID) (dto.NotifyParams, bool) {
	target, err := reference.Resolve()
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("kind", string(reference.Kind)).Msg("resolve comment reference failed")
		return dto.NotifyParams{}, false
	}

	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("actor_id", actorID.String()).Msg("load comment actor failed")
		return dto.NotifyParams{}, false
	}
	if actor == nil {
		return dto.NotifyParams{}, false
	}

	return dto.NotifyParams{
		ReferenceID:   target.ReferenceID(reference),
		ReferenceType: target.ReferenceType(reference),
		ActorID:       actorID,
		EmailActor:    actor.DisplayLabel(),
		EmailLink:     target.LinkURL(reference),
	}, true
}

func (s *service) send(ctx context.Context, base dto.NotifyParams, recipientID uuid.UUID, notifType dto.NotificationType, action string) {
	params := base
	params.RecipientID = recipientID
	params.Type = notifType
	params.EmailAction = action

	if err := s.notifSvc.Notify(ctx, params); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("type", string(notifType)).Str("recipient_id", recipientID.String()).Msg("comment notification failed")
	}
}

func toReference(target spec.CommentTarget, commentID uuid.UUID) mention.Reference {
	return mention.Reference{
		Kind:        mention.Kind(target.Kind),
		EntityID:    target.ID,
		EntityKey:   target.Key,
		EntryNumber: target.EntryNumber,
		ChildID:     commentID,
	}
}

func toResponse(row model.CommentRow, media []model.PostMediaRow, ownerID uuid.UUID) dto.CommentResponse {
	return dto.CommentResponse{
		ID:       row.ID,
		ParentID: row.ParentID,
		EntryID:  row.EntryID,
		Author: dto.UserResponse{
			ID:          row.UserID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarURL,
			Role:        role.Role(row.AuthorRole),
			Banned:      row.AuthorBanned,
		},
		Body:      row.Body,
		Media:     model.MediaRowsToResponse(media),
		LikeCount: row.LikeCount,
		UserLiked: row.UserLiked,
		IsAuthor:  ownerID != uuid.Nil && row.UserID == ownerID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func auditEntry(ref spec.CommentRef, action string, actorID uuid.UUID, authorID uuid.UUID) audit.NewEntry {
	family := familyOf(ref.Kind)

	return audit.NewEntry{
		ActorID:    actorID,
		Action:     audit.Action(family + "_comment_" + action),
		TargetType: audit.TargetType(family + "_comment"),
		TargetID:   ref.CommentID.String(),
		SubjectID:  authorID,
	}
}

func familyOf(kind spec.CommentKind) string {
	family, _, _ := strings.Cut(string(kind), "_")

	return family
}

func nounOf(family string) string {
	if len(family) <= 2 {
		return strings.ToUpper(family)
	}

	return family
}
