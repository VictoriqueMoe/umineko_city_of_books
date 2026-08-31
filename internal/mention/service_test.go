package mention

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const emailHTTPTimeout = 30 * time.Second

type serviceMocks struct {
	userRepo     *repository.MockUserRepository
	blockSvc     *block.MockService
	notifSvc     *notification.MockService
	shipComments *repository.MockCommentDAO[uuid.UUID]
}

var (
	testActorID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testAliceID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testBobID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testCarolID = uuid.MustParse("66666666-6666-6666-6666-666666666666")

	errCreateFailed = errors.New("create failed")
)

func newTestService(t *testing.T) (Service, serviceMocks) {
	t.Helper()

	m := serviceMocks{
		userRepo:     repository.NewMockUserRepository(t),
		blockSvc:     block.NewMockService(t),
		notifSvc:     notification.NewMockService(t),
		shipComments: repository.NewMockCommentDAO[uuid.UUID](t),
	}

	comments := repository.CommentDAOs{
		ByID: map[string]repository.CommentDAO[uuid.UUID]{string(KindShipComment): m.shipComments},
	}

	return NewService(m.userRepo, m.blockSvc, m.notifSvc, comments), m
}

func testArtComment() Reference {
	return Reference{Kind: KindArtComment, EntityID: testEntityID, ChildID: testChildID}
}

func TestServiceNotifyGuards(t *testing.T) {
	human := &model.User{ID: testActorID, Username: "kujo", DisplayName: "Kujo Kazuya"}
	bot := &model.User{ID: testActorID, Username: "bern", DisplayName: "Bernkastel", IsBot: true}

	tests := []struct {
		name         string
		actor        *model.User
		body         string
		wantLookup   []string
		resolved     []model.User
		blocked      map[uuid.UUID]bool
		wantNotified []uuid.UUID
	}{
		{
			name:  "a bot actor never fans out",
			actor: bot,
			body:  "@alice look at this",
		},
		{
			name:  "a missing actor never fans out",
			actor: nil,
			body:  "@alice look at this",
		},
		{
			name:         "the actor's own username is skipped",
			actor:        human,
			body:         "@kujo and @alice",
			wantLookup:   []string{"kujo", "alice"},
			resolved:     []model.User{{ID: testActorID, Username: "kujo"}, {ID: testAliceID, Username: "alice"}},
			wantNotified: []uuid.UUID{testAliceID},
		},
		{
			name:         "a blocked pair is skipped in either direction",
			actor:        human,
			body:         "@alice and @bob",
			wantLookup:   []string{"alice", "bob"},
			resolved:     []model.User{{ID: testAliceID, Username: "alice"}, {ID: testBobID, Username: "bob"}},
			blocked:      map[uuid.UUID]bool{testAliceID: true},
			wantNotified: []uuid.UUID{testBobID},
		},
		{
			name:         "a repeated username is notified once",
			actor:        human,
			body:         "@alice @alice @alice",
			wantLookup:   []string{"alice"},
			resolved:     []model.User{{ID: testAliceID, Username: "alice"}},
			wantNotified: []uuid.UUID{testAliceID},
		},
		{
			name:         "an unknown username is dropped",
			actor:        human,
			body:         "@alice and @nobodyhere",
			wantLookup:   []string{"alice", "nobodyhere"},
			resolved:     []model.User{{ID: testAliceID, Username: "alice"}},
			wantNotified: []uuid.UUID{testAliceID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			m.userRepo.EXPECT().GetByID(mock.Anything, testActorID).Return(tt.actor, nil)

			if tt.wantLookup != nil {
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, tt.wantLookup).Return(tt.resolved, nil).Once()
			}

			for _, u := range tt.resolved {
				if u.ID == testActorID {
					continue
				}

				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, testActorID, u.ID).Return(tt.blocked[u.ID], nil)
			}

			var notified []uuid.UUID
			for _, id := range tt.wantNotified {
				m.notifSvc.EXPECT().
					Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool { return p.RecipientID == id })).
					Run(func(_ context.Context, p dto.NotifyParams) { notified = append(notified, p.RecipientID) }).
					Return(nil)
			}

			// when
			err := svc.Notify(t.Context(), testArtComment(), testActorID, tt.body)

			// then
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantNotified, notified)
		})
	}
}

func TestServiceNotifyResolvesInASingleQuery(t *testing.T) {
	// given
	svc, m := newTestService(t)

	m.userRepo.EXPECT().GetByID(mock.Anything, testActorID).
		Return(&model.User{ID: testActorID, DisplayName: "Kujo Kazuya"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice", "bob", "carol"}).
		Return([]model.User{{ID: testAliceID}, {ID: testBobID}, {ID: testCarolID}}, nil).
		Once()
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, testActorID, mock.Anything).Return(false, nil)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil)

	// when
	err := svc.Notify(t.Context(), testArtComment(), testActorID, "@alice @bob @carol")

	// then
	require.NoError(t, err)
	m.userRepo.AssertNumberOfCalls(t, "GetByUsernames", 1)
	m.userRepo.AssertNotCalled(t, "GetByUsername")
	m.notifSvc.AssertNumberOfCalls(t, "Notify", 3)
}

func TestServiceNotifyCapsUsernamesAtTwenty(t *testing.T) {
	// given
	svc, m := newTestService(t)

	var body strings.Builder
	want := make([]string, 0, MaxMatches)
	for i := range 25 {
		body.WriteString("@user" + strconv.Itoa(i) + " ")
		if i < MaxMatches {
			want = append(want, "user"+strconv.Itoa(i))
		}
	}

	m.userRepo.EXPECT().GetByID(mock.Anything, testActorID).
		Return(&model.User{ID: testActorID, DisplayName: "Kujo Kazuya"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, want).Return(nil, nil).Once()

	// when
	err := svc.Notify(t.Context(), testArtComment(), testActorID, body.String())

	// then
	require.NoError(t, err)
	m.userRepo.AssertNumberOfCalls(t, "GetByUsernames", 1)
}

func TestServiceNotifyCarriesTheRegistryReference(t *testing.T) {
	// given
	svc, m := newTestService(t)

	m.userRepo.EXPECT().GetByID(mock.Anything, testActorID).
		Return(&model.User{ID: testActorID, DisplayName: "Kujo Kazuya"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).
		Return([]model.User{{ID: testAliceID, Username: "alice"}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, testActorID, testAliceID).Return(false, nil)

	var got dto.NotifyParams
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		Run(func(_ context.Context, p dto.NotifyParams) { got = p }).
		Return(nil)

	ref := Reference{Kind: KindSecretComment, EntityKey: testEntityKey, ChildID: testChildID}

	// when
	err := svc.Notify(t.Context(), ref, testActorID, "@alice")

	// then
	require.NoError(t, err)
	assert.Equal(t, dto.NotifMention, got.Type)
	assert.Equal(t, uuid.Nil, got.ReferenceID)
	assert.Equal(t, "secret_comment:gohda-recipe:55555555-5555-5555-5555-555555555555", got.ReferenceType)
	assert.Equal(t, "/secrets/gohda-recipe#comment-55555555-5555-5555-5555-555555555555", got.EmailLink)
	assert.Equal(t, "Kujo Kazuya", got.EmailActor)
	assert.Equal(t, "mentioned you", got.EmailAction)
	assert.Equal(t, testActorID, got.ActorID)
}

func TestServiceNotifyRejectsAnUnregisteredKind(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.Notify(t.Context(), Reference{Kind: Kind("ship_wheel"), EntityID: testEntityID}, testActorID, "@alice")

	// then
	require.ErrorIs(t, err, ErrUnknownKind)
}

func TestServiceCreateComment(t *testing.T) {
	tests := []struct {
		name       string
		createID   uuid.UUID
		createErr  error
		wantID     uuid.UUID
		wantFanOut bool
	}{
		{
			name:       "a created comment fans its mentions out",
			createID:   testChildID,
			wantID:     testChildID,
			wantFanOut: true,
		},
		{
			name:      "a failed create fans nothing out",
			createErr: errCreateFailed,
			wantID:    uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			notified := make(chan dto.NotifyParams, 1)

			if tt.wantFanOut {
				m.userRepo.EXPECT().GetByID(mock.Anything, testActorID).
					Return(&model.User{ID: testActorID, DisplayName: "Kujo Kazuya"}, nil)
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).
					Return([]model.User{{ID: testAliceID, Username: "alice"}}, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, testActorID, testAliceID).Return(false, nil)
				m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
					RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
						notified <- p
						return nil
					})
			}

			spec := CommentSpec{
				Kind:     KindShipComment,
				EntityID: testEntityID,
				AuthorID: testActorID,
				Body:     "nice one @alice",
			}

			var row *repository.CommentRow
			if tt.createErr == nil {
				row = &repository.CommentRow{ID: tt.createID}
			}

			m.shipComments.EXPECT().
				CreateComment(mock.Anything, testEntityID, (*uuid.UUID)(nil), testActorID, "nice one @alice").
				Return(row, tt.createErr).
				Once()

			// when
			got, err := svc.CreateComment(t.Context(), spec)

			// then
			assert.Equal(t, tt.wantID, got)

			if tt.createErr != nil {
				require.ErrorIs(t, err, tt.createErr)
				return
			}

			require.NoError(t, err)

			select {
			case p := <-notified:
				assert.Equal(t, testAliceID, p.RecipientID)
				assert.Equal(t, "ship_comment:"+testChildID.String(), p.ReferenceType)
				assert.Equal(t, "/ships/"+testEntityID.String()+"#comment-"+testChildID.String(), p.EmailLink)
			case <-time.After(5 * time.Second):
				t.Fatal("mention fan-out did not reach the notification service")
			}
		})
	}
}

func TestServiceCreateCommentWithoutARegisteredDAO(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
	}{
		{
			name: "an unregistered entity kind is reported",
			kind: KindArtComment,
		},
		{
			name: "an unregistered journal kind is reported",
			kind: KindJournalComment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			got, err := svc.CreateComment(t.Context(), CommentSpec{
				Kind:     tt.kind,
				EntityID: testEntityID,
				AuthorID: testActorID,
				Body:     "hello @alice",
			})

			// then
			require.ErrorIs(t, err, ErrNoCommentDAO)
			assert.Equal(t, uuid.Nil, got)
		})
	}
}

func TestFanOutBudgetScalesWithRecipientCount(t *testing.T) {
	// given a detached fan-out whose deadline the goroutine can report back
	svc, m := newTestService(t)

	budgets := make(chan time.Duration, 1)
	m.userRepo.EXPECT().GetByID(mock.Anything, testActorID).
		RunAndReturn(func(ctx context.Context, _ uuid.UUID, _ ...*sql.Tx) (*model.User, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				budgets <- 0
				return nil, nil
			}

			budgets <- time.Until(deadline)

			return nil, nil
		})

	// when the fan-out is detached from the caller
	svc.NotifyAsync(t.Context(), testArtComment(), testActorID, "@alice @bob @carol")

	// then its budget covers every recipient the parser can yield, not one deadline shared across them all
	select {
	case budget := <-budgets:
		assert.GreaterOrEqual(t, budget, MaxMatches*emailHTTPTimeout,
			"fan-out budget must scale with the recipient cap, not be a single shared deadline")
		assert.Greater(t, recipientBudget, emailHTTPTimeout,
			"one recipient's budget must outlive the email client's own http timeout")
		assert.Equal(t, MaxMatches*recipientBudget, fanOutTimeout,
			"the fan-out budget must stay the per-recipient budget times the recipient cap")
	case <-time.After(5 * time.Second):
		t.Fatal("mention fan-out never started")
	}
}

func TestValidateCommentDAOs(t *testing.T) {
	complete := func(t *testing.T) repository.CommentDAOs {
		t.Helper()

		byID := make(map[string]repository.CommentDAO[uuid.UUID])
		for _, kind := range []Kind{KindPostComment, KindArtComment, KindShipComment, KindOCComment, KindMysteryComment, KindFanficComment, KindAnnouncementComment} {
			byID[string(kind)] = repository.NewMockCommentDAO[uuid.UUID](t)
		}

		return repository.CommentDAOs{
			ByID:    byID,
			BySlug:  map[string]repository.CommentDAO[string]{string(KindSecretComment): repository.NewMockCommentDAO[string](t)},
			Journal: repository.NewMockJournalCommentWriter(t),
		}
	}

	tests := []struct {
		name    string
		drop    func(c *repository.CommentDAOs)
		wantErr bool
	}{
		{
			name: "every comment kind has a dao",
		},
		{
			name:    "a missing entity dao is reported",
			drop:    func(c *repository.CommentDAOs) { delete(c.ByID, string(KindOCComment)) },
			wantErr: true,
		},
		{
			name:    "a missing journal dao is reported",
			drop:    func(c *repository.CommentDAOs) { c.Journal = nil },
			wantErr: true,
		},
		{
			name:    "a missing secret dao is reported",
			drop:    func(c *repository.CommentDAOs) { delete(c.BySlug, string(KindSecretComment)) },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			comments := complete(t)
			if tt.drop != nil {
				tt.drop(&comments)
			}

			// when
			err := ValidateCommentDAOs(comments)

			// then
			if tt.wantErr {
				require.ErrorIs(t, err, ErrNoCommentDAO)
				return
			}

			require.NoError(t, err)
		})
	}
}
