package gameroom

import "errors"

var (
	ErrNotFound         = errors.New("game room not found")
	ErrNotParticipant   = errors.New("not a participant")
	ErrNotInvitee       = errors.New("not the invitee")
	ErrNotInviter       = errors.New("not the inviter")
	ErrRoomNotPending   = errors.New("room is not pending")
	ErrRoomNotActive    = errors.New("room is not active")
	ErrNotYourTurn      = errors.New("not your turn")
	ErrInvalidOpponent  = errors.New("invalid opponent")
	ErrSelfInvite       = errors.New("cannot invite yourself")
	ErrUnknownGameType  = errors.New("unknown game type")
	ErrOpponentBlocked  = errors.New("cannot invite this user")
	ErrOpponentDmsOff   = errors.New("opponent does not accept game invites")
	ErrOpponentInactive = errors.New("opponent not found")
	ErrEmptyChat        = errors.New("chat message is empty")
	ErrPlayersCantChat  = errors.New("players cannot use spectator chat")
)
