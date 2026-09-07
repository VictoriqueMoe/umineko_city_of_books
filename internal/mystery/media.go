package mystery

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

func (s *service) UploadAttachment(ctx context.Context, mysteryID uuid.UUID, userID uuid.UUID, fileName string, fileSize int64, reader io.Reader) (*dto.MysteryAttachment, error) {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return nil, ErrNotAuthor
	}

	existing, _ := s.mysteryRepo.GetAttachments(ctx, mysteryID)
	for _, a := range existing {
		if a.FileName == fileName {
			return nil, fmt.Errorf("a file named %q is already attached", fileName)
		}
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxGeneralSize))
	subDir := "mystery-attachments/" + mysteryID.String()

	urlPath, err := s.uploadSvc.SaveAttachment(ctx, subDir, fileSize, maxSize, reader)
	if err != nil {
		return nil, err
	}

	dbID, err := s.mysteryRepo.AddAttachment(ctx, spec.NewMysteryAttachment{
		MysteryID: mysteryID,
		FileURL:   urlPath,
		FileName:  fileName,
		FileSize:  int(fileSize),
	})
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

func (s *service) UploadMedia(
	ctx context.Context,
	mysteryID uuid.UUID,
	userID uuid.UUID,
	contentType string,
	filename string,
	fileSize int64,
	reader io.Reader,
	isSpoiler bool,
) (*dto.PostMediaResponse, error) {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return nil, ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
		return nil, ErrNotAuthor
	}

	existing, _ := s.mysteryRepo.GetMedia(ctx, mysteryID)
	sortOrder := len(existing)

	resp, err := s.uploader.SaveAndRecord(ctx, "mysteries", contentType, filename, fileSize, reader, isSpoiler,
		func(mediaURL, mediaType, _, filename string, _ int) (int64, error) {
			return s.mysteryRepo.AddMedia(ctx, spec.NewMedia{
				TargetID:  mysteryID,
				MediaURL:  mediaURL,
				MediaType: mediaType,
				Filename:  filename,
				SortOrder: sortOrder,
				IsSpoiler: isSpoiler,
			})
		},
		s.mysteryRepo.UpdateMediaURL,
		s.mysteryRepo.UpdateMediaThumbnail,
	)
	if err != nil {
		return nil, err
	}
	resp.SortOrder = sortOrder
	return resp, nil
}

func (s *service) DeleteMedia(ctx context.Context, mediaID int64, mysteryID uuid.UUID, userID uuid.UUID) error {
	authorID, err := s.mysteryRepo.GetAuthorID(ctx, mysteryID)
	if err != nil {
		return ErrNotFound
	}
	if authorID != userID && !s.authz.Can(ctx, userID, authz.PermEditAnyTheory) {
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
		if int64(a.ID) == attachmentID {
			fileURL = a.FileURL
			break
		}
	}

	if err := s.mysteryRepo.DeleteAttachment(ctx, spec.MysteryAttachmentDeletion{ID: attachmentID, MysteryID: mysteryID}); err != nil {
		return err
	}

	if fileURL != "" {
		diskPath := s.uploadSvc.GetUploadDir() + strings.TrimPrefix(fileURL, "/uploads")
		os.Remove(diskPath)
	}

	return nil
}
