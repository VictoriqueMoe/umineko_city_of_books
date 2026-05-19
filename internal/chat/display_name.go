package chat

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s *service) displayNameFor(ctx context.Context, userID, roomID uuid.UUID) string {
	if roomID != uuid.Nil {
		if nickname, _ := s.chatRepo.GetMemberNickname(ctx, roomID, userID); strings.TrimSpace(nickname) != "" {
			return nickname
		}
	}
	user, _ := s.userRepo.GetByID(ctx, userID)
	return user.DisplayLabel()
}
