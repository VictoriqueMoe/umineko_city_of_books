package chatbot

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildMessages_GameBoardChain(t *testing.T) {
	postID := uuid.New()
	otherPostID := uuid.New()
	botID := uuid.New()
	humanID := uuid.New()
	triggerID := uuid.New()
	first := uuid.New()
	second := uuid.New()

	firstRow := &repository.CommentRow{
		ID:                first,
		EntityID:          postID.String(),
		UserID:            humanID,
		Body:              "who did it",
		AuthorDisplayName: "Kujo",
		AuthorUsername:    "kujo",
	}
	secondRow := &repository.CommentRow{
		ID:       second,
		EntityID: postID.String(),
		UserID:   botID,
		Body:     "the culprit is not human",
		ParentID: &first,
	}

	cases := []struct {
		name  string
		depth int
		rows  map[uuid.UUID]*repository.CommentRow
		want  []openai.Message
	}{
		{
			name:  "chain is ordered oldest first with the trigger last",
			depth: 25,
			rows:  map[uuid.UUID]*repository.CommentRow{first: firstRow, second: secondRow},
			want: []openai.Message{
				{Role: "user", Content: "Kujo: who did it"},
				{Role: "assistant", Content: "the culprit is not human"},
				{Role: "user", Content: "explain"},
			},
		},
		{
			name:  "depth caps how far the walk climbs",
			depth: 1,
			rows:  map[uuid.UUID]*repository.CommentRow{first: firstRow, second: secondRow},
			want: []openai.Message{
				{Role: "assistant", Content: "the culprit is not human"},
				{Role: "user", Content: "explain"},
			},
		},
		{
			name:  "a parent on another post ends the walk",
			depth: 25,
			rows: map[uuid.UUID]*repository.CommentRow{
				second: {ID: second, EntityID: otherPostID.String(), UserID: botID, Body: "stray"},
			},
			want: []openai.Message{{Role: "user", Content: "explain"}},
		},
		{
			name:  "a missing parent ends the walk",
			depth: 25,
			rows:  map[uuid.UUID]*repository.CommentRow{second: nil},
			want:  []openai.Message{{Role: "user", Content: "explain"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			postRepo := repository.NewMockPostRepository(t)
			for id, row := range tc.rows {
				postRepo.EXPECT().GetCommentByID(mock.Anything, id).Return(row, nil).Maybe()
			}

			svc := &service{postRepo: postRepo}
			j := job{
				ev: botEvent{
					Surface:  SurfacePostComment,
					ScopeID:  postID,
					ItemID:   triggerID,
					Body:     "explain",
					ParentID: &second,
				},
				bot:      repository.Chatbot{UserID: botID},
				useChain: true,
			}

			// when
			got := svc.buildMessages(context.Background(), j, tuning{maxReplyChain: tc.depth})

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildMessages_MentionOnlySendsOneMessage(t *testing.T) {
	// given
	parentID := uuid.New()

	cases := []struct {
		name    string
		surface Surface
		parent  *uuid.UUID
	}{
		{"post body mention", SurfacePost, nil},
		{"post comment mention with a human parent", SurfacePostComment, &parentID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := &service{postRepo: repository.NewMockPostRepository(t)}
			j := job{
				ev: botEvent{
					Surface:  tc.surface,
					ScopeID:  uuid.New(),
					Body:     "@beatrice hello",
					ParentID: tc.parent,
				},
				bot:      repository.Chatbot{UserID: uuid.New()},
				useChain: false,
			}

			// when
			got := svc.buildMessages(context.Background(), j, tuning{maxReplyChain: 25, contextMessages: 20})

			// then
			assert.Equal(t, []openai.Message{{Role: "user", Content: "@beatrice hello"}}, got)
		})
	}
}

func TestBuildMessages_NamesWhoeverTriggeredTheBot(t *testing.T) {
	cases := []struct {
		name       string
		surface    Surface
		senderName string
		want       string
	}{
		{"a room mention carries the sender name", SurfaceChat, "Featherine Augustus Aurora", "Featherine Augustus Aurora: @beatrice who am i?"},
		{"a room nickname is used when the sender has one", SurfaceChat, "Feather", "Feather: @beatrice who am i?"},
		{"a game board mention carries the author name", SurfacePost, "Kujo", "Kujo: @beatrice who am i?"},
		{"an unknown name falls back to the bare body", SurfaceChat, "", "@beatrice who am i?"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := &service{postRepo: repository.NewMockPostRepository(t)}
			j := job{
				ev: botEvent{
					Surface:    tc.surface,
					ScopeID:    uuid.New(),
					SenderName: tc.senderName,
					Body:       "@beatrice who am i?",
				},
				bot:      repository.Chatbot{UserID: uuid.New()},
				useChain: false,
			}

			// when
			got := svc.buildMessages(context.Background(), j, tuning{maxReplyChain: 25, contextMessages: 20})

			// then
			assert.Equal(t, []openai.Message{{Role: "user", Content: tc.want}}, got)
		})
	}
}

func TestBuildMessages_ReplyChainStillNamesTheTrigger(t *testing.T) {
	// given
	roomID := uuid.New()
	botID := uuid.New()
	parentID := uuid.New()

	chatRepo := repository.NewMockChatRepository(t)
	chatRepo.EXPECT().GetMessageByID(mock.Anything, parentID).Return(&repository.ChatMessageRow{
		ID:       parentID,
		RoomID:   roomID,
		SenderID: botID,
		Body:     "the culprit is not human",
	}, nil)

	svc := &service{chatRepo: chatRepo}
	j := job{
		ev: botEvent{
			Surface:    SurfaceChat,
			ScopeID:    roomID,
			SenderName: "Feather",
			Body:       "explain",
			ParentID:   &parentID,
		},
		bot:      repository.Chatbot{UserID: botID},
		useChain: true,
	}

	// when
	got := svc.buildMessages(context.Background(), j, tuning{maxReplyChain: 25})

	// then
	assert.Equal(t, []openai.Message{
		{Role: "assistant", Content: "the culprit is not human"},
		{Role: "user", Content: "Feather: explain"},
	}, got)
}

func TestChatPromptRow_PrefersTheRoomNickname(t *testing.T) {
	cases := []struct {
		name     string
		nickname string
		display  string
		username string
		want     string
	}{
		{"a nickname wins over the display name", "Feather", "Featherine Augustus Aurora", "featherine", "Feather"},
		{"a blank nickname falls back to the display name", "", "Featherine Augustus Aurora", "featherine", "Featherine Augustus Aurora"},
		{"a whitespace nickname falls back to the display name", "   ", "Featherine Augustus Aurora", "featherine", "Featherine Augustus Aurora"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			row := repository.ChatMessageRow{
				SenderNickname:    tc.nickname,
				SenderDisplayName: tc.display,
				SenderUsername:    tc.username,
			}

			// when
			got := chatPromptRow(row)

			// then
			assert.Equal(t, tc.want, got.DisplayName)
		})
	}
}
