package chat

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/hyperbeam"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func expectGroupRoomLookup(m *testMocks, roomID, userID uuid.UUID) {
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "group", IsSystem: false,
	}, nil)
}

func TestStartWatchParty_Disabled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.hyperbeamSvc.EXPECT().Enabled().Return(false)

	// when
	_, err := svc.StartWatchParty(context.Background(), roomID, userID, "", "", "")

	// then
	require.ErrorIs(t, err, ErrWatchPartyDisabled)
}

func TestStartWatchParty_NotMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(false, nil)

	// when
	_, err := svc.StartWatchParty(context.Background(), roomID, userID, "", "", "")

	// then
	require.ErrorIs(t, err, ErrNotMember)
}

func TestStartWatchParty_RejectsDM(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "dm", IsSystem: false,
	}, nil)

	// when
	_, err := svc.StartWatchParty(context.Background(), roomID, userID, "", "", "")

	// then
	require.ErrorIs(t, err, ErrWatchPartyWrongRoomType)
}

func TestStartWatchParty_RejectsSystemRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, roomID, userID).Return(&repository.ChatRoomRow{
		ID: roomID, Type: "group", IsSystem: true,
	}, nil)

	// when
	_, err := svc.StartWatchParty(context.Background(), roomID, userID, "", "", "")

	// then
	require.ErrorIs(t, err, ErrWatchPartyWrongRoomType)
}

func TestStartWatchParty_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	expectGroupRoomLookup(m, roomID, userID)
	m.hyperbeamSvc.EXPECT().CreateVM(mock.Anything, mock.MatchedBy(func(opts hyperbeam.CreateVMOptions) bool {
		return opts.Timeout != nil && opts.Timeout.Offline == defaultOfflineTimeout && opts.Timeout.Absolute == defaultSessionTimeout
	})).Return(&hyperbeam.VM{SessionID: "hb_sess_1", EmbedURL: "https://hb/embed", AdminToken: "admin"}, nil)
	m.watchPartyRepo.EXPECT().CreateSession(mock.Anything, mock.MatchedBy(func(r repository.ChatWatchPartySessionRow) bool {
		return r.RoomID == roomID && r.ControllerID == userID && r.HyperbeamSessionID == "hb_sess_1" && r.Title == "Movie night" && r.VMBaseURL == "https://hb/embed"
	})).Return(sessionID, nil)
	m.watchPartyRepo.EXPECT().UpsertParticipant(mock.Anything, sessionID, userID, true, "").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, userID, "watch_party.start", "chat_watch_party_session", sessionID.String(), mock.Anything).Return(nil)

	stored := &repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, StartedBy: userID, ControllerID: userID,
		HyperbeamSessionID: "hb_sess_1", EmbedURL: "https://hb/embed", Status: "active", Title: "Movie night",
	}
	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(stored, nil)
	m.watchPartyRepo.EXPECT().GetActiveParticipants(mock.Anything, sessionID).Return(nil, nil).Twice()
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, userID).Return(&repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: userID, HasControl: true}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, userID, mock.Anything).Return(nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	resp, err := svc.StartWatchParty(context.Background(), roomID, userID, "", "", "Movie night")

	// then
	require.NoError(t, err)
	require.Equal(t, "https://hb/embed", resp.EmbedURL)
	require.Equal(t, sessionID, resp.Session.ID)
	require.Equal(t, "Movie night", resp.Session.Title)
	require.NotNil(t, resp.Session.Viewer)
	require.True(t, resp.Session.Viewer.HasControl)
}

func TestStartWatchParty_HyperbeamFailureCleansUp(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()

	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	expectGroupRoomLookup(m, roomID, userID)
	m.hyperbeamSvc.EXPECT().CreateVM(mock.Anything, mock.Anything).Return(nil, errors.New("hyperbeam down"))

	// when
	_, err := svc.StartWatchParty(context.Background(), roomID, userID, "", "", "")

	// then
	require.Error(t, err)
}

func TestJoinWatchParty_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	controllerID := uuid.New()
	joinerID := uuid.New()

	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, joinerID).Return(true, nil)
	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, ControllerID: controllerID, HyperbeamSessionID: "hb", Status: "active",
		VMBaseURL: "https://hb.example/sess", EmbedURL: "https://hb.example/sess?token=u",
	}, nil)
	m.hyperbeamSvc.EXPECT().GetVMStatus(mock.Anything, "hb").Return(&hyperbeam.VMStatus{SessionID: "hb"}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, joinerID).Return(nil, nil).Once()
	m.watchPartyRepo.EXPECT().UpsertParticipant(mock.Anything, sessionID, joinerID, false, "").Return(nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, joinerID).Return(&repository.ChatWatchPartyParticipantRow{
		SessionID: sessionID, UserID: joinerID, Username: "joiner", DisplayName: "Joiner",
	}, nil)
	m.roleRepo.EXPECT().GetRole(mock.Anything, joinerID).Return("", nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, joinerID).Return(nil, nil)
	m.watchPartyRepo.EXPECT().GetActiveParticipants(mock.Anything, sessionID).Return(nil, nil)

	// when
	resp, err := svc.JoinWatchParty(context.Background(), roomID, sessionID, joinerID)

	// then
	require.NoError(t, err)
	require.Equal(t, "https://hb.example/sess?token=u", resp.EmbedURL)
}

func TestJoinWatchParty_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	userID := uuid.New()
	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, userID).Return(true, nil)
	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(nil, nil)

	// when
	_, err := svc.JoinWatchParty(context.Background(), roomID, sessionID, userID)

	// then
	require.ErrorIs(t, err, ErrWatchPartyNotActive)
}

func TestGrantWatchPartyControl_NotController(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	controllerID := uuid.New()
	targetID := uuid.New()

	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	otherOwnerID := uuid.New()
	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, StartedBy: otherOwnerID, ControllerID: targetID, HyperbeamSessionID: "hb", Status: "active",
	}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, controllerID).Return(&repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: controllerID, HasControl: false}, nil)
	m.roleRepo.EXPECT().GetRole(mock.Anything, controllerID).Return("", nil)

	// when
	err := svc.GrantWatchPartyControl(context.Background(), roomID, sessionID, controllerID, targetID)

	// then
	require.ErrorIs(t, err, ErrWatchPartyNotController)
}

func TestGrantWatchPartyControl_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	controllerID := uuid.New()
	targetID := uuid.New()

	m.hyperbeamSvc.EXPECT().Enabled().Return(true)
	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, StartedBy: controllerID, ControllerID: controllerID, HyperbeamSessionID: "hb_sess", Status: "active",
		VMBaseURL: "https://hb.example/sess",
	}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, controllerID).Return(&repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: controllerID, HasControl: true, HyperbeamIdentifier: "id-controller"}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, targetID).Return(&repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: targetID, HasControl: false, HyperbeamIdentifier: "id-target"}, nil)
	m.watchPartyRepo.EXPECT().GetActiveParticipants(mock.Anything, sessionID).Return([]repository.ChatWatchPartyParticipantRow{
		{SessionID: sessionID, UserID: controllerID, HasControl: true, HyperbeamIdentifier: "id-controller"},
		{SessionID: sessionID, UserID: targetID, HasControl: false, HyperbeamIdentifier: "id-target"},
	}, nil)
	m.hyperbeamSvc.EXPECT().SetControlRole(mock.Anything, "https://hb.example/sess", "id-controller", false).Return(nil)
	m.watchPartyRepo.EXPECT().SetParticipantControl(mock.Anything, sessionID, controllerID, false).Return(nil)
	m.hyperbeamSvc.EXPECT().SetControlRole(mock.Anything, "https://hb.example/sess", "id-target", true).Return(nil)
	m.watchPartyRepo.EXPECT().SetParticipantControl(mock.Anything, sessionID, targetID, true).Return(nil)
	m.watchPartyRepo.EXPECT().SetControllerID(mock.Anything, sessionID, targetID).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, controllerID, "watch_party.grant_control", "chat_watch_party_session", sessionID.String(), mock.Anything).Return(nil)

	// when
	err := svc.GrantWatchPartyControl(context.Background(), roomID, sessionID, controllerID, targetID)

	// then
	require.NoError(t, err)
}

func TestLeaveWatchParty_OwnerLeavesEndsSession(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	ownerID := uuid.New()
	session := &repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, StartedBy: ownerID, ControllerID: ownerID,
		HyperbeamSessionID: "hb_sess", Status: "active",
	}

	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(session, nil).Twice()
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, ownerID).Return(&repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: ownerID, HasControl: true, LeftAt: sql.NullString{}}, nil).Twice()
	m.hyperbeamSvc.EXPECT().TerminateVM(mock.Anything, "hb_sess").Return(nil)
	m.watchPartyRepo.EXPECT().MarkAllParticipantsLeft(mock.Anything, sessionID).Return(nil)
	m.watchPartyRepo.EXPECT().EndSession(mock.Anything, sessionID, "owner_left").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, ownerID, "watch_party.end", "chat_watch_party_session", sessionID.String(), mock.Anything).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, ownerID).Return(nil, nil)
	m.chatRepo.EXPECT().InsertSystemMessage(mock.Anything, mock.Anything, roomID, ownerID, mock.Anything).Return(nil)
	m.chatRepo.EXPECT().GetMessageByID(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	err := svc.LeaveWatchParty(context.Background(), roomID, sessionID, ownerID)

	// then
	require.NoError(t, err)
}

func TestLeaveWatchParty_NonControllerJustLeaves(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	controllerID := uuid.New()
	memberID := uuid.New()

	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, ControllerID: controllerID, Status: "active",
	}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, memberID).Return(&repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: memberID, HasControl: false}, nil)
	m.watchPartyRepo.EXPECT().MarkParticipantLeft(mock.Anything, sessionID, memberID).Return(nil)

	// when
	err := svc.LeaveWatchParty(context.Background(), roomID, sessionID, memberID)

	// then
	require.NoError(t, err)
}

func TestSendWatchPartyMessage_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	senderID := uuid.New()

	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, Status: "active",
	}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, senderID).Return(&repository.ChatWatchPartyParticipantRow{
		SessionID: sessionID, UserID: senderID,
	}, nil)
	m.watchPartyRepo.EXPECT().InsertMessage(mock.Anything, mock.Anything, sessionID, senderID, "hello").Return(nil)
	m.watchPartyRepo.EXPECT().GetMessageByID(mock.Anything, mock.Anything).Return(&repository.ChatWatchPartyMessageRow{
		ID: uuid.New(), SessionID: sessionID, SenderID: senderID, SenderUsername: "sender", SenderDisplayName: "Sender", Body: "hello",
	}, nil)
	m.roleRepo.EXPECT().GetRole(mock.Anything, senderID).Return("", nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, senderID).Return(nil, nil)
	m.watchPartyRepo.EXPECT().GetActiveParticipants(mock.Anything, sessionID).Return(nil, nil)

	// when
	msg, err := svc.SendWatchPartyMessage(context.Background(), roomID, sessionID, senderID, "hello")

	// then
	require.NoError(t, err)
	require.Equal(t, "hello", msg.Body)
}

func TestSendWatchPartyMessage_NotParticipant(t *testing.T) {
	// given
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	senderID := uuid.New()

	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, Status: "active",
	}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, senderID).Return(nil, nil)

	// when
	_, err := svc.SendWatchPartyMessage(context.Background(), roomID, sessionID, senderID, "hello")

	// then
	require.ErrorIs(t, err, ErrWatchPartyNotParticipant)
}
