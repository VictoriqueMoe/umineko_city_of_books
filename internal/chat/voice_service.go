package chat

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	wsVoicePresence = "voice_presence"

	voiceSessionRoomPrefix = "wp_"
)

type voiceService struct {
	*core

	mu       sync.Mutex
	presence map[uuid.UUID]map[uuid.UUID]any
}

func newVoiceService(c *core) *voiceService {
	return &voiceService{
		core:     c,
		presence: make(map[uuid.UUID]map[uuid.UUID]any),
	}
}

func (s *voiceService) VoiceEnabled() bool {
	return s.settingsSvc.GetBool(context.Background(), config.SettingVoiceEnabled) && s.livekitSvc.Enabled()
}

func (s *voiceService) MintVoiceToken(ctx context.Context, roomID, userID uuid.UUID) (token, url string, err error) {
	if !s.VoiceEnabled() {
		return "", "", ErrVoiceDisabled
	}

	room, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return "", "", fmt.Errorf("get room: %w", err)
	}
	if room == nil {
		return "", "", ErrRoomNotFound
	}

	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return "", "", fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return "", "", ErrNotMember
	}

	if room.Type == "dm" {
		if err := s.assertDMNotBlocked(ctx, roomID, userID); err != nil {
			return "", "", err
		}
	}

	displayName := s.displayNameFor(ctx, userID, roomID)
	canMic := !s.isVoiceMuted(roomID.String(), userID)

	token, err = s.livekitSvc.MintToken(roomID.String(), userID.String(), displayName, canMic, false)
	if err != nil {
		return "", "", err
	}

	return token, s.livekitSvc.URL(), nil
}

func (s *voiceService) ForceMuteVoice(ctx context.Context, roomID, actorID, targetID uuid.UUID, muted bool) error {
	if !s.VoiceEnabled() {
		return ErrVoiceDisabled
	}

	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrVoiceMuteForbidden
	}

	s.setVoiceMuted(roomID.String(), targetID, muted)

	return s.livekitSvc.SetCanPublish(ctx, roomID.String(), targetID.String(), !muted, false)
}

func (s *voiceService) assertDMNotBlocked(ctx context.Context, roomID, userID uuid.UUID) error {
	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room members: %w", err)
	}

	for i := range members {
		if members[i] == userID {
			continue
		}

		if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, members[i]); blocked {
			return ErrUserBlocked
		}
	}

	return nil
}

func (s *voiceService) HandleVoiceWebhook(ctx context.Context, authHeader string, body []byte) error {
	event, err := s.livekitSvc.ParseWebhook(authHeader, body)
	if err != nil {
		return err
	}

	logger.Log.Debug().Str("event", event.Type).Str("room", event.RoomName).Str("identity", event.Identity).Msg("livekit webhook received")

	if strings.HasPrefix(event.RoomName, voiceSessionRoomPrefix) {
		return nil
	}

	roomID, err := uuid.Parse(event.RoomName)
	if err != nil {
		return nil
	}

	switch event.Type {
	case livekit.EventParticipantJoined:
		userID, err := uuid.Parse(event.Identity)
		if err != nil {
			return nil
		}
		s.addParticipant(roomID, userID)
		s.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s joined the voice chat.", s.displayNameFor(ctx, userID, roomID)))
		s.broadcastVoicePresence(ctx, roomID)

	case livekit.EventParticipantLeft:
		userID, err := uuid.Parse(event.Identity)
		if err != nil {
			return nil
		}
		s.removeParticipant(roomID, userID)
		s.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s left the voice chat.", s.displayNameFor(ctx, userID, roomID)))
		s.broadcastVoicePresence(ctx, roomID)

	case livekit.EventRoomFinished:
		s.clearRoom(roomID)
		s.clearVoiceMuted(roomID.String())
		s.broadcastVoicePresence(ctx, roomID)
	}

	return nil
}

func (s *voiceService) VoiceParticipants(roomID uuid.UUID) []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]uuid.UUID, 0, len(s.presence[roomID]))
	for id := range s.presence[roomID] {
		ids = append(ids, id)
	}

	slices.SortFunc(ids, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	return ids
}

func (s *voiceService) VoiceCount(roomID uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.presence[roomID])
}

func (s *voiceService) addParticipant(roomID, userID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.presence[roomID] == nil {
		s.presence[roomID] = make(map[uuid.UUID]any)
	}
	s.presence[roomID][userID] = struct{}{}
}

func (s *voiceService) removeParticipant(roomID, userID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.presence[roomID], userID)
	if len(s.presence[roomID]) == 0 {
		delete(s.presence, roomID)
	}
}

func (s *voiceService) clearRoom(roomID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.presence, roomID)
}

func (s *voiceService) ReconcilePresence(ctx context.Context) (int, error) {
	if !s.VoiceEnabled() {
		return 0, nil
	}

	rooms, err := s.livekitSvc.ActiveRooms(ctx)
	if err != nil {
		return 0, fmt.Errorf("reconcile voice presence: %w", err)
	}

	next := make(map[uuid.UUID]map[uuid.UUID]any, len(rooms))
	for name, identities := range rooms {
		roomID, err := uuid.Parse(name)
		if err != nil {
			continue
		}

		members := make(map[uuid.UUID]any, len(identities))
		for i := range identities {
			userID, err := uuid.Parse(identities[i])
			if err != nil {
				continue
			}
			members[userID] = struct{}{}
		}

		if len(members) > 0 {
			next[roomID] = members
		}
	}

	affected := s.swapPresence(next)

	for i := range affected {
		s.broadcastVoicePresence(ctx, affected[i])
	}

	return len(affected), nil
}

func (s *voiceService) swapPresence(next map[uuid.UUID]map[uuid.UUID]any) []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev := s.presence
	s.presence = next

	roomIDs := make(map[uuid.UUID]any, len(prev)+len(next))
	for id := range prev {
		roomIDs[id] = struct{}{}
	}
	for id := range next {
		roomIDs[id] = struct{}{}
	}

	changed := make([]uuid.UUID, 0)
	for roomID := range roomIDs {
		if !sameMemberSet(prev[roomID], next[roomID]) {
			changed = append(changed, roomID)
		}
	}

	return changed
}

func sameMemberSet(a, b map[uuid.UUID]any) bool {
	if len(a) != len(b) {
		return false
	}

	for id := range a {
		if _, ok := b[id]; !ok {
			return false
		}
	}

	return true
}

func (s *voiceService) broadcastVoicePresence(ctx context.Context, roomID uuid.UUID) {
	participants := s.VoiceParticipants(roomID)

	event := ws.Message{
		Type: wsVoicePresence,
		Data: dto.VoicePresenceEvent{
			RoomID:       roomID,
			Participants: participants,
			Count:        len(participants),
		},
	}

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return
	}

	for i := range members {
		s.hub.SendToUser(members[i], event)
	}
}
