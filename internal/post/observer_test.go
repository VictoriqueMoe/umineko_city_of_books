package post

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type captureObserver struct {
	mu       sync.Mutex
	events   []BotContentEvent
	seen     chan struct{}
	disabled bool
}

func newCaptureObserver() *captureObserver {
	return &captureObserver{seen: make(chan struct{}, 4)}
}

func (c *captureObserver) Enabled() bool { return !c.disabled }

func (c *captureObserver) ObserveComment(ev BotContentEvent) {
	c.mu.Lock()
	c.events = append(c.events, ev)
	c.mu.Unlock()

	c.seen <- struct{}{}
}

func (c *captureObserver) captured() []BotContentEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]BotContentEvent(nil), c.events...)
}

func TestObserveForBot(t *testing.T) {
	cases := []struct {
		name             string
		body             string
		withParent       bool
		expectAuthorRead bool
		author           *model.User
		authorErr        error
		expectMentions   bool
		mentioned        []model.User
		mentionSelf      bool
		mentionErr       error
		parentAuthorSet  bool
		wantEvent        bool
		wantMentionCount int
	}{
		{
			name: "plain comment with no mention and no parent never touches the database",
			body: "just thinking out loud",
		},
		{
			name:             "bot authored comment is dropped before mentions are resolved",
			body:             "@beatrice what do you think",
			expectAuthorRead: true,
			author:           &model.User{IsBot: true},
		},
		{
			name:             "bot authored reply is dropped so bots cannot ping pong",
			body:             "thank you",
			withParent:       true,
			expectAuthorRead: true,
			author:           &model.User{IsBot: true},
		},
		{
			name:             "author lookup failure drops the trigger",
			body:             "@beatrice hi",
			expectAuthorRead: true,
			authorErr:        errors.New("db down"),
		},
		{
			name:             "human mention is emitted",
			body:             "@beatrice hi",
			expectAuthorRead: true,
			author:           &model.User{},
			expectMentions:   true,
			mentioned:        []model.User{{ID: uuid.New()}},
			wantEvent:        true,
			wantMentionCount: 1,
		},
		{
			name:             "reply with no mention is emitted with the parent author",
			body:             "and then what",
			withParent:       true,
			expectAuthorRead: true,
			author:           &model.User{},
			parentAuthorSet:  true,
			wantEvent:        true,
		},
		{
			name:             "mention that resolves to nobody with no parent is dropped",
			body:             "@nobody hello",
			expectAuthorRead: true,
			author:           &model.User{},
			expectMentions:   true,
			mentionErr:       errors.New("no such user"),
		},
		{
			name:             "self mention alone is dropped",
			body:             "@me talking to myself",
			expectAuthorRead: true,
			author:           &model.User{},
			expectMentions:   true,
			mentionSelf:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			obs := newCaptureObserver()
			svc.SetCommentObserver(obs)

			postID := uuid.New()
			commentID := uuid.New()
			authorID := uuid.New()
			parentAuthorID := uuid.New()

			var parentID *uuid.UUID
			if tc.withParent {
				parentID = new(uuid.New())
			}

			if tc.expectAuthorRead {
				author := tc.author
				if author != nil {
					author.ID = authorID
				}
				m.userRepo.EXPECT().GetByID(mock.Anything, authorID).Return(author, tc.authorErr).Once()
			}

			if tc.withParent && tc.author != nil && tc.authorErr == nil && !tc.author.IsBot {
				var row *repository.CommentRow
				if tc.parentAuthorSet {
					row = &repository.CommentRow{ID: *parentID, EntityID: postID.String(), UserID: parentAuthorID}
				}
				m.postRepo.EXPECT().GetCommentByID(mock.Anything, *parentID).Return(row, nil).Once()
			}

			if tc.expectMentions {
				mentioned := tc.mentioned
				if tc.mentionSelf {
					mentioned = []model.User{{ID: authorID}}
				}
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, mock.Anything).Return(mentioned, tc.mentionErr).Once()
			}

			// when
			svc.observeForBot(postID, new(commentID), authorID, tc.body, parentID)

			// then
			events := obs.captured()
			if !tc.wantEvent {
				assert.Empty(t, events)

				return
			}

			require.Len(t, events, 1)
			assert.Equal(t, postID, events[0].PostID)
			assert.Equal(t, commentID, *events[0].CommentID)
			assert.Equal(t, authorID, events[0].AuthorID)
			assert.Equal(t, tc.author.DisplayLabel(), events[0].AuthorName, "the bot must be told who is talking to it")
			assert.Equal(t, tc.body, events[0].Body)
			assert.Len(t, events[0].MentionedIDs, tc.wantMentionCount)

			if tc.parentAuthorSet {
				assert.Equal(t, parentAuthorID, events[0].ParentAuthor)
			}
		})
	}
}

func TestObserveForBot_ParentOnAnotherPostIsIgnored(t *testing.T) {
	// given
	svc, m := newTestService(t)
	obs := newCaptureObserver()
	svc.SetCommentObserver(obs)

	postID := uuid.New()
	otherPostID := uuid.New()
	authorID := uuid.New()
	parentID := uuid.New()
	commentID := uuid.New()

	m.userRepo.EXPECT().GetByID(mock.Anything, authorID).Return(&model.User{ID: authorID}, nil).Once()
	m.postRepo.EXPECT().GetCommentByID(mock.Anything, parentID).
		Return(&repository.CommentRow{ID: parentID, EntityID: otherPostID.String(), UserID: uuid.New()}, nil).Once()

	// when
	svc.observeForBot(postID, new(commentID), authorID, "no mention here", new(parentID))

	// then
	assert.Empty(t, obs.captured(), "a parent comment belonging to another post must not summon a bot")
}

func TestObserveForBot_DisabledObserverDoesNoDatabaseWork(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	obs := newCaptureObserver()
	obs.disabled = true
	svc.SetCommentObserver(obs)

	parentID := uuid.New()
	commentID := uuid.New()

	// when
	svc.observeForBot(uuid.New(), new(commentID), uuid.New(), "@beatrice hi", new(parentID))

	// then
	assert.Empty(t, obs.captured())
}

func TestObserveForBot_NoObserverDoesNothing(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	svc.observeForBot(uuid.New(), nil, uuid.New(), "@beatrice hi", nil)

	// then
	assert.Nil(t, svc.botObserver)
}

func TestCreateComment_EmitsBotTrigger(t *testing.T) {
	// given
	svc, m := newTestService(t)
	obs := newCaptureObserver()
	svc.SetCommentObserver(obs)

	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	botID := uuid.New()

	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil).Maybe()
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	m.postRepo.EXPECT().CreateComment(mock.Anything, mock.Anything, postID, (*uuid.UUID)(nil), userID, "@beatrice hello").Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID}, nil).Maybe()
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"beatrice"}).Return([]model.User{{ID: botID}}, nil).Maybe()
	expectBackgroundSocial(m)

	// when
	id, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "@beatrice hello"})

	// then
	require.NoError(t, err)

	select {
	case <-obs.seen:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the bot trigger")
	}

	events := obs.captured()
	require.Len(t, events, 1)
	assert.Equal(t, postID, events[0].PostID)
	assert.Equal(t, id, *events[0].CommentID)
	assert.Contains(t, events[0].MentionedIDs, botID)
}

func TestCreatePost_EmitsBotTriggerOutsideSuggestions(t *testing.T) {
	cases := []struct {
		name      string
		corner    string
		wantEvent bool
	}{
		{"general corner triggers a bot", "general", true},
		{"suggestions corner never triggers a bot", "suggestions", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			obs := newCaptureObserver()
			svc.SetCommentObserver(obs)

			userID := uuid.New()
			botID := uuid.New()

			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
			m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, tc.corner, "@beatrice hello", (*string)(nil), (*string)(nil)).Return(nil)
			m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID}, nil).Maybe()
			m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"beatrice"}).Return([]model.User{{ID: botID}}, nil).Maybe()
			expectBackgroundSocial(m)

			// when
			id, err := svc.CreatePost(context.Background(), userID, dto.CreatePostRequest{Corner: tc.corner, Body: "@beatrice hello"})

			// then
			require.NoError(t, err)

			if !tc.wantEvent {
				assert.Empty(t, obs.captured())

				return
			}

			select {
			case <-obs.seen:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for the bot trigger")
			}

			events := obs.captured()
			require.Len(t, events, 1)
			assert.Equal(t, id, events[0].PostID)
			assert.Nil(t, events[0].CommentID)
			assert.Nil(t, events[0].ParentID)
		})
	}
}
