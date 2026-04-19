package chat

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) lookupDisplayName(ctx context.Context, userID uuid.UUID) string {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return "A member"
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return user.Username
}

func (s *service) isTargetProtected(ctx context.Context, roomID, targetID uuid.UUID) (bool, error) {
	memberRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return false, fmt.Errorf("get target role: %w", err)
	}
	if memberRole == "host" {
		return true, nil
	}
	siteRole, err := s.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return false, fmt.Errorf("get target site role: %w", err)
	}
	return siteRole.IsSiteStaff(), nil
}

func (s *service) enforceBannedWords(ctx context.Context, roomID, senderID uuid.UUID, body string) error {
	if body == "" {
		return nil
	}
	match, err := s.bannedWordsRule.CheckForRoom(ctx, roomID, body)
	if err != nil {
		return err
	}
	if match == nil {
		return nil
	}
	immune, err := s.canModerateRoom(ctx, roomID, senderID)
	if err != nil {
		return err
	}
	if immune {
		return nil
	}
	details := fmt.Sprintf("room=%s user=%s pattern=%q match=%q", roomID, senderID, match.Pattern, match.MatchedOn)
	_ = s.auditRepo.CreateSystem(ctx, "chat_word_filter_"+match.Action, "chat_room", roomID.String(), details)
	if match.Action == contentfilter.BannedWordActionKick {
		targetName := s.lookupDisplayName(ctx, senderID)
		s.postRoomActionMessage(ctx, roomID, senderID, fmt.Sprintf("%s was kicked by the word filter.", targetName))
		_ = s.evictUserFromRoom(ctx, roomID, senderID, "the word filter matched a banned word")
		s.notifyAutomatedKick(roomID, senderID, match.Pattern)
	}
	return &ErrBannedWordMatch{Pattern: match.Pattern, Action: match.Action}
}

func (s *service) evictUserFromRoom(ctx context.Context, roomID, targetID uuid.UUID, reason string) error {
	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)

	if err := s.chatRepo.RemoveMember(ctx, roomID, targetID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	s.hub.LeaveRoom(roomID, targetID)

	leftEvent := ws.Message{
		Type: "chat_member_left",
		Data: map[string]interface{}{
			"room_id": roomID,
			"user_id": targetID,
		},
	}
	for _, mid := range members {
		if mid == targetID {
			continue
		}
		s.hub.SendToUser(mid, leftEvent)
	}

	kickData := map[string]interface{}{
		"room_id": roomID,
	}
	if reason != "" {
		kickData["reason"] = reason
	}
	s.hub.SendToUser(targetID, ws.Message{Type: "chat_kicked", Data: kickData})

	return nil
}

func (s *service) banUserFromRoom(ctx context.Context, roomID, targetID uuid.UUID, actorID *uuid.UUID, reason string) error {
	if err := s.banRepo.Ban(ctx, roomID, targetID, actorID, reason); err != nil {
		return err
	}

	if err := s.evictUserFromRoom(ctx, roomID, targetID, reason); err != nil {
		return err
	}

	if actorID != nil {
		s.notifyModerationAction(roomID, targetID, *actorID, "banned", reason)
	}

	return nil
}

func (s *service) notifyModerationAction(roomID, targetID, actorID uuid.UUID, action, reason string) {
	roomName := s.lookupRoomName(context.Background(), roomID)
	var message string
	if strings.TrimSpace(reason) != "" {
		message = fmt.Sprintf("%s you from %s because %s.", action, roomName, strings.TrimSpace(reason))
	} else {
		message = fmt.Sprintf("%s you from %s.", action, roomName)
	}
	var notifType dto.NotificationType
	var refType string
	switch action {
	case "kicked":
		notifType = dto.NotifChatRoomKicked
		refType = "chat_room_kick"
	case "unbanned":
		notifType = dto.NotifChatRoomUnbanned
		refType = "chat_room_unban"
	default:
		notifType = dto.NotifChatRoomBanned
		refType = "chat_room_ban"
	}
	go func(msg string) {
		s.notifSvc.Notify(context.Background(), dto.NotifyParams{
			RecipientID:   targetID,
			ActorID:       actorID,
			Type:          notifType,
			ReferenceID:   roomID,
			ReferenceType: refType,
			Message:       msg,
		})
	}(message)
}

func (s *service) notifyAutomatedKick(roomID, targetID uuid.UUID, pattern string) {
	roomName := s.lookupRoomName(context.Background(), roomID)
	message := fmt.Sprintf("You were kicked from %s by the word filter.", roomName)
	go func(msg string) {
		s.notifSvc.Notify(context.Background(), dto.NotifyParams{
			RecipientID:   targetID,
			ActorID:       uuid.Nil,
			Type:          dto.NotifChatRoomKicked,
			ReferenceID:   roomID,
			ReferenceType: "chat_room_kick",
			Message:       msg,
		})
	}(message)
	_ = pattern
}

func (s *service) lookupRoomName(ctx context.Context, roomID uuid.UUID) string {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, uuid.Nil)
	if err != nil || row == nil || row.Name == "" {
		return "the chat room"
	}
	return row.Name
}

func (s *service) BanMember(ctx context.Context, actorID, roomID, targetID uuid.UUID, reason string) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return ErrRoomNotFound
	}
	if row.IsSystem {
		return ErrSystemRoom
	}

	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}

	if actorID == targetID {
		return ErrCannotBanStaff
	}

	protected, err := s.isTargetProtected(ctx, roomID, targetID)
	if err != nil {
		return err
	}
	if protected {
		return ErrCannotBanStaff
	}

	targetName := s.lookupDisplayName(ctx, targetID)
	var message string
	trimmed := strings.TrimSpace(reason)
	if trimmed != "" {
		message = fmt.Sprintf("%s was banned because %s.", targetName, trimmed)
	} else {
		message = fmt.Sprintf("%s was banned.", targetName)
	}
	s.postRoomActionMessage(ctx, roomID, actorID, message)

	if err := s.banUserFromRoom(ctx, roomID, targetID, &actorID, reason); err != nil {
		return err
	}

	details := fmt.Sprintf("target=%s reason=%s", targetID, reason)
	if err := s.auditRepo.Create(ctx, actorID, "chat_room_ban", "chat_room", roomID.String(), details); err != nil {
		return fmt.Errorf("audit ban: %w", err)
	}
	return nil
}

func (s *service) UnbanMember(ctx context.Context, actorID, roomID, targetID uuid.UUID) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return ErrRoomNotFound
	}

	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}

	if err := s.banRepo.Unban(ctx, roomID, targetID); err != nil {
		return err
	}

	targetName := s.lookupDisplayName(ctx, targetID)
	s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s was unbanned.", targetName))

	s.notifyModerationAction(roomID, targetID, actorID, "unbanned", "")

	return s.auditRepo.Create(ctx, actorID, "chat_room_unban", "chat_room", roomID.String(), "target="+targetID.String())
}

func (s *service) ListRoomBans(ctx context.Context, actorID, roomID uuid.UUID) ([]dto.ChatRoomBanResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}

	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}

	rows, err := s.banRepo.ListForRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ChatRoomBanResponse, len(rows))
	for i, r := range rows {
		entry := dto.ChatRoomBanResponse{
			User: dto.UserResponse{
				ID:          r.UserID,
				Username:    r.Username,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Role:        role.Role(r.Role),
			},
			Reason:    r.Reason,
			CreatedAt: r.CreatedAt,
		}
		if r.BannedByID != nil {
			entry.BannedBy = &dto.UserResponse{
				ID:          *r.BannedByID,
				Username:    r.BannedByUsername,
				DisplayName: r.BannedByDisplay,
				AvatarURL:   r.BannedByAvatarURL,
			}
		}
		out[i] = entry
	}
	return out, nil
}

func validateCreateBannedWord(req dto.CreateBannedWordRequest) error {
	trimmed := strings.TrimSpace(req.Pattern)
	if trimmed == "" {
		return ErrMissingFields
	}
	if err := contentfilter.ValidateBannedWordMode(req.MatchMode); err != nil {
		return ErrInvalidBannedWordMode
	}
	if err := contentfilter.ValidateBannedWordAction(req.Action); err != nil {
		return ErrInvalidBannedWordAction
	}
	if _, err := contentfilter.CompileBannedWordPattern(trimmed, req.MatchMode, req.CaseSensitive); err != nil {
		return ErrInvalidBannedWordRegex
	}
	return nil
}

func bannedWordRowToResponse(row repository.ChatBannedWordRow) dto.BannedWordRuleResponse {
	resp := dto.BannedWordRuleResponse{
		ID:            row.ID.String(),
		Scope:         row.Scope,
		Pattern:       row.Pattern,
		MatchMode:     row.MatchMode,
		CaseSensitive: row.CaseSensitive,
		Action:        row.Action,
		CreatedByName: row.CreatedByName,
		CreatedAt:     row.CreatedAt,
	}
	if row.RoomID != nil {
		s := row.RoomID.String()
		resp.RoomID = &s
	}
	if row.CreatedBy != nil {
		s := row.CreatedBy.String()
		resp.CreatedByID = &s
	}
	return resp
}

func (s *service) ListRoomBannedWords(ctx context.Context, actorID, roomID uuid.UUID) ([]dto.BannedWordRuleResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}
	rows, err := s.bannedWordRepo.ListApplicable(ctx, roomID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BannedWordRuleResponse, len(rows))
	for i, r := range rows {
		out[i] = bannedWordRowToResponse(r)
	}
	return out, nil
}

func (s *service) CreateRoomBannedWord(ctx context.Context, actorID, roomID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}
	if err := validateCreateBannedWord(req); err != nil {
		return nil, err
	}
	spec := repository.ChatBannedWordSpec{
		Scope:         "room",
		RoomID:        &roomID,
		Pattern:       strings.TrimSpace(req.Pattern),
		MatchMode:     req.MatchMode,
		CaseSensitive: req.CaseSensitive,
		Action:        req.Action,
		CreatedBy:     &actorID,
	}
	id, err := s.bannedWordRepo.Create(ctx, spec)
	if err != nil {
		return nil, err
	}
	created, err := s.bannedWordRepo.GetByID(ctx, id)
	if err != nil || created == nil {
		return nil, fmt.Errorf("fetch created banned word: %w", err)
	}
	details := fmt.Sprintf("room=%s pattern=%s mode=%s case=%t action=%s", roomID, req.Pattern, req.MatchMode, req.CaseSensitive, req.Action)
	if err := s.auditRepo.Create(ctx, actorID, "chat_room_banned_word_create", "chat_room", roomID.String(), details); err != nil {
		return nil, fmt.Errorf("audit create banned word: %w", err)
	}
	resp := bannedWordRowToResponse(*created)
	return &resp, nil
}

func (s *service) UpdateRoomBannedWord(ctx context.Context, actorID, roomID, ruleID uuid.UUID, req dto.UpdateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}
	existing, err := s.bannedWordRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrRoomNotFound
	}
	if existing.Scope != "room" || existing.RoomID == nil || *existing.RoomID != roomID {
		return nil, ErrBannedWordRuleMismatch
	}
	updated, err := s.updateBannedWord(ctx, ruleID, dto.CreateBannedWordRequest(req))
	if err != nil {
		return nil, err
	}
	details := fmt.Sprintf("room=%s pattern=%s mode=%s case=%t action=%s", roomID, req.Pattern, req.MatchMode, req.CaseSensitive, req.Action)
	if err := s.auditRepo.Create(ctx, actorID, "chat_room_banned_word_update", "chat_room", roomID.String(), details); err != nil {
		return nil, fmt.Errorf("audit update banned word: %w", err)
	}
	return updated, nil
}

func (s *service) DeleteRoomBannedWord(ctx context.Context, actorID, roomID, ruleID uuid.UUID) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return ErrRoomNotFound
	}
	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}
	existing, err := s.bannedWordRepo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRoomNotFound
	}
	if existing.Scope != "room" || existing.RoomID == nil || *existing.RoomID != roomID {
		return ErrBannedWordRuleMismatch
	}
	if err := s.bannedWordRepo.Delete(ctx, ruleID); err != nil {
		return err
	}
	s.bannedWordsRule.Invalidate(ruleID)
	return s.auditRepo.Create(ctx, actorID, "chat_room_banned_word_delete", "chat_room", roomID.String(), "rule="+ruleID.String())
}

func (s *service) ensureCanManageGlobalBannedWords(ctx context.Context, actorID uuid.UUID) error {
	if !s.authzSvc.Can(ctx, actorID, authz.PermManageBannedWords) {
		return ErrModRoleRequired
	}
	return nil
}

func (s *service) ListGlobalBannedWords(ctx context.Context, actorID uuid.UUID) ([]dto.BannedWordRuleResponse, error) {
	if err := s.ensureCanManageGlobalBannedWords(ctx, actorID); err != nil {
		return nil, err
	}
	rows, err := s.bannedWordRepo.ListGlobal(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BannedWordRuleResponse, len(rows))
	for i, r := range rows {
		out[i] = bannedWordRowToResponse(r)
	}
	return out, nil
}

func (s *service) CreateGlobalBannedWord(ctx context.Context, actorID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	if err := s.ensureCanManageGlobalBannedWords(ctx, actorID); err != nil {
		return nil, err
	}
	if err := validateCreateBannedWord(req); err != nil {
		return nil, err
	}
	spec := repository.ChatBannedWordSpec{
		Scope:         "global",
		Pattern:       strings.TrimSpace(req.Pattern),
		MatchMode:     req.MatchMode,
		CaseSensitive: req.CaseSensitive,
		Action:        req.Action,
		CreatedBy:     &actorID,
	}
	id, err := s.bannedWordRepo.Create(ctx, spec)
	if err != nil {
		return nil, err
	}
	created, err := s.bannedWordRepo.GetByID(ctx, id)
	if err != nil || created == nil {
		return nil, fmt.Errorf("fetch created banned word: %w", err)
	}
	details := fmt.Sprintf("pattern=%s mode=%s case=%t action=%s", req.Pattern, req.MatchMode, req.CaseSensitive, req.Action)
	if err := s.auditRepo.Create(ctx, actorID, "chat_global_banned_word_create", "banned_word", id.String(), details); err != nil {
		return nil, fmt.Errorf("audit create banned word: %w", err)
	}
	resp := bannedWordRowToResponse(*created)
	return &resp, nil
}

func (s *service) UpdateGlobalBannedWord(ctx context.Context, actorID, ruleID uuid.UUID, req dto.UpdateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	if err := s.ensureCanManageGlobalBannedWords(ctx, actorID); err != nil {
		return nil, err
	}
	existing, err := s.bannedWordRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrRoomNotFound
	}
	if existing.Scope != "global" {
		return nil, ErrBannedWordRuleMismatch
	}
	updated, err := s.updateBannedWord(ctx, ruleID, dto.CreateBannedWordRequest(req))
	if err != nil {
		return nil, err
	}
	details := fmt.Sprintf("pattern=%s mode=%s case=%t action=%s", req.Pattern, req.MatchMode, req.CaseSensitive, req.Action)
	if err := s.auditRepo.Create(ctx, actorID, "chat_global_banned_word_update", "banned_word", ruleID.String(), details); err != nil {
		return nil, fmt.Errorf("audit update banned word: %w", err)
	}
	return updated, nil
}

func (s *service) updateBannedWord(ctx context.Context, ruleID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	if err := validateCreateBannedWord(req); err != nil {
		return nil, err
	}
	update := repository.ChatBannedWordUpdate{
		Pattern:       strings.TrimSpace(req.Pattern),
		MatchMode:     req.MatchMode,
		CaseSensitive: req.CaseSensitive,
		Action:        req.Action,
	}
	if err := s.bannedWordRepo.Update(ctx, ruleID, update); err != nil {
		return nil, err
	}
	s.bannedWordsRule.Invalidate(ruleID)
	row, err := s.bannedWordRepo.GetByID(ctx, ruleID)
	if err != nil || row == nil {
		return nil, fmt.Errorf("fetch updated banned word: %w", err)
	}
	resp := bannedWordRowToResponse(*row)
	return &resp, nil
}

func (s *service) DeleteGlobalBannedWord(ctx context.Context, actorID, ruleID uuid.UUID) error {
	if err := s.ensureCanManageGlobalBannedWords(ctx, actorID); err != nil {
		return err
	}
	existing, err := s.bannedWordRepo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRoomNotFound
	}
	if existing.Scope != "global" {
		return ErrBannedWordRuleMismatch
	}
	if err := s.bannedWordRepo.Delete(ctx, ruleID); err != nil {
		return err
	}
	s.bannedWordsRule.Invalidate(ruleID)
	return s.auditRepo.Create(ctx, actorID, "chat_global_banned_word_delete", "banned_word", ruleID.String(), "")
}
