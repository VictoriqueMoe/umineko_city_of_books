package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/hyperbeam"
	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	moderatorKindHost  = "host"
	moderatorKindStaff = "staff"

	wsChatRoomUpdated = "chat_room_updated"

	maxRoomNameLength        = 80
	maxRoomDescriptionLength = 500
)

var (
	tagAllowedRegex  = regexp.MustCompile(`[^a-z0-9-]+`)
	timeoutUnitYears = map[string]int{
		"year":      1,
		"years":     1,
		"decade":    10,
		"decades":   10,
		"century":   100,
		"centuries": 100,
	}
	maxTimeoutUntil = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
)

type (
	core struct {
		chatRepo        repository.ChatRepository
		userRepo        repository.UserRepository
		roleRepo        repository.RoleRepository
		vanityRoleRepo  repository.VanityRoleRepository
		banRepo         repository.ChatRoomBanRepository
		bannedWordRepo  repository.ChatBannedWordRepository
		watchPartyRepo  repository.ChatWatchPartyRepository
		auditRepo       repository.AuditLogRepository
		authzSvc        authz.Service
		notifSvc        notification.Service
		blockSvc        block.Service
		settingsSvc     settings.Service
		uploadSvc       upload.Service
		uploader        *media.Uploader
		hub             *ws.Hub
		hyperbeamSvc    hyperbeam.Service
		livekitSvc      livekit.Service
		contentFilter   *contentfilter.Manager
		bannedWordsRule *contentfilter.ChatBannedWordsRule
		sideEffectsWG   sync.WaitGroup
		botObserver     MessageObserver
	}

	MessageObserver interface {
		ObserveMessage(ev BotMessageEvent)
	}

	BotMessageEvent struct {
		RoomID        uuid.UUID
		RoomType      string
		SenderID      uuid.UUID
		SenderName    string
		SenderHandle  string
		MessageID     uuid.UUID
		Body          string
		Members       []uuid.UUID
		MentionedIDs  map[uuid.UUID]struct{}
		ReplyToID     *uuid.UUID
		ReplyToAuthor uuid.UUID
	}

	FileUpload struct {
		ContentType string
		Size        int64
		Open        func() (io.ReadCloser, error)
	}
)

func (c *core) SetMessageObserver(obs MessageObserver) {
	c.botObserver = obs
}

func (c *core) clearVoiceMuted(ctx context.Context, roomID uuid.UUID) {
	if err := c.chatRepo.ClearVoiceForceMutes(ctx, roomID); err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("clear voice force mutes failed")
	}
}

func (c *core) dropFromLiveKitRoom(ctx context.Context, roomName, identity string) {
	if c.livekitSvc == nil {
		return
	}

	err := c.livekitSvc.RemoveParticipant(ctx, roomName, identity)
	if err == nil || errors.Is(err, livekit.ErrDisabled) || livekit.IsNotFound(err) {
		return
	}

	logger.Log.Warn().Err(err).Str("livekit_room", roomName).Str("identity", identity).Msg("remove livekit participant failed")
}

func (c *core) deleteRoomWithMedia(ctx context.Context, roomID uuid.UUID) error {
	members, err := c.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("list room members for hub cleanup failed")
	}

	paths, err := c.chatRepo.DeleteRoomWithMessages(ctx, roomID)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}

	c.uploadSvc.Delete(paths...)

	for i := range members {
		c.hub.LeaveRoom(roomID, members[i])
	}

	return nil
}

func (c *core) cleanupDeadSession(session *repository.ChatWatchPartySessionRow, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Log.Warn().
		Str("session_id", session.ID.String()).
		Str("room_id", session.RoomID.String()).
		Str("hyperbeam_session_id", session.HyperbeamSessionID).
		Str("type", session.Type).
		Str("started_at", session.StartedAt).
		Str("reason", reason).
		Msg("watch party ended")

	if err := c.watchPartyRepo.MarkAllParticipantsLeft(ctx, session.ID); err != nil {
		logger.Log.Warn().Err(err).Msg("cleanup dead session: mark participants left failed")
	}
	if err := c.watchPartyRepo.EndSession(ctx, session.ID, reason); err != nil {
		logger.Log.Warn().Err(err).Msg("cleanup dead session: end session failed")
	}
	if err := c.deleteRoomWithMedia(ctx, session.ID); err != nil {
		logger.Log.Warn().Err(err).Msg("cleanup dead session: delete watch party chat room failed")
	}

	c.clearVoiceMuted(ctx, session.ID)

	c.hub.BroadcastToRoom(session.RoomID, ws.Message{
		Type: wsWatchPartyEnded,
		Data: dto.WatchPartyEndedEvent{
			SessionID: session.ID,
			RoomID:    session.RoomID,
			Reason:    reason,
		},
	}, uuid.Nil)
}

func (c *core) endWatchPartiesForRoom(ctx context.Context, roomID uuid.UUID, reason string) {
	if c.watchPartyRepo == nil {
		return
	}

	sessions, err := c.watchPartyRepo.ListActiveByRoom(ctx, roomID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("end watch parties for room: list failed")
		return
	}

	for i := range sessions {
		session := sessions[i]
		if c.hyperbeamSvc != nil && session.HyperbeamSessionID != "" {
			if err := c.hyperbeamSvc.TerminateVM(ctx, session.HyperbeamSessionID); err != nil {
				logger.Log.Warn().Err(err).Str("hyperbeam_session_id", session.HyperbeamSessionID).Msg("end watch parties for room: terminate vm failed")
			}
		}
		c.cleanupDeadSession(&session, reason)
	}
}

func (c *core) clearWatchPartyParticipation(ctx context.Context, roomID, userID uuid.UUID) {
	if c.watchPartyRepo == nil {
		return
	}

	sessions, err := c.watchPartyRepo.ListActiveByRoom(ctx, roomID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("evict: list active watch parties failed")
		return
	}

	for i := range sessions {
		sessionID := sessions[i].ID

		participant, err := c.watchPartyRepo.GetParticipant(ctx, sessionID, userID)
		if err != nil || participant == nil || participant.LeftAt.Valid {
			continue
		}

		if err := c.watchPartyRepo.RemoveParticipant(ctx, sessionID, userID); err != nil {
			logger.Log.Warn().Err(err).Str("session_id", sessionID.String()).Msg("evict: mark watch party participant left failed")
			continue
		}

		c.dropFromLiveKitRoom(ctx, voiceSessionRoomPrefix+sessionID.String(), userID.String())

		c.hub.LeaveRoom(sessionID, userID)

		c.hub.BroadcastToRoom(roomID, ws.Message{
			Type: wsWatchPartyParticipantLeft,
			Data: dto.WatchPartyParticipantLeftEvent{
				SessionID: sessionID,
				RoomID:    roomID,
				UserID:    userID,
			},
		}, uuid.Nil)
	}
}

func (c *core) filterTexts(ctx context.Context, texts ...string) error {
	if c.contentFilter == nil {
		return nil
	}
	return c.contentFilter.Check(ctx, texts...)
}

func resolveSenderName(nickname, displayName, username string) string {
	if strings.TrimSpace(nickname) != "" {
		return nickname
	}
	if strings.TrimSpace(displayName) != "" {
		return displayName
	}
	return username
}

func (c *core) ensureLockAllowsRoom(ctx context.Context, senderID, roomID uuid.UUID) error {
	locked, err := c.userRepo.IsLocked(ctx, senderID)
	if err != nil {
		return fmt.Errorf("check lock: %w", err)
	}
	if !locked {
		return nil
	}
	room, err := c.chatRepo.GetRoomByID(ctx, roomID, senderID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room == nil || room.Type != dto.RoomTypeDM {
		return ErrLockedNonStaffDM
	}
	members, err := c.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room members: %w", err)
	}
	others := make([]uuid.UUID, 0, len(members))
	for i := range members {
		if members[i] != senderID {
			others = append(others, members[i])
		}
	}
	if len(others) == 0 {
		return ErrLockedNonStaffDM
	}
	roles, err := c.authzSvc.GetRoles(ctx, others)
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

func (c *core) moderatorKind(ctx context.Context, roomID, userID uuid.UUID) (string, error) {
	memberRole, err := c.chatRepo.GetMemberRole(ctx, roomID, userID)
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}
	if memberRole == "host" {
		return moderatorKindHost, nil
	}

	siteRole, err := c.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get site role: %w", err)
	}
	if siteRole.IsSiteStaff() {
		return moderatorKindStaff, nil
	}

	return "", nil
}

func (c *core) canModerateRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	kind, err := c.moderatorKind(ctx, roomID, userID)
	if err != nil {
		return false, err
	}

	return kind != "", nil
}

func (c *core) evictUserFromRoom(ctx context.Context, roomID, targetID uuid.UUID, reason string) error {
	members, _ := c.chatRepo.GetRoomMembers(ctx, roomID)

	if err := c.chatRepo.RemoveMember(ctx, roomID, targetID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	c.clearWatchPartyParticipation(ctx, roomID, targetID)
	c.dropFromLiveKitRoom(ctx, roomID.String(), targetID.String())

	c.hub.LeaveRoom(roomID, targetID)

	leftEvent := ws.Message{
		Type: "chat_member_left",
		Data: map[string]any{
			"room_id": roomID,
			"user_id": targetID,
		},
	}
	for _, mid := range members {
		if mid == targetID {
			continue
		}
		c.hub.SendToUser(mid, leftEvent)
	}

	kickData := map[string]any{
		"room_id": roomID,
	}
	if reason != "" {
		kickData["reason"] = reason
	}
	c.hub.SendToUser(targetID, ws.Message{Type: "chat_kicked", Data: kickData})

	return nil
}

func (c *core) normaliseRoomInput(ctx context.Context, rawName, rawDescription string, rawTags []string) (string, string, []string, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", "", nil, ErrMissingFields
	}

	if err := c.filterTexts(ctx, name, rawDescription); err != nil {
		return "", "", nil, err
	}

	if len(name) > maxRoomNameLength {
		name = name[:maxRoomNameLength]
	}

	description := strings.TrimSpace(rawDescription)
	if len(description) > maxRoomDescriptionLength {
		description = description[:maxRoomDescriptionLength]
	}

	return name, description, sanitizeTags(rawTags), nil
}

func (c *core) broadcastRoomUpdated(ctx context.Context, roomID uuid.UUID, name, description string, tags []string, isPublic, isRP bool) {
	c.broadcastToRoomMembers(ctx, roomID, ws.Message{
		Type: wsChatRoomUpdated,
		Data: map[string]any{
			"room_id":     roomID,
			"name":        name,
			"description": description,
			"tags":        tags,
			"is_public":   isPublic,
			"is_rp":       isRP,
		},
	})
}

func (c *core) rejectBotsOutsideRP(ctx context.Context, isRP bool, userIDs []uuid.UUID) error {
	if isRP || len(userIDs) == 0 {
		return nil
	}

	users, err := c.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return fmt.Errorf("get invited users: %w", err)
	}

	for _, user := range users {
		if user.IsBot {
			return ErrBotsRPRoomsOnly
		}
	}

	return nil
}

func isAuditableRoom(row *repository.ChatRoomRow) bool {
	if row == nil {
		return false
	}

	return row.Type == dto.RoomTypeGroup && !row.IsSystem && row.IsPublic
}

func isAuditableSendContext(row *repository.ChatRoomSendContext) bool {
	if row == nil {
		return false
	}

	return row.Type == dto.RoomTypeGroup && !row.IsSystem && row.IsPublic
}

func (c *core) writeAudit(ctx context.Context, entry repository.NewAuditEntry) {
	if err := c.auditRepo.Create(ctx, entry); err != nil {
		logger.Log.Error().Err(err).Str("action", string(entry.Action)).Msg("failed to write audit log")
	}
}

func (c *core) toVanityRoleResponses(rows []repository.VanityRoleRow) []dto.VanityRoleResponse {
	if len(rows) == 0 {
		return nil
	}
	out := make([]dto.VanityRoleResponse, len(rows))
	for i, r := range rows {
		out[i] = dto.VanityRoleResponse{
			ID:        r.ID,
			Label:     r.Label,
			Color:     r.Color,
			IsSystem:  r.IsSystem,
			SortOrder: r.SortOrder,
		}
	}
	return out
}

func (c *core) rowToResponse(row repository.ChatRoomRow) dto.ChatRoomResponse {
	return dto.ChatRoomResponse{
		ID:            row.ID,
		Name:          row.Name,
		Description:   row.Description,
		Type:          row.Type,
		IsPublic:      row.IsPublic,
		IsRP:          row.IsRP,
		IsSystem:      row.IsSystem,
		SystemKind:    row.SystemKind,
		Tags:          row.Tags,
		ViewerRole:    row.ViewerRole,
		ViewerMuted:   row.ViewerMuted,
		ViewerGhost:   row.ViewerGhost,
		IsMember:      row.IsMember,
		MemberCount:   row.MemberCount,
		HotScore:      row.HotScore,
		CreatedAt:     row.CreatedAt,
		LastMessageAt: nullStr(row.LastMessageAt),
		ArchivedAt:    nullStr(row.ArchivedAt),
		Unread:        isUnread(row.LastMessageAt, row.LastReadAt),
	}
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func (c *core) actionDisplayName(ctx context.Context, userID uuid.UUID, fallback string) string {
	name, _ := c.nameAndPossessive(ctx, userID)
	if name == "" {
		return fallback
	}
	return name
}

func (c *core) nameAndPossessive(ctx context.Context, userID uuid.UUID) (string, string) {
	u, err := c.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "", "their"
	}
	possessive := strings.TrimSpace(u.PronounPossessive)
	if possessive == "" {
		possessive = "their"
	}
	return u.DisplayLabel(), possessive
}

func (c *core) postRoomActionMessage(ctx context.Context, roomID, actorID uuid.UUID, body string) {
	actionBody := c.roomActionMessageBody(ctx, roomID, actorID, body)
	if actionBody == "" {
		return
	}

	row, err := c.chatRepo.InsertSystemMessage(ctx, roomID, actorID, actionBody)
	if err != nil {
		return
	}

	c.broadcastRoomActionMessage(ctx, roomID, actorID, row)
}

func (c *core) roomActionMessageBody(ctx context.Context, roomID, actorID uuid.UUID, body string) string {
	actionBody := strings.TrimSpace(body)
	if actionBody == "" {
		return ""
	}

	if timedOut, _ := c.chatRepo.HasActiveMemberTimeout(ctx, roomID, actorID); timedOut {
		return ""
	}

	return actionBody
}

func (c *core) broadcastRoomActionMessage(ctx context.Context, roomID, actorID uuid.UUID, row *repository.ChatMessageRow) {
	if row == nil {
		return
	}

	vanityRows, _ := c.vanityRoleRepo.GetRolesForUser(ctx, actorID)
	msg := c.messageRowToResponse(*row, nil, nil, c.toVanityRoleResponses(vanityRows))

	members, err := c.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return
	}

	event := ws.Message{Type: "chat_message", Data: msg}
	for i := range members {
		c.hub.SendToUser(members[i], event)
	}
}

func (c *core) hydrateMessageRows(ctx context.Context, viewerID uuid.UUID, rows []repository.ChatMessageRow) []dto.ChatMessageResponse {
	messageIDs := make([]uuid.UUID, len(rows))
	senderIDs := make([]uuid.UUID, 0, len(rows))
	seenSender := make(map[uuid.UUID]struct{})
	for i := range rows {
		messageIDs[i] = rows[i].ID
		if _, ok := seenSender[rows[i].SenderID]; !ok {
			seenSender[rows[i].SenderID] = struct{}{}
			senderIDs = append(senderIDs, rows[i].SenderID)
		}
	}
	mediaBatch, _ := c.chatRepo.GetMessageMediaBatch(ctx, messageIDs)
	reactionBatch, _ := c.chatRepo.GetReactionsBatch(ctx, messageIDs, viewerID)
	vanityMap, _ := c.vanityRoleRepo.GetRolesForUsersBatch(ctx, senderIDs)

	messages := make([]dto.ChatMessageResponse, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, c.messageRowToResponse(row, mediaBatch[row.ID], reactionBatch[row.ID], c.toVanityRoleResponses(vanityMap[row.SenderID])))
	}
	return messages
}

func (c *core) messageRowToResponse(row repository.ChatMessageRow, media []dto.PostMediaResponse, reactions []repository.ReactionGroup, vanityRoles []dto.VanityRoleResponse) dto.ChatMessageResponse {
	resp := dto.ChatMessageResponse{
		ID:     row.ID,
		RoomID: row.RoomID,
		Sender: dto.UserResponse{
			ID:          row.SenderID,
			Username:    row.SenderUsername,
			DisplayName: row.SenderDisplayName,
			AvatarURL:   row.SenderAvatarURL,
			Role:        row.SenderRoleTyped,
			VanityRoles: vanityRoles,
		},
		SenderNickname:        row.SenderNickname,
		SenderMemberAvatarURL: row.SenderMemberAvatar,
		Body:                  row.Body,
		IsSystem:              row.IsSystem,
		CreatedAt:             row.CreatedAt,
		Media:                 media,
		Pinned:                row.PinnedAt != nil,
		PinnedAt:              row.PinnedAt,
		PinnedBy:              row.PinnedBy,
		EditedAt:              row.EditedAt,
		Reactions:             toDTOReactions(reactions),
	}
	if row.ReplyToID != nil && row.ReplyToSenderID != nil && row.ReplyToSenderName != nil && row.ReplyToBody != nil {
		preview := *row.ReplyToBody
		if len(preview) > 140 {
			preview = preview[:140] + "..."
		}
		resp.ReplyTo = &dto.ChatMessageReplyPreview{
			ID:          *row.ReplyToID,
			SenderID:    *row.ReplyToSenderID,
			SenderName:  *row.ReplyToSenderName,
			BodyPreview: preview,
		}
	}
	return resp
}

func toDTOReactions(groups []repository.ReactionGroup) []dto.ReactionGroup {
	if len(groups) == 0 {
		return []dto.ReactionGroup{}
	}
	out := make([]dto.ReactionGroup, len(groups))
	for i, g := range groups {
		out[i] = dto.ReactionGroup{
			Emoji:         g.Emoji,
			Count:         g.Count,
			ViewerReacted: g.ViewerReacted,
			DisplayNames:  g.DisplayNames,
		}
	}
	return out
}

func isUnread(lastMessageAt, lastReadAt sql.NullString) bool {
	if !lastMessageAt.Valid {
		return false
	}
	if !lastReadAt.Valid {
		return true
	}
	return lastMessageAt.String > lastReadAt.String
}

func (c *core) effectiveLocked(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	siteRole, err := c.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get site role: %w", err)
	}
	if siteRole.IsSiteStaff() {
		return false, nil
	}
	locked, err := c.chatRepo.IsMemberNicknameLocked(ctx, roomID, userID)
	if err != nil {
		return false, fmt.Errorf("check nickname locked: %w", err)
	}
	return locked, nil
}

func (c *core) requireSiteMod(ctx context.Context, userID uuid.UUID) error {
	siteRole, err := c.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return fmt.Errorf("get site role: %w", err)
	}
	if !siteRole.IsSiteStaff() {
		return ErrModRoleRequired
	}
	return nil
}

func (c *core) assertTargetEditable(ctx context.Context, roomID, targetID uuid.UUID) error {
	targetRole, err := c.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return ErrNotMember
	}
	siteRole, err := c.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get target site role: %w", err)
	}
	if siteRole.IsSiteStaff() {
		return ErrTargetImmune
	}
	return nil
}

func (c *core) displayNameFor(ctx context.Context, userID, roomID uuid.UUID) string {
	if roomID != uuid.Nil {
		if nickname, _ := c.chatRepo.GetMemberNickname(ctx, roomID, userID); strings.TrimSpace(nickname) != "" {
			return nickname
		}
	}
	user, _ := c.userRepo.GetByID(ctx, userID)
	return user.DisplayLabel()
}

func stringOrEmpty(resp *dto.ChatRoomMemberResponse, get func(*dto.ChatRoomMemberResponse) string) string {
	if resp == nil {
		return ""
	}
	return get(resp)
}

func validateEmoji(emoji string) error {
	if emoji == "" || len(emoji) > 16 {
		return ErrInvalidEmoji
	}
	return nil
}

func sanitizeTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		t = strings.ReplaceAll(t, " ", "-")
		t = tagAllowedRegex.ReplaceAllString(t, "")
		t = strings.Trim(t, "-")
		if t == "" {
			continue
		}
		if len(t) > 30 {
			t = t[:30]
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= 10 {
			break
		}
	}
	return out
}

func timeoutDurationLabel(amount int, unit string) string {
	if amount == 1 {
		switch unit {
		case "second", "seconds":
			return "1 second"
		case "hour", "hours":
			return "1 hour"
		case "week", "weeks":
			return "1 week"
		case "year", "years":
			return "1 year"
		case "decade", "decades":
			return "1 decade"
		case "century", "centuries":
			return "1 century"
		}
	}

	suffix := unit
	switch unit {
	case "second":
		suffix = "seconds"
	case "hour":
		suffix = "hours"
	case "week":
		suffix = "weeks"
	case "year":
		suffix = "years"
	case "decade":
		suffix = "decades"
	case "century":
		suffix = "centuries"
	}
	return fmt.Sprintf("%d %s", amount, suffix)
}

func capTimeout(t time.Time) time.Time {
	if t.After(maxTimeoutUntil) {
		return maxTimeoutUntil
	}
	return t
}

func computeTimeoutUntil(now time.Time, amount int, unit string) (time.Time, string, error) {
	if amount <= 0 {
		return time.Time{}, "", ErrInvalidTimeoutDuration
	}

	normalized := strings.ToLower(strings.TrimSpace(unit))
	if normalized == "" {
		return time.Time{}, "", ErrInvalidTimeoutDuration
	}

	switch normalized {
	case "second", "seconds":
		return capTimeout(now.Add(time.Duration(amount) * time.Second)), timeoutDurationLabel(amount, normalized), nil
	case "hour", "hours":
		return capTimeout(now.Add(time.Duration(amount) * time.Hour)), timeoutDurationLabel(amount, normalized), nil
	case "week", "weeks":
		return capTimeout(now.Add(time.Duration(amount) * 7 * 24 * time.Hour)), timeoutDurationLabel(amount, normalized), nil
	}

	years, ok := timeoutUnitYears[normalized]
	if ok {
		return capTimeout(now.AddDate(amount*years, 0, 0)), timeoutDurationLabel(amount, normalized), nil
	}

	return time.Time{}, "", ErrInvalidTimeoutDuration
}

func formatTimeoutUntilForUser(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	layouts := []string{time.RFC3339Nano, time.RFC3339, time.DateTime}
	for i := range layouts {
		parsed, err := time.Parse(layouts[i], trimmed)
		if err == nil {
			return parsed.UTC().Format("02 January 2006 15:04 UTC")
		}
	}

	return trimmed
}

func (c *core) memberRowToMemberResponse(m repository.ChatRoomMemberRow, vanityRoles []dto.VanityRoleResponse, presence string) dto.ChatRoomMemberResponse {
	return dto.ChatRoomMemberResponse{
		User: dto.UserResponse{
			ID:          m.UserID,
			Username:    m.Username,
			DisplayName: m.DisplayName,
			AvatarURL:   m.AvatarURL,
			Role:        m.AuthorRoleTyped,
			VanityRoles: vanityRoles,
		},
		Role:            m.Role,
		JoinedAt:        m.JoinedAt,
		Nickname:        m.Nickname,
		NicknameLocked:  m.NicknameLocked && !m.AuthorRoleTyped.IsSiteStaff(),
		MemberAvatarURL: m.MemberAvatarURL,
		TimeoutUntil:    m.TimeoutUntil,
		TimeoutByStaff:  m.TimeoutByStaff,
		Presence:        presence,
	}
}

func (c *core) broadcastToRoomMembers(ctx context.Context, roomID uuid.UUID, msg ws.Message) {
	members, err := c.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return
	}

	for i := range members {
		c.hub.SendToUser(members[i], msg)
	}
}

func (c *core) checkSenderTimeout(ctx context.Context, roomID, senderID uuid.UUID) error {
	activeTimeout, timeoutUntil, _, err := c.chatRepo.GetMemberTimeoutState(ctx, roomID, senderID)
	if err != nil {
		return fmt.Errorf("get timeout state: %w", err)
	}
	if !activeTimeout {
		return nil
	}

	if timeoutUntil != "" {
		return fmt.Errorf("%w until %s", ErrTimedOut, formatTimeoutUntilForUser(timeoutUntil))
	}
	return ErrTimedOut
}

func (c *core) notifyInvited(inviterID, roomID uuid.UUID, roomName string, invitedIDs []uuid.UUID) {
	bgCtx := context.Background()

	actorName := "Someone"
	if inviter, err := c.userRepo.GetByID(bgCtx, inviterID); err == nil && inviter != nil {
		actorName = inviter.DisplayName
	}

	for _, invitedID := range invitedIDs {
		_ = c.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   invitedID,
			ActorID:       inviterID,
			Type:          dto.NotifChatRoomInvite,
			ReferenceID:   roomID,
			ReferenceType: "chat_room",
			EmailActor:    actorName,
			EmailAction:   "added you to a chat room",
			EmailTitle:    roomName,
			EmailLink:     fmt.Sprintf("/rooms/%s", roomID),
		})
		c.hub.SendToUser(invitedID, ws.Message{
			Type: "chat_room_invited",
			Data: map[string]any{
				"room_id": roomID,
			},
		})
	}
}

func (c *core) broadcastAndBuildMember(ctx context.Context, roomID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	rows, err := c.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	vanityMap, _ := c.vanityRoleRepo.GetRolesForUsersBatch(ctx, []uuid.UUID{targetID})

	var resp *dto.ChatRoomMemberResponse
	for _, m := range rows {
		if m.UserID != targetID {
			continue
		}
		resp = new(c.memberRowToMemberResponse(m, c.toVanityRoleResponses(vanityMap[m.UserID]), ""))
		break
	}

	event := ws.Message{
		Type: "chat_member_updated",
		Data: map[string]any{
			"room_id":              roomID,
			"user_id":              targetID,
			"nickname":             stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.Nickname }),
			"display_name":         stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.User.DisplayName }),
			"username":             stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.User.Username }),
			"member_avatar_url":    stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.MemberAvatarURL }),
			"nickname_locked":      resp != nil && resp.NicknameLocked,
			"timeout_until":        stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.TimeoutUntil }),
			"timeout_set_by_staff": resp != nil && resp.TimeoutByStaff,
		},
	}
	for _, r := range rows {
		c.hub.SendToUser(r.UserID, event)
	}

	if resp == nil {
		return nil, ErrNotMember
	}
	return resp, nil
}

func (c *core) getRoomMemberResponses(ctx context.Context, roomID, viewerID uuid.UUID) ([]dto.UserResponse, int, error) {
	memberIDs, err := c.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, 0, fmt.Errorf("get room members: %w", err)
	}

	hasGhost, _ := c.chatRepo.HasGhostMembers(ctx, roomID)
	var viewerIsStaff bool
	if hasGhost {
		r, _ := c.authzSvc.GetRole(ctx, viewerID)
		viewerIsStaff = r.IsSiteStaff()
	}

	members := make([]dto.UserResponse, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if hasGhost && !viewerIsStaff {
			ghost, _ := c.chatRepo.IsGhostMember(ctx, roomID, memberID)
			if ghost {
				continue
			}
		}
		user, err := c.userRepo.GetByID(ctx, memberID)
		if err != nil || user == nil {
			continue
		}
		members = append(members, *user.ToResponse())
	}
	return members, len(members), nil
}

func (c *core) loadRoomForMod(ctx context.Context, roomID, actorID uuid.UUID) (*repository.ChatRoomRow, error) {
	row, err := c.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	if row.IsSystem {
		return nil, ErrSystemRoom
	}

	canMod, err := c.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}

	return row, nil
}
