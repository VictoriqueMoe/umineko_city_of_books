package chatbot

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func botsByID(ids ...uuid.UUID) map[uuid.UUID]model.Chatbot {
	out := make(map[uuid.UUID]model.Chatbot, len(ids))
	for _, id := range ids {
		out[id] = model.Chatbot{UserID: id, Username: id.String()}
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
		cache    *cache.Manager
		window   time.Duration
		previous bool
		want     bool
	}{
		{"no window always allows", cache.NewManager(engines.NewInMemory(0)), 0, true, true},
		{"first summon is allowed", cache.NewManager(engines.NewInMemory(0)), time.Minute, false, true},
		{"second summon inside the window is refused", cache.NewManager(engines.NewInMemory(0)), time.Minute, true, false},
		{"an unreachable coordination store lets the summon through", cache.NewManager(), time.Minute, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			svc := &service{cache: tc.cache}
			userID := uuid.New()
			if tc.previous {
				svc.takeCooldown(ctx, userID, time.Minute)
			}

			// when
			got := svc.takeCooldown(ctx, userID, tc.window)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTakeCooldown_IsPerMember(t *testing.T) {
	// given
	ctx := context.Background()
	svc := &service{cache: cache.NewManager(engines.NewInMemory(0))}
	svc.takeCooldown(ctx, uuid.New(), time.Minute)

	// when
	got := svc.takeCooldown(ctx, uuid.New(), time.Minute)

	// then
	assert.True(t, got, "one member's cooldown must not hold back another")
}

func TestTakeNoticeSlot(t *testing.T) {
	userID := uuid.New()

	cases := []struct {
		name      string
		cache     *cache.Manager
		previous  Reason
		requested Reason
		want      bool
	}{
		{"the first notice is sent", cache.NewManager(engines.NewInMemory(0)), "", reasonNotPermitted, true},
		{"a repeat of the same reason inside the window is suppressed", cache.NewManager(engines.NewInMemory(0)), reasonNotPermitted, reasonNotPermitted, false},
		{"a different reason is still explained", cache.NewManager(engines.NewInMemory(0)), reasonNotPermitted, reasonQuotaUser, true},
		{"an unreachable coordination store still explains", cache.NewManager(), reasonNotPermitted, reasonNotPermitted, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			svc := &service{cache: tc.cache}
			if tc.previous != "" {
				svc.takeNoticeSlot(ctx, userID, tc.previous)
			}

			// when
			got := svc.takeNoticeSlot(ctx, userID, tc.requested)

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
		cache       *cache.Manager
		alreadyTold bool
		wantSends   int
	}{
		{"the first refused summon is explained", cache.NewManager(engines.NewInMemory(0)), false, 1},
		{"a second refusal inside the window stays quiet", cache.NewManager(engines.NewInMemory(0)), true, 0},
		{"an unreachable coordination store still explains the refusal", cache.NewManager(), true, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			authzSvc := authz.NewMockService(t)
			authzSvc.EXPECT().Can(mock.Anything, senderID, authz.PermUseChatbot).Return(false)

			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://whentheycry.social").Maybe()

			chatSvc := chat.NewMockService(t)
			sends := 0
			chatSvc.EXPECT().SendMessage(mock.Anything, botID, event.ScopeID, mock.Anything, mock.Anything).
				Run(func(context.Context, uuid.UUID, uuid.UUID, dto.SendMessageRequest, []upload.FileUpload) {
					sends++
				}).Return(nil, nil).Maybe()

			svc := &service{
				jobs:        make(chan job, 4),
				bots:        botsByID(botID),
				tune:        tuning{enabled: true, requirePermission: true},
				authzSvc:    authzSvc,
				chatSvc:     chatSvc,
				settingsSvc: settingsSvc,
				cache:       tc.cache,
			}
			svc.loaded = true
			if tc.alreadyTold {
				svc.takeNoticeSlot(context.Background(), senderID, reasonNotPermitted)
			}

			// when
			svc.observe(event)

			// then
			assert.Equal(t, tc.wantSends, sends, "the refusal is delivered directly, never through the reply queue")
			assert.Empty(t, svc.jobs, "a notice must not consume a worker slot")
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
		cache:    cache.NewManager(engines.NewInMemory(0)),
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
	assert.Equal(t, senderID, queued.ev.SenderID)
}

func TestObserve_CooldownAppliesEverywhereIncludingDMs(t *testing.T) {
	botID := uuid.New()
	senderID := uuid.New()
	scopeID := uuid.New()

	cases := []struct {
		name           string
		event          botEvent
		summonedBefore bool
		wantQueue      int
	}{
		{
			name: "a DM inside the cooldown window is held back, because a paid model is on the other end",
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
			summonedBefore: true,
			wantQueue:      0,
		},
		{
			name: "a DM once the cooldown window has lapsed is queued",
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
			summonedBefore: false,
			wantQueue:      1,
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
			summonedBefore: true,
			wantQueue:      0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := &service{
				jobs:  make(chan job, 4),
				bots:  botsByID(botID),
				tune:  tuning{enabled: true, cooldown: time.Hour},
				cache: cache.NewManager(engines.NewInMemory(0)),
			}
			svc.loaded = true
			if tc.summonedBefore {
				svc.takeCooldown(context.Background(), senderID, time.Hour)
			}

			// when
			svc.observe(tc.event)

			// then
			assert.Len(t, svc.jobs, tc.wantQueue)
		})
	}
}

func TestObserve_SummonsInTheSameRoomAreAllQueued(t *testing.T) {
	// given two members summoning in one room before either reply is written
	botID := uuid.New()
	scopeID := uuid.New()
	svc := &service{
		jobs:  make(chan job, 4),
		bots:  botsByID(botID),
		tune:  tuning{enabled: true},
		cache: cache.NewManager(engines.NewInMemory(0)),
	}
	svc.loaded = true

	// when
	for range 2 {
		svc.observe(botEvent{
			Surface:      SurfaceChat,
			ScopeID:      scopeID,
			ItemID:       uuid.New(),
			SenderID:     uuid.New(),
			Body:         "@bot hello",
			MentionedIDs: mentions(botID),
		})
	}

	// then neither summon is dropped
	assert.Len(t, svc.jobs, 2)
}

func TestAbandonQueued_DrainsEveryQueuedJob(t *testing.T) {
	// given
	svc := &service{jobs: make(chan job, 4)}
	svc.jobs <- job{ev: botEvent{Surface: SurfaceChat, ScopeID: uuid.New()}}
	svc.jobs <- job{ev: botEvent{Surface: SurfacePost, ScopeID: uuid.New()}}

	// when
	abandoned := svc.abandonQueued()

	// then
	assert.Equal(t, 2, abandoned)
	assert.Empty(t, svc.jobs)
}
