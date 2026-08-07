package chatbot

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRowToMessage_CarriesReplyContext(t *testing.T) {
	botID := uuid.New()

	cases := []struct {
		name string
		row  promptRow
		want string
	}{
		{
			name: "plain message is unchanged",
			row:  promptRow{AuthorID: uuid.New(), DisplayName: "Battler", Body: "hello"},
			want: "Battler: hello",
		},
		{
			name: "reply names the target and quotes it",
			row: promptRow{
				AuthorID:    uuid.New(),
				DisplayName: "Battler",
				Body:        "what did you mean?",
				ReplyToName: "Beatrice",
				ReplyToBody: "the golden truth",
			},
			want: `Battler (replying to Beatrice: "the golden truth"): what did you mean?`,
		},
		{
			name: "reply with an unknown author still quotes it",
			row: promptRow{
				AuthorID:    uuid.New(),
				DisplayName: "Battler",
				Body:        "no",
				ReplyToBody: "the golden truth",
			},
			want: `Battler (replying to "the golden truth"): no`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := rowToMessage(tc.row, botID, messageBodyMax)

			// then
			assert.Equal(t, "user", got.Role)
			assert.Equal(t, tc.want, got.Content)
		})
	}
}

func TestRowToMessage_BotOwnMessagesStayClean(t *testing.T) {
	// given
	botID := uuid.New()
	row := promptRow{AuthorID: botID, DisplayName: "Beatrice", Body: "ufufu", ReplyToName: "Battler", ReplyToBody: "prove it"}

	// when
	got := rowToMessage(row, botID, messageBodyMax)

	// then
	assert.Equal(t, "assistant", got.Role)
	assert.Equal(t, "ufufu", got.Content)
}

func TestFitBudget_DropsWholeOldMessagesAndAlwaysKeepsTheTrigger(t *testing.T) {
	big := strings.Repeat("x", promptCharBudget)

	cases := []struct {
		name     string
		messages []openai.Message
		wantLen  int
		wantLast string
	}{
		{
			name:     "under budget is untouched",
			messages: []openai.Message{{Content: "one"}, {Content: "two"}, {Content: "three"}},
			wantLen:  3,
			wantLast: "three",
		},
		{
			name:     "oldest are dropped whole, never truncated",
			messages: []openai.Message{{Content: big}, {Content: big}, {Content: "trigger"}},
			wantLen:  1,
			wantLast: "trigger",
		},
		{
			name:     "an oversized trigger alone is still sent",
			messages: []openai.Message{{Content: big}},
			wantLen:  1,
			wantLast: big,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := fitBudget(tc.messages)

			// then
			assert.Len(t, got, tc.wantLen)
			assert.Equal(t, tc.wantLast, got[len(got)-1].Content)
			for i := range got {
				assert.NotContains(t, got[i].Content, "...", "fitBudget must drop messages, never truncate them")
			}
		})
	}
}

func TestDMHistory_ReadsOnlyWhatTheMemberCanSee(t *testing.T) {
	// given
	roomID := uuid.New()
	senderID := uuid.New()
	botID := uuid.New()

	chatRepo := repository.NewMockChatRepository(t)
	chatRepo.EXPECT().GetMessagesForMember(mock.Anything, roomID, senderID, 20).
		Return([]repository.ChatMessageRow{
			{SenderID: senderID, SenderDisplayName: "Kujo", Body: "hello again"},
		}, nil).Once()

	svc := &service{chatRepo: chatRepo}
	ev := botEvent{Surface: SurfaceChat, IsDM: true, ScopeID: roomID, SenderID: senderID, Body: "hello again"}

	// when
	got := svc.dmHistory(context.Background(), ev, botID, tuning{contextMessages: 20})

	// then
	require.Len(t, got, 1)
	assert.Equal(t, "Kujo: hello again", got[0].Content)
	chatRepo.AssertNotCalled(t, "GetMessages", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
