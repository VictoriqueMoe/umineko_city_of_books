package chat

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/hyperbeam"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
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
		contentFilter   *contentfilter.Manager
		bannedWordsRule *contentfilter.ChatBannedWordsRule
		sideEffectsWG   sync.WaitGroup
	}

	FileUpload struct {
		ContentType string
		Size        int64
		Open        func() (io.ReadCloser, error)
	}
)

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
	if room == nil || room.Type != "dm" {
		return ErrLockedNonStaffDM
	}
	members, err := c.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get room members: %w", err)
	}
	others := make([]uuid.UUID, 0, len(members))
	for i := 0; i < len(members); i++ {
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

func (c *core) canModerateRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	memberRole, err := c.chatRepo.GetMemberRole(ctx, roomID, userID)
	if err != nil {
		return false, fmt.Errorf("get member role: %w", err)
	}
	if memberRole == "host" {
		return true, nil
	}
	siteRole, err := c.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get site role: %w", err)
	}
	return siteRole.IsSiteStaff(), nil
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
	actionBody := strings.TrimSpace(body)
	if actionBody == "" {
		return
	}

	if timedOut, _ := c.chatRepo.HasActiveMemberTimeout(ctx, roomID, actorID); timedOut {
		return
	}

	messageID := uuid.New()
	if err := c.chatRepo.InsertSystemMessage(ctx, messageID, roomID, actorID, actionBody); err != nil {
		return
	}

	row, err := c.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return
	}
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
	for i := 0; i < len(members); i++ {
		c.hub.SendToUser(members[i], event)
	}
}

func (c *core) hydrateMessageRows(ctx context.Context, viewerID uuid.UUID, rows []repository.ChatMessageRow) []dto.ChatMessageResponse {
	messageIDs := make([]uuid.UUID, len(rows))
	senderIDs := make([]uuid.UUID, 0, len(rows))
	seenSender := make(map[uuid.UUID]struct{})
	for i := 0; i < len(rows); i++ {
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
	for i := 0; i < len(rows); i++ {
		row := rows[i]
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
	for i := 0; i < len(layouts); i++ {
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

	for i := 0; i < len(members); i++ {
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
	baseURL := c.settingsSvc.Get(bgCtx, config.SettingBaseURL)
	linkURL := fmt.Sprintf("%s/rooms/%s", baseURL, roomID)

	actorName := "Someone"
	if inviter, err := c.userRepo.GetByID(bgCtx, inviterID); err == nil && inviter != nil {
		actorName = inviter.DisplayName
	}
	subject, body := notification.NotifEmail(actorName, "added you to a chat room", roomName, linkURL)

	for _, invitedID := range invitedIDs {
		_ = c.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   invitedID,
			ActorID:       inviterID,
			Type:          dto.NotifChatRoomInvite,
			ReferenceID:   roomID,
			ReferenceType: "chat_room",
			EmailSubject:  subject,
			EmailBody:     body,
		})
		c.hub.SendToUser(invitedID, ws.Message{
			Type: "chat_room_invited",
			Data: map[string]interface{}{
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
		Data: map[string]interface{}{
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
