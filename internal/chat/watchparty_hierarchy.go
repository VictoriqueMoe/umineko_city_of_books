package chat

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const watchPartyOwnerRank = 1

func watchPartyEffectiveRank(siteRole role.Role, isOwner bool) int {
	rank := siteRole.Rank()
	if isOwner && rank < watchPartyOwnerRank {
		return watchPartyOwnerRank
	}
	return rank
}

func (s *watchPartyService) watchPartyRankOf(ctx context.Context, session *repository.ChatWatchPartySessionRow, userID uuid.UUID) int {
	siteRole, _ := s.roleRepo.GetRole(ctx, userID)
	return watchPartyEffectiveRank(siteRole, session.StartedBy == userID)
}

func (s *watchPartyService) postWatchPartySystemMessage(ctx context.Context, roomID, sessionID uuid.UUID, body string) {
	if body == "" {
		return
	}
	id := uuid.New()
	if err := s.watchPartyRepo.InsertSystemMessage(ctx, id, sessionID, body); err != nil {
		logger.Log.Warn().Err(err).Msg("insert watch party system message failed")
		return
	}
	row, err := s.watchPartyRepo.GetMessageByID(ctx, id)
	if err != nil || row == nil {
		logger.Log.Warn().Err(err).Msg("reload watch party system message failed")
		return
	}
	msgDTO := s.buildWatchPartyMessageDTO(ctx, row)
	s.hub.BroadcastToRoom(roomID, ws.Message{
		Type: wsWatchPartyMessage,
		Data: dto.WatchPartyMessageEvent{
			SessionID: sessionID,
			RoomID:    roomID,
			Message:   msgDTO,
		},
	}, uuid.Nil)
}

func (s *watchPartyService) postControlChangeSystemMessage(ctx context.Context, roomID, sessionID, callerID, targetID uuid.UUID, reason string) {
	switch reason {
	case "reclaim":
		body := fmt.Sprintf("%s took control.", s.displayNameFor(ctx, callerID, roomID))
		s.postWatchPartySystemMessage(ctx, roomID, sessionID, body)
	case "pass":
		body := fmt.Sprintf("%s gave control to %s.", s.displayNameFor(ctx, callerID, roomID), s.displayNameFor(ctx, targetID, roomID))
		s.postWatchPartySystemMessage(ctx, roomID, sessionID, body)
	case "auto_owner_return":
		body := fmt.Sprintf("Control returned to %s.", s.displayNameFor(ctx, targetID, roomID))
		s.postWatchPartySystemMessage(ctx, roomID, sessionID, body)
	}
}
