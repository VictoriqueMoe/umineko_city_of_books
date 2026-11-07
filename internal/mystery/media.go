package mystery

import (
	"context"
	"fmt"
	"io"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
)

func (s *service) UploadAttachment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, fileName string, fileSize int64, reader io.Reader) (*dto.MysteryAttachment, error) {
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
	for _, a := range existing {
		if a.FileName == fileName {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateAttachment, fileName)
		}
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxGeneralSize))

	attachment, err := s.storeAttachment(ctx, maxSize, fileName, fileSize, reader)
	if err != nil {
		return nil, err
	}

	attachment.MysteryID = mysteryID

	dbID, err := s.mysteryRepo.AddAttachment(ctx, attachment)
	if err != nil {
		s.uploadSvc.Delete(attachment.FileURL)

		return nil, err
	}

	return &dto.MysteryAttachment{
		ID:       int(dbID),
		FileURL:  attachment.FileURL,
		FileName: attachment.FileName,
		FileSize: attachment.FileSize,
	}, nil
}

func (s *service) storeAttachment(ctx context.Context, maxSize int64, fileName string, fileSize int64, reader io.Reader) (spec.NewMysteryAttachment, error) {
	urlPath, err := s.uploadSvc.SaveAttachment(ctx, "mystery-attachments", fileSize, maxSize, reader)
	if err != nil {
		return spec.NewMysteryAttachment{}, err
	}

	return spec.NewMysteryAttachment{FileURL: urlPath, FileName: fileName, FileSize: int(fileSize)}, nil
}

func (s *service) validateAttachments(ctx context.Context, attachments []upload.FileUpload) error {
	if len(attachments) == 0 {
		return nil
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxGeneralSize))

	seen := make(map[string]struct{}, len(attachments))
	for _, f := range attachments {
		if _, dup := seen[f.Filename]; dup {
			return fmt.Errorf("%w: %q", ErrDuplicateAttachment, f.Filename)
		}
		seen[f.Filename] = struct{}{}

		if err := upload.ValidateAttachmentFile(f, maxSize); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) storeAttachments(ctx context.Context, attachments []upload.FileUpload) ([]spec.NewMysteryAttachment, error) {
	if len(attachments) == 0 {
		return nil, nil
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxGeneralSize))

	var stored []spec.NewMysteryAttachment
	for _, f := range attachments {
		attachment, err := s.storeAttachmentFile(ctx, maxSize, f)
		if err != nil {
			s.discardAttachments(stored)

			return nil, err
		}

		stored = append(stored, attachment)
	}

	return stored, nil
}

func (s *service) storeAttachmentFile(ctx context.Context, maxSize int64, f upload.FileUpload) (spec.NewMysteryAttachment, error) {
	r, err := f.Open()
	if err != nil {
		return spec.NewMysteryAttachment{}, fmt.Errorf("open attachment: %w", err)
	}
	defer r.Close()

	return s.storeAttachment(ctx, maxSize, f.Filename, f.Size, r)
}

func (s *service) discardAttachments(stored []spec.NewMysteryAttachment) {
	for _, a := range stored {
		s.uploadSvc.Delete(a.FileURL)
	}
}

func (s *service) UploadMedia(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, file upload.FileUpload) (*dto.PostMediaResponse, error) {
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
	stored, err := s.uploadSvc.StoreMedia(ctx, "mysteries", []upload.FileUpload{file})
	if err != nil {
		return nil, err
	}

	stored[0].SortOrder = len(existing)

	mediaID, err := s.mysteryRepo.AddMedia(ctx, stored[0].NewMedia(mysteryID))
	if err != nil {
		s.uploadSvc.DiscardMedia(stored...)

		return nil, err
	}

	s.uploadSvc.TranscodeMedia(ctx, "mysteries", stored, []int64{mediaID}, s.mysteryRepo.UpdateMediaURL, s.mysteryRepo.UpdateMediaThumbnail)

	resp := stored[0].Response(mediaID)

	return &resp, nil
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
