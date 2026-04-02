package chat

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		GetOrCreateDMRoom(ctx context.Context, senderID uuid.UUID, req dto.CreateDMRequest) (*dto.ChatRoomResponse, error)
		CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error)
		ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error)
		GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error)

		SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest) (*dto.ChatMessageResponse, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error
	}

	service struct {
		chatRepo  repository.ChatRepository
		userRepo  repository.UserRepository
		notifRepo repository.NotificationRepository
		hub       *ws.Hub
	}
)

func NewService(chatRepo repository.ChatRepository, userRepo repository.UserRepository, notifRepo repository.NotificationRepository, hub *ws.Hub) Service {
	return &service{
		chatRepo:  chatRepo,
		userRepo:  userRepo,
		notifRepo: notifRepo,
		hub:       hub,
	}
}

func (s *service) GetOrCreateDMRoom(ctx context.Context, senderID uuid.UUID, req dto.CreateDMRequest) (*dto.ChatRoomResponse, error) {
	if senderID == req.RecipientID {
		return nil, ErrCannotDMSelf
	}

	recipient, err := s.userRepo.GetByID(ctx, req.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("get recipient: %w", err)
	}
	if recipient == nil {
		return nil, ErrUserNotFound
	}
	if !recipient.DmsEnabled {
		return nil, ErrDmsDisabled
	}

	existingID, err := s.chatRepo.FindDMRoom(ctx, senderID, req.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("find dm room: %w", err)
	}
	if existingID != uuid.Nil {
		return s.buildRoomResponse(ctx, existingID)
	}

	roomID := uuid.New()
	if err := s.chatRepo.CreateRoom(ctx, roomID, "", "dm", senderID); err != nil {
		return nil, fmt.Errorf("create dm room: %w", err)
	}
	if err := s.chatRepo.AddMember(ctx, roomID, senderID); err != nil {
		return nil, fmt.Errorf("add sender to dm: %w", err)
	}
	if err := s.chatRepo.AddMember(ctx, roomID, req.RecipientID); err != nil {
		return nil, fmt.Errorf("add recipient to dm: %w", err)
	}

	return s.buildRoomResponse(ctx, roomID)
}

func (s *service) CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error) {
	if req.Name == "" {
		return nil, ErrMissingFields
	}

	roomID := uuid.New()
	if err := s.chatRepo.CreateRoom(ctx, roomID, req.Name, "group", creatorID); err != nil {
		return nil, fmt.Errorf("create group room: %w", err)
	}
	if err := s.chatRepo.AddMember(ctx, roomID, creatorID); err != nil {
		return nil, fmt.Errorf("add creator to group: %w", err)
	}

	for _, memberID := range req.MemberIDs {
		if memberID == creatorID {
			continue
		}
		if err := s.chatRepo.AddMember(ctx, roomID, memberID); err != nil {
			return nil, fmt.Errorf("add member to group: %w", err)
		}
	}

	return s.buildRoomResponse(ctx, roomID)
}

func (s *service) ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error) {
	rows, err := s.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}

	var rooms []dto.ChatRoomResponse
	for _, row := range rows {
		members, err := s.getRoomMemberResponses(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, dto.ChatRoomResponse{
			ID:        row.ID,
			Name:      row.Name,
			Type:      row.Type,
			Members:   members,
			CreatedAt: row.CreatedAt,
		})
	}

	if rooms == nil {
		rooms = []dto.ChatRoomResponse{}
	}

	return &dto.ChatRoomListResponse{Rooms: rooms}, nil
}

func (s *service) GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, total, err := s.chatRepo.GetMessages(ctx, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	var messages []dto.ChatMessageResponse
	for _, row := range rows {
		messages = append(messages, dto.ChatMessageResponse{
			ID:     row.ID,
			RoomID: row.RoomID,
			Sender: dto.UserResponse{
				ID:          row.SenderID,
				Username:    row.SenderUsername,
				DisplayName: row.SenderDisplayName,
				AvatarURL:   row.SenderAvatarURL,
			},
			Body:      row.Body,
			CreatedAt: row.CreatedAt,
		})
	}

	if messages == nil {
		messages = []dto.ChatMessageResponse{}
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (s *service) SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest) (*dto.ChatMessageResponse, error) {
	if req.Body == "" {
		return nil, ErrMissingFields
	}

	isMember, err := s.chatRepo.IsMember(ctx, roomID, senderID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	sender, err := s.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrUserNotFound
	}

	msgID := uuid.New()
	if err := s.chatRepo.InsertMessage(ctx, msgID, roomID, senderID, req.Body); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	resp := &dto.ChatMessageResponse{
		ID:     msgID,
		RoomID: roomID,
		Sender: dto.UserResponse{
			ID:          sender.ID,
			Username:    sender.Username,
			DisplayName: sender.DisplayName,
			AvatarURL:   sender.AvatarURL,
		},
		Body:      req.Body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err == nil {
		msg := ws.Message{
			Type: "chat_message",
			Data: resp,
		}
		for _, memberID := range members {
			if memberID != senderID {
				s.hub.SendToUser(memberID, msg)
				s.notifRepo.Create(ctx, memberID, dto.NotifChatMessage, roomID, "chat", senderID, "")
			}
		}
	}

	return resp, nil
}

func (s *service) GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user: %w", err)
	}

	var roomIDs []uuid.UUID
	for _, row := range rows {
		roomIDs = append(roomIDs, row.ID)
	}
	return roomIDs, nil
}

func (s *service) DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := s.chatRepo.DeleteMessages(ctx, roomID); err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}

	return s.chatRepo.RemoveMember(ctx, roomID, userID)
}

func (s *service) buildRoomResponse(ctx context.Context, roomID uuid.UUID) (*dto.ChatRoomResponse, error) {
	rows, err := s.chatRepo.GetRoomsByUser(ctx, uuid.Nil)
	if err != nil {
		rows = nil
	}

	_ = rows

	var name, roomType, createdAt string
	members, err := s.getRoomMemberResponses(ctx, roomID)
	if err != nil {
		return nil, err
	}

	allRooms, err := s.chatRepo.GetRoomsByUser(ctx, members[0].ID)
	if err != nil {
		return nil, fmt.Errorf("get rooms: %w", err)
	}
	for _, r := range allRooms {
		if r.ID == roomID {
			name = r.Name
			roomType = r.Type
			createdAt = r.CreatedAt
			break
		}
	}

	return &dto.ChatRoomResponse{
		ID:        roomID,
		Name:      name,
		Type:      roomType,
		Members:   members,
		CreatedAt: createdAt,
	}, nil
}

func (s *service) getRoomMemberResponses(ctx context.Context, roomID uuid.UUID) ([]dto.UserResponse, error) {
	memberIDs, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}

	var members []dto.UserResponse
	for _, memberID := range memberIDs {
		user, err := s.userRepo.GetByID(ctx, memberID)
		if err != nil || user == nil {
			continue
		}
		members = append(members, *user.ToResponse())
	}
	return members, nil
}
