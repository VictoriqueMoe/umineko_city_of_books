package chat

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type moderationService struct {
	*core
}

func (s *moderationService) enforceBannedWords(ctx context.Context, roomID, senderID uuid.UUID, body string) error {
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
	details := fmt.Sprintf("pattern=%q", match.Pattern)
	if err := s.auditRepo.CreateSystem(ctx, repository.NewAuditEntry{
		Action:     wordFilterAuditAction(match.Action),
		TargetType: repository.AuditTargetChatRoom,
		TargetID:   roomID.String(),
		Details:    details,
		SubjectID:  senderID,
	}); err != nil {
		logger.Log.Error().Err(err).Str("room_id", roomID.String()).Msg("failed to audit word filter hit")
	}
	if match.Action == contentfilter.BannedWordActionKick && !s.isBotSender(ctx, senderID) {
		targetName := s.displayNameFor(ctx, senderID, roomID)
		s.postRoomActionMessage(ctx, roomID, senderID, fmt.Sprintf("%s was kicked by the word filter.", targetName))
		_ = s.evictUserFromRoom(ctx, roomID, senderID, "the word filter matched a banned word")
		s.notifyAutomatedKick(roomID, senderID, match.Pattern)
	}
	return &ErrBannedWordMatch{Pattern: match.Pattern, Action: match.Action}
}

func (s *moderationService) isBotSender(ctx context.Context, senderID uuid.UUID) bool {
	sender, err := s.userRepo.GetByID(ctx, senderID)
	if err != nil || sender == nil {
		return false
	}

	return sender.IsBot
}

func (s *moderationService) banUserFromRoom(ctx context.Context, roomID, targetID uuid.UUID, actorID *uuid.UUID, reason string) error {
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

func (s *moderationService) notifyModerationAction(roomID, targetID, actorID uuid.UUID, action, reason string) {
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

func (s *moderationService) notifyAutomatedKick(roomID, targetID uuid.UUID, pattern string) {
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

func (s *moderationService) lookupRoomName(ctx context.Context, roomID uuid.UUID) string {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, uuid.Nil)
	if err != nil || row == nil || row.Name == "" {
		return "the chat room"
	}
	return row.Name
}

func (s *moderationService) BanMember(ctx context.Context, actorID, roomID, targetID uuid.UUID, reason string) error {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return err
	}

	if actorID == targetID {
		return ErrCannotBanStaff
	}

	protected, err := s.canModerateRoom(ctx, roomID, targetID)
	if err != nil {
		return err
	}
	if protected {
		return ErrCannotBanStaff
	}

	targetName := s.displayNameFor(ctx, targetID, roomID)
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

	details := fmt.Sprintf("reason=%s", reason)
	if err := s.auditRepo.Create(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatRoomBan,
		TargetType: repository.AuditTargetChatRoom,
		TargetID:   roomID.String(),
		Details:    details,
		SubjectID:  targetID,
	}); err != nil {
		logger.Log.Error().Err(err).Str("room_id", roomID.String()).Msg("failed to audit room ban")
	}
	return nil
}

func (s *moderationService) UnbanMember(ctx context.Context, actorID, roomID, targetID uuid.UUID) error {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return err
	}

	if err := s.banRepo.UnbanWithAudit(ctx, roomID, targetID, actorID); err != nil {
		return err
	}

	targetName := s.displayNameFor(ctx, targetID, roomID)
	s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s was unbanned.", targetName))

	s.notifyModerationAction(roomID, targetID, actorID, "unbanned", "")

	return nil
}

func (s *moderationService) ListRoomBans(ctx context.Context, actorID, roomID uuid.UUID) ([]dto.ChatRoomBanResponse, error) {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return nil, err
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
		resp.RoomID = new(row.RoomID.String())
	}
	if row.CreatedBy != nil {
		resp.CreatedByID = new(row.CreatedBy.String())
	}
	return resp
}

func (s *moderationService) ListRoomBannedWords(ctx context.Context, actorID, roomID uuid.UUID) ([]dto.BannedWordRuleResponse, error) {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return nil, err
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

func (s *moderationService) CreateRoomBannedWord(ctx context.Context, actorID, roomID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return nil, err
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

	details := fmt.Sprintf("room=%s pattern=%s mode=%s case=%t action=%s", roomID, req.Pattern, req.MatchMode, req.CaseSensitive, req.Action)

	created, err := s.bannedWordRepo.CreateWithAudit(ctx, spec, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatRoomBannedWordCreate,
		TargetType: repository.AuditTargetChatRoom,
		TargetID:   roomID.String(),
		Details:    details,
	})
	if err != nil {
		return nil, err
	}

	return new(bannedWordRowToResponse(*created)), nil
}

func (s *moderationService) UpdateRoomBannedWord(ctx context.Context, actorID, roomID, ruleID uuid.UUID, req dto.UpdateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return nil, err
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
	if err := s.auditRepo.Create(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatRoomBannedWordUpdate,
		TargetType: repository.AuditTargetChatRoom,
		TargetID:   roomID.String(),
		Details:    details,
	}); err != nil {
		logger.Log.Error().Err(err).Str("room_id", roomID.String()).Msg("failed to audit banned word update")
	}
	return updated, nil
}

func (s *moderationService) DeleteRoomBannedWord(ctx context.Context, actorID, roomID, ruleID uuid.UUID) error {
	if _, err := s.loadRoomForMod(ctx, roomID, actorID); err != nil {
		return err
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

	if err := s.bannedWordRepo.DeleteWithAudit(ctx, ruleID, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatRoomBannedWordDelete,
		TargetType: repository.AuditTargetChatRoom,
		TargetID:   roomID.String(),
		Details:    "rule=" + ruleID.String(),
	}); err != nil {
		return err
	}

	s.bannedWordsRule.Invalidate(ruleID)

	return nil
}

func (s *moderationService) ensureCanManageGlobalBannedWords(ctx context.Context, actorID uuid.UUID) error {
	if !s.authzSvc.Can(ctx, actorID, authz.PermManageBannedWords) {
		return ErrModRoleRequired
	}
	return nil
}

func (s *moderationService) ListGlobalBannedWords(ctx context.Context, actorID uuid.UUID) ([]dto.BannedWordRuleResponse, error) {
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

func (s *moderationService) CreateGlobalBannedWord(ctx context.Context, actorID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
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

	details := fmt.Sprintf("pattern=%s mode=%s case=%t action=%s", req.Pattern, req.MatchMode, req.CaseSensitive, req.Action)

	created, err := s.bannedWordRepo.CreateWithAudit(ctx, spec, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatGlobalBannedWordCreate,
		TargetType: repository.AuditTargetBannedWord,
		Details:    details,
	})
	if err != nil {
		return nil, err
	}

	return new(bannedWordRowToResponse(*created)), nil
}

func (s *moderationService) UpdateGlobalBannedWord(ctx context.Context, actorID, ruleID uuid.UUID, req dto.UpdateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
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
	if err := s.auditRepo.Create(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatGlobalBannedWordUpdate,
		TargetType: repository.AuditTargetBannedWord,
		TargetID:   ruleID.String(),
		Details:    details,
	}); err != nil {
		logger.Log.Error().Err(err).Str("rule_id", ruleID.String()).Msg("failed to audit banned word update")
	}
	return updated, nil
}

func (s *moderationService) updateBannedWord(ctx context.Context, ruleID uuid.UUID, req dto.CreateBannedWordRequest) (*dto.BannedWordRuleResponse, error) {
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
	return new(bannedWordRowToResponse(*row)), nil
}

func (s *moderationService) DeleteGlobalBannedWord(ctx context.Context, actorID, ruleID uuid.UUID) error {
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

	if err := s.bannedWordRepo.DeleteWithAudit(ctx, ruleID, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     repository.AuditActionChatGlobalBannedWordDelete,
		TargetType: repository.AuditTargetBannedWord,
		TargetID:   ruleID.String(),
	}); err != nil {
		return err
	}

	s.bannedWordsRule.Invalidate(ruleID)

	return nil
}

func wordFilterAuditAction(action string) repository.AuditAction {
	switch action {
	case contentfilter.BannedWordActionKick:
		return repository.AuditActionChatWordFilterKick
	default:
		return repository.AuditActionChatWordFilterDelete
	}
}
