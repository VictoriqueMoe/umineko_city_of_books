package chatbot

import (
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func botsByID(ids ...uuid.UUID) map[uuid.UUID]repository.Chatbot {
	out := make(map[uuid.UUID]repository.Chatbot, len(ids))
	for _, id := range ids {
		out[id] = repository.Chatbot{UserID: id, Username: id.String()}
	}

	return out
}

func mentions(ids ...uuid.UUID) map[uuid.UUID]struct{} {
	out := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}

	return out
}

func TestSelectBot_PrecedenceAcrossSurfaces(t *testing.T) {
	botA := uuid.New()
	botB := uuid.New()
	human := uuid.New()
	sender := uuid.New()
	parentID := uuid.New()

	cases := []struct {
		name       string
		ev         botEvent
		wantBot    uuid.UUID
		wantThread bool
	}{
		{
			name: "post comment replying to a bot beats a mention of another bot",
			ev: botEvent{
				Surface:      SurfacePostComment,
				SenderID:     sender,
				ParentID:     &parentID,
				ParentAuthor: botA,
				MentionedIDs: mentions(botB),
			},
			wantBot:    botA,
			wantThread: true,
		},
		{
			name: "post comment mentioning a bot picks it without threading",
			ev: botEvent{
				Surface:      SurfacePostComment,
				SenderID:     sender,
				ParentID:     &parentID,
				ParentAuthor: human,
				MentionedIDs: mentions(botB),
			},
			wantBot: botB,
		},
		{
			name: "post body mentioning a bot picks it",
			ev: botEvent{
				Surface:      SurfacePost,
				SenderID:     sender,
				MentionedIDs: mentions(botA),
			},
			wantBot: botA,
		},
		{
			name: "post comment replying to a human with no bot mention selects nothing",
			ev: botEvent{
				Surface:      SurfacePostComment,
				SenderID:     sender,
				ParentID:     &parentID,
				ParentAuthor: human,
				MentionedIDs: mentions(human),
			},
		},
		{
			name: "chat reply to a bot still threads",
			ev: botEvent{
				Surface:      SurfaceChat,
				SenderID:     sender,
				ParentID:     &parentID,
				ParentAuthor: botB,
			},
			wantBot:    botB,
			wantThread: true,
		},
		{
			name: "dm picks the bot member that is not the sender",
			ev: botEvent{
				Surface:  SurfaceChat,
				IsDM:     true,
				SenderID: sender,
				Audience: []uuid.UUID{sender, botA},
			},
			wantBot: botA,
		},
		{
			name: "dm without a bot member never falls through to mentions",
			ev: botEvent{
				Surface:      SurfaceChat,
				IsDM:         true,
				SenderID:     sender,
				Audience:     []uuid.UUID{sender, human},
				MentionedIDs: mentions(botA),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			bots := botsByID(botA, botB)

			// when
			bot, threaded, ok := selectBot(tc.ev, bots)

			// then
			if tc.wantBot == uuid.Nil {
				assert.False(t, ok)

				return
			}

			assert.True(t, ok)
			assert.Equal(t, tc.wantBot, bot.UserID)
			assert.Equal(t, tc.wantThread, threaded)
		})
	}
}

func TestUseChain(t *testing.T) {
	parentID := uuid.New()

	cases := []struct {
		name      string
		surface   Surface
		hasParent bool
		threaded  bool
		want      bool
	}{
		{"chat reply to a human still sends the chain", SurfaceChat, true, false, true},
		{"chat reply to a bot sends the chain", SurfaceChat, true, true, true},
		{"chat mention with no reply sends one message", SurfaceChat, false, false, false},
		{"post comment replying to a bot sends the chain", SurfacePostComment, true, true, true},
		{"post comment replying to a human sends one message", SurfacePostComment, true, false, false},
		{"post body mention sends one message", SurfacePost, false, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ev := botEvent{Surface: tc.surface}
			if tc.hasParent {
				ev.ParentID = &parentID
			}

			// when
			got := useChain(ev, tc.threaded)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTakeCooldown(t *testing.T) {
	cases := []struct {
		name     string
		window   time.Duration
		previous bool
		want     bool
	}{
		{"no window always allows", 0, true, true},
		{"first summon is allowed", time.Minute, false, true},
		{"second summon inside the window is refused", time.Minute, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := new(service)
			userID := uuid.New()
			if tc.previous {
				svc.lastUse.Store(userID, time.Now())
			}

			// when
			got := svc.takeCooldown(userID, tc.window)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAllowedToSummon(t *testing.T) {
	cases := []struct {
		name       string
		require    bool
		canChatbot bool
		want       bool
	}{
		{"gate off lets anyone summon", false, false, true},
		{"gate on and permission granted", true, true, true},
		{"gate on and permission missing", true, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			authzSvc := authz.NewMockService(t)
			userID := uuid.New()
			if tc.require {
				authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermUseChatbot).Return(tc.canChatbot)
			}
			svc := &service{authzSvc: authzSvc}

			// when
			got := svc.allowedToSummon(userID, tuning{requirePermission: tc.require})

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestObserve_CooldownAppliesToSharedSpacesButNotDMs(t *testing.T) {
	botID := uuid.New()
	senderID := uuid.New()
	scopeID := uuid.New()

	cases := []struct {
		name      string
		event     botEvent
		wantQueue int
	}{
		{
			name: "a DM is never held back by the cooldown",
			event: botEvent{
				Surface:      SurfaceChat,
				IsDM:         true,
				ScopeID:      scopeID,
				ItemID:       uuid.New(),
				SenderID:     senderID,
				Body:         "again",
				Audience:     []uuid.UUID{senderID, botID},
				MentionedIDs: mentions(botID),
			},
			wantQueue: 1,
		},
		{
			name: "a room mention still honours the cooldown",
			event: botEvent{
				Surface:      SurfaceChat,
				ScopeID:      scopeID,
				ItemID:       uuid.New(),
				SenderID:     senderID,
				Body:         "@bot again",
				MentionedIDs: mentions(botID),
			},
			wantQueue: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := &service{
				jobs: make(chan job, 4),
				bots: botsByID(botID),
				tune: tuning{enabled: true, cooldown: time.Hour},
			}
			svc.loaded = true
			svc.lastUse.Store(senderID, time.Now())

			// when
			svc.observe(tc.event)

			// then
			assert.Len(t, svc.jobs, tc.wantQueue)
		})
	}
}
