package mystery

import (
	"context"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

func (s *service) GetLeaderboard(ctx context.Context, page bounds.Page) (*dto.MysteryLeaderboardResponse, error) {
	rows, err := s.mysteryRepo.GetLeaderboard(ctx, page.Limit())
	if err != nil {
		return nil, err
	}

	entries := make([]dto.MysteryLeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = dto.MysteryLeaderboardEntry{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			Score:           r.Score,
			EasySolved:      r.EasySolved,
			MediumSolved:    r.MediumSolved,
			HardSolved:      r.HardSolved,
			NightmareSolved: r.NightmareSolved,
			ScoreAdjustment: r.ScoreAdjustment,
		}
	}
	return &dto.MysteryLeaderboardResponse{Entries: entries}, nil
}

func (s *service) GetTopDetectiveIDs(ctx context.Context) ([]string, error) {
	return s.mysteryRepo.GetTopDetectiveIDs(ctx)
}

func (s *service) GetGMLeaderboard(ctx context.Context, page bounds.Page) (*dto.GMLeaderboardResponse, error) {
	rows, err := s.mysteryRepo.GetGMLeaderboard(ctx, page.Limit())
	if err != nil {
		return nil, err
	}
	entries := make([]dto.GMLeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = dto.GMLeaderboardEntry{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			Score:        r.Score,
			MysteryCount: r.MysteryCount,
			PlayerCount:  r.PlayerCount,
		}
	}
	return &dto.GMLeaderboardResponse{Entries: entries}, nil
}

func (s *service) GetTopGMIDs(ctx context.Context) ([]string, error) {
	return s.mysteryRepo.GetTopGMIDs(ctx)
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID, page bounds.Page) (*dto.MysteryListResponse, error) {
	rows, total, err := s.mysteryRepo.ListByUser(ctx, spec.MysteryUserListFilter{
		UserID: userID,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	})
	if err != nil {
		return nil, err
	}

	return s.buildMysteryList(rows, total, page.Limit(), page.Offset()), nil
}
