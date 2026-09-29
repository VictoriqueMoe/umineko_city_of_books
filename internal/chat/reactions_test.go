package chat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	reactionsFixture struct {
		svc       *service
		m         *testMocks
		messageID uuid.UUID
		roomID    uuid.UUID
		userID    uuid.UUID
	}
)

const (
	reactionsThumbsUp = "👍"
)

var (
	reactionsPinnedAt = new("2024-01-01T00:00:00Z")
)

func reactionsFixtureFor(t *testing.T) reactionsFixture {
	svc, m := newTestService(t)

	return reactionsFixture{svc: svc, m: m, messageID: uuid.New(), roomID: uuid.New(), userID: uuid.New()}
}

func reactionsRequireErr(t *testing.T, err, want error) {
	t.Helper()

	if want == nil {
		require.NoError(t, err)

		return
	}

	require.ErrorIs(t, err, want)
}

func (f reactionsFixture) member() spec.ChatMemberRef {
	return spec.ChatMemberRef{RoomID: f.roomID, UserID: f.userID}
}

func (f reactionsFixture) expectMessage(pinnedAt *string) {
	f.m.chatRepo.EXPECT().GetMessageByID(mock.Anything, f.messageID).Return(&model.ChatMessageRow{ID: f.messageID, RoomID: f.roomID, PinnedAt: pinnedAt}, nil)
}

func (f reactionsFixture) expectPinnable(roomType dto.RoomType, pinnedAt *string) {
	f.expectMessage(pinnedAt)
	expectRoomKind(f.m, f.roomID, roomType)
}

func (f reactionsFixture) expectRoles(memberRole string, siteRole role.Role) {
	f.m.chatRepo.EXPECT().GetMemberRole(mock.Anything, f.member()).Return(memberRole, nil)
	f.m.authzSvc.EXPECT().GetRole(mock.Anything, f.userID).Return(siteRole, nil)
}

func (f reactionsFixture) expectPinned() {
	f.m.chatRepo.EXPECT().PinMessage(mock.Anything, spec.ChatMessagePin{MessageID: f.messageID, PinnedBy: f.userID}).Return(nil)
	f.m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, f.roomID).Return(nil, nil)
}

func (f reactionsFixture) expectUnpinned() {
	f.m.chatRepo.EXPECT().UnpinMessage(mock.Anything, f.messageID).Return(nil)
	f.m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, f.roomID).Return(nil, nil)
}

func (f reactionsFixture) expectReactingMember(isMember bool) {
	f.expectMessage(nil)
	f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(isMember, nil)
}

func (f reactionsFixture) expectReactionAnnounced(count int) {
	f.m.userRepo.EXPECT().GetByID(mock.Anything, f.userID).Return(sampleUser(f.userID), nil)
	f.m.chatRepo.EXPECT().CountReactions(mock.Anything, spec.ChatReactionCount{MessageID: f.messageID, Emoji: reactionsThumbsUp}).Return(count, nil)
	f.m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, f.roomID).Return(nil, nil)
}

func TestPinMessage(t *testing.T) {
	errDB := errors.New("db")
	cases := []struct {
		name    string
		wantErr error
		given   func(f reactionsFixture)
	}{
		{name: "a missing message reads as a missing room", wantErr: ErrRoomNotFound, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().GetMessageByID(mock.Anything, f.messageID).Return(nil, nil)
		}},
		{name: "a failed message lookup is reported", wantErr: errDB, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().GetMessageByID(mock.Anything, f.messageID).Return(nil, errDB)
		}},
		{name: "a failed role lookup is reported and nothing is pinned", wantErr: errDB, given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, nil)
			f.m.chatRepo.EXPECT().GetMemberRole(mock.Anything, f.member()).Return("", errDB)
		}},
		{name: "a failed pin write is reported and nothing is broadcast", wantErr: errDB, given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, nil)
			f.m.chatRepo.EXPECT().GetMemberRole(mock.Anything, f.member()).Return("host", nil)
			f.m.chatRepo.EXPECT().PinMessage(mock.Anything, spec.ChatMessagePin{MessageID: f.messageID, PinnedBy: f.userID}).Return(errDB)
		}},
		{name: "a host pins a message in their room", given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, nil)
			f.m.chatRepo.EXPECT().GetMemberRole(mock.Anything, f.member()).Return("host", nil)
			f.expectPinned()
		}},
		{name: "a site moderator pins in a room where they are only a member", given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, nil)
			f.expectRoles("member", authz.RoleModerator)
			f.expectPinned()
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := reactionsFixtureFor(t)
			tc.given(f)

			// when
			err := f.svc.PinMessage(context.Background(), f.messageID, f.userID)

			// then
			reactionsRequireErr(t, err, tc.wantErr)
		})
	}
}

func TestPinMessage_BothParticipantsMayPinTheirOwnThread(t *testing.T) {
	cases := []struct {
		name     string
		roomType dto.RoomType
		isMember bool
		wantErr  error
	}{
		{name: "either party may pin inside their own pair without holding a role", roomType: dto.RoomTypeDM, isMember: true},
		{name: "somebody outside the pair still has to be staff", roomType: dto.RoomTypeDM, wantErr: ErrNotHost},
		{name: "an ordinary member of a room still cannot pin", roomType: dto.RoomTypeGroup, isMember: true, wantErr: ErrNotHost},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a message in a room of this kind
			f := reactionsFixtureFor(t)
			f.expectPinnable(tc.roomType, nil)

			if tc.roomType == dto.RoomTypeDM {
				f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(tc.isMember, nil)
			}

			if tc.wantErr != nil {
				f.expectRoles("member", "")
			} else {
				f.expectPinned()
			}

			// when they pin it
			err := f.svc.PinMessage(context.Background(), f.messageID, f.userID)

			// then pinning is a participant capability in a pair and a moderator one everywhere else
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			f.m.chatRepo.AssertNotCalled(t, "GetMemberRole", mock.Anything, f.member())
		})
	}
}

func TestPinMessage_SiteStaffStillPinInsideAPair(t *testing.T) {
	// given a member of staff who is not part of the pair
	f := reactionsFixtureFor(t)
	f.expectPinnable(dto.RoomTypeDM, nil)
	f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(false, nil)
	f.expectRoles("", authz.RoleModerator)
	f.expectPinned()

	// when they pin a message inside the pair
	err := f.svc.PinMessage(context.Background(), f.messageID, f.userID)

	// then site-staff moderation inside a pair is unchanged
	require.NoError(t, err)
}

func TestUnpinMessage(t *testing.T) {
	errDB := errors.New("db")
	cases := []struct {
		name    string
		wantErr error
		given   func(f reactionsFixture)
	}{
		{name: "a missing message reads as a missing room", wantErr: ErrRoomNotFound, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().GetMessageByID(mock.Anything, f.messageID).Return(nil, nil)
		}},
		{name: "a message that is not pinned is refused before any permission check", wantErr: ErrMessageNotPinned, given: func(f reactionsFixture) {
			f.expectMessage(nil)
		}},
		{name: "an ordinary member cannot unpin", wantErr: ErrNotHost, given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, reactionsPinnedAt)
			f.expectRoles("member", "")
		}},
		{name: "a failed unpin write is reported and nothing is broadcast", wantErr: errDB, given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, reactionsPinnedAt)
			f.m.chatRepo.EXPECT().GetMemberRole(mock.Anything, f.member()).Return("host", nil)
			f.m.chatRepo.EXPECT().UnpinMessage(mock.Anything, f.messageID).Return(errDB)
		}},
		{name: "a host unpins a pinned message", given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, reactionsPinnedAt)
			f.m.chatRepo.EXPECT().GetMemberRole(mock.Anything, f.member()).Return("host", nil)
			f.expectUnpinned()
		}},
		{name: "a site admin unpins in a room where they are only a member", given: func(f reactionsFixture) {
			f.expectPinnable(dto.RoomTypeGroup, reactionsPinnedAt)
			f.expectRoles("member", authz.RoleAdmin)
			f.expectUnpinned()
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := reactionsFixtureFor(t)
			tc.given(f)

			// when
			err := f.svc.UnpinMessage(context.Background(), f.messageID, f.userID)

			// then
			reactionsRequireErr(t, err, tc.wantErr)
		})
	}
}

func TestListPinnedMessages(t *testing.T) {
	// given
	f := reactionsFixtureFor(t)
	f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(true, nil)
	f.m.chatRepo.EXPECT().ListPinnedMessages(mock.Anything, spec.ChatRoomViewer{RoomID: f.roomID, ViewerID: f.userID}).Return([]model.ChatMessageRow{
		{ID: f.messageID, RoomID: f.roomID, SenderID: uuid.New(), Body: "pinned"},
	}, nil)
	f.m.chatRepo.EXPECT().GetMessageMediaBatch(mock.Anything, []uuid.UUID{f.messageID}).Return(nil, nil)
	f.m.chatRepo.EXPECT().GetReactionsBatch(mock.Anything, spec.ChatReactionsQuery{MessageIDs: []uuid.UUID{f.messageID}, ViewerID: f.userID}).Return(nil, nil)
	f.m.vanityRoleRepo.EXPECT().GetRolesForUsersBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	res, err := f.svc.ListPinnedMessages(context.Background(), f.roomID, f.userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, "pinned", res.Messages[0].Body)
}

func TestListPinnedMessages_Rejections(t *testing.T) {
	errDB := errors.New("db")
	cases := []struct {
		name    string
		wantErr error
		given   func(f reactionsFixture)
	}{
		{name: "a non-member cannot list the pins", wantErr: ErrNotMember, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(false, nil)
		}},
		{name: "a failed membership check is reported", wantErr: errDB, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(false, errDB)
		}},
		{name: "a failed listing is reported", wantErr: errDB, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().IsMember(mock.Anything, f.member()).Return(true, nil)
			f.m.chatRepo.EXPECT().ListPinnedMessages(mock.Anything, spec.ChatRoomViewer{RoomID: f.roomID, ViewerID: f.userID}).Return(nil, errDB)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := reactionsFixtureFor(t)
			tc.given(f)

			// when
			res, err := f.svc.ListPinnedMessages(context.Background(), f.roomID, f.userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, res)
		})
	}
}

func TestAddReaction(t *testing.T) {
	errDB := errors.New("db")
	cases := []struct {
		name    string
		emoji   string
		wantErr error
		given   func(f reactionsFixture)
	}{
		{name: "an empty emoji is rejected before the message is looked up", emoji: "", wantErr: ErrInvalidEmoji},
		{name: "an emoji over the rune limit is rejected before the message is looked up", emoji: strings.Repeat("x", 20), wantErr: ErrInvalidEmoji},
		{name: "a missing message reads as a missing room", emoji: reactionsThumbsUp, wantErr: ErrRoomNotFound, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().GetMessageByID(mock.Anything, f.messageID).Return(nil, nil)
		}},
		{name: "a non-member cannot react", emoji: reactionsThumbsUp, wantErr: ErrNotMember, given: func(f reactionsFixture) {
			f.expectReactingMember(false)
		}},
		{name: "a timed-out member cannot react", emoji: reactionsThumbsUp, wantErr: ErrTimedOut, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, f.member()).Return(true, "2099-01-01T00:00:00Z", false, nil)
		}},
		{name: "a failed write is reported and nothing is broadcast", emoji: reactionsThumbsUp, wantErr: errDB, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, f.member()).Return(false, "", false, nil)
			f.m.chatRepo.EXPECT().AddReaction(mock.Anything, spec.ChatMessageReaction{MessageID: f.messageID, UserID: f.userID, Emoji: reactionsThumbsUp}).Return(false, errDB)
		}},
		{name: "a member's reaction is stored and announced", emoji: reactionsThumbsUp, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, f.member()).Return(false, "", false, nil)
			f.m.chatRepo.EXPECT().AddReaction(mock.Anything, spec.ChatMessageReaction{MessageID: f.messageID, UserID: f.userID, Emoji: reactionsThumbsUp}).Return(true, nil)
			f.expectReactionAnnounced(1)
		}},
		{name: "a failed count is reported rather than broadcasting a count of zero", emoji: reactionsThumbsUp, wantErr: errDB, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, f.member()).Return(false, "", false, nil)
			f.m.chatRepo.EXPECT().AddReaction(mock.Anything, spec.ChatMessageReaction{MessageID: f.messageID, UserID: f.userID, Emoji: reactionsThumbsUp}).Return(true, nil)
			f.m.userRepo.EXPECT().GetByID(mock.Anything, f.userID).Return(sampleUser(f.userID), nil)
			f.m.chatRepo.EXPECT().CountReactions(mock.Anything, spec.ChatReactionCount{MessageID: f.messageID, Emoji: reactionsThumbsUp}).Return(0, errDB)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := reactionsFixtureFor(t)

			if tc.given != nil {
				tc.given(f)
			}

			// when
			err := f.svc.AddReaction(context.Background(), f.messageID, f.userID, tc.emoji)

			// then
			reactionsRequireErr(t, err, tc.wantErr)
		})
	}
}

func TestRemoveReaction(t *testing.T) {
	errDB := errors.New("db")
	cases := []struct {
		name    string
		emoji   string
		wantErr error
		given   func(f reactionsFixture)
	}{
		{name: "an empty emoji is rejected before the message is looked up", emoji: "", wantErr: ErrInvalidEmoji},
		{name: "an emoji over the rune limit is rejected before the message is looked up", emoji: strings.Repeat("x", 20), wantErr: ErrInvalidEmoji},
		{name: "a missing message reads as a missing room", emoji: reactionsThumbsUp, wantErr: ErrRoomNotFound, given: func(f reactionsFixture) {
			f.m.chatRepo.EXPECT().GetMessageByID(mock.Anything, f.messageID).Return(nil, nil)
		}},
		{name: "a non-member cannot remove a reaction", emoji: reactionsThumbsUp, wantErr: ErrNotMember, given: func(f reactionsFixture) {
			f.expectReactingMember(false)
		}},
		{name: "a failed write is reported and nothing is broadcast", emoji: reactionsThumbsUp, wantErr: errDB, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().RemoveReaction(mock.Anything, spec.ChatMessageReaction{MessageID: f.messageID, UserID: f.userID, Emoji: reactionsThumbsUp}).Return(false, errDB)
		}},
		{name: "a member's reaction is removed and announced", emoji: reactionsThumbsUp, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().RemoveReaction(mock.Anything, spec.ChatMessageReaction{MessageID: f.messageID, UserID: f.userID, Emoji: reactionsThumbsUp}).Return(true, nil)
			f.expectReactionAnnounced(0)
		}},
		{name: "a failed count is reported rather than broadcasting a count of zero", emoji: reactionsThumbsUp, wantErr: errDB, given: func(f reactionsFixture) {
			f.expectReactingMember(true)
			f.m.chatRepo.EXPECT().RemoveReaction(mock.Anything, spec.ChatMessageReaction{MessageID: f.messageID, UserID: f.userID, Emoji: reactionsThumbsUp}).Return(true, nil)
			f.m.userRepo.EXPECT().GetByID(mock.Anything, f.userID).Return(sampleUser(f.userID), nil)
			f.m.chatRepo.EXPECT().CountReactions(mock.Anything, spec.ChatReactionCount{MessageID: f.messageID, Emoji: reactionsThumbsUp}).Return(0, errDB)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := reactionsFixtureFor(t)

			if tc.given != nil {
				tc.given(f)
			}

			// when
			err := f.svc.RemoveReaction(context.Background(), f.messageID, f.userID, tc.emoji)

			// then
			reactionsRequireErr(t, err, tc.wantErr)
		})
	}
}
