package controllers

import (
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/health"

	healthgo "github.com/hellofresh/health-go/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newHealthHarness(t *testing.T) (*testutil.Harness, *health.MockService) {
	h := testutil.NewHarness(t)
	svc := health.NewMockService(t)
	s := &Service{
		HealthService: svc,
	}
	for _, setup := range s.getAllHealthRoutes() {
		setup(h.App)
	}
	return h, svc
}

func TestHealthController_AllChecksPass_Returns200(t *testing.T) {
	// given
	h, svc := newHealthHarness(t)
	svc.EXPECT().Measure(mock.Anything).Return(healthgo.Check{Status: healthgo.StatusOK})

	// when
	status, body := h.NewRequest(http.MethodGet, "/health").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), string(healthgo.StatusOK))
}

func TestHealthController_CriticalCheckDown_Returns503(t *testing.T) {
	// given
	h, svc := newHealthHarness(t)
	svc.EXPECT().Measure(mock.Anything).Return(healthgo.Check{
		Status:   healthgo.StatusUnavailable,
		Failures: map[string]string{"postgres": "connection refused"},
	})

	// when
	status, body := h.NewRequest(http.MethodGet, "/health").Do()

	// then
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Contains(t, string(body), "postgres")
}

func TestHealthController_NonCriticalDegraded_Returns200(t *testing.T) {
	// given
	h, svc := newHealthHarness(t)
	svc.EXPECT().Measure(mock.Anything).Return(healthgo.Check{
		Status:   healthgo.StatusPartiallyAvailable,
		Failures: map[string]string{"livekit": "timeout"},
	})

	// when
	status, _ := h.NewRequest(http.MethodGet, "/health").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
}

func TestHealthController_Livez_IsShallowAndAlways200(t *testing.T) {
	// given
	h, _ := newHealthHarness(t)

	// when
	status, body := h.NewRequest(http.MethodGet, "/livez").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "ok")
}
