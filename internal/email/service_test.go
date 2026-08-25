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

	"github.com/wneessen/go-mail"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newServiceWithSettings(t *testing.T, values map[*config.SiteSettingDef]string, rt http.RoundTripper) *service {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, def *config.SiteSettingDef) string {
			if v, ok := values[def]; ok {
				return v
			}

			return def.Default
		}).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, def *config.SiteSettingDef) int {
			raw, ok := values[def]
			if !ok {
				raw = def.Default
			}

			n, err := strconv.Atoi(raw)
			if err != nil {
				return 0
			}

			return n
		}).Maybe()

	svc := NewService(settingsSvc).(*service)
	if rt != nil {
		svc.httpClient = &http.Client{Transport: rt}
	}

	return svc
}

func cloudflareConfigured(t *testing.T, rt http.RoundTripper) *service {
	t.Helper()

	return newServiceWithSettings(t, map[*config.SiteSettingDef]string{
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
	svc := newServiceWithSettings(t, nil, nil)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Subject", "<p>Body</p>")

	// then
	require.NoError(t, err)
}

func TestSendTest_SMTPNotConfigured_ReturnsNotConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(t, nil, nil)

	// when
	err := svc.SendTest(context.Background(), "user@example.com", "Subject", "<p>Body</p>")

	// then
	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestSendTest_CloudflareNotConfigured_ReturnsNotConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(t, map[*config.SiteSettingDef]string{
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
	svc := cloudflareConfigured(t, rt)

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
	svc := cloudflareConfigured(t, rt)

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
	svc := cloudflareConfigured(t, rt)

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
	svc := cloudflareConfigured(t, rt)

	// when
	err := svc.Send(context.Background(), "user@example.com", "Welcome", "<p>Body</p>")

	// then
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotConfigured)
}

func TestEnabled_SMTPConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(t, map[*config.SiteSettingDef]string{
		config.SettingSMTPHost: "127.0.0.1",
	}, nil)

	// when / then
	assert.True(t, svc.Enabled(context.Background()))
}

func TestEnabled_SMTPNotConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(t, nil, nil)

	// when / then
	assert.False(t, svc.Enabled(context.Background()))
}

func TestEnabled_CloudflareConfigured(t *testing.T) {
	// given
	svc := newServiceWithSettings(t, map[*config.SiteSettingDef]string{
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
	svc := newServiceWithSettings(t, map[*config.SiteSettingDef]string{
		config.SettingEmailProvider:       string(config.EmailProviderCloudflare),
		config.SettingCloudflareAccountID: "acct-123",
		config.SettingCloudflareEmailFrom: "noreply@books.test",
	}, nil)

	// when / then
	assert.False(t, svc.Enabled(context.Background()))
}

func TestBuildClient_TLSPolicyForUncredentialedHosts(t *testing.T) {
	// given
	cases := []struct {
		name string
		host string
		want string
	}{
		{name: "loopback ip may stay in cleartext", host: "127.0.0.1", want: mail.NoTLS.String()},
		{name: "ipv6 loopback may stay in cleartext", host: "::1", want: mail.NoTLS.String()},
		{name: "localhost may stay in cleartext", host: "localhost", want: mail.NoTLS.String()},
		{name: "remote host must not disable tls", host: "smtp.example.com", want: mail.TLSOpportunistic.String()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			svc := newServiceWithSettings(t, map[*config.SiteSettingDef]string{
				config.SettingSMTPHost: tc.host,
			}, nil)

			// then
			require.NotNil(t, svc.client)
			assert.Equal(t, tc.want, svc.client.TLSPolicy())
		})
	}
}

func TestBuildClient_CredentialsKeepMandatoryTLS(t *testing.T) {
	// given
	svc := newServiceWithSettings(t, map[*config.SiteSettingDef]string{
		config.SettingSMTPHost:     "smtp.example.com",
		config.SettingSMTPUsername: "beatrice",
		config.SettingSMTPPassword: "golden",
	}, nil)

	// then
	require.NotNil(t, svc.client)
	assert.Equal(t, mail.TLSMandatory.String(), svc.client.TLSPolicy())
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
