package credibility

import (
	"context"
	"errors"
	"math"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(t *testing.T) (*Service, *repository.MockTheoryRepository) {
	repo := repository.NewMockTheoryRepository(t)
	return NewService(repo), repo
}

func TestCalculate_BalancedWeightsReturns50(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.calculate(3.0, 3.0)

	// then
	assert.InDelta(t, 50.0, got, 1e-9)
}

func TestCalculate_MoreSupportingEvidenceAboveMidpoint(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.calculate(10.0, 0.0)

	// then
	assert.Greater(t, got, 50.0)
	assert.LessOrEqual(t, got, 100.0)
}

func TestCalculate_MoreContraryEvidenceBelowMidpoint(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.calculate(0.0, 10.0)

	// then
	assert.Less(t, got, 50.0)
	assert.GreaterOrEqual(t, got, 0.0)
}

func TestCalculate_BoundedBetween0And100(t *testing.T) {
	cases := []struct {
		name        string
		withLove    float64
		withoutLove float64
	}{
		{"extreme positive", 1000.0, 0.0},
		{"extreme negative", 0.0, 1000.0},
		{"both large equal", 1000.0, 1000.0},
		{"both zero", 0.0, 0.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			got := svc.calculate(tc.withLove, tc.withoutLove)

			// then
			assert.GreaterOrEqual(t, got, 0.0)
			assert.LessOrEqual(t, got, 100.0)
			assert.False(t, math.IsNaN(got))
		})
	}
}

func TestRecalculate_HappyPath(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	theoryID := uuid.New()
	repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(5.0, 5.0, nil)
	repo.EXPECT().UpdateCredibilityScore(mock.Anything, theoryID, mock.MatchedBy(func(score float64) bool {
		return math.Abs(score-50.0) < 1e-9
	})).Return(nil)

	// when
	svc.Recalculate(context.Background(), theoryID)

	// then — mock expectations asserted by cleanup
}

func TestRecalculate_WeightsLookupErrorAborts(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	theoryID := uuid.New()
	repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(0.0, 0.0, errors.New("db down"))

	// when
	svc.Recalculate(context.Background(), theoryID)

	// then — no UpdateCredibilityScore call expected
}

func TestRecalculate_UpdateErrorSwallowed(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	theoryID := uuid.New()
	repo.EXPECT().GetResponseEvidenceWeights(mock.Anything, theoryID).Return(1.0, 0.0, nil)
	repo.EXPECT().UpdateCredibilityScore(mock.Anything, theoryID, mock.Anything).Return(errors.New("write failed"))

	// when
	svc.Recalculate(context.Background(), theoryID)

	// then — no panic, error swallowed
}
