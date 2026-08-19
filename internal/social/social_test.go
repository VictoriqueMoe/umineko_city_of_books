package social

import (
	"errors"
	"testing"
	"umineko_city_of_books/internal/block"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMentionRegex_MatchesUsernames(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"single mention", "hello @alice", []string{"alice"}},
		{"multiple mentions", "@alice said hi to @bob", []string{"alice", "bob"}},
		{"underscores and digits", "ping @user_42", []string{"user_42"}},
		{"no mentions", "just some text", nil},
		{"dash not part of name", "@alice-bob", []string{"alice"}},
		{"email address is not a mention", "mail me at kujo@alice.example", nil},
		{"mention after punctuation still counts", "(@alice)", []string{"alice"}},
		{"mention at the very start still counts", "@alice hello", []string{"alice"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			matches := MentionRegex.FindAllStringSubmatch(tc.body, -1)

			// when
			var got []string
			for _, m := range matches {
				got = append(got, m[1])
			}

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestProcessMentions_NoMentionsDoesNothing(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, uuid.New(), "no mentions", uuid.New(), "post", "/p/1")

	// then — no mock calls expected
}

func TestProcessMentions_UnknownUsernameSkipped(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	actorID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, Username: "bob"}, nil)
	userRepo.EXPECT().GetByUsername(mock.Anything, "ghost").Return(nil, errors.New("not found"))

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, actorID, "@ghost", uuid.New(), "post", "/p/1")

	// then — no notification sent
}

func TestProcessMentions_SelfMentionSkipped(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	actorID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, Username: "me", DisplayName: "Me"}, nil)
	userRepo.EXPECT().GetByUsername(mock.Anything, "me").Return(&model.User{ID: actorID, Username: "me", DisplayName: "Me"}, nil)

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, actorID, "@me", uuid.New(), "post", "/p/1")

	// then — no notification sent
}

func TestProcessMentions_DuplicateUsernameOnlyNotifiedOnce(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	actorID := uuid.New()
	mentionedID := uuid.New()
	refID := uuid.New()
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: mentionedID, Username: "alice", DisplayName: "Alice"}, nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, Username: "bob", DisplayName: "Bob"}, nil).Once()
	notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == mentionedID && p.ActorID == actorID && p.Type == dto.NotifMention && p.ReferenceID == refID && p.ReferenceType == "post"
	})).Return(nil).Once()

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, actorID, "@alice and @alice again", refID, "post", "/p/1")

	// then — only one notification call (enforced by .Once())
}

func TestProcessMentions_ActorLookupErrorSkipped(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	actorID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(nil, errors.New("boom"))

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, actorID, "@alice", uuid.New(), "post", "/p/1")

	// then — no notification sent
}

func TestProcessMentions_BotAuthoredMentionsNotifyNobody(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	botID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, botID).Return(&model.User{ID: botID, Username: "beatrice", IsBot: true}, nil)

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, botID, "@alice come look at this", uuid.New(), "post", "/p/1")

	// then
	notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything)
	userRepo.AssertNotCalled(t, "GetByUsername", mock.Anything, mock.Anything)
	blockSvc.AssertNotCalled(t, "IsBlockedEither", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMentions_NotifyErrorSwallowed(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	actorID := uuid.New()
	mentionedID := uuid.New()
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: mentionedID, Username: "alice"}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "Bob"}, nil)
	notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(errors.New("notify failed"))

	// when — should not panic
	require.NotPanics(t, func() {
		ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, actorID, "@alice", uuid.New(), "post", "/p/1")
	})

	// then — error swallowed
}

func TestProcessMentions_BlockedUserIsNotNotified(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	blockSvc := block.NewMockService(t)
	actorID := uuid.New()
	mentionedID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, Username: "bob"}, nil)
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: mentionedID, Username: "alice"}, nil)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, mentionedID).Return(true, nil)

	// when
	ProcessMentions(userRepo, blockSvc, notifSvc, settingsSvc, actorID, "@alice", uuid.New(), "post", "/p/1")

	// then
	notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything)
}
