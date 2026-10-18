package announcement

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const deletedAuthorName = "Deleted user"

type (
	Service interface {
		List(ctx context.Context, page bounds.Page) (*dto.AnnouncementListResponse, error)
		GetDetail(ctx context.Context, id, viewerID uuid.UUID) (*dto.AnnouncementDetailResponse, error)
		GetLatest(ctx context.Context) (*dto.AnnouncementResponse, error)
		Create(ctx context.Context, userID uuid.UUID, title, body string) (uuid.UUID, error)
		Update(ctx context.Context, actorID uuid.UUID, id uuid.UUID, title, body string) error
		Delete(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error
		SetPinned(ctx context.Context, actorID uuid.UUID, id uuid.UUID, pinned bool) error
	}

	service struct {
		repo       repository.AnnouncementRepository
		auditRepo  repository.AuditLogRepository
		mentionSvc mention.Service
		commentSvc comment.Service
		hub        *ws.Hub
		uploadSvc  upload.Service
		ogCache    *og.Resolver
	}
)

func NewService(
	repo repository.AnnouncementRepository,
	auditRepo repository.AuditLogRepository,
	mentionSvc mention.Service,
	commentSvc comment.Service,
	hub *ws.Hub,
	uploadSvc upload.Service,
	ogCache *og.Resolver,
) Service {
	return &service{
		repo:       repo,
		auditRepo:  auditRepo,
		mentionSvc: mentionSvc,
		commentSvc: commentSvc,
		hub:        hub,
		uploadSvc:  uploadSvc,
		ogCache:    ogCache,
	}
}

func rowToResponse(r model.AnnouncementRow) dto.AnnouncementResponse {
	if r.AuthorID == uuid.Nil {
		r.AuthorDisplayName = deletedAuthorName
	}

	return dto.AnnouncementResponse{
		ID:        r.ID,
		Title:     r.Title,
		Body:      r.Body,
		Pinned:    r.Pinned,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Author: dto.UserResponse{
			ID:          r.AuthorID,
			Username:    r.AuthorUsername,
			DisplayName: r.AuthorDisplayName,
			AvatarURL:   r.AuthorAvatarURL,
			Role:        role.Role(r.AuthorRole),
		},
	}
}

func (s *service) List(ctx context.Context, page bounds.Page) (*dto.AnnouncementListResponse, error) {
	rows, total, err := s.repo.List(ctx, spec.AnnouncementListQuery{Limit: page.Limit(), Offset: page.Offset()})
	if err != nil {
		return nil, err
	}
	items := make([]dto.AnnouncementResponse, len(rows))
	for i, r := range rows {
		items[i] = rowToResponse(r)
	}
	return &dto.AnnouncementListResponse{
		Announcements: items,
		Total:         total,
		Limit:         page.Limit(),
		Offset:        page.Offset(),
	}, nil
}

func (s *service) GetDetail(ctx context.Context, id, viewerID uuid.UUID) (*dto.AnnouncementDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}

	comments, err := s.commentSvc.Thread(ctx, spec.CommentTarget{Kind: spec.CommentKindAnnouncement, ID: id}, viewerID)
	if err != nil {
		return nil, fmt.Errorf("announcement comments: %w", err)
	}

	return &dto.AnnouncementDetailResponse{
		AnnouncementResponse: rowToResponse(*row),
		Comments:             comments,
	}, nil
}

func (s *service) GetLatest(ctx context.Context) (*dto.AnnouncementResponse, error) {
	row, err := s.repo.GetLatest(ctx)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return new(rowToResponse(*row)), nil
}

func (s *service) audit(ctx context.Context, actorID uuid.UUID, action audit.Action, id, subjectID uuid.UUID, details string) {
	if err := s.auditRepo.Create(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: audit.TargetAnnouncement,
		TargetID:   id.String(),
		Details:    details,
		SubjectID:  subjectID,
	}); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(action)).Msg("failed to write audit log")
	}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, title, body string) (uuid.UUID, error) {
	if title == "" || body == "" {
		return uuid.Nil, ErrEmptyTitleOrBody
	}
	created, err := s.repo.Create(ctx, spec.NewAnnouncement{AuthorID: userID, Title: title, Body: body})
	if err != nil {
		return uuid.Nil, err
	}

	s.hub.Broadcast(ws.Message{
		Type: "new_announcement",
		Data: map[string]interface{}{
			"id":        created.ID,
			"title":     title,
			"author_id": userID,
		},
	})

	s.audit(ctx, userID, audit.ActionAnnouncementCreate, created.ID, userID, fmt.Sprintf("title=%s", title))

	s.mentionSvc.NotifyAsync(ctx, mention.Reference{Kind: mention.KindAnnouncement, EntityID: created.ID}, userID, body)

	return created.ID, nil
}

func (s *service) Update(ctx context.Context, actorID uuid.UUID, id uuid.UUID, title, body string) error {
	if title == "" || body == "" {
		return ErrEmptyTitleOrBody
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrNotFound
	}

	if err := s.repo.Update(ctx, spec.AnnouncementUpdate{ID: id, Title: title, Body: body}); err != nil {
		return err
	}

	details := fmt.Sprintf("title=%s", title)
	if current.Title != title {
		details = fmt.Sprintf("title=%s -> %s", current.Title, title)
	}

	s.audit(ctx, actorID, audit.ActionAnnouncementUpdate, id, current.AuthorID, details)

	return nil
}

func (s *service) Delete(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	doomed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if doomed == nil {
		return ErrNotFound
	}

	paths, err := s.repo.DeleteWithMedia(ctx, id)
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(paths...)

	s.audit(ctx, actorID, audit.ActionAnnouncementDelete, id, doomed.AuthorID, fmt.Sprintf("title=%s", doomed.Title))

	if err := s.ogCache.ClearMetaCache(ctx, og.KindAnnouncement, id.String()); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("announcement_id", id.String()).Msg("clear og meta cache failed")
	}

	return nil
}

func (s *service) SetPinned(ctx context.Context, actorID uuid.UUID, id uuid.UUID, pinned bool) error {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrNotFound
	}

	if err := s.repo.SetPinned(ctx, spec.AnnouncementPinUpdate{ID: id, Pinned: pinned}); err != nil {
		return err
	}

	s.audit(ctx, actorID, audit.ActionAnnouncementPin, id, current.AuthorID, fmt.Sprintf("title=%s pinned=%t", current.Title, pinned))

	return nil
}
