package admin

import (
	"context"
	"fmt"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/giphy/banlist"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

func (s *service) ListBannedGifs(ctx context.Context) (*dto.BannedGiphyListResponse, error) {
	entries, err := s.giphyBanlist.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BannedGiphyEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, dto.BannedGiphyEntry{
			Kind:      string(e.Kind),
			Value:     e.Value,
			Reason:    e.Reason,
			CreatedAt: e.CreatedAt,
			CreatedBy: e.CreatedBy,
		})
	}
	return &dto.BannedGiphyListResponse{Entries: out}, nil
}

func (s *service) AddBannedGif(ctx context.Context, actorID uuid.UUID, req dto.AddBannedGiphyRequest) (*dto.AddBannedGiphyResponse, error) {
	kind, value, ok := banlist.ParseInput(req.Input)
	if !ok {
		return nil, ErrBannedGiphyInvalidInput
	}
	if req.Kind != "" && req.Kind != string(kind) {
		return nil, ErrBannedGiphyKindMismatch
	}
	actor := actorID.String()
	if err := s.giphyBanlist.Add(ctx, kind, value, req.Reason, &actor); err != nil {
		return nil, err
	}

	s.auditDetails(ctx, actorID, repository.AuditActionBannedGifCreate, repository.AuditTargetBannedGif, value, fmt.Sprintf("kind=%s id=%s", kind, value))

	return &dto.AddBannedGiphyResponse{
		Entry: dto.BannedGiphyEntry{
			Kind:      string(kind),
			Value:     value,
			Reason:    req.Reason,
			CreatedBy: &actor,
		},
	}, nil
}

func (s *service) RemoveBannedGif(ctx context.Context, actorID uuid.UUID, kind, value string) error {
	k := banlist.Kind(kind)
	if k != banlist.KindGif && k != banlist.KindUser {
		return banlist.ErrInvalidKind
	}

	if err := s.giphyBanlist.Remove(ctx, k, value); err != nil {
		return err
	}

	s.auditDetails(ctx, actorID, repository.AuditActionBannedGifDelete, repository.AuditTargetBannedGif, value, fmt.Sprintf("kind=%s id=%s", k, value))

	return nil
}
