package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	wireRequest struct {
		Model           string         `json:"model"`
		Input           []wireInput    `json:"input"`
		Reasoning       *wireReasoning `json:"reasoning"`
		MaxOutputTokens int            `json:"max_output_tokens"`
		PromptCacheKey  string         `json:"prompt_cache_key"`

		PromptCacheOptions *wireCacheOptions `json:"prompt_cache_options"`
	}

	wireCacheOptions struct {
		Mode string `json:"mode"`
	}

	wireInput struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}

	wireReasoning struct {
		Effort string `json:"effort"`
	}
)

const (
	testAPIKey   = "sk-test-key"
	testAdminKey = "sk-admin-key"

	completeBody = `{
		"status": "completed",
		"output": [
			{"type": "reasoning", "content": []},
			{"type": "message", "content": [{"type": "output_text", "text": "Hello, Kujo."}]}
		],
		"usage": {
			"input_tokens": 120,
			"input_tokens_details": {"cached_tokens": 80},
			"output_tokens": 34,
			"output_tokens_details": {"reasoning_tokens": 12}
		}
	}`

	incompleteBody = `{
		"status": "incomplete",
		"output": [{"type": "message", "content": [{"type": "output_text", "text": "half a sent"}]}],
		"usage": {
			"input_tokens": 5,
			"input_tokens_details": {"cached_tokens": 0},
			"output_tokens": 2000,
			"output_tokens_details": {"reasoning_tokens": 1990}
		}
	}`
)

func newTestService(t *testing.T, apiKey, adminKey string, handler http.HandlerFunc) *service {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAPIKey).Return(apiKey).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAdminKey).Return(adminKey).Maybe()

	return newService(settingsSvc, srv.URL)
}

func TestEnabled(t *testing.T) {
	// given
	cases := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"key set", testAPIKey, true},
		{"key blank", "", false},
		{"key whitespace", "   ", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, tc.apiKey, "", func(w http.ResponseWriter, r *http.Request) {
				t.Error("Enabled must not make an http request")
			})

			// when
			got := svc.Enabled()

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestComplete(t *testing.T) {
	// given
	cases := []struct {
		name      string
		apiKey    string
		body      string
		wantCalls int
		wantErr   error
		want      *CompletionResult
	}{
		{
			name:      "disabled when key blank",
			apiKey:    "",
			body:      completeBody,
			wantCalls: 0,
			wantErr:   ErrDisabled,
		},
		{
			name:      "successful completion",
			apiKey:    testAPIKey,
			body:      completeBody,
			wantCalls: 1,
			want: &CompletionResult{
				Text:               "Hello, Kujo.",
				PromptTokens:       120,
				CachedPromptTokens: 80,
				CompletionTokens:   34,
				ReasoningTokens:    12,
				Incomplete:         false,
			},
		},
		{
			name:      "incomplete response is flagged",
			apiKey:    testAPIKey,
			body:      incompleteBody,
			wantCalls: 1,
			want: &CompletionResult{
				Text:               "half a sent",
				PromptTokens:       5,
				CachedPromptTokens: 0,
				CompletionTokens:   2000,
				ReasoningTokens:    1990,
				Incomplete:         true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				calls int
				got   wireRequest
			)

			svc := newTestService(t, tc.apiKey, "", func(w http.ResponseWriter, r *http.Request) {
				calls++

				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/responses", r.URL.Path)
				assert.Equal(t, "Bearer "+tc.apiKey, r.Header.Get("Authorization"))
				require.NoError(t, json.NewDecoder(r.Body).Decode(&got))

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			})

			// when
			res, err := svc.Complete(context.Background(), CompletionRequest{
				Model:           "gpt-5.6-luna",
				SystemPrompt:    "you are Beatrice",
				Messages:        []Message{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "hello"}},
				ReasoningEffort: "low",
				MaxOutputTokens: 2000,
				CacheKey:        "bot-1",
			})

			// then
			assert.Equal(t, tc.wantCalls, calls)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, res)

			require.Len(t, got.Input, 3)
			assert.Equal(t, "system", got.Input[0].Role)
			assert.JSONEq(t,
				`[{"type":"input_text","text":"you are Beatrice","prompt_cache_breakpoint":{"mode":"explicit"}}]`,
				string(got.Input[0].Content),
				"the persona must end an explicitly marked cache prefix")
			assert.Equal(t, "user", got.Input[1].Role)
			assert.JSONEq(t, `"hi"`, string(got.Input[1].Content))
			assert.Equal(t, "assistant", got.Input[2].Role)
			assert.JSONEq(t, `"hello"`, string(got.Input[2].Content))
			require.NotNil(t, got.PromptCacheOptions)
			assert.Equal(t, "explicit", got.PromptCacheOptions.Mode,
				"implicit mode would put the breakpoint on the changing message and cache nothing reusable")
			assert.Equal(t, "gpt-5.6-luna", got.Model)
			assert.Equal(t, "bot-1", got.PromptCacheKey)
			assert.Equal(t, 2000, got.MaxOutputTokens)
			require.NotNil(t, got.Reasoning)
			assert.Equal(t, "low", got.Reasoning.Effort)
		})
	}
}

func TestComplete_OptionalFieldsOmitted(t *testing.T) {
	// given
	var got wireRequest

	svc := newTestService(t, testAPIKey, "", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completeBody))
	})

	// when
	_, err := svc.Complete(context.Background(), CompletionRequest{
		Model:    "gpt-5.6-luna",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	// then
	require.NoError(t, err)
	assert.Nil(t, got.Reasoning)
	assert.Empty(t, got.PromptCacheKey)
	assert.Zero(t, got.MaxOutputTokens)
	require.Len(t, got.Input, 1)
	assert.Equal(t, "user", got.Input[0].Role)
	assert.JSONEq(t, `"hi"`, string(got.Input[0].Content))
	assert.Nil(t, got.PromptCacheOptions, "with no persona there is nothing stable to cache")
}

func TestComplete_RateLimitTripsBreaker(t *testing.T) {
	// given
	var calls int

	svc := newTestService(t, testAPIKey, "", func(w http.ResponseWriter, r *http.Request) {
		calls++

		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	// when
	_, err := svc.Complete(context.Background(), CompletionRequest{Model: "gpt-5.6-luna"})

	// then
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Equal(t, 1, calls)

	// when
	for range 5 {
		_, err = svc.Complete(context.Background(), CompletionRequest{Model: "gpt-5.6-luna"})
		require.ErrorIs(t, err, ErrRateLimited)
	}

	// then
	assert.Equal(t, 1, calls, "calls during the blackout must not reach upstream")
}

func TestCosts_RateLimitTripsBreaker(t *testing.T) {
	// given
	var calls int

	svc := newTestService(t, testAPIKey, testAdminKey, func(w http.ResponseWriter, r *http.Request) {
		calls++

		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	// when
	_, err := svc.Costs(context.Background(), time.Unix(1_700_000_000, 0))

	// then
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Equal(t, 1, calls)

	// when
	_, err = svc.Costs(context.Background(), time.Unix(1_700_000_000, 0))

	// then
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Equal(t, 1, calls, "calls during the blackout must not reach upstream")
}

func TestErrorsHideRequestDetails(t *testing.T) {
	// given
	cases := []struct {
		name string
		body string
		call func(svc *service) error
	}{
		{
			name: "complete json error body",
			body: `{"error":{"message":"bad model"}}`,
			call: func(svc *service) error {
				_, err := svc.Complete(context.Background(), CompletionRequest{Model: "nope"})

				return err
			},
		},
		{
			name: "complete empty error body",
			body: "",
			call: func(svc *service) error {
				_, err := svc.Complete(context.Background(), CompletionRequest{Model: "nope"})

				return err
			},
		},
		{
			name: "costs json error body",
			body: `{"error":{"message":"no access"}}`,
			call: func(svc *service) error {
				_, err := svc.Costs(context.Background(), time.Unix(1_700_000_000, 0))

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, testAPIKey, testAdminKey, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(tc.body))
			})

			// when
			err := tc.call(svc)

			// then
			apiErr, ok := errors.AsType[*APIError](err)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.LessOrEqual(t, len(apiErr.Body), maxErrorBodyBytes)

			assert.NotContains(t, err.Error(), svc.baseURL)
			assert.NotContains(t, err.Error(), testAPIKey)
			assert.NotContains(t, err.Error(), testAdminKey)
		})
	}
}

func TestComplete_TransportErrorHidesURL(t *testing.T) {
	// given
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAPIKey).Return(testAPIKey).Once()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAdminKey).Return("").Once()

	svc := newService(settingsSvc, "http://127.0.0.1:1/")

	// when
	_, err := svc.Complete(context.Background(), CompletionRequest{Model: "nope"})

	// then
	require.Error(t, err)
	assert.NotContains(t, err.Error(), svc.baseURL)
	assert.NotContains(t, err.Error(), testAPIKey)
}

func TestCosts(t *testing.T) {
	// given
	cases := []struct {
		name      string
		adminKey  string
		body      string
		wantCalls int
		want      *CostsResult
	}{
		{
			name:      "blank admin key makes no request",
			adminKey:  "",
			body:      `{"data":[]}`,
			wantCalls: 0,
			want:      nil,
		},
		{
			name:     "sums every bucket result",
			adminKey: testAdminKey,
			body: `{"data":[
				{"results":[
					{"object":"organization.costs.result","amount":{"value":0.25,"currency":"usd"}},
					{"object":"organization.costs.result","amount":{"value":0.5,"currency":"usd"}}
				]},
				{"results":[{"object":"organization.costs.result","amount":{"value":1.25,"currency":"usd"}}]}
			]}`,
			wantCalls: 1,
			want:      &CostsResult{AmountUSD: 2, Currency: "usd"},
		},
	}

	since := time.Unix(1_700_000_000, 0)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls int

			svc := newTestService(t, testAPIKey, tc.adminKey, func(w http.ResponseWriter, r *http.Request) {
				calls++

				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/organization/costs", r.URL.Path)
				assert.Equal(t, "1700000000", r.URL.Query().Get("start_time"))
				assert.Equal(t, "180", r.URL.Query().Get("limit"))
				assert.Empty(t, r.URL.Query().Get("bucket_width"))
				assert.Equal(t, "Bearer "+tc.adminKey, r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			})

			// when
			res, err := svc.Costs(context.Background(), since)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantCalls, calls)

			if tc.want == nil {
				assert.Nil(t, res)

				return
			}

			require.NotNil(t, res)
			assert.InDelta(t, tc.want.AmountUSD, res.AmountUSD, 0.0001)
			assert.Equal(t, tc.want.Currency, res.Currency)
		})
	}
}

func TestModels(t *testing.T) {
	// given
	cases := []struct {
		name      string
		apiKey    string
		body      string
		wantCalls int
		want      []string
	}{
		{
			name:      "empty catalogue",
			apiKey:    testAPIKey,
			body:      `{"object":"list","data":[]}`,
			wantCalls: 1,
			want:      []string{},
		},
		{
			name:   "newest first",
			apiKey: testAPIKey,
			body: `{"object":"list","data":[
				{"id":"gpt-5.6-terra","created":300,"owned_by":"openai"},
				{"id":"gpt-5.6-luna","created":500,"owned_by":"openai"},
				{"id":"whisper-1","created":100,"owned_by":"openai"}
			]}`,
			wantCalls: 1,
			want:      []string{"gpt-5.6-luna", "gpt-5.6-terra", "whisper-1"},
		},
		{
			name:   "nothing is filtered out",
			apiKey: testAPIKey,
			body: `{"object":"list","data":[
				{"id":"dall-e-3","created":400,"owned_by":"openai"},
				{"id":"beatrice-golden-witch","created":900,"owned_by":"rokkenjima"},
				{"id":"text-embedding-3-large","created":200,"owned_by":"openai"},
				{"id":"ft:gpt-5.6-luna:umineko::abc123","created":700,"owned_by":"user"}
			]}`,
			wantCalls: 1,
			want:      []string{"beatrice-golden-witch", "ft:gpt-5.6-luna:umineko::abc123", "dall-e-3", "text-embedding-3-large"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls int

			svc := newTestService(t, tc.apiKey, "", func(w http.ResponseWriter, r *http.Request) {
				calls++

				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/models", r.URL.Path)
				assert.Equal(t, "Bearer "+tc.apiKey, r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			})

			// when
			got, err := svc.Models(context.Background())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantCalls, calls)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestModels_BlankAPIKeyReportsNotConfigured(t *testing.T) {
	// given
	var calls int

	svc := newTestService(t, "", "", func(w http.ResponseWriter, r *http.Request) {
		calls++
	})

	// when
	got, err := svc.Models(context.Background())

	// then
	require.ErrorIs(t, err, ErrDisabled)
	assert.Nil(t, got)
	assert.Equal(t, 0, calls)
}

func TestReason_NamesTheActualCause(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"no error", nil, ""},
		{"no key saved", ErrDisabled, "No OpenAI API key is saved on the server."},
		{
			"provider rejection carries its own sentence",
			&APIError{StatusCode: 401, Body: `{"error":{"message":"Incorrect API key provided."}}`},
			"OpenAI answered 401: Incorrect API key provided.",
		},
		{
			"missing scope is reported as the provider stated it",
			&APIError{StatusCode: 403, Body: `{"error":{"message":"You have insufficient permissions for this operation. Missing scopes: api.model.read."}}`},
			"OpenAI answered 403: You have insufficient permissions for this operation. Missing scopes: api.model.read.",
		},
		{
			"rate limit names the reset time",
			&RateLimitError{ResetAt: time.Unix(1_700_000_000, 0).UTC()},
			"OpenAI rate limited this server until 2023-11-14T22:13:20Z.",
		},
		{"anything else is not blamed on the key", errors.New("dial tcp: lookup api.openai.com: no such host"), "OpenAI could not be reached from the server."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := Reason(tc.err)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestModels_RateLimitTripsBreaker(t *testing.T) {
	// given
	var calls int

	svc := newTestService(t, testAPIKey, "", func(w http.ResponseWriter, r *http.Request) {
		calls++

		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	// when
	_, err := svc.Models(context.Background())

	// then
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Equal(t, 1, calls)

	// when
	_, err = svc.Models(context.Background())

	// then
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Equal(t, 1, calls, "calls during the blackout must not reach upstream")
}

func TestModels_ErrorHidesRequestDetails(t *testing.T) {
	// given
	svc := newTestService(t, testAPIKey, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	})

	// when
	got, err := svc.Models(context.Background())

	// then
	assert.Nil(t, got)

	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.NotContains(t, err.Error(), svc.baseURL)
	assert.NotContains(t, err.Error(), testAPIKey)
}

func TestSettingsAreReadOnceNotPerCall(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completeBody))
	}))
	t.Cleanup(srv.Close)

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAPIKey).Return(testAPIKey).Once()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAdminKey).Return("").Once()

	svc := newService(settingsSvc, srv.URL)

	// when
	for range 3 {
		_, err := svc.Complete(context.Background(), CompletionRequest{Model: "gpt-5.6-luna"})
		require.NoError(t, err)
	}

	assert.True(t, svc.Enabled())

	// then
	settingsSvc.AssertNumberOfCalls(t, "Get", 2)
}

func TestOnSettingsBatchChangedRebuildsClients(t *testing.T) {
	// given
	const rotatedKey = "sk-rotated-key"

	var seen []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completeBody))
	}))
	t.Cleanup(srv.Close)

	settingsSvc := settings.NewMockService(t)
	apiKey := testAPIKey
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAPIKey).RunAndReturn(func(_ context.Context, _ *config.SiteSettingDef) string {
		return apiKey
	})
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAdminKey).Return("")

	svc := newService(settingsSvc, srv.URL)

	_, err := svc.Complete(context.Background(), CompletionRequest{Model: "gpt-5.6-luna"})
	require.NoError(t, err)

	// when
	apiKey = rotatedKey
	svc.OnSettingsBatchChanged([]config.SiteSettingKey{config.SettingChatbotAPIKey.Key})

	_, err = svc.Complete(context.Background(), CompletionRequest{Model: "gpt-5.6-luna"})
	require.NoError(t, err)

	// then
	require.Len(t, seen, 2)
	assert.Equal(t, "Bearer "+testAPIKey, seen[0])
	assert.Equal(t, "Bearer "+rotatedKey, seen[1])
}

func TestOnSettingsBatchChangedIgnoresUnrelatedKeys(t *testing.T) {
	// given
	cases := []struct {
		name       string
		key        config.SiteSettingKey
		wantReload bool
	}{
		{"chatbot key reloads", config.SettingChatbotAPIKey.Key, true},
		{"unrelated key is ignored", config.SiteSettingKey("site_name"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var loads int

			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAPIKey).RunAndReturn(func(_ context.Context, _ *config.SiteSettingDef) string {
				loads++

				return testAPIKey
			})
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotAdminKey).Return("")

			svc := newService(settingsSvc, "http://127.0.0.1:1/")
			require.Equal(t, 1, loads)

			// when
			svc.OnSettingsBatchChanged([]config.SiteSettingKey{tc.key})

			// then
			want := 1
			if tc.wantReload {
				want = 2
			}

			assert.Equal(t, want, loads)
		})
	}
}

func TestParseRateLimitReset(t *testing.T) {
	// given
	now := time.Unix(1_000_000, 0)

	cases := []struct {
		name       string
		retryAfter string
		want       time.Time
	}{
		{"seconds", "120", now.Add(2 * time.Minute)},
		{"missing header", "", now.Add(defaultRateLimitHold)},
		{"garbage header", "soon", now.Add(defaultRateLimitHold)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}

			if tc.retryAfter != "" {
				h.Set("Retry-After", tc.retryAfter)
			}

			// when
			got := parseRateLimitReset(h, now)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
