package mystery

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
)

func (s *service) AttachFile(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, ref dto.UploadRef) (*dto.MysteryAttachment, error) {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return nil, err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return nil, ErrNotAuthor
	}

	existing, err := s.mysteryRepo.GetAttachments(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("existing attachments: %w", err)
	}

	refs := []dto.UploadRef{ref}
	if err := s.checkAttachmentNames(ctx, userID, refs, existing); err != nil {
		return nil, err
	}

	return s.mysteryRepo.AttachFile(ctx, spec.MediaAttachment{TargetID: mysteryID, Uploads: upload.FileClaim(userID, refs)})
}

func (s *service) checkAttachmentNames(ctx context.Context, userID uuid.UUID, refs []dto.UploadRef, existing []dto.MysteryAttachment) error {
	if len(refs) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}

	pending, err := s.uploadSvc.UploadStatuses(ctx, userID, ids)
	if err != nil {
		return fmt.Errorf("pending attachments: %w", err)
	}

	seen := make(map[string]struct{}, len(existing)+len(pending))
	for _, a := range existing {
		seen[a.FileName] = struct{}{}
	}

	for _, p := range pending {
		if _, dup := seen[p.Filename]; dup {
			return fmt.Errorf("%w: %q", ErrDuplicateAttachment, p.Filename)
		}

		seen[p.Filename] = struct{}{}
	}

	return nil
}

func (s *service) AttachMedia(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, ref dto.UploadRef) (*dto.PostMediaResponse, error) {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return nil, err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return nil, ErrNotAuthor
	}

	existing, err := s.mysteryRepo.GetMedia(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("existing media: %w", err)
	}

	attached, err := s.mysteryRepo.AttachMedia(ctx, spec.MediaAttachment{TargetID: mysteryID, SortOrder: len(existing), Uploads: upload.MediaClaim(userID, []dto.UploadRef{ref})})
	if err != nil {
		return nil, err
	}

	return new(attached.ToResponse()), nil
}

func (s *service) DeleteMedia(ctx context.Context, mediaID int64, mysteryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return ErrNotAuthor
	}

	mediaURL, err := s.mysteryRepo.DeleteMedia(ctx, spec.MediaDeletion{ID: mediaID, TargetID: mysteryID})
	if err != nil {
		return err
	}

	s.uploadSvc.Delete(mediaURL)
	return nil
}

func (s *service) DeleteAttachment(ctx context.Context, attachmentID int64, mysteryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.mysteryAuthor(ctx, mysteryID)
	if err != nil {
		return err
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyWork) {
		return ErrNotAuthor
	}

	attachments, err := s.mysteryRepo.GetAttachments(ctx, mysteryID)
	if err != nil {
		return fmt.Errorf("existing attachments: %w", err)
	}
	var fileURL string
	for _, a := range attachments {
		if int64(a.ID) == attachmentID {
			fileURL = a.FileURL
			break
		}
	}

	if err := s.mysteryRepo.DeleteAttachment(ctx, spec.MysteryAttachmentDeletion{ID: attachmentID, MysteryID: mysteryID}); err != nil {
		return err
	}

	if fileURL != "" {
		s.uploadSvc.Delete(fileURL)
	}

	return nil
}
