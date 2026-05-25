package chat

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type roomsService struct {
	*core
}

func (r *roomsService) CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrMissingFields
	}
	if err := r.filterTexts(ctx, name, req.Description); err != nil {
		return nil, err
	}
	if len(name) > 80 {
		name = name[:80]
	}
	description := strings.TrimSpace(req.Description)
	if len(description) > 500 {
		description = description[:500]
	}
	tags := sanitizeTags(req.Tags)

	roomID := uuid.New()
	if err := r.chatRepo.CreateRoom(ctx, roomID, name, description, "group", req.IsPublic, req.IsRP, creatorID); err != nil {
		return nil, fmt.Errorf("create group room: %w", err)
	}
	if len(tags) > 0 {
		if err := r.chatRepo.AddRoomTags(ctx, roomID, tags); err != nil {
			return nil, fmt.Errorf("add room tags: %w", err)
		}
	}
	if err := r.chatRepo.AddMemberWithRole(ctx, roomID, creatorID, "host", false); err != nil {
		return nil, fmt.Errorf("add creator to group: %w", err)
	}

	invitedIDs := make([]uuid.UUID, 0, len(req.MemberIDs))
	for _, memberID := range req.MemberIDs {
		if memberID == creatorID {
			continue
		}
		if blocked, _ := r.blockSvc.IsBlockedEither(ctx, creatorID, memberID); blocked {
			continue
		}
		if err := r.chatRepo.AddMemberWithRole(ctx, roomID, memberID, "member", false); err != nil {
			return nil, fmt.Errorf("add member to group: %w", err)
		}
		invitedIDs = append(invitedIDs, memberID)
	}

	if len(invitedIDs) > 0 {
		go r.notifyInvited(creatorID, roomID, name, invitedIDs)
	}

	return r.buildRoomResponse(ctx, roomID, creatorID)
}

func (r *roomsService) ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, includeArchived bool, limit, offset int) (*dto.ChatRoomListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	blockedIDs, _ := r.blockSvc.GetBlockedIDs(ctx, viewerID)
	tag = strings.ToLower(strings.TrimSpace(tag))
	rows, total, err := r.chatRepo.ListPublicRooms(ctx, search, isRPOnly, tag, viewerID, blockedIDs, includeArchived, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public rooms: %w", err)
	}

	bannedRoomIDs, _ := r.banRepo.BannedRoomIDsForUser(ctx, viewerID)
	bannedSet := make(map[uuid.UUID]struct{}, len(bannedRoomIDs))
	for _, id := range bannedRoomIDs {
		bannedSet[id] = struct{}{}
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := range rows {
		if _, banned := bannedSet[rows[i].ID]; banned {
			if total > 0 {
				total--
			}
			continue
		}
		room := r.rowToResponse(rows[i])
		rooms = append(rooms, room)
	}
	return &dto.ChatRoomListResponse{Rooms: rooms, Total: total}, nil
}

func (r *roomsService) ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, roleFilter string, includeArchived bool, limit, offset int) (*dto.ChatRoomListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if roleFilter != "host" && roleFilter != "member" {
		roleFilter = ""
	}

	tag = strings.ToLower(strings.TrimSpace(tag))
	rows, total, err := r.chatRepo.ListUserGroupRooms(ctx, userID, search, isRPOnly, tag, roleFilter, includeArchived, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user group rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := range rows {
		rooms = append(rooms, r.rowToResponse(rows[i]))
	}
	return &dto.ChatRoomListResponse{Rooms: rooms, Total: total}, nil
}

func (r *roomsService) SetRoomMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error {
	isMember, err := r.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := r.chatRepo.SetMuted(ctx, roomID, userID, muted); err != nil {
		return fmt.Errorf("set muted: %w", err)
	}
	return nil
}

func (r *roomsService) IsRoomMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.chatRepo.IsMuted(ctx, roomID, userID)
}

func (r *roomsService) JoinRoom(ctx context.Context, roomID, userID uuid.UUID, ghost bool) (*dto.ChatRoomResponse, error) {
	row, err := r.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	if row.Type != "group" {
		return nil, ErrNotGroupRoom
	}
	if row.IsSystem {
		return nil, ErrSystemRoom
	}
	if !row.IsPublic {
		return nil, ErrNotPublic
	}

	banned, err := r.banRepo.IsBanned(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check ban: %w", err)
	}
	if banned {
		return nil, ErrBannedFromRoom
	}

	if ghost {
		viewerSiteRole, err := r.authzSvc.GetRole(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get site role: %w", err)
		}
		if !viewerSiteRole.IsSiteStaff() {
			return nil, ErrGhostRequiresStaff
		}
	}
	if row.IsMember {
		return r.buildRoomResponse(ctx, roomID, userID)
	}

	if blocked, _ := r.blockSvc.IsBlockedEither(ctx, userID, row.CreatedBy); blocked {
		return nil, ErrUserBlocked
	}

	cap := r.settingsSvc.GetInt(ctx, config.SettingMaxChatRoomMembers)
	if cap > 0 && row.MemberCount >= cap {
		return nil, ErrRoomFull
	}

	if err := r.chatRepo.AddMemberWithRole(ctx, roomID, userID, "member", ghost); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	resp, err := r.buildRoomResponse(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	members, _ := r.chatRepo.GetRoomMembers(ctx, roomID)
	joiner, _ := r.userRepo.GetByID(ctx, userID)
	if joiner != nil {
		event := ws.Message{
			Type: "chat_member_joined",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user":    joiner.ToResponse(),
				"ghost":   ghost,
			},
		}
		if ghost {
			r.broadcastToStaff(ctx, members, event)
		} else {
			r.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s joined the room.", joiner.DisplayName))
			for _, mid := range members {
				r.hub.SendToUser(mid, event)
			}
		}
	}
	return resp, nil
}

func (r *roomsService) broadcastToStaff(ctx context.Context, memberIDs []uuid.UUID, msg ws.Message) {
	for _, mid := range memberIDs {
		role, err := r.authzSvc.GetRole(ctx, mid)
		if err != nil {
			continue
		}
		if role.IsSiteStaff() {
			r.hub.SendToUser(mid, msg)
		}
	}
}

func (r *roomsService) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := r.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil || !row.IsMember {
		return ErrNotMember
	}
	if row.IsSystem {
		return ErrSystemRoom
	}
	if row.ViewerRole == "host" {
		return ErrCannotLeaveAsHost
	}

	members, _ := r.chatRepo.GetRoomMembers(ctx, roomID)
	var wasGhost bool
	if hasGhost, _ := r.chatRepo.HasGhostMembers(ctx, roomID); hasGhost {
		wasGhost, _ = r.chatRepo.IsGhostMember(ctx, roomID, userID)
	}

	if err := r.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	leaver, _ := r.userRepo.GetByID(ctx, userID)
	if leaver != nil {
		if !wasGhost {
			r.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s left the room.", leaver.DisplayName))
		}
		event := ws.Message{
			Type: "chat_member_left",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user_id": userID,
				"ghost":   wasGhost,
			},
		}
		if wasGhost {
			r.broadcastToStaff(ctx, members, event)
		} else {
			for _, mid := range members {
				r.hub.SendToUser(mid, event)
			}
		}
	}
	return nil
}

func (r *roomsService) ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error) {
	rows, err := r.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		members, count, err := r.getRoomMemberResponses(ctx, row.ID, userID)
		if err != nil {
			return nil, err
		}
		resp := r.rowToResponse(row)
		resp.Members = members
		resp.MemberCount = count
		rooms = append(rooms, resp)
	}

	return &dto.ChatRoomListResponse{Rooms: rooms}, nil
}

func (r *roomsService) ArchiveStale(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	ids, err := r.chatRepo.ArchiveStaleGroupRooms(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("archive stale chat rooms: %w", err)
	}
	return len(ids), nil
}

func (r *roomsService) DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := r.chatRepo.GetRoomByID(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return ErrNotMember
	}
	if row.IsSystem {
		return ErrSystemRoom
	}

	canMod := false
	if row.Type == "group" {
		mod, modErr := r.canModerateRoom(ctx, roomID, userID)
		if modErr != nil {
			return modErr
		}
		canMod = mod
	}
	if !row.IsMember && !canMod {
		return ErrNotMember
	}

	if row.Type == "group" && canMod {
		members, _ := r.chatRepo.GetRoomMembers(ctx, roomID)
		if err := r.chatRepo.DeleteMessages(ctx, roomID); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := r.chatRepo.DeleteRoom(ctx, roomID); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}
		event := ws.Message{
			Type: "chat_room_deleted",
			Data: map[string]interface{}{
				"room_id": roomID,
			},
		}
		for _, mid := range members {
			r.hub.SendToUser(mid, event)
		}
		return nil
	}

	if err := r.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	remaining, err := r.chatRepo.CountRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("count remaining members: %w", err)
	}
	if remaining == 0 {
		if err := r.chatRepo.DeleteMessages(ctx, roomID); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := r.chatRepo.DeleteRoom(ctx, roomID); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}
	}

	return nil
}

func (r *roomsService) buildRoomResponse(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatRoomResponse, error) {
	row, err := r.chatRepo.GetRoomByID(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrNotMember
	}

	members, count, err := r.getRoomMemberResponses(ctx, roomID, viewerID)
	if err != nil {
		return nil, err
	}

	resp := r.rowToResponse(*row)
	resp.Members = members
	resp.MemberCount = count
	return &resp, nil
}

func (r *roomsService) SetRoomNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error) {
	if err := r.filterTexts(ctx, nickname); err != nil {
		return nil, err
	}

	isMember, err := r.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	locked, err := r.effectiveLocked(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, ErrNicknameLocked
	}

	nickname = strings.TrimSpace(nickname)
	if len(nickname) > 32 {
		nickname = nickname[:32]
	}

	if err := r.chatRepo.SetMemberNickname(ctx, roomID, userID, nickname); err != nil {
		return nil, fmt.Errorf("set member nickname: %w", err)
	}

	name, possessive := r.nameAndPossessive(ctx, userID)
	if name != "" {
		if nickname == "" {
			r.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s cleared %s alias.", name, possessive))
		} else {
			r.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s changed %s alias.", name, possessive))
		}
	}

	return r.broadcastAndBuildMember(ctx, roomID, userID)
}

func (r *roomsService) SetRoomAvatar(ctx context.Context, roomID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.ChatRoomMemberResponse, error) {
	isMember, err := r.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	locked, err := r.effectiveLocked(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, ErrNicknameLocked
	}

	maxSize := int64(r.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	subDir := fmt.Sprintf("chat-avatars/%s", roomID.String())
	avatarURL, err := r.uploadSvc.SaveImage(ctx, subDir, userID, fileSize, maxSize, reader)
	if err != nil {
		return nil, err
	}

	if err := r.chatRepo.SetMemberAvatar(ctx, roomID, userID, avatarURL); err != nil {
		return nil, fmt.Errorf("set member avatar: %w", err)
	}

	return r.broadcastAndBuildMember(ctx, roomID, userID)
}

func (r *roomsService) ClearRoomAvatar(ctx context.Context, roomID, userID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	isMember, err := r.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	locked, err := r.effectiveLocked(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, ErrNicknameLocked
	}

	rows, err := r.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	if err == nil {
		for _, row := range rows {
			if row.UserID == userID && row.MemberAvatarURL != "" {
				_ = r.uploadSvc.Delete(row.MemberAvatarURL)
				break
			}
		}
	}

	if err := r.chatRepo.SetMemberAvatar(ctx, roomID, userID, ""); err != nil {
		return nil, fmt.Errorf("clear member avatar: %w", err)
	}

	return r.broadcastAndBuildMember(ctx, roomID, userID)
}
