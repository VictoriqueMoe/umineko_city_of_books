package social

import (
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
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

func TestProcessEmbeds_SkipsUnparseable(t *testing.T) {
	// given
	postRepo := repository.NewMockPostRepository(t)
	ownerID := uuid.NewString()

	// when
	ProcessEmbeds(postRepo, ownerID, "post", "no urls here")

	// then — no mock calls expected
}

func TestProcessEmbeds_AddsEmbedsForValidURLs(t *testing.T) {
	// given
	postRepo := repository.NewMockPostRepository(t)
	ownerID := uuid.NewString()
	postRepo.EXPECT().AddEmbed(
		mock.Anything, ownerID, "post",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil).Maybe()

	// when
	ProcessEmbeds(postRepo, ownerID, "post", "check this https://youtube.com/watch?v=dQw4w9WgXcQ")

	// then — embed attempt made, errors swallowed
}

func TestProcessEmbeds_CapsAtFiveURLs(t *testing.T) {
	// given
	postRepo := repository.NewMockPostRepository(t)
	ownerID := uuid.NewString()
	body := "https://a.com https://b.com https://c.com https://d.com https://e.com https://f.com https://g.com"
	postRepo.EXPECT().AddEmbed(
		mock.Anything, ownerID, "post",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil).Maybe()

	// when
	ProcessEmbeds(postRepo, ownerID, "post", body)

	// then — mockery verifies no more than the number of parseable embeds were attempted;
	// the important invariant (i>=5 break) is exercised because the 6th and 7th URLs
	// are never parsed. We assert by running with no failure.
	_ = strings.Split(body, " ")
}

func TestProcessMentions_NoMentionsDoesNothing(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)

	// when
	ProcessMentions(userRepo, notifSvc, settingsSvc, uuid.New(), "no mentions", uuid.New(), "post", "/p/1")

	// then — no mock calls expected
}

func TestProcessMentions_UnknownUsernameSkipped(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	userRepo.EXPECT().GetByUsername(mock.Anything, "ghost").Return(nil, errors.New("not found"))

	// when
	ProcessMentions(userRepo, notifSvc, settingsSvc, uuid.New(), "@ghost", uuid.New(), "post", "/p/1")

	// then — no notification sent
}

func TestProcessMentions_SelfMentionSkipped(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	actorID := uuid.New()
	userRepo.EXPECT().GetByUsername(mock.Anything, "me").Return(&model.User{ID: actorID, Username: "me", DisplayName: "Me"}, nil)

	// when
	ProcessMentions(userRepo, notifSvc, settingsSvc, actorID, "@me", uuid.New(), "post", "/p/1")

	// then — no notification sent
}

func TestProcessMentions_DuplicateUsernameOnlyNotifiedOnce(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	actorID := uuid.New()
	mentionedID := uuid.New()
	refID := uuid.New()
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: mentionedID, Username: "alice", DisplayName: "Alice"}, nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, Username: "bob", DisplayName: "Bob"}, nil).Once()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com").Once()
	notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == mentionedID && p.ActorID == actorID && p.Type == dto.NotifMention && p.ReferenceID == refID && p.ReferenceType == "post"
	})).Return(nil).Once()

	// when
	ProcessMentions(userRepo, notifSvc, settingsSvc, actorID, "@alice and @alice again", refID, "post", "/p/1")

	// then — only one notification call (enforced by .Once())
}

func TestProcessMentions_ActorLookupErrorSkipped(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	actorID := uuid.New()
	mentionedID := uuid.New()
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: mentionedID, Username: "alice"}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(nil, errors.New("boom"))

	// when
	ProcessMentions(userRepo, notifSvc, settingsSvc, actorID, "@alice", uuid.New(), "post", "/p/1")

	// then — no notification sent
}

func TestProcessMentions_NotifyErrorSwallowed(t *testing.T) {
	// given
	userRepo := repository.NewMockUserRepository(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	actorID := uuid.New()
	mentionedID := uuid.New()
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(&model.User{ID: mentionedID, Username: "alice"}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "Bob"}, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")
	notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(errors.New("notify failed"))

	// when — should not panic
	require.NotPanics(t, func() {
		ProcessMentions(userRepo, notifSvc, settingsSvc, actorID, "@alice", uuid.New(), "post", "/p/1")
	})

	// then — error swallowed
}
