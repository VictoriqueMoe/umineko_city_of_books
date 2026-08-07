package chatbot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubReloader struct {
	reloads int
}

func (s *stubReloader) ObserveMessage(_ chat.BotMessageEvent) {}

func (s *stubReloader) ObserveComment(_ post.BotContentEvent) {}

func (s *stubReloader) Enabled() bool { return true }

func (s *stubReloader) OnSettingChanged(_ config.SiteSettingKey, _ string) {}

func (s *stubReloader) OnSettingsBatchChanged(_ []config.SiteSettingKey) {}

func (s *stubReloader) Reload() { s.reloads++ }

func (s *stubReloader) Shutdown(_ context.Context) error { return nil }

func validRequest(model string) dto.ChatbotUpsertRequest {
	return dto.ChatbotUpsertRequest{
		Username:     "beatrice",
		DisplayName:  "Beatrice",
		SystemPrompt: "you are the golden witch",
		Model:        model,
		Enabled:      true,
	}
}

func TestValidateModel_FailsOpen(t *testing.T) {
	// given
	cases := []struct {
		name      string
		model     string
		available []string
		listErr   error
		wantErr   bool
	}{
		{
			name:      "known model is accepted",
			model:     "gpt-5.6-luna",
			available: []string{"gpt-5.6-luna", "gpt-5.6-terra"},
		},
		{
			name:      "unknown model is rejected",
			model:     "gpt-4o-typo",
			available: []string{"gpt-5.6-luna", "gpt-5.6-terra"},
			wantErr:   true,
		},
		{
			name:      "blank model inherits and is always accepted",
			model:     "   ",
			available: []string{"gpt-5.6-luna"},
		},
		{
			name:    "provider unreachable accepts anything",
			model:   "gpt-9-unreleased",
			listErr: errors.New("dial tcp: connection refused"),
		},
		{
			name:      "empty catalogue accepts anything",
			model:     "gpt-9-unreleased",
			available: []string{},
		},
		{
			name:      "nil catalogue accepts anything",
			model:     "gpt-9-unreleased",
			available: nil,
		},
		{
			name:      "model not named gpt is accepted when the provider lists it",
			model:     "beatrice-golden-witch",
			available: []string{"beatrice-golden-witch"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Models(mock.Anything).Return(tc.available, tc.listErr).Maybe()

			// when
			err := validateModel(context.Background(), openaiSvc, tc.model)

			// then
			if !tc.wantErr {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, ErrBotUnknownModel)
			assert.Contains(t, err.Error(), tc.model)
		})
	}
}

func TestModelValidator_MatchesUpsertValidation(t *testing.T) {
	// given
	cases := []struct {
		name      string
		value     string
		available []string
		listErr   error
		wantErr   bool
	}{
		{"known model", "gpt-5.6-luna", []string{"gpt-5.6-luna"}, nil, false},
		{"unknown model", "gpt-5.6-nope", []string{"gpt-5.6-luna"}, nil, true},
		{"blank stays allowed", "", []string{"gpt-5.6-luna"}, nil, false},
		{"provider down", "gpt-5.6-nope", nil, errors.New("boom"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Models(mock.Anything).Return(tc.available, tc.listErr).Maybe()

			validate := ModelValidator(openaiSvc)

			// when
			err := validate(context.Background(), tc.value)

			// then
			if tc.wantErr {
				require.ErrorIs(t, err, ErrBotUnknownModel)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCreate_RejectsUnknownModelBeforeTouchingTheDatabase(t *testing.T) {
	// given
	botRepo := repository.NewMockChatbotRepository(t)
	vanityRepo := repository.NewMockVanityRoleRepository(t)
	userSvc := user.NewMockService(t)
	openaiSvc := openai.NewMockService(t)
	openaiSvc.EXPECT().Models(mock.Anything).Return([]string{"gpt-5.6-luna"}, nil).Once()

	svc := NewAdminService(botRepo, vanityRepo, userSvc, openaiSvc, new(stubReloader))

	// when
	bot, err := svc.Create(context.Background(), validRequest("gpt-5.6-mirage"))

	// then
	require.ErrorIs(t, err, ErrBotUnknownModel)
	assert.Nil(t, bot)
	userSvc.AssertNotCalled(t, "CheckUsernameAvailable", mock.Anything, mock.Anything)
	botRepo.AssertNotCalled(t, "CreateBot", mock.Anything, mock.Anything)
}

func TestCreate_ProviderOutageDoesNotBlockTheSave(t *testing.T) {
	// given
	reloader := new(stubReloader)

	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().CreateBotAccount(mock.Anything, mock.Anything, "beatrice", "Beatrice", "").Return(nil).Once()
	botRepo.EXPECT().CreateBot(mock.Anything, mock.Anything).Return(nil).Once()
	botRepo.EXPECT().GetBotByUserID(mock.Anything, mock.Anything).Return(&repository.Chatbot{
		Username: "beatrice",
		Model:    "gpt-9-unreleased",
	}, nil).Once()

	vanityRepo := repository.NewMockVanityRoleRepository(t)
	vanityRepo.EXPECT().AssignToUser(mock.Anything, mock.Anything, botVanityRoleID).Return(nil).Once()

	userSvc := user.NewMockService(t)
	userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, "beatrice").Return(nil).Once()

	openaiSvc := openai.NewMockService(t)
	openaiSvc.EXPECT().Models(mock.Anything).Return(nil, errors.New("provider unreachable")).Once()

	svc := NewAdminService(botRepo, vanityRepo, userSvc, openaiSvc, reloader)

	// when
	bot, err := svc.Create(context.Background(), validRequest("gpt-9-unreleased"))

	// then
	require.NoError(t, err)
	require.NotNil(t, bot)
	assert.Equal(t, "gpt-9-unreleased", bot.Model)
	assert.Equal(t, 1, reloader.reloads)
}

func TestDelete_MissingBotIsReportedAsNotFound(t *testing.T) {
	// given
	reloader := new(stubReloader)

	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().DeleteBot(mock.Anything, mock.Anything).Return(repository.ErrBotNotFound).Once()

	svc := NewAdminService(botRepo, repository.NewMockVanityRoleRepository(t), user.NewMockService(t), openai.NewMockService(t), reloader)

	// when
	err := svc.Delete(context.Background(), uuid.New())

	// then
	require.ErrorIs(t, err, ErrBotNotFound)
	assert.Equal(t, 0, reloader.reloads)
}

func TestTest_ProviderFailureIsReportedNotPropagated(t *testing.T) {
	// given
	cases := []struct {
		name        string
		model       string
		result      *openai.CompletionResult
		completeErr error
		wantOK      bool
		wantMessage string
		wantCall    bool
	}{
		{
			name:     "success",
			model:    "gpt-5.6-luna",
			result:   &openai.CompletionResult{Text: "pong"},
			wantOK:   true,
			wantCall: true,
		},
		{
			name:        "blank model never calls the provider",
			model:       "  ",
			wantMessage: "pick a model first",
		},
		{
			name:        "provider error becomes the sentence the provider wrote",
			model:       "gpt-5.6-luna",
			completeErr: &openai.APIError{StatusCode: 400, Body: `{"error":{"message":"The requested model 'gpe' does not exist.","type":"invalid_request_error"}}`},
			wantMessage: "The requested model 'gpe' does not exist.",
			wantCall:    true,
		},
		{
			name:        "unreadable provider body becomes the status alone",
			model:       "gpt-5.6-luna",
			completeErr: &openai.APIError{StatusCode: 502, Body: "<html>Bad Gateway</html>"},
			wantMessage: "the provider rejected the request with status 502",
			wantCall:    true,
		},
		{
			name:        "other failures keep their own wording",
			model:       "gpt-5.6-luna",
			completeErr: openai.ErrDisabled,
			wantMessage: "openai integration is not configured",
			wantCall:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			botRepo := repository.NewMockChatbotRepository(t)
			vanityRepo := repository.NewMockVanityRoleRepository(t)
			userSvc := user.NewMockService(t)
			openaiSvc := openai.NewMockService(t)

			if tc.wantCall {
				openaiSvc.EXPECT().Complete(mock.Anything, mock.MatchedBy(func(req openai.CompletionRequest) bool {
					return req.Model == "gpt-5.6-luna" && req.MaxOutputTokens == testMaxOutputTokens && len(req.Messages) == 1
				})).Return(tc.result, tc.completeErr).Once()
			}

			svc := NewAdminService(botRepo, vanityRepo, userSvc, openaiSvc, new(stubReloader))

			// when
			ok, message, err := svc.Test(context.Background(), tc.model)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantMessage, message)
		})
	}
}

func TestUpsert_ClampsDisplayName(t *testing.T) {
	// given
	cases := []struct {
		name    string
		given   string
		want    string
		wantErr error
	}{
		{
			name:  "a plain name is passed through",
			given: "Beatrice",
			want:  "Beatrice",
		},
		{
			name:  "markup is stripped",
			given: `Beatrice<script>alert("xss")</script>`,
			want:  "Beatrice",
		},
		{
			name:  "surrounding and repeated whitespace is collapsed",
			given: "  Lady   Beatrice  ",
			want:  "Lady Beatrice",
		},
		{
			name:  "an over-long name is capped at forty runes",
			given: strings.Repeat("ぞ", 60),
			want:  strings.Repeat("ぞ", user.MaxDisplayNameRunes),
		},
		{
			name:    "a name that is only markup is rejected",
			given:   "<b></b><img src=''>",
			wantErr: ErrBotInvalid,
		},
	}

	for _, tc := range cases {
		t.Run("create/"+tc.name, func(t *testing.T) {
			botRepo := repository.NewMockChatbotRepository(t)
			vanityRepo := repository.NewMockVanityRoleRepository(t)
			userSvc := user.NewMockService(t)
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Models(mock.Anything).Return([]string{"gpt-5.6-luna"}, nil).Maybe()

			if tc.wantErr == nil {
				botRepo.EXPECT().CreateBotAccount(mock.Anything, mock.Anything, "beatrice", tc.want, "").Return(nil).Once()
				botRepo.EXPECT().CreateBot(mock.Anything, mock.Anything).Return(nil).Once()
				botRepo.EXPECT().GetBotByUserID(mock.Anything, mock.Anything).Return(&repository.Chatbot{DisplayName: tc.want}, nil).Once()
				vanityRepo.EXPECT().AssignToUser(mock.Anything, mock.Anything, botVanityRoleID).Return(nil).Once()
				userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, "beatrice").Return(nil).Once()
			}

			svc := NewAdminService(botRepo, vanityRepo, userSvc, openaiSvc, new(stubReloader))

			req := validRequest("gpt-5.6-luna")
			req.DisplayName = tc.given

			// when
			bot, err := svc.Create(context.Background(), req)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, bot)
				botRepo.AssertNotCalled(t, "CreateBotAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, tc.want, bot.DisplayName)
		})
	}

	for _, tc := range cases {
		t.Run("update/"+tc.name, func(t *testing.T) {
			botID := uuid.New()
			botUserID := uuid.New()

			botRepo := repository.NewMockChatbotRepository(t)
			vanityRepo := repository.NewMockVanityRoleRepository(t)
			userSvc := user.NewMockService(t)
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Models(mock.Anything).Return([]string{"gpt-5.6-luna"}, nil).Maybe()

			if tc.wantErr == nil {
				botRepo.EXPECT().ListBots(mock.Anything).Return([]repository.Chatbot{{ID: botID, UserID: botUserID}}, nil).Once()
				botRepo.EXPECT().UpdateBot(mock.Anything, mock.Anything).Return(nil).Once()
				botRepo.EXPECT().UpdateBotAccount(mock.Anything, botUserID, tc.want, "").Return(nil).Once()
				botRepo.EXPECT().GetBotByUserID(mock.Anything, botUserID).Return(&repository.Chatbot{DisplayName: tc.want}, nil).Once()
			}

			svc := NewAdminService(botRepo, vanityRepo, userSvc, openaiSvc, new(stubReloader))

			req := validRequest("gpt-5.6-luna")
			req.DisplayName = tc.given

			// when
			bot, err := svc.Update(context.Background(), botID, req)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, bot)
				botRepo.AssertNotCalled(t, "UpdateBotAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, tc.want, bot.DisplayName)
		})
	}
}

func TestUsage_ReportsFailureCounts(t *testing.T) {
	// given
	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().StatsSince(mock.Anything, mock.Anything).Return(&repository.ChatbotStats{
		Invocations: 10,
		Failed:      3,
		Quota:       2,
	}, nil).Once()

	vanityRepo := repository.NewMockVanityRoleRepository(t)
	userSvc := user.NewMockService(t)

	openaiSvc := openai.NewMockService(t)
	openaiSvc.EXPECT().Costs(mock.Anything, mock.Anything).Return(nil, errors.New("no admin key")).Once()

	svc := NewAdminService(botRepo, vanityRepo, userSvc, openaiSvc, new(stubReloader))

	// when
	usage, err := svc.Usage(context.Background(), time.Unix(1_700_000_000, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 10, usage.Invocations)
	assert.Equal(t, 3, usage.Failed)
	assert.Equal(t, 2, usage.Quota)
	assert.Nil(t, usage.BilledUSD)
}
