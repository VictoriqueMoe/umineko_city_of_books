package chatbot

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestObserve_RefusedSummonIsExplainedRatherThanSilent(t *testing.T) {
	botID := uuid.New()
	senderID := uuid.New()

	event := botEvent{
		Surface:      SurfaceChat,
		ScopeID:      uuid.New(),
		ItemID:       uuid.New(),
		SenderID:     senderID,
		Body:         "@bot hello",
		MentionedIDs: mentions(botID),
	}

	cases := []struct {
		name        string
		alreadyTold bool
		wantQueue   int
	}{
		{"the first refused summon is explained", false, 1},
		{"a second refusal inside the window stays quiet", true, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			authzSvc := authz.NewMockService(t)
			authzSvc.EXPECT().Can(mock.Anything, senderID, authz.PermUseChatbot).Return(false)

			svc := &service{
				jobs:     make(chan job, 4),
				bots:     botsByID(botID),
				tune:     tuning{enabled: true, requirePermission: true},
				authzSvc: authzSvc,
			}
			svc.loaded = true
			if tc.alreadyTold {
				svc.lastRefusal.Store(senderID, time.Now())
			}

			// when
			svc.observe(event)

			// then
			assert.Len(t, svc.jobs, tc.wantQueue)

			if tc.wantQueue == 0 {
				return
			}

			queued := <-svc.jobs
			assert.True(t, queued.refusal, "the job must be marked as a refusal so no model call is made")
			assert.Equal(t, botID, queued.bot.UserID)
		})
	}
}

func TestObserve_PermittedSummonQueuesARealReply(t *testing.T) {
	// given
	botID := uuid.New()
	senderID := uuid.New()
	authzSvc := authz.NewMockService(t)
	authzSvc.EXPECT().Can(mock.Anything, senderID, authz.PermUseChatbot).Return(true)

	svc := &service{
		jobs:     make(chan job, 4),
		bots:     botsByID(botID),
		tune:     tuning{enabled: true, requirePermission: true},
		authzSvc: authzSvc,
	}
	svc.loaded = true

	// when
	svc.observe(botEvent{
		Surface:      SurfaceChat,
		ScopeID:      uuid.New(),
		ItemID:       uuid.New(),
		SenderID:     senderID,
		Body:         "@bot hello",
		MentionedIDs: mentions(botID),
	})

	// then
	require.Len(t, svc.jobs, 1)
	queued := <-svc.jobs
	assert.False(t, queued.refusal)
}

func TestHumaniseWait(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"already clear", -time.Minute, "shortly"},
		{"seconds away", 30 * time.Second, "in less than a minute"},
		{"a single minute", 70 * time.Second, "in about a minute"},
		{"some minutes", 20 * time.Minute, "in about 20 minutes"},
		{"just under an hour", 59 * time.Minute, "in about 59 minutes"},
		{"an hour", 62 * time.Minute, "in about an hour"},
		{"several hours", 3*time.Hour + 10*time.Minute, "in about 3 hours"},
		{"most of a day", 23 * time.Hour, "in about 23 hours"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given the duration from the table

			// when
			got := humaniseWait(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestQuotaMessage_SaysWhoseLimitAndWhenItClears(t *testing.T) {
	cases := []struct {
		name        string
		state       quotaState
		wantPhrases []string
	}{
		{
			name:        "a member out of their own allowance",
			state:       quotaState{over: true, clearsAt: time.Now().Add(2 * time.Hour)},
			wantPhrases: []string{"your message limit", "in about 2 hours"},
		},
		{
			name:        "the whole site out of allowance",
			state:       quotaState{over: true, global: true, clearsAt: time.Now().Add(30 * time.Minute)},
			wantPhrases: []string{"whole site", "in about 30 minutes"},
		},
		{
			name:        "an unknown clearing time still says something useful",
			state:       quotaState{over: true},
			wantPhrases: []string{"shortly"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := new(service)

			// when
			got := svc.quotaMessage(tc.state)

			// then
			for _, phrase := range tc.wantPhrases {
				assert.Contains(t, got, phrase)
			}
		})
	}
}

func TestQuotaClearsAt(t *testing.T) {
	// given
	oldest := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)

	// when
	got := quotaClearsAt(oldest)

	// then
	assert.Equal(t, oldest.Add(24*time.Hour), got)
	assert.True(t, quotaClearsAt(time.Time{}).IsZero(), "an unknown oldest invocation must not invent a clearing time")
}

func TestRefusalMessage_PointsAtTheSettingsPage(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{"a plain base url", "https://whentheycry.social", "https://whentheycry.social/settings"},
		{"a trailing slash is not doubled", "https://whentheycry.social/", "https://whentheycry.social/settings"},
		{"surrounding whitespace is trimmed", "  https://whentheycry.social  ", "https://whentheycry.social/settings"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return(tc.baseURL)
			svc := &service{settingsSvc: settingsSvc}

			// when
			got := svc.refusalMessage(context.Background())

			// then
			assert.Contains(t, got, tc.want)
			assert.NotContains(t, got, "//settings")
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
