package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const (
	SystemKindLiveStream = "live_stream"
	SystemKindWatchParty = "watch_party"
)

type streamChatService struct {
	*core
}

func (s *streamChatService) CreateStreamRoom(ctx context.Context, streamID, streamerID uuid.UUID, title string) error {
	name := title
	if name == "" {
		name = "Live stream"
	}

	if err := s.chatRepo.CreateSystemRoom(ctx, streamID, name, "", SystemKindLiveStream, streamerID); err != nil {
		return fmt.Errorf("create stream chat room: %w", err)
	}

	if err := s.chatRepo.AddMemberWithRole(ctx, streamID, streamerID, "host", false); err != nil {
		return fmt.Errorf("add streamer to chat room: %w", err)
	}

	s.hub.JoinRoom(streamID, streamerID)

	return nil
}

func (s *streamChatService) JoinStreamChat(ctx context.Context, streamID, userID uuid.UUID) error {
	room, err := s.chatRepo.GetRoomByID(ctx, streamID, userID)
	if err != nil {
		return fmt.Errorf("get stream chat room: %w", err)
	}
	if room == nil || !room.IsSystem || room.SystemKind != SystemKindLiveStream {
		return ErrRoomNotFound
	}

	alreadyMember, err := s.chatRepo.IsMember(ctx, streamID, userID)
	if err != nil {
		return fmt.Errorf("check stream chat membership: %w", err)
	}

	if !alreadyMember {
		if err := s.chatRepo.AddMemberWithRole(ctx, streamID, userID, "member", false); err != nil {
			return fmt.Errorf("join stream chat room: %w", err)
		}
	}

	s.hub.JoinRoom(streamID, userID)

	return nil
}

func (s *streamChatService) DeleteStreamRoom(ctx context.Context, streamID uuid.UUID) error {
	if err := s.deleteRoomWithMedia(ctx, streamID); err != nil {
		return fmt.Errorf("delete stream chat room: %w", err)
	}

	return nil
}
