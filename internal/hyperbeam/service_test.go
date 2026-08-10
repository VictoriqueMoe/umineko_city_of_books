package hyperbeam

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T, apiKey, baseURL string) *service {
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingHyperbeamAPIKey).Return(apiKey).Maybe()

	return &service{
		settingsSvc: settingsSvc,
		baseURL:     baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func TestService_Enabled(t *testing.T) {
	// given
	cases := []struct {
		name string
		key  string
		want bool
	}{
		{"empty key disabled", "", false},
		{"populated key enabled", "sk_test_abc", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			svc := newTestService(t, tc.key, "http://example")

			// then
			require.Equal(t, tc.want, svc.Enabled())
		})
	}
}

func TestService_CreateVM(t *testing.T) {
	// given
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotBody   map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session_id":"sess_123","embed_url":"https://hb.example/embed","admin_token":"admin_abc"}`))
	}))
	defer server.Close()

	svc := newTestService(t, "sk_test_abc", server.URL)

	// when
	vm, err := svc.CreateVM(context.Background(), CreateVMOptions{
		StartURL: "https://youtube.com",
		Region:   "NA",
		Timeout:  &VMTimeoutOpts{Offline: 1800, Absolute: 14400},
	})

	// then
	require.NoError(t, err)
	require.Equal(t, "sess_123", vm.SessionID)
	require.Equal(t, "https://hb.example/embed", vm.EmbedURL)
	require.Equal(t, "admin_abc", vm.AdminToken)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/vm", gotPath)
	require.Equal(t, "Bearer sk_test_abc", gotAuth)
	require.Equal(t, "https://youtube.com", gotBody["start_url"])
	require.Equal(t, "NA", gotBody["region"])
	timeout, ok := gotBody["timeout"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1800), timeout["offline"])
	require.Equal(t, float64(14400), timeout["absolute"])
}

func TestService_SetControlRole_RequiresAnAdminToken(t *testing.T) {
	// given
	svc := newTestService(t, "sk_test_abc", "http://engine")

	// when
	err := svc.SetControlRole(context.Background(), "https://vm.hyperbeam.com/abc", "", "user_xyz", true)

	// then
	require.ErrorIs(t, err, ErrMissingAdminToken)
}

func TestService_SetControlRole_AddsControlRole(t *testing.T) {
	// given
	var gotPath string
	var gotAuth string
	var gotBody []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := newTestService(t, "sk_test_abc", "http://engine")

	// when
	err := svc.SetControlRole(context.Background(), server.URL+"/session_path", "vm_admin_token", "user_xyz", true)

	// then
	require.NoError(t, err)
	require.Equal(t, "Bearer vm_admin_token", gotAuth, "vm scoped endpoints authenticate with the session admin token, not the account api key")
	require.Equal(t, "/session_path/addRoles", gotPath)
	require.Len(t, gotBody, 2)
	userIDs, ok := gotBody[0].([]any)
	require.True(t, ok)
	require.Equal(t, []any{"user_xyz"}, userIDs)
	roles, ok := gotBody[1].([]any)
	require.True(t, ok)
	require.Equal(t, []any{"control"}, roles)
}

func TestService_SetControlRole_RemovesControlRole(t *testing.T) {
	// given
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := newTestService(t, "sk_test_abc", "http://engine")

	// when
	err := svc.SetControlRole(context.Background(), server.URL+"/session_path", "vm_admin_token", "user_xyz", false)

	// then
	require.NoError(t, err)
	require.Equal(t, "/session_path/removeRoles", gotPath)
}

func TestService_TerminateVM_Success(t *testing.T) {
	// given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/vm/sess_123", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := newTestService(t, "sk_test_abc", server.URL)

	// when
	err := svc.TerminateVM(context.Background(), "sess_123")

	// then
	require.NoError(t, err)
}

func TestService_APIError(t *testing.T) {
	// given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	svc := newTestService(t, "sk_test_abc", server.URL)

	// when
	_, err := svc.CreateVM(context.Background(), CreateVMOptions{})

	// then
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.True(t, strings.Contains(apiErr.Body, "bad request"))
}

func TestService_NoCapacityIsDistinguishable(t *testing.T) {
	cases := []struct {
		name           string
		status         int
		body           string
		wantNoCapacity bool
	}{
		{"a full region is recognised by its error code", http.StatusServiceUnavailable, `{"code":"err_no_available_vm","message":"No VMs are currently available, please try again"}`, true},
		{"another 503 is not mistaken for a full region", http.StatusServiceUnavailable, `{"code":"err_something_else","message":"nope"}`, false},
		{"a body without a code is not a capacity problem", http.StatusBadRequest, `{"error":"bad request"}`, false},
		{"a non-json body does not panic", http.StatusBadGateway, `<html>gateway</html>`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			svc := newTestService(t, "sk_test_abc", server.URL)

			// when
			_, err := svc.CreateVM(context.Background(), CreateVMOptions{})

			// then
			require.Error(t, err)
			require.Equal(t, tc.wantNoCapacity, IsNoCapacity(err))
		})
	}
}

func TestExtractVMBaseURL(t *testing.T) {
	// given
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"with query token", "https://abc.hyperbeam.com/SESSPATH/?token=xyz", "https://abc.hyperbeam.com/SESSPATH", false},
		{"no trailing slash", "https://abc.hyperbeam.com/SESSPATH?token=xyz", "https://abc.hyperbeam.com/SESSPATH", false},
		{"empty", "", "", true},
		{"missing host", "/just/path", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := ExtractVMBaseURL(tc.input)

			// then
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
