package email

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSettings struct {
	values map[*config.SiteSettingDef]string
}

func (s *stubSettings) Get(_ context.Context, def *config.SiteSettingDef) string {
	if v, ok := s.values[def]; ok {
		return v
	}
	return def.Default
}

func (s *stubSettings) GetInt(ctx context.Context, def *config.SiteSettingDef) int {
	v, err := strconv.Atoi(s.Get(ctx, def))
	if err != nil {
		return 0
	}
	return v
}

func (s *stubSettings) GetBool(ctx context.Context, def *config.SiteSettingDef) bool {
	return s.Get(ctx, def) == "true"
}

func (s *stubSettings) GetAll(_ context.Context) map[config.SiteSettingKey]string {
	return nil
}

func (s *stubSettings) Set(_ context.Context, _ *config.SiteSettingDef, _ string, _ uuid.UUID) error {
	return nil
}

func (s *stubSettings) SetMultiple(_ context.Context, _ map[config.SiteSettingKey]string, _ uuid.UUID) error {
	return nil
}

func (s *stubSettings) Subscribe(_ settings.Listener) {}

func (s *stubSettings) Refresh(_ context.Context) error {
	return nil
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newServiceWithSettings(values map[*config.SiteSettingDef]string, rt http.RoundTripper) *service {
	svc := NewService(&stubSettings{values: values}).(*service)
	if rt != nil {
		svc.httpClient = &http.Client{Transport: rt}
	}
	return svc
}

func cloudflareConfigured(rt http.RoundTripper) *service {
	return newServiceWithSettings(map[*config.SiteSettingDef]string{
		config.SettingEmailProvider:       string(config.EmailProviderCloudflare),
		config.SettingCloudflareAccountID: "acct-123",
		config.SettingCloudflareAPIToken:  "secret-token",
		config.SettingCloudflareEmailFrom: "noreply@books.test",
		config.SettingSiteName:            "City of Books",
	}, rt)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestSend_SMTPNotConfigured_IsSwallowed(t *testing.T) {
	// given
	svc := newServiceWithSettings(nil, nil)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Subject", "<p>Body</p>")

	// then
	require.NoError(t, err)
}

func TestSendTest_SMTPNotConfigured_ReturnsNotConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(nil, nil)

	// when
	err := svc.SendTest(context.Background(), "user@example.com", "Subject", "<p>Body</p>")

	// then
	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestSendTest_CloudflareNotConfigured_ReturnsNotConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(map[*config.SiteSettingDef]string{
		config.SettingEmailProvider: string(config.EmailProviderCloudflare),
	}, nil)

	// when
	err := svc.SendTest(context.Background(), "user@example.com", "Subject", "<p>Body</p>")

	// then
	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestSend_CloudflareSuccess_BuildsRequest(t *testing.T) {
	// given
	var captured *http.Request
	var capturedBody []byte
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		captured = req
		capturedBody, _ = io.ReadAll(req.Body)
		return jsonResponse(http.StatusOK, `{"success":true,"result":{"delivered":["user@example.com"]}}`), nil
	})
	svc := cloudflareConfigured(rt)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Welcome", "<p>Hi &amp; bye</p>")

	// then
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, http.MethodPost, captured.Method)
	assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acct-123/email/sending/send", captured.URL.String())
	assert.Equal(t, "Bearer secret-token", captured.Header.Get("Authorization"))
	assert.Equal(t, "application/json", captured.Header.Get("Content-Type"))

	var payload cloudflareSendRequest
	require.NoError(t, json.Unmarshal(capturedBody, &payload))
	assert.Equal(t, "user@example.com", payload.To)
	assert.Equal(t, "noreply@books.test", payload.From.Address)
	assert.Equal(t, "City of Books", payload.From.Name)
	assert.Equal(t, "Welcome", payload.Subject)
	assert.Equal(t, "<p>Hi &amp; bye</p>", payload.HTML)
	assert.Equal(t, "Hi & bye", payload.Text)
}

func TestSend_CloudflareHTTPStatusError_Returns(t *testing.T) {
	// given
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"success":false,"errors":[{"code":1000,"message":"bad token"}]}`), nil
	})
	svc := cloudflareConfigured(rt)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Welcome", "<p>Body</p>")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestSend_CloudflareSuccessFalse_Returns(t *testing.T) {
	// given
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"success":false,"errors":[{"code":1000,"message":"domain not verified"}]}`), nil
	})
	svc := cloudflareConfigured(rt)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Welcome", "<p>Body</p>")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rejected")
}

func TestSend_CloudflareTransportError_Returns(t *testing.T) {
	// given
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	svc := cloudflareConfigured(rt)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Welcome", "<p>Body</p>")

	// then
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotConfigured)
}

func TestEnabled_SMTPConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(map[*config.SiteSettingDef]string{
		config.SettingSMTPHost: "127.0.0.1",
	}, nil)

	// when / then
	assert.True(t, svc.Enabled(context.Background()))
}

func TestEnabled_SMTPNotConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(nil, nil)

	// when / then
	assert.False(t, svc.Enabled(context.Background()))
}

func TestEnabled_CloudflareConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(map[*config.SiteSettingDef]string{
		config.SettingEmailProvider:       string(config.EmailProviderCloudflare),
		config.SettingCloudflareAccountID: "acct-123",
		config.SettingCloudflareAPIToken:  "secret-token",
		config.SettingCloudflareEmailFrom: "noreply@books.test",
	}, nil)

	// when / then
	assert.True(t, svc.Enabled(context.Background()))
}

func TestEnabled_CloudflareMissingToken(t *testing.T) {
	// given
	svc := newServiceWithSettings(map[*config.SiteSettingDef]string{
		config.SettingEmailProvider:       string(config.EmailProviderCloudflare),
		config.SettingCloudflareAccountID: "acct-123",
		config.SettingCloudflareEmailFrom: "noreply@books.test",
	}, nil)

	// when / then
	assert.False(t, svc.Enabled(context.Background()))
}

func TestHtmlToText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "strips tags", in: "<p>Hello <b>world</b></p>", want: "Hello world"},
		{name: "unescapes entities", in: "<p>Tom &amp; Jerry &lt;3 &gt;_&lt;</p>", want: "Tom & Jerry <3 >_<"},
		{name: "replaces nbsp", in: "a&nbsp;b", want: "a b"},
		{name: "trims surrounding whitespace", in: "  <p>  hi  </p>  ", want: "hi"},
		{name: "plain text unchanged", in: "no tags here", want: "no tags here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given / when
			got := htmlToText(tt.in)

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}
