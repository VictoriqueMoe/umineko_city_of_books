package chat

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type roomCapabilities struct {
	announcesDepartures    bool
	destroyableByModerator bool
	evictsOnBannedWord     bool
	participantsMayPin     bool
	requiresRecipientOptIn bool
	countsTowardsUnread    bool
	pairwiseReadReceipts   bool
}

func capabilitiesFor(roomType dto.RoomType) roomCapabilities {
	if roomType == dto.RoomTypeDM {
		return roomCapabilities{
			participantsMayPin:     true,
			requiresRecipientOptIn: true,
			countsTowardsUnread:    true,
			pairwiseReadReceipts:   true,
		}
	}

	return roomCapabilities{
		announcesDepartures:    true,
		destroyableByModerator: roomType == dto.RoomTypeGroup,
		evictsOnBannedWord:     true,
	}
}

func sendContextCapabilities(roomRow *repository.ChatRoomSendContext) roomCapabilities {
	if roomRow == nil {
		return roomCapabilities{}
	}

	return capabilitiesFor(roomRow.Type)
}

func hostBlockApplies(roomRow *repository.ChatRoomSendContext) bool {
	if !roomRow.IsSystem {
		return true
	}

	return roomRow.SystemKind == SystemKindLiveStream || roomRow.SystemKind == SystemKindWatchParty
}

func plainMessageNotification(roomRow *repository.ChatRoomSendContext) (dto.NotificationType, string) {
	if roomRow == nil || roomRow.Type != dto.RoomTypeGroup {
		return dto.NotifChatMessage, ""
	}

	return dto.NotifChatRoomMessage, fmt.Sprintf("sent a message in %s", roomRow.Name)
}

func (c *core) assertPairNotBlocked(ctx context.Context, userID, otherID uuid.UUID) error {
	if blocked, _ := c.blockSvc.IsBlockedEither(ctx, userID, otherID); blocked {
		return ErrUserBlocked
	}

	return nil
}

func (c *core) assertBlocksAllowParticipation(ctx context.Context, roomRow *repository.ChatRoomSendContext, userID uuid.UUID, members []uuid.UUID) error {
	if roomRow == nil {
		return nil
	}

	if roomRow.Type == dto.RoomTypeDM {
		for _, memberID := range members {
			if memberID == userID {
				continue
			}

			if err := c.assertPairNotBlocked(ctx, userID, memberID); err != nil {
				return err
			}
		}

		return nil
	}

	if !hostBlockApplies(roomRow) || roomRow.CreatedBy == userID {
		return nil
	}

	if blocked, _ := c.blockSvc.IsBlocked(ctx, roomRow.CreatedBy, userID); blocked {
		return ErrBlockedByRoomHost
	}

	return nil
}

func (c *core) assertBlocksAllowRoomEntry(ctx context.Context, roomRow *repository.ChatRoomSendContext, userID uuid.UUID) error {
	if roomRow == nil {
		return nil
	}

	members, err := c.chatRepo.GetRoomMembers(ctx, roomRow.ID)
	if err != nil {
		return fmt.Errorf("get room members: %w", err)
	}

	return c.assertBlocksAllowParticipation(ctx, roomRow, userID, members)
}

func (c *core) senderLocked(ctx context.Context, senderID uuid.UUID) (bool, error) {
	locked, err := c.userRepo.IsLocked(ctx, senderID)
	if err != nil {
		return false, fmt.Errorf("check lock: %w", err)
	}

	return locked, nil
}

func (c *core) assertAudienceHasStaff(ctx context.Context, audience []uuid.UUID) error {
	if len(audience) == 0 {
		return ErrLockedNonStaffDM
	}

	roles, err := c.authzSvc.GetRoles(ctx, audience)
	if err != nil {
		return fmt.Errorf("get member roles: %w", err)
	}

	for _, r := range roles {
		if r.IsSiteStaff() {
			return nil
		}
	}

	return ErrLockedNonStaffDM
}
