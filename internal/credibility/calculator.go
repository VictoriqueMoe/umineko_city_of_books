package credibility

import (
	"context"
	"math"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type Service struct {
	theoryRepo repository.TheoryRepository
}

const k = 5.0

func NewService(theoryRepo repository.TheoryRepository) *Service {
	return &Service{theoryRepo: theoryRepo}
}

func (*Service) calculate(withLoveSum, withoutLoveSum float64) float64 {
	raw := withLoveSum - withoutLoveSum
	return 50.0 + 50.0*math.Tanh(raw/k)
}

func (s *Service) Recalculate(ctx context.Context, theoryID uuid.UUID) {
	withLove, withoutLove, err := s.theoryRepo.GetResponseEvidenceWeights(ctx, theoryID)
	if err != nil {
		logger.Log.Error().Err(err).Str("theory_id", theoryID.String()).Msg("failed to get evidence weights for credibility")
		return
	}

	score := s.calculate(withLove, withoutLove)

	if err := s.theoryRepo.UpdateCredibilityScore(ctx, theoryID, score); err != nil {
		logger.Log.Error().Err(err).Str("theory_id", theoryID.String()).Msg("failed to update credibility score")
	}
}
