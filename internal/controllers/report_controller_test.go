package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/report"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type reportDeps struct {
	ReportService *report.MockService
}

func newReportHarness(t *testing.T) (*testutil.Harness, reportDeps) {
	h := testutil.NewHarness(t)
	rs := report.NewMockService(t)

	s := &Service{
		ReportService: rs,
		AuthSession:   h.SessionManager,
		AuthzService:  h.AuthzService,
	}
	for _, setup := range s.getAllReportRoutes() {
		setup(h.App)
	}
	return h, reportDeps{ReportService: rs}
}

func reportFactory(t *testing.T) (*testutil.Harness, reportDeps) {
	return newReportHarness(t)
}

func TestCreateReport_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, reportFactory, "POST", "/report", report.CreateReportRequest{
		TargetType: "mystery",
		TargetID:   uuid.NewString(),
		Reason:     "spam",
	})
}

func TestCreateReport_OK(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := report.CreateReportRequest{
		TargetType: "mystery",
		TargetID:   uuid.NewString(),
		Reason:     "spam",
	}
	deps.ReportService.EXPECT().Create(mock.Anything, userID, req).Return(nil)

	// when
	status, body := h.NewRequest("POST", "/report").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, "ok", resp["status"])
}

func TestCreateReport_BadJSON(t *testing.T) {
	// given
	h, _ := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/report").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestCreateReport_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{
			name:     "missing fields",
			err:      report.ErrMissingFields,
			wantCode: http.StatusBadRequest,
			wantBody: report.ErrMissingFields.Error(),
		},
		{
			name:     "internal error",
			err:      errors.New("boom"),
			wantCode: http.StatusInternalServerError,
			wantBody: "failed to create report",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newReportHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := report.CreateReportRequest{
				TargetType: "mystery",
				TargetID:   uuid.NewString(),
				Reason:     "spam",
			}
			deps.ReportService.EXPECT().Create(mock.Anything, userID, req).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/report").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListReports_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, reportFactory, "GET", "/admin/reports", nil, authz.PermViewUsers)
}

func TestListReports_OK(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	expected := &report.ReportListResponse{
		Reports: []report.ReportResponse{{ID: 1, Reason: "spam"}},
		Total:   1,
		Limit:   50,
		Offset:  0,
	}
	deps.ReportService.EXPECT().List(mock.Anything, "open", 50, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/reports").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[report.ReportListResponse](t, body)
	assert.Equal(t, 1, got.Total)
	require.Len(t, got.Reports, 1)
	assert.Equal(t, 1, got.Reports[0].ID)
}

func TestListReports_CustomQuery(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	deps.ReportService.EXPECT().List(mock.Anything, "resolved", 10, 20).Return(&report.ReportListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/admin/reports?status=resolved&limit=10&offset=20").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListReports_InternalError(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	deps.ReportService.EXPECT().List(mock.Anything, "open", 50, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/admin/reports").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list reports")
}

func TestResolveReport_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, reportFactory, "POST", "/admin/reports/42/resolve", nil, authz.PermViewUsers)
}

func TestResolveReport_OK(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	reportID := 42
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	deps.ReportService.EXPECT().Resolve(mock.Anything, reportID, userID, "looks fine").Return(nil)

	// when
	status, body := h.NewRequest("POST", "/admin/reports/"+strconv.Itoa(reportID)+"/resolve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"comment": "looks fine"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "ok")
}

func TestResolveReport_EmptyCommentOnMissingBody(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	reportID := 7
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	deps.ReportService.EXPECT().Resolve(mock.Anything, reportID, userID, "").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/reports/"+strconv.Itoa(reportID)+"/resolve").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestResolveReport_BadJSONIgnored(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	reportID := 9
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	deps.ReportService.EXPECT().Resolve(mock.Anything, reportID, userID, "").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/reports/"+strconv.Itoa(reportID)+"/resolve").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestResolveReport_InvalidID(t *testing.T) {
	// given
	h, _ := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)

	// when
	status, body := h.NewRequest("POST", "/admin/reports/not-an-int/resolve").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid report ID")
}

func TestResolveReport_ZeroID(t *testing.T) {
	// given
	h, _ := newReportHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)

	// when
	status, body := h.NewRequest("POST", "/admin/reports/0/resolve").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid report ID")
}

func TestResolveReport_InternalError(t *testing.T) {
	// given
	h, deps := newReportHarness(t)
	userID := uuid.New()
	reportID := 42
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	deps.ReportService.EXPECT().Resolve(mock.Anything, reportID, userID, "").Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/admin/reports/"+strconv.Itoa(reportID)+"/resolve").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to resolve report")
}
