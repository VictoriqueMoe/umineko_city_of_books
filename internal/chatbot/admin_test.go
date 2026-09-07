package chatbot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func anyReloader(t *testing.T) *MockService {
	t.Helper()

	m := NewMockService(t)
	m.EXPECT().Reload().Maybe()

	return m
}

func validRequest(modelName string) dto.ChatbotUpsertRequest {
	return dto.ChatbotUpsertRequest{
		Username:     "beatrice",
		DisplayName:  "Beatrice",
		SystemPrompt: "you are the golden witch",
		Model:        modelName,
		Enabled:      true,
	}
}

func botAccount(displayName string) spec.NewUser {
	return spec.NewUser{
		Username:      "beatrice",
		PasswordHash:  botPasswordHash,
		DisplayName:   displayName,
		HomePage:      botHomePage,
		IsBot:         true,
		DMsEnabled:    true,
		EmailVerified: true,
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
	userSvc := user.NewMockService(t)
	openaiSvc := openai.NewMockService(t)
	openaiSvc.EXPECT().Models(mock.Anything).Return([]string{"gpt-5.6-luna"}, nil).Once()

	svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), repository.NewMockAuditLogRepository(t), userSvc, openaiSvc, anyReloader(t))

	// when
	bot, err := svc.Create(context.Background(), uuid.New(), validRequest("gpt-5.6-mirage"))

	// then
	require.ErrorIs(t, err, ErrBotUnknownModel)
	assert.Nil(t, bot)
	userSvc.AssertNotCalled(t, "CheckUsernameAvailable", mock.Anything, mock.Anything)
	botRepo.AssertNotCalled(t, "CreateBotWithAccount", mock.Anything, mock.Anything)
}

func TestCreate_ProviderOutageDoesNotBlockTheSave(t *testing.T) {
	// given
	reloader := NewMockService(t)
	reloader.EXPECT().Reload().Once()

	actor := uuid.New()
	botID := uuid.New()
	botUserID := uuid.New()

	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().CreateBotWithAccount(mock.Anything, spec.NewChatbotWithAccount{
		Account: botAccount("Beatrice"),
		Bot: model.Chatbot{
			SystemPrompt: "you are the golden witch",
			Model:        "gpt-9-unreleased",
			Enabled:      true,
		},
		VanityRoleID: botVanityRoleID,
	}).Return(&model.Chatbot{
		ID:       botID,
		UserID:   botUserID,
		Username: "beatrice",
		Model:    "gpt-9-unreleased",
	}, nil).Once()

	userSvc := user.NewMockService(t)
	userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, "beatrice").Return(nil).Once()

	openaiSvc := openai.NewMockService(t)
	openaiSvc.EXPECT().Models(mock.Anything).Return(nil, errors.New("provider unreachable")).Once()

	auditRepo := repository.NewMockAuditLogRepository(t)
	auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionChatbotCreate,
		TargetType: audit.TargetChatbot,
		TargetID:   botID.String(),
		Details:    "username=beatrice name= model=gpt-9-unreleased enabled=false",
		SubjectID:  botUserID,
	}).Return(nil).Once()

	svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), auditRepo, userSvc, openaiSvc, reloader)

	// when
	bot, err := svc.Create(context.Background(), actor, validRequest("gpt-9-unreleased"))

	// then
	require.NoError(t, err)
	require.NotNil(t, bot)
	assert.Equal(t, "gpt-9-unreleased", bot.Model)
	reloader.AssertExpectations(t)
}

func TestDelete_MissingBotIsReportedAsNotFound(t *testing.T) {
	// given
	reloader := NewMockService(t)

	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().ListBots(mock.Anything).Return(nil, nil).Once()
	botRepo.EXPECT().DeleteBot(mock.Anything, mock.Anything).Return(dao.ErrBotNotFound).Once()

	svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), repository.NewMockAuditLogRepository(t), user.NewMockService(t), openai.NewMockService(t), reloader)

	// when
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	// then
	require.ErrorIs(t, err, ErrBotNotFound)
	reloader.AssertExpectations(t)
}

func TestDelete_KeepsTheBotNameTheDeleteDestroys(t *testing.T) {
	// given
	reloader := NewMockService(t)
	reloader.EXPECT().Reload().Once()

	actor := uuid.New()
	botID := uuid.New()
	botUserID := uuid.New()

	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().ListBots(mock.Anything).Return([]model.Chatbot{
		{ID: botID, UserID: botUserID, Username: "beatrice", DisplayName: "Beatrice"},
	}, nil).Once()
	botRepo.EXPECT().DeleteBot(mock.Anything, botID).Return(nil).Once()

	auditRepo := repository.NewMockAuditLogRepository(t)
	auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionChatbotDelete,
		TargetType: audit.TargetChatbot,
		TargetID:   botID.String(),
		Details:    "username=beatrice name=Beatrice",
		SubjectID:  botUserID,
	}).Return(nil).Once()

	svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), auditRepo, user.NewMockService(t), openai.NewMockService(t), reloader)

	// when
	err := svc.Delete(context.Background(), actor, botID)

	// then
	require.NoError(t, err)
	reloader.AssertExpectations(t)
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
			userSvc := user.NewMockService(t)
			openaiSvc := openai.NewMockService(t)

			if tc.wantCall {
				openaiSvc.EXPECT().Complete(mock.Anything, mock.MatchedBy(func(req openai.CompletionRequest) bool {
					return req.Model == "gpt-5.6-luna" && req.MaxOutputTokens == testMaxOutputTokens && len(req.Messages) == 1
				})).Return(tc.result, tc.completeErr).Once()
			}

			svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), repository.NewMockAuditLogRepository(t), userSvc, openaiSvc, anyReloader(t))

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
			userSvc := user.NewMockService(t)
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Models(mock.Anything).Return([]string{"gpt-5.6-luna"}, nil).Maybe()
			auditRepo := repository.NewMockAuditLogRepository(t)

			if tc.wantErr == nil {
				botRepo.EXPECT().CreateBotWithAccount(mock.Anything, spec.NewChatbotWithAccount{
					Account: botAccount(tc.want),
					Bot: model.Chatbot{
						SystemPrompt: "you are the golden witch",
						Model:        "gpt-5.6-luna",
						Enabled:      true,
					},
					VanityRoleID: botVanityRoleID,
				}).Return(&model.Chatbot{DisplayName: tc.want}, nil).Once()
				userSvc.EXPECT().CheckUsernameAvailable(mock.Anything, "beatrice").Return(nil).Once()
				auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
					return entry.Action == audit.ActionChatbotCreate && strings.Contains(entry.Details, "name="+tc.want)
				})).Return(nil).Once()
			}

			svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), auditRepo, userSvc, openaiSvc, anyReloader(t))

			req := validRequest("gpt-5.6-luna")
			req.DisplayName = tc.given

			// when
			bot, err := svc.Create(context.Background(), uuid.New(), req)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, bot)
				botRepo.AssertNotCalled(t, "CreateBotWithAccount", mock.Anything, mock.Anything)

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
			userSvc := user.NewMockService(t)
			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Models(mock.Anything).Return([]string{"gpt-5.6-luna"}, nil).Maybe()
			auditRepo := repository.NewMockAuditLogRepository(t)

			if tc.wantErr == nil {
				botRepo.EXPECT().ListBots(mock.Anything).Return([]model.Chatbot{{ID: botID, UserID: botUserID}}, nil).Once()
				botRepo.EXPECT().UpdateBotWithAccount(mock.Anything, spec.ChatbotAccountUpdate{
					Bot: model.Chatbot{
						ID:           botID,
						UserID:       botUserID,
						SystemPrompt: "you are the golden witch",
						Model:        "gpt-5.6-luna",
						Enabled:      true,
					},
					DisplayName: tc.want,
				}).Return(&model.Chatbot{DisplayName: tc.want}, nil).Once()
				auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
					return entry.Action == audit.ActionChatbotUpdate &&
						entry.TargetID == botID.String() &&
						entry.SubjectID == botUserID &&
						strings.Contains(entry.Details, "display_name="+tc.want) &&
						!strings.Contains(entry.Details, validRequest("gpt-5.6-luna").SystemPrompt)
				})).Return(nil).Once()
			}

			svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), auditRepo, userSvc, openaiSvc, anyReloader(t))

			req := validRequest("gpt-5.6-luna")
			req.DisplayName = tc.given

			// when
			bot, err := svc.Update(context.Background(), uuid.New(), botID, req)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, bot)
				botRepo.AssertNotCalled(t, "UpdateBotWithAccount", mock.Anything, mock.Anything)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, tc.want, bot.DisplayName)
		})
	}
}

func TestUpdateBasePrompt_AuditCarriesTheNameNotThePrompt(t *testing.T) {
	// given
	const secret = "you are the golden witch and you never lose"

	actor := uuid.New()
	promptID := uuid.New()

	basePromptRepo := repository.NewMockChatbotBasePromptRepository(t)
	basePromptRepo.EXPECT().Update(mock.Anything, spec.ChatbotBasePromptUpdate{ID: promptID, Name: "Witches", Prompt: secret}).
		Return(&model.ChatbotBasePrompt{ID: promptID, Name: "Witches", Prompt: secret, BotCount: 3}, nil).Once()

	auditRepo := repository.NewMockAuditLogRepository(t)
	auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionChatbotBasePromptUpdate,
		TargetType: audit.TargetChatbotBasePrompt,
		TargetID:   promptID.String(),
		Details:    "name=Witches bots=3",
	}).Return(nil).Once()

	svc := NewAdminService(repository.NewMockChatbotRepository(t), basePromptRepo, auditRepo, user.NewMockService(t), openai.NewMockService(t), anyReloader(t))

	// when
	updated, err := svc.UpdateBasePrompt(context.Background(), actor, promptID, dto.ChatbotBasePromptUpsertRequest{Name: "Witches", Prompt: secret})

	// then
	require.NoError(t, err)
	require.NotNil(t, updated)
}

func TestDeleteBasePrompt_KeepsTheNameTheDeleteDestroys(t *testing.T) {
	// given
	actor := uuid.New()
	promptID := uuid.New()

	basePromptRepo := repository.NewMockChatbotBasePromptRepository(t)
	basePromptRepo.EXPECT().GetByID(mock.Anything, promptID).
		Return(&model.ChatbotBasePrompt{ID: promptID, Name: "Witches", Prompt: "you are the golden witch"}, nil).Once()
	basePromptRepo.EXPECT().Delete(mock.Anything, promptID).Return(nil).Once()

	auditRepo := repository.NewMockAuditLogRepository(t)
	auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionChatbotBasePromptDelete,
		TargetType: audit.TargetChatbotBasePrompt,
		TargetID:   promptID.String(),
		Details:    "name=Witches bots=0",
	}).Return(nil).Once()

	svc := NewAdminService(repository.NewMockChatbotRepository(t), basePromptRepo, auditRepo, user.NewMockService(t), openai.NewMockService(t), anyReloader(t))

	// when
	err := svc.DeleteBasePrompt(context.Background(), actor, promptID)

	// then
	require.NoError(t, err)
}

func TestUsage_ReportsFailureCounts(t *testing.T) {
	// given
	botRepo := repository.NewMockChatbotRepository(t)
	botRepo.EXPECT().StatsSince(mock.Anything, mock.Anything).Return(&model.ChatbotStats{
		Invocations: 10,
		Failed:      3,
		Quota:       2,
	}, nil).Once()

	userSvc := user.NewMockService(t)

	openaiSvc := openai.NewMockService(t)
	openaiSvc.EXPECT().Costs(mock.Anything, mock.Anything).Return(nil, errors.New("no admin key")).Once()

	svc := NewAdminService(botRepo, repository.NewMockChatbotBasePromptRepository(t), repository.NewMockAuditLogRepository(t), userSvc, openaiSvc, anyReloader(t))

	// when
	usage, err := svc.Usage(context.Background(), time.Unix(1_700_000_000, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 10, usage.Invocations)
	assert.Equal(t, 3, usage.Failed)
	assert.Equal(t, 2, usage.Quota)
	assert.Nil(t, usage.BilledUSD)
}
