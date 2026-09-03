package admin

import (
	"context"
	"fmt"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

func (s *service) CreateInvite(ctx context.Context, actorID uuid.UUID) (*dto.InviteResponse, error) {
	code := uuid.New().String()[:8]
	if err := s.inviteRepo.Create(ctx, code, actorID); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	s.audit(ctx, actorID, repository.AuditActionCreateInvite, repository.AuditTargetInvite, code)

	return &dto.InviteResponse{
		Code:      code,
		CreatedBy: actorID,
		CreatedAt: "just now",
	}, nil
}

func (s *service) ListInvites(ctx context.Context, page bounds.Page) (*dto.InviteListResponse, error) {
	invites, total, err := s.inviteRepo.List(ctx, page.Limit(), page.Offset())
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}

	items := make([]dto.InviteResponse, len(invites))
	for i, inv := range invites {
		items[i] = dto.InviteResponse{
			Code:      inv.Code,
			CreatedBy: inv.CreatedBy,
			UsedBy:    inv.UsedBy,
			UsedAt:    inv.UsedAt,
			CreatedAt: inv.CreatedAt,
		}
	}

	return &dto.InviteListResponse{
		Invites: items,
		Total:   total,
		Limit:   page.Limit(),
		Offset:  page.Offset(),
	}, nil
}

func (s *service) DeleteInvite(ctx context.Context, actorID uuid.UUID, code string) error {
	if err := s.inviteRepo.Delete(ctx, code); err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	s.audit(ctx, actorID, repository.AuditActionDeleteInvite, repository.AuditTargetInvite, code)
	return nil
}
