package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type membersService struct {
	*core
	parent *service
}

func (m *membersService) InviteMembers(ctx context.Context, hostID, roomID uuid.UUID, userIDs []uuid.UUID) (*dto.InviteMembersResponse, error) {
	row, err := m.loadRoomForMod(ctx, roomID, hostID)
	if err != nil {
		return nil, err
	}
	if row.Type != dto.RoomTypeGroup {
		return nil, ErrNotGroupRoom
	}

	cap := m.settingsSvc.GetInt(ctx, config.SettingMaxChatRoomMembers)
	memberCount := row.MemberCount

	existingMembers, _ := m.chatRepo.GetRoomMembers(ctx, roomID)
	inviterName := "Someone"
	if inviter, err := m.userRepo.GetByID(ctx, hostID); err == nil && inviter != nil {
		inviterName = inviter.DisplayName
	}

	invitedIDs := make([]uuid.UUID, 0, len(userIDs))
	seen := make(map[uuid.UUID]bool, len(userIDs))
	skipped := 0

	for _, targetID := range userIDs {
		if targetID == hostID || seen[targetID] {
			skipped++
			continue
		}
		seen[targetID] = true

		if cap > 0 && memberCount >= cap {
			skipped++
			continue
		}

		existingRole, err := m.chatRepo.GetMemberRole(ctx, roomID, targetID)
		if err != nil {
			return nil, fmt.Errorf("get member role: %w", err)
		}
		if existingRole != "" {
			skipped++
			continue
		}

		banned, err := m.banRepo.IsBanned(ctx, roomID, targetID)
		if err != nil {
			return nil, fmt.Errorf("check ban: %w", err)
		}
		if banned {
			skipped++
			continue
		}

		target, err := m.userRepo.GetByID(ctx, targetID)
		if err != nil || target == nil {
			skipped++
			continue
		}

		if blocked, _ := m.blockSvc.IsBlockedEither(ctx, hostID, targetID); blocked {
			skipped++
			continue
		}

		actionBody := m.roomActionMessageBody(ctx, roomID, hostID, fmt.Sprintf("%s invited %s to the room.", inviterName, target.DisplayName))

		actionRow, err := m.chatRepo.AddMemberWithSystemMessage(ctx,
			repository.NewChatRoomMember{RoomID: roomID, UserID: targetID, Role: "member", Ghost: false},
			repository.NewChatMessage{RoomID: roomID, SenderID: hostID, Body: actionBody, IsSystem: true},
		)
		if err != nil {
			return nil, fmt.Errorf("add member: %w", err)
		}

		m.hub.JoinRoom(roomID, targetID)
		memberCount++
		invitedIDs = append(invitedIDs, targetID)

		joinedEvent := ws.Message{
			Type: "chat_member_joined",
			Data: map[string]any{
				"room_id": roomID,
				"user":    target.ToResponse(),
			},
		}
		for _, mid := range existingMembers {
			m.hub.SendToUser(mid, joinedEvent)
		}
		m.hub.SendToUser(targetID, joinedEvent)
		existingMembers = append(existingMembers, targetID)

		m.broadcastRoomActionMessage(ctx, roomID, hostID, actionRow)
	}

	if len(invitedIDs) > 0 {
		go m.notifyInvited(hostID, roomID, row.Name, invitedIDs)
	}

	return &dto.InviteMembersResponse{
		InvitedCount: len(invitedIDs),
		SkippedCount: skipped,
	}, nil
}

func (m *membersService) KickMember(ctx context.Context, hostID, roomID, targetID uuid.UUID) error {
	if _, err := m.loadRoomForMod(ctx, roomID, hostID); err != nil {
		return err
	}

	targetRole, err := m.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return ErrNotMember
	}
	if targetRole == "host" {
		return ErrCannotKickHost
	}

	targetSiteRole, err := m.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get target site role: %w", err)
	}
	if targetSiteRole.IsSiteStaff() {
		return ErrTargetImmune
	}

	if err := m.parent.evictUserFromRoom(ctx, roomID, targetID, ""); err != nil {
		return err
	}

	m.postRoomActionMessage(ctx, roomID, hostID, "A member was kicked from the room.")

	m.parent.notifyModerationAction(roomID, targetID, hostID, "kicked", "")

	return nil
}

func (m *membersService) SetMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID, req dto.SetMemberTimeoutRequest) (*dto.ChatRoomMemberResponse, error) {
	if _, err := m.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return nil, err
	}

	targetRole, err := m.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return nil, ErrNotMember
	}

	actorSiteRole, err := m.authzSvc.GetRole(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("get actor site role: %w", err)
	}
	actorIsStaff := actorSiteRole.IsSiteStaff()
	if !actorIsStaff && targetRole == "host" {
		return nil, ErrCannotKickHost
	}

	targetSiteRole, err := m.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target site role: %w", err)
	}
	if targetSiteRole.IsSiteStaff() {
		return nil, ErrTargetImmune
	}

	activeTimeout, _, timeoutByStaff, err := m.chatRepo.GetMemberTimeoutState(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get timeout state: %w", err)
	}
	if activeTimeout && timeoutByStaff && !actorIsStaff {
		return nil, ErrTimeoutLockedByStaff
	}

	now := time.Now().UTC()
	until, label, err := computeTimeoutUntil(now, req.Amount, req.Unit)
	if err != nil {
		return nil, err
	}

	if err := m.chatRepo.SetMemberTimeout(ctx, roomID, targetID, until.Format(time.DateTime), actorIsStaff); err != nil {
		return nil, fmt.Errorf("set member timeout: %w", err)
	}

	actorName := m.actionDisplayName(ctx, actorID, "A moderator")
	targetName := m.actionDisplayName(ctx, targetID, "a member")
	m.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s timed out %s for %s.", actorName, targetName, label))

	return m.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (m *membersService) ClearMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	if _, err := m.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return nil, err
	}

	targetRole, err := m.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return nil, ErrNotMember
	}

	actorSiteRole, err := m.authzSvc.GetRole(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("get actor site role: %w", err)
	}
	actorIsStaff := actorSiteRole.IsSiteStaff()

	activeTimeout, _, timeoutByStaff, err := m.chatRepo.GetMemberTimeoutState(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get timeout state: %w", err)
	}
	if activeTimeout && timeoutByStaff && !actorIsStaff {
		return nil, ErrTimeoutLockedByStaff
	}

	if err := m.chatRepo.ClearMemberTimeout(ctx, roomID, targetID); err != nil {
		return nil, fmt.Errorf("clear member timeout: %w", err)
	}

	actorName := m.actionDisplayName(ctx, actorID, "A moderator")
	targetName := m.actionDisplayName(ctx, targetID, "a member")
	m.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s removed %s's timeout.", actorName, targetName))

	return m.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (m *membersService) GetMembers(ctx context.Context, viewerID, roomID uuid.UUID) ([]dto.ChatRoomMemberResponse, error) {
	isMember, err := m.chatRepo.IsMember(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, err := m.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	var viewerIsStaff bool
	hasGhost := false
	for i := range rows {
		if rows[i].Ghost {
			hasGhost = true
			break
		}
	}
	if hasGhost {
		r, _ := m.authzSvc.GetRole(ctx, viewerID)
		viewerIsStaff = r.IsSiteStaff()
	}

	userIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		if rows[i].Ghost && !viewerIsStaff {
			continue
		}
		userIDs = append(userIDs, rows[i].UserID)
	}
	vanityMap, _ := m.vanityRoleRepo.GetRolesForUsersBatch(ctx, userIDs)
	presence := m.hub.GetRoomPresence(roomID)

	members := make([]dto.ChatRoomMemberResponse, 0, len(rows))
	for i := range rows {
		mr := rows[i]
		if mr.Ghost && !viewerIsStaff {
			continue
		}
		state := presence[mr.UserID]
		if state == "" && m.hub.IsAlwaysOnline(mr.UserID) {
			state = ws.ViewerStateActive
		}

		resp := m.memberRowToMemberResponse(mr, m.toVanityRoleResponses(vanityMap[mr.UserID]), state)
		resp.Ghost = mr.Ghost
		members = append(members, resp)
	}
	return members, nil
}

func (m *membersService) SetMemberNicknameAsMod(ctx context.Context, roomID, actorID, targetID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error) {
	if err := m.requireSiteMod(ctx, actorID); err != nil {
		return nil, err
	}

	if err := m.assertTargetEditable(ctx, roomID, targetID); err != nil {
		return nil, err
	}

	if err := m.filterTexts(ctx, nickname); err != nil {
		return nil, err
	}

	nickname = strings.TrimSpace(nickname)
	if len(nickname) > 32 {
		nickname = nickname[:32]
	}

	locked := nickname != ""
	if err := m.chatRepo.SetMemberNicknameWithLock(ctx, roomID, targetID, nickname, locked); err != nil {
		return nil, fmt.Errorf("set member nickname as mod: %w", err)
	}

	targetName, targetPoss := m.nameAndPossessive(ctx, targetID)
	actorName, _ := m.nameAndPossessive(ctx, actorID)
	if targetName != "" && actorName != "" {
		m.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s has had %s alias locked by %s.", targetName, targetPoss, actorName))
	}

	return m.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (m *membersService) UnlockMemberNickname(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	if err := m.requireSiteMod(ctx, actorID); err != nil {
		return nil, err
	}

	if err := m.assertTargetEditable(ctx, roomID, targetID); err != nil {
		return nil, err
	}

	if err := m.chatRepo.SetMemberNicknameWithLock(ctx, roomID, targetID, "", false); err != nil {
		return nil, fmt.Errorf("unlock nickname: %w", err)
	}

	targetName, targetPoss := m.nameAndPossessive(ctx, targetID)
	actorName, _ := m.nameAndPossessive(ctx, actorID)
	if targetName != "" && actorName != "" {
		m.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s has had %s alias reset by %s.", targetName, targetPoss, actorName))
	}

	return m.broadcastAndBuildMember(ctx, roomID, targetID)
}
