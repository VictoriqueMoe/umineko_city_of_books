package chat

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

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

func (s *watchPartyService) postControlChangeSystemMessage(ctx context.Context, roomID, sessionID, callerID, targetID uuid.UUID, reason string) {
	switch reason {
	case "reclaim":
		body := fmt.Sprintf("%s took control.", s.displayNameFor(ctx, callerID, roomID))
		s.postRoomActionMessage(ctx, sessionID, callerID, body)
	case "pass":
		body := fmt.Sprintf("%s gave control to %s.", s.displayNameFor(ctx, callerID, roomID), s.displayNameFor(ctx, targetID, roomID))
		s.postRoomActionMessage(ctx, sessionID, callerID, body)
	case "auto_owner_return":
		body := fmt.Sprintf("Control returned to %s.", s.displayNameFor(ctx, targetID, roomID))
		s.postRoomActionMessage(ctx, sessionID, targetID, body)
	}
}
