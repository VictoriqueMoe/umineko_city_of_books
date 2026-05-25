package chat

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type dmService struct {
	*core
	parent *service
}

func (d *dmService) ensureLockAllowsDMTo(ctx context.Context, senderID, recipientID uuid.UUID) error {
	locked, err := d.userRepo.IsLocked(ctx, senderID)
	if err != nil {
		return fmt.Errorf("check lock: %w", err)
	}
	if !locked {
		return nil
	}

	recipientRole, err := d.authzSvc.GetRole(ctx, recipientID)
	if err != nil {
		return fmt.Errorf("get recipient role: %w", err)
	}
	if !recipientRole.IsSiteStaff() {
		return ErrLockedNonStaffDM
	}
	return nil
}

func (d *dmService) checkDMPreconditions(ctx context.Context, senderID, recipientID uuid.UUID) (*model.User, error) {
	if senderID == recipientID {
		return nil, ErrCannotDMSelf
	}

	recipient, err := d.userRepo.GetByID(ctx, recipientID)
	if err != nil {
		return nil, fmt.Errorf("get recipient: %w", err)
	}
	if recipient == nil {
		return nil, ErrUserNotFound
	}
	if !recipient.DmsEnabled {
		return nil, ErrDmsDisabled
	}

	if blocked, _ := d.blockSvc.IsBlockedEither(ctx, senderID, recipientID); blocked {
		return nil, ErrUserBlocked
	}
	return recipient, nil
}

func (d *dmService) ResolveDMRoom(ctx context.Context, senderID, recipientID uuid.UUID) (*dto.ResolveDMResponse, error) {
	recipient, err := d.checkDMPreconditions(ctx, senderID, recipientID)
	if err != nil {
		return nil, err
	}

	resp := &dto.ResolveDMResponse{
		Recipient: *recipient.ToResponse(),
	}

	existingID, err := d.chatRepo.FindDMRoom(ctx, senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("find dm room: %w", err)
	}
	if existingID == uuid.Nil {
		return resp, nil
	}

	room, err := d.parent.buildRoomResponse(ctx, existingID, senderID)
	if err != nil {
		return nil, err
	}
	resp.Room = room
	return resp, nil
}

func (d *dmService) SendDMMessage(ctx context.Context, senderID, recipientID uuid.UUID, body string, files []FileUpload) (*dto.SendDMResponse, error) {
	if body == "" && len(files) == 0 {
		return nil, ErrMissingFields
	}
	if body != "" {
		if err := d.filterTexts(ctx, body); err != nil {
			return nil, err
		}
	}

	if err := d.ensureLockAllowsDMTo(ctx, senderID, recipientID); err != nil {
		return nil, err
	}
	if _, err := d.checkDMPreconditions(ctx, senderID, recipientID); err != nil {
		return nil, err
	}

	roomID, err := d.chatRepo.CreateDMRoomAtomic(ctx, uuid.New(), senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("create dm room: %w", err)
	}

	msgResp, err := d.parent.SendMessage(ctx, senderID, roomID, dto.SendMessageRequest{Body: body}, files)
	if err != nil {
		return nil, err
	}

	roomResp, err := d.parent.buildRoomResponse(ctx, roomID, senderID)
	if err != nil {
		return nil, err
	}

	return &dto.SendDMResponse{
		Room:    *roomResp,
		Message: *msgResp,
	}, nil
}
