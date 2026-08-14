package chatbot

import (
	"context"
	"errors"
	"testing"

	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOverQuota(t *testing.T) {
	cases := []struct {
		name          string
		perUserPerDay int
		perDay        int
		userUsed      int
		userErr       error
		siteUsed      int
		siteErr       error
		want          bool
		wantGlobal    bool
		wantErr       bool
	}{
		{name: "both ceilings disabled"},
		{name: "under both ceilings", perUserPerDay: 20, perDay: 500, userUsed: 3, siteUsed: 40},
		{name: "per user ceiling reached", perUserPerDay: 20, perDay: 500, userUsed: 20, want: true},
		{name: "site ceiling reached", perUserPerDay: 20, perDay: 500, userUsed: 1, siteUsed: 500, want: true, wantGlobal: true},
		{name: "per user count failure is an error", perUserPerDay: 20, userErr: errors.New("db down"), wantErr: true},
		{name: "site count failure is an error", perDay: 500, siteErr: errors.New("db down"), wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			userID := uuid.New()
			botRepo := repository.NewMockChatbotRepository(t)
			if tc.perUserPerDay > 0 {
				botRepo.EXPECT().CountUserInvocationsToday(mock.Anything, userID).Return(tc.userUsed, tc.userErr).Once()
			}
			if tc.perDay > 0 {
				botRepo.EXPECT().CountInvocationsToday(mock.Anything).Return(tc.siteUsed, tc.siteErr).Maybe()
			}
			botRepo.EXPECT().OldestUserInvocationToday(mock.Anything, userID).Return(time.Now().Add(-20*time.Hour), nil).Maybe()
			botRepo.EXPECT().OldestInvocationToday(mock.Anything).Return(time.Now().Add(-20*time.Hour), nil).Maybe()

			svc := &service{botRepo: botRepo}

			// when
			quota, err := svc.overQuota(context.Background(), userID, tuning{perUserPerDay: tc.perUserPerDay, perDay: tc.perDay})

			// then
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, quota.over)
			assert.Equal(t, tc.wantGlobal, quota.global)
		})
	}
}

func TestReply_IncompleteTextIsDeliveredNotBinned(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		incomplete bool
		reason     string
		wantBody   string
		wantStatus repository.InvocationStatus
	}{
		{
			name:       "a reply cut short by the token cap is still delivered",
			text:       "The culprit is not human, and the proof begins with the",
			incomplete: true,
			reason:     openai.IncompleteMaxOutputTokens,
			wantBody:   "The culprit is not human, and the proof begins with the",
			wantStatus: repository.InvocationReplied,
		},
		{
			name:       "a complete reply is unaffected",
			text:       "The culprit is not human.",
			wantBody:   "The culprit is not human.",
			wantStatus: repository.InvocationReplied,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			postID := uuid.New()
			botUserID := uuid.New()
			invocationID := uuid.New()

			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Complete(mock.Anything, mock.Anything).Return(&openai.CompletionResult{
				Text:             tc.text,
				CompletionTokens: 120,
				ReasoningTokens:  400,
				Incomplete:       tc.incomplete,
				IncompleteReason: tc.reason,
			}, nil).Once()

			botRepo := repository.NewMockChatbotRepository(t)
			botRepo.EXPECT().CreateInvocation(mock.Anything, mock.MatchedBy(func(spec repository.NewInvocation) bool {
				return spec.BotUserID == botUserID &&
					spec.RoomID == nil &&
					spec.Channel == string(SurfacePost) &&
					spec.Model == "gpt-5.6"
			})).Return(&repository.ChatbotInvocation{ID: invocationID}, nil).Once()
			botRepo.EXPECT().CompleteInvocation(mock.Anything, invocationID, mock.Anything, tc.wantStatus).Return(nil).Once()

			postSvc := post.NewMockService(t)
			postSvc.EXPECT().CreateComment(mock.Anything, postID, botUserID, mock.MatchedBy(func(req dto.CreateCommentRequest) bool {
				return req.Body == tc.wantBody
			})).Return(uuid.New(), nil).Once()

			svc := &service{openaiSvc: openaiSvc, botRepo: botRepo, postSvc: postSvc, quit: make(chan struct{})}
			j := job{
				ev:  botEvent{Surface: SurfacePost, ScopeID: postID, ItemID: postID, SenderID: uuid.New(), Body: "@beatrice who did it?"},
				bot: repository.Chatbot{UserID: botUserID, Username: "beatrice"},
			}

			// when
			out := svc.reply(context.Background(), j, tuning{}, "gpt-5.6")

			// then
			assert.Equal(t, tc.wantStatus, out.status)
			assert.Equal(t, reasonNone, out.reason, "a delivered answer needs no explanation")
		})
	}
}

func TestReply_EmptyTextBecomesAnExplainableOutcome(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		reason string
	}{
		{"the token cap left no visible text", "   ", openai.IncompleteMaxOutputTokens},
		{"a content filter stopped it", "", openai.IncompleteContentFilter},
		{"no text and no stated reason", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			postID := uuid.New()
			botUserID := uuid.New()
			invocationID := uuid.New()

			openaiSvc := openai.NewMockService(t)
			openaiSvc.EXPECT().Complete(mock.Anything, mock.Anything).Return(&openai.CompletionResult{
				Text:             tc.text,
				ReasoningTokens:  1800,
				Incomplete:       true,
				IncompleteReason: tc.reason,
			}, nil).Once()

			botRepo := repository.NewMockChatbotRepository(t)
			botRepo.EXPECT().CreateInvocation(mock.Anything, mock.MatchedBy(func(spec repository.NewInvocation) bool {
				return spec.BotUserID == botUserID &&
					spec.RoomID == nil &&
					spec.Channel == string(SurfacePost) &&
					spec.Model == "gpt-5.6"
			})).Return(&repository.ChatbotInvocation{ID: invocationID}, nil).Once()
			botRepo.EXPECT().CompleteInvocation(mock.Anything, invocationID, mock.Anything, repository.InvocationRefused).Return(nil).Once()

			postSvc := post.NewMockService(t)

			svc := &service{openaiSvc: openaiSvc, botRepo: botRepo, postSvc: postSvc, quit: make(chan struct{})}
			j := job{
				ev:  botEvent{Surface: SurfacePost, ScopeID: postID, ItemID: postID, SenderID: uuid.New(), Body: "@beatrice who did it?"},
				bot: repository.Chatbot{UserID: botUserID, Username: "beatrice"},
			}

			// when
			out := svc.reply(context.Background(), j, tuning{}, "gpt-5.6")

			// then
			assert.Equal(t, reasonEmptyReply, out.reason)
			assert.Equal(t, stagePostModel, out.stage)
			assert.Equal(t, tc.reason, out.detail, "the provider's reason must survive for the message and the log")
			postSvc.AssertNotCalled(t, "CreateComment", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestSettle_EveryOutcomeReachesTheMember(t *testing.T) {
	postID := uuid.New()
	botUserID := uuid.New()
	senderID := uuid.New()

	cases := []struct {
		name     string
		out      outcome
		wantBody string
	}{
		{"a provider outage", outcome{reason: reasonProviderDown, stage: stagePostModel}, "I cannot reach my thoughts just now. Try me again in a moment."},
		{"the model timed out", outcome{reason: reasonTimeout, stage: stagePostModel}, "I took too long thinking and ran out of time. Try me again."},
		{"no api key", outcome{reason: reasonNotConfigured, stage: stagePostModel}, "I have no voice configured at the moment. A site admin needs to set the chatbot API key before I can answer."},
		{"a database fault", outcome{reason: reasonInternal, stage: stagePreModel}, "Something went wrong on my side and I could not answer. The staff have been told."},
		{"the word filter ate the answer", outcome{reason: reasonFiltered, stage: stagePostModel}, "I wrote an answer that the word filter would not let through. Ask me another way."},
		{"an empty reply", outcome{reason: reasonEmptyReply, stage: stagePostModel, detail: openai.IncompleteContentFilter}, emptyReplyMessage(openai.IncompleteContentFilter)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			postSvc := post.NewMockService(t)
			postSvc.EXPECT().CreateComment(mock.Anything, postID, botUserID, mock.MatchedBy(func(req dto.CreateCommentRequest) bool {
				return req.Body == tc.wantBody
			})).Return(uuid.New(), nil).Once()

			svc := &service{postSvc: postSvc, quit: make(chan struct{})}
			j := job{
				ev:  botEvent{Surface: SurfacePost, ScopeID: postID, ItemID: postID, SenderID: senderID},
				bot: repository.Chatbot{UserID: botUserID, Username: "beatrice"},
			}

			// when
			svc.settle(context.Background(), j, tc.out)

			// then the expectation above is the assertion: an error outcome always speaks
		})
	}
}

func TestSettle_ErrorOutcomesAreNeverSuppressedByTheNoticeCooldown(t *testing.T) {
	// given
	postID := uuid.New()
	botUserID := uuid.New()
	senderID := uuid.New()

	postSvc := post.NewMockService(t)
	postSvc.EXPECT().CreateComment(mock.Anything, postID, botUserID, mock.Anything).Return(uuid.New(), nil).Times(3)

	svc := &service{postSvc: postSvc, quit: make(chan struct{})}
	j := job{
		ev:  botEvent{Surface: SurfacePost, ScopeID: postID, ItemID: postID, SenderID: senderID},
		bot: repository.Chatbot{UserID: botUserID, Username: "beatrice"},
	}

	// when three provider failures land back to back for the same member
	for range 3 {
		svc.settle(context.Background(), j, outcome{reason: reasonProviderDown, stage: stagePostModel})
	}

	// then all three are explained, because a broken bot must not go quiet
}

func TestStartTyping_OnlyChatFansOut(t *testing.T) {
	cases := []struct {
		name    string
		surface Surface
	}{
		{"post body trigger sends no typing", SurfacePost},
		{"post comment trigger sends no typing", SurfacePostComment},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := &service{quit: make(chan struct{})}
			ev := botEvent{Surface: tc.surface, ScopeID: uuid.New(), Audience: []uuid.UUID{uuid.New()}}

			// when
			stop := svc.startTyping(ev, uuid.New())

			// then
			require.NotNil(t, stop)
			stop()
		})
	}
}
