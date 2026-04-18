package chat

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	SystemKindMods   = "mods"
	SystemKindAdmins = "admins"

	systemModsName   = "Moderators"
	systemAdminsName = "Administrators"
	systemModsDesc   = "Private staff room for moderators, admins, and super admins. Membership is managed automatically."
	systemAdminsDesc = "Private room for admins and super admins. Membership is managed automatically."
)

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
var tagAllowedRegex = regexp.MustCompile(`[^a-z0-9-]+`)

var timeoutUnitYears = map[string]int{
	"year":      1,
	"years":     1,
	"decade":    10,
	"decades":   10,
	"century":   100,
	"centuries": 100,
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

var maxTimeoutUntil = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

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

type (
	FileUpload struct {
		ContentType string
		Size        int64
		Open        func() (io.ReadCloser, error)
	}

	Service interface {
		EnsureSystemRooms(ctx context.Context) error
		SyncSystemRoomMembership(ctx context.Context, userID uuid.UUID, newRole role.Role) error

		ResolveDMRoom(ctx context.Context, senderID, recipientID uuid.UUID) (*dto.ResolveDMResponse, error)
		SendDMMessage(ctx context.Context, senderID, recipientID uuid.UUID, body string, files []FileUpload) (*dto.SendDMResponse, error)
		CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error)
		ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error)
		ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, limit, offset int) (*dto.ChatRoomListResponse, error)
		ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, limit, offset int) (*dto.ChatRoomListResponse, error)
		GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) (*dto.ChatMessageListResponse, error)
		GetMessagesBefore(ctx context.Context, userID, roomID uuid.UUID, before string, limit int) (*dto.ChatMessageListResponse, error)

		SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest, files []FileUpload) (*dto.ChatMessageResponse, error)
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		DeleteChat(ctx context.Context, roomID, userID uuid.UUID) error
		JoinRoom(ctx context.Context, roomID, userID uuid.UUID, ghost bool) (*dto.ChatRoomResponse, error)
		SetRoomMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error
		IsRoomMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error
		KickMember(ctx context.Context, hostID, roomID, targetID uuid.UUID) error
		InviteMembers(ctx context.Context, hostID, roomID uuid.UUID, userIDs []uuid.UUID) (*dto.InviteMembersResponse, error)
		SetMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID, req dto.SetMemberTimeoutRequest) (*dto.ChatRoomMemberResponse, error)
		ClearMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error)
		GetMembers(ctx context.Context, viewerID, roomID uuid.UUID) ([]dto.ChatRoomMemberResponse, error)
		GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
		MarkRead(ctx context.Context, roomID, userID uuid.UUID) error

		SetRoomNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error)
		SetRoomAvatar(ctx context.Context, roomID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.ChatRoomMemberResponse, error)
		ClearRoomAvatar(ctx context.Context, roomID, userID uuid.UUID) (*dto.ChatRoomMemberResponse, error)
		SetMemberNicknameAsMod(ctx context.Context, roomID, actorID, targetID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error)
		UnlockMemberNickname(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error)
		PinMessage(ctx context.Context, messageID, userID uuid.UUID) error
		UnpinMessage(ctx context.Context, messageID, userID uuid.UUID) error
		ListPinnedMessages(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatMessageListResponse, error)
		AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
		RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
		DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error
		EditMessage(ctx context.Context, messageID, actorID uuid.UUID, body string) (*dto.ChatMessageResponse, error)
	}

	service struct {
		chatRepo       repository.ChatRepository
		userRepo       repository.UserRepository
		roleRepo       repository.RoleRepository
		vanityRoleRepo repository.VanityRoleRepository
		authzSvc       authz.Service
		notifSvc       notification.Service
		blockSvc       block.Service
		settingsSvc    settings.Service
		uploadSvc      upload.Service
		uploader       *media.Uploader
		hub            *ws.Hub
		contentFilter  *contentfilter.Manager
	}
)

func NewService(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	vanityRoleRepo repository.VanityRoleRepository,
	authzSvc authz.Service,
	notifSvc notification.Service,
	blockSvc block.Service,
	uploadSvc upload.Service,
	settingsSvc settings.Service,
	mediaProc *media.Processor,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
) Service {
	return &service{
		chatRepo:       chatRepo,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		vanityRoleRepo: vanityRoleRepo,
		authzSvc:       authzSvc,
		notifSvc:       notifSvc,
		blockSvc:       blockSvc,
		settingsSvc:    settingsSvc,
		uploadSvc:      uploadSvc,
		uploader:       media.NewUploader(uploadSvc, settingsSvc, mediaProc),
		hub:            hub,
		contentFilter:  contentFilter,
	}
}

func (s *service) filterTexts(ctx context.Context, texts ...string) error {
	if s.contentFilter == nil {
		return nil
	}
	return s.contentFilter.Check(ctx, texts...)
}

func (s *service) checkDMPreconditions(ctx context.Context, senderID, recipientID uuid.UUID) (*model.User, error) {
	if senderID == recipientID {
		return nil, ErrCannotDMSelf
	}
	recipient, err := s.userRepo.GetByID(ctx, recipientID)
	if err != nil {
		return nil, fmt.Errorf("get recipient: %w", err)
	}
	if recipient == nil {
		return nil, ErrUserNotFound
	}
	if !recipient.DmsEnabled {
		return nil, ErrDmsDisabled
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, senderID, recipientID); blocked {
		return nil, ErrUserBlocked
	}
	return recipient, nil
}

func (s *service) ResolveDMRoom(ctx context.Context, senderID, recipientID uuid.UUID) (*dto.ResolveDMResponse, error) {
	recipient, err := s.checkDMPreconditions(ctx, senderID, recipientID)
	if err != nil {
		return nil, err
	}

	resp := &dto.ResolveDMResponse{
		Recipient: *recipient.ToResponse(),
	}

	existingID, err := s.chatRepo.FindDMRoom(ctx, senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("find dm room: %w", err)
	}
	if existingID == uuid.Nil {
		return resp, nil
	}

	room, err := s.buildRoomResponse(ctx, existingID, senderID)
	if err != nil {
		return nil, err
	}
	resp.Room = room
	return resp, nil
}

func (s *service) SendDMMessage(ctx context.Context, senderID, recipientID uuid.UUID, body string, files []FileUpload) (*dto.SendDMResponse, error) {
	if body == "" && len(files) == 0 {
		return nil, ErrMissingFields
	}
	if body != "" {
		if err := s.filterTexts(ctx, body); err != nil {
			return nil, err
		}
	}
	if _, err := s.checkDMPreconditions(ctx, senderID, recipientID); err != nil {
		return nil, err
	}

	roomID, err := s.chatRepo.CreateDMRoomAtomic(ctx, uuid.New(), senderID, recipientID)
	if err != nil {
		return nil, fmt.Errorf("create dm room: %w", err)
	}

	msgResp, err := s.SendMessage(ctx, senderID, roomID, dto.SendMessageRequest{Body: body}, files)
	if err != nil {
		return nil, err
	}

	roomResp, err := s.buildRoomResponse(ctx, roomID, senderID)
	if err != nil {
		return nil, err
	}

	return &dto.SendDMResponse{
		Room:    *roomResp,
		Message: *msgResp,
	}, nil
}

func (s *service) CreateGroupRoom(ctx context.Context, creatorID uuid.UUID, req dto.CreateGroupRoomRequest) (*dto.ChatRoomResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrMissingFields
	}
	if err := s.filterTexts(ctx, name, req.Description); err != nil {
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
	if err := s.chatRepo.CreateRoom(ctx, roomID, name, description, "group", req.IsPublic, req.IsRP, creatorID); err != nil {
		return nil, fmt.Errorf("create group room: %w", err)
	}
	if len(tags) > 0 {
		if err := s.chatRepo.AddRoomTags(ctx, roomID, tags); err != nil {
			return nil, fmt.Errorf("add room tags: %w", err)
		}
	}
	if err := s.chatRepo.AddMemberWithRole(ctx, roomID, creatorID, "host", false); err != nil {
		return nil, fmt.Errorf("add creator to group: %w", err)
	}

	invitedIDs := make([]uuid.UUID, 0, len(req.MemberIDs))
	for _, memberID := range req.MemberIDs {
		if memberID == creatorID {
			continue
		}
		if blocked, _ := s.blockSvc.IsBlockedEither(ctx, creatorID, memberID); blocked {
			continue
		}
		if err := s.chatRepo.AddMemberWithRole(ctx, roomID, memberID, "member", false); err != nil {
			return nil, fmt.Errorf("add member to group: %w", err)
		}
		invitedIDs = append(invitedIDs, memberID)
	}

	if len(invitedIDs) > 0 {
		go s.notifyInvited(creatorID, roomID, name, invitedIDs)
	}

	return s.buildRoomResponse(ctx, roomID, creatorID)
}

func (s *service) notifyInvited(inviterID, roomID uuid.UUID, roomName string, invitedIDs []uuid.UUID) {
	bgCtx := context.Background()
	baseURL := s.settingsSvc.Get(bgCtx, config.SettingBaseURL)
	linkURL := fmt.Sprintf("%s/rooms/%s", baseURL, roomID)

	actorName := "Someone"
	if inviter, err := s.userRepo.GetByID(bgCtx, inviterID); err == nil && inviter != nil {
		actorName = inviter.DisplayName
	}
	subject, body := notification.NotifEmail(actorName, "added you to a chat room", roomName, linkURL)

	for _, invitedID := range invitedIDs {
		_ = s.notifSvc.Notify(bgCtx, dto.NotifyParams{
			RecipientID:   invitedID,
			ActorID:       inviterID,
			Type:          dto.NotifChatRoomInvite,
			ReferenceID:   roomID,
			ReferenceType: "chat_room",
			EmailSubject:  subject,
			EmailBody:     body,
		})
		s.hub.SendToUser(invitedID, ws.Message{
			Type: "chat_room_invited",
			Data: map[string]interface{}{
				"room_id": roomID,
			},
		})
	}
}

func (s *service) InviteMembers(ctx context.Context, hostID, roomID uuid.UUID, userIDs []uuid.UUID) (*dto.InviteMembersResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, hostID)
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

	canMod, err := s.canModerateRoom(ctx, roomID, hostID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}

	cap := s.settingsSvc.GetInt(ctx, config.SettingMaxChatRoomMembers)
	memberCount := row.MemberCount

	existingMembers, _ := s.chatRepo.GetRoomMembers(ctx, roomID)
	inviterName := "Someone"
	if inviter, err := s.userRepo.GetByID(ctx, hostID); err == nil && inviter != nil {
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

		existingRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
		if err != nil {
			return nil, fmt.Errorf("get member role: %w", err)
		}
		if existingRole != "" {
			skipped++
			continue
		}

		target, err := s.userRepo.GetByID(ctx, targetID)
		if err != nil || target == nil {
			skipped++
			continue
		}

		if blocked, _ := s.blockSvc.IsBlockedEither(ctx, hostID, targetID); blocked {
			skipped++
			continue
		}

		if err := s.chatRepo.AddMemberWithRole(ctx, roomID, targetID, "member", false); err != nil {
			return nil, fmt.Errorf("add member: %w", err)
		}
		memberCount++
		invitedIDs = append(invitedIDs, targetID)

		joinedEvent := ws.Message{
			Type: "chat_member_joined",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user":    target.ToResponse(),
			},
		}
		for _, mid := range existingMembers {
			s.hub.SendToUser(mid, joinedEvent)
		}
		s.hub.SendToUser(targetID, joinedEvent)
		existingMembers = append(existingMembers, targetID)

		s.postRoomActionMessage(ctx, roomID, hostID, fmt.Sprintf("%s invited %s to the room.", inviterName, target.DisplayName))
	}

	if len(invitedIDs) > 0 {
		go s.notifyInvited(hostID, roomID, row.Name, invitedIDs)
	}

	return &dto.InviteMembersResponse{
		InvitedCount: len(invitedIDs),
		SkippedCount: skipped,
	}, nil
}

func (s *service) ListPublicRooms(ctx context.Context, search string, isRPOnly bool, tag string, viewerID uuid.UUID, limit, offset int) (*dto.ChatRoomListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	blockedIDs, _ := s.blockSvc.GetBlockedIDs(ctx, viewerID)
	tag = strings.ToLower(strings.TrimSpace(tag))
	rows, total, err := s.chatRepo.ListPublicRooms(ctx, search, isRPOnly, tag, viewerID, blockedIDs, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := range rows {
		room := s.rowToResponse(rows[i])
		rooms = append(rooms, room)
	}
	return &dto.ChatRoomListResponse{Rooms: rooms, Total: total}, nil
}

func (s *service) ListUserGroupRooms(ctx context.Context, userID uuid.UUID, search string, isRPOnly bool, tag, role string, limit, offset int) (*dto.ChatRoomListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if role != "host" && role != "member" {
		role = ""
	}
	tag = strings.ToLower(strings.TrimSpace(tag))
	rows, total, err := s.chatRepo.ListUserGroupRooms(ctx, userID, search, isRPOnly, tag, role, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user group rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := range rows {
		rooms = append(rooms, s.rowToResponse(rows[i]))
	}
	return &dto.ChatRoomListResponse{Rooms: rooms, Total: total}, nil
}

func (s *service) SetRoomMuted(ctx context.Context, roomID, userID uuid.UUID, muted bool) error {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}
	if err := s.chatRepo.SetMuted(ctx, roomID, userID, muted); err != nil {
		return fmt.Errorf("set muted: %w", err)
	}
	return nil
}

func (s *service) IsRoomMuted(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return s.chatRepo.IsMuted(ctx, roomID, userID)
}

func (s *service) JoinRoom(ctx context.Context, roomID, userID uuid.UUID, ghost bool) (*dto.ChatRoomResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
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
	if ghost {
		viewerSiteRole, err := s.authzSvc.GetRole(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get site role: %w", err)
		}
		if !viewerSiteRole.IsSiteStaff() {
			return nil, ErrGhostRequiresStaff
		}
	}
	if row.IsMember {
		return s.buildRoomResponse(ctx, roomID, userID)
	}
	if blocked, _ := s.blockSvc.IsBlockedEither(ctx, userID, row.CreatedBy); blocked {
		return nil, ErrUserBlocked
	}
	cap := s.settingsSvc.GetInt(ctx, config.SettingMaxChatRoomMembers)
	if cap > 0 && row.MemberCount >= cap {
		return nil, ErrRoomFull
	}

	if err := s.chatRepo.AddMemberWithRole(ctx, roomID, userID, "member", ghost); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	resp, err := s.buildRoomResponse(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)
	joiner, _ := s.userRepo.GetByID(ctx, userID)
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
			s.broadcastToStaff(ctx, members, event)
		} else {
			s.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s joined the room.", joiner.DisplayName))
			for _, mid := range members {
				s.hub.SendToUser(mid, event)
			}
		}
	}
	return resp, nil
}

func (s *service) broadcastToStaff(ctx context.Context, memberIDs []uuid.UUID, msg ws.Message) {
	for _, mid := range memberIDs {
		r, err := s.authzSvc.GetRole(ctx, mid)
		if err != nil {
			continue
		}
		if r.IsSiteStaff() {
			s.hub.SendToUser(mid, msg)
		}
	}
}

func (s *service) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
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

	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)
	var wasGhost bool
	if hasGhost, _ := s.chatRepo.HasGhostMembers(ctx, roomID); hasGhost {
		wasGhost, _ = s.chatRepo.IsGhostMember(ctx, roomID, userID)
	}

	if err := s.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	leaver, _ := s.userRepo.GetByID(ctx, userID)
	if leaver != nil {
		if !wasGhost {
			s.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s left the room.", leaver.DisplayName))
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
			s.broadcastToStaff(ctx, members, event)
		} else {
			for _, mid := range members {
				s.hub.SendToUser(mid, event)
			}
		}
	}
	return nil
}

func (s *service) KickMember(ctx context.Context, hostID, roomID, targetID uuid.UUID) error {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, hostID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return ErrRoomNotFound
	}
	if row.IsSystem {
		return ErrSystemRoom
	}

	canMod, err := s.canModerateRoom(ctx, roomID, hostID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}
	targetRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return ErrNotMember
	}
	if targetRole == "host" {
		return ErrCannotKickHost
	}
	targetSiteRole, err := s.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get target site role: %w", err)
	}
	if targetSiteRole.IsSiteStaff() {
		return ErrTargetImmune
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)

	if err := s.chatRepo.RemoveMember(ctx, roomID, targetID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	s.postRoomActionMessage(ctx, roomID, hostID, "A member was kicked from the room.")

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
	s.hub.SendToUser(targetID, ws.Message{
		Type: "chat_kicked",
		Data: map[string]interface{}{
			"room_id": roomID,
		},
	})
	return nil
}

func (s *service) SetMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID, req dto.SetMemberTimeoutRequest) (*dto.ChatRoomMemberResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	if row.IsSystem {
		return nil, ErrSystemRoom
	}

	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}

	targetRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return nil, ErrNotMember
	}

	actorSiteRole, err := s.authzSvc.GetRole(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("get actor site role: %w", err)
	}
	actorIsStaff := actorSiteRole.IsSiteStaff()
	if !actorIsStaff && targetRole == "host" {
		return nil, ErrCannotKickHost
	}

	targetSiteRole, err := s.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target site role: %w", err)
	}
	if targetSiteRole.IsSiteStaff() {
		return nil, ErrTargetImmune
	}

	activeTimeout, _, timeoutByStaff, err := s.chatRepo.GetMemberTimeoutState(ctx, roomID, targetID)
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

	if err := s.chatRepo.SetMemberTimeout(ctx, roomID, targetID, until.Format(time.DateTime), actorIsStaff); err != nil {
		return nil, fmt.Errorf("set member timeout: %w", err)
	}

	actorName := s.actionDisplayName(ctx, actorID, "A moderator")
	targetName := s.actionDisplayName(ctx, targetID, "a member")
	s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s timed out %s for %s.", actorName, targetName, label))

	return s.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (s *service) ClearMemberTimeout(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrRoomNotFound
	}
	if row.IsSystem {
		return nil, ErrSystemRoom
	}

	canMod, err := s.canModerateRoom(ctx, roomID, actorID)
	if err != nil {
		return nil, err
	}
	if !canMod {
		return nil, ErrNotHost
	}

	targetRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return nil, ErrNotMember
	}

	actorSiteRole, err := s.authzSvc.GetRole(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("get actor site role: %w", err)
	}
	actorIsStaff := actorSiteRole.IsSiteStaff()

	activeTimeout, _, timeoutByStaff, err := s.chatRepo.GetMemberTimeoutState(ctx, roomID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get timeout state: %w", err)
	}
	if activeTimeout && timeoutByStaff && !actorIsStaff {
		return nil, ErrTimeoutLockedByStaff
	}

	if err := s.chatRepo.ClearMemberTimeout(ctx, roomID, targetID); err != nil {
		return nil, fmt.Errorf("clear member timeout: %w", err)
	}

	actorName := s.actionDisplayName(ctx, actorID, "A moderator")
	targetName := s.actionDisplayName(ctx, targetID, "a member")
	s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s removed %s's timeout.", actorName, targetName))

	return s.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (s *service) GetMembers(ctx context.Context, viewerID, roomID uuid.UUID) ([]dto.ChatRoomMemberResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, err := s.chatRepo.GetRoomMembersDetailed(ctx, roomID)
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
		r, _ := s.authzSvc.GetRole(ctx, viewerID)
		viewerIsStaff = r.IsSiteStaff()
	}

	userIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		if rows[i].Ghost && !viewerIsStaff {
			continue
		}
		userIDs = append(userIDs, rows[i].UserID)
	}
	vanityMap, _ := s.vanityRoleRepo.GetRolesForUsersBatch(ctx, userIDs)
	presence := s.hub.GetRoomPresence(roomID)

	members := make([]dto.ChatRoomMemberResponse, 0, len(rows))
	for i := range rows {
		m := rows[i]
		if m.Ghost && !viewerIsStaff {
			continue
		}
		locked := m.NicknameLocked && !m.AuthorRoleTyped.IsSiteStaff()
		members = append(members, dto.ChatRoomMemberResponse{
			User: dto.UserResponse{
				ID:          m.UserID,
				Username:    m.Username,
				DisplayName: m.DisplayName,
				AvatarURL:   m.AvatarURL,
				Role:        m.AuthorRoleTyped,
				VanityRoles: s.toVanityRoleResponses(vanityMap[m.UserID]),
			},
			Role:            m.Role,
			JoinedAt:        m.JoinedAt,
			Nickname:        m.Nickname,
			NicknameLocked:  locked,
			MemberAvatarURL: m.MemberAvatarURL,
			TimeoutUntil:    m.TimeoutUntil,
			TimeoutByStaff:  m.TimeoutByStaff,
			Presence:        presence[m.UserID],
			Ghost:           m.Ghost,
		})
	}
	return members, nil
}

func (s *service) ListRooms(ctx context.Context, userID uuid.UUID) (*dto.ChatRoomListResponse, error) {
	rows, err := s.chatRepo.GetRoomsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}

	rooms := make([]dto.ChatRoomResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		members, count, err := s.getRoomMemberResponses(ctx, row.ID, userID)
		if err != nil {
			return nil, err
		}
		resp := s.rowToResponse(row)
		resp.Members = members
		resp.MemberCount = count
		rooms = append(rooms, resp)
	}

	return &dto.ChatRoomListResponse{Rooms: rooms}, nil
}

func (s *service) canModerateRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	memberRole, err := s.chatRepo.GetMemberRole(ctx, roomID, userID)
	if err != nil {
		return false, fmt.Errorf("get member role: %w", err)
	}
	if memberRole == "host" {
		return true, nil
	}
	siteRole, err := s.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get site role: %w", err)
	}
	return siteRole.IsSiteStaff(), nil
}

func (s *service) toVanityRoleResponses(rows []repository.VanityRoleRow) []dto.VanityRoleResponse {
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

func eligibleForMods(r role.Role) bool {
	return r == authz.RoleModerator || r == authz.RoleAdmin || r == authz.RoleSuperAdmin
}

func eligibleForAdmins(r role.Role) bool {
	return r == authz.RoleAdmin || r == authz.RoleSuperAdmin
}

func memberRoleForSystem(r role.Role) string {
	if r == authz.RoleSuperAdmin {
		return "host"
	}
	return "member"
}

func (s *service) EnsureSystemRooms(ctx context.Context) error {
	modsID, err := s.chatRepo.GetSystemRoomID(ctx, SystemKindMods)
	if err != nil {
		return fmt.Errorf("get mods room: %w", err)
	}
	adminsID, err := s.chatRepo.GetSystemRoomID(ctx, SystemKindAdmins)
	if err != nil {
		return fmt.Errorf("get admins room: %w", err)
	}
	if modsID != uuid.Nil && adminsID != uuid.Nil {
		return nil
	}

	supers, err := s.roleRepo.GetUsersByRoles(ctx, []role.Role{authz.RoleSuperAdmin})
	if err != nil {
		return fmt.Errorf("find super admin: %w", err)
	}
	if len(supers) == 0 {
		return nil
	}
	creator := supers[0]

	if modsID == uuid.Nil {
		if err := s.chatRepo.CreateSystemRoom(ctx, uuid.New(), systemModsName, systemModsDesc, SystemKindMods, creator); err != nil {
			return err
		}
	}
	if adminsID == uuid.Nil {
		if err := s.chatRepo.CreateSystemRoom(ctx, uuid.New(), systemAdminsName, systemAdminsDesc, SystemKindAdmins, creator); err != nil {
			return err
		}
	}

	staff, err := s.roleRepo.GetUsersByRoles(ctx, []role.Role{authz.RoleModerator, authz.RoleAdmin, authz.RoleSuperAdmin})
	if err != nil {
		return fmt.Errorf("list staff: %w", err)
	}
	for _, uid := range staff {
		r, rErr := s.roleRepo.GetRole(ctx, uid)
		if rErr != nil {
			logger.Log.Error().Err(rErr).Str("user_id", uid.String()).Msg("get role during system room seed")
			continue
		}
		if err := s.SyncSystemRoomMembership(ctx, uid, r); err != nil {
			logger.Log.Error().Err(err).Str("user_id", uid.String()).Msg("sync system room membership during seed")
		}
	}
	return nil
}

func (s *service) SyncSystemRoomMembership(ctx context.Context, userID uuid.UUID, newRole role.Role) error {
	modsID, err := s.chatRepo.GetSystemRoomID(ctx, SystemKindMods)
	if err != nil {
		return fmt.Errorf("get mods room: %w", err)
	}
	adminsID, err := s.chatRepo.GetSystemRoomID(ctx, SystemKindAdmins)
	if err != nil {
		return fmt.Errorf("get admins room: %w", err)
	}

	desired := memberRoleForSystem(newRole)
	if err := s.syncOneSystemRoom(ctx, modsID, userID, eligibleForMods(newRole), desired); err != nil {
		return err
	}
	if err := s.syncOneSystemRoom(ctx, adminsID, userID, eligibleForAdmins(newRole), desired); err != nil {
		return err
	}
	return nil
}

func (s *service) syncOneSystemRoom(ctx context.Context, roomID, userID uuid.UUID, shouldBeMember bool, desiredRole string) error {
	if roomID == uuid.Nil {
		return nil
	}
	currentRole, err := s.chatRepo.GetMemberRole(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("get current role: %w", err)
	}
	wasMember := currentRole != ""

	switch {
	case shouldBeMember && !wasMember:
		if err := s.chatRepo.AddMemberWithRole(ctx, roomID, userID, desiredRole, false); err != nil {
			return err
		}
		s.hub.JoinRoom(roomID, userID)
		s.hub.SendToUser(userID, ws.Message{
			Type: "chat_room_invited",
			Data: map[string]interface{}{"room_id": roomID},
		})
	case !shouldBeMember && wasMember:
		if err := s.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
			return err
		}
		s.hub.LeaveRoom(roomID, userID)
		s.hub.SendToUser(userID, ws.Message{
			Type: "chat_kicked",
			Data: map[string]interface{}{"room_id": roomID},
		})
	case shouldBeMember && wasMember && currentRole != desiredRole:
		if err := s.chatRepo.SetMemberRole(ctx, roomID, userID, desiredRole); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) rowToResponse(row repository.ChatRoomRow) dto.ChatRoomResponse {
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
		CreatedAt:     row.CreatedAt,
		LastMessageAt: nullStr(row.LastMessageAt),
		Unread:        isUnread(row.LastMessageAt, row.LastReadAt),
	}
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func (s *service) actionDisplayName(ctx context.Context, userID uuid.UUID, fallback string) string {
	name, _ := s.nameAndPossessive(ctx, userID)
	if name == "" {
		return fallback
	}
	return name
}

func (s *service) nameAndPossessive(ctx context.Context, userID uuid.UUID) (string, string) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "", "their"
	}
	name := strings.TrimSpace(u.DisplayName)
	if name == "" {
		name = strings.TrimSpace(u.Username)
	}
	possessive := strings.TrimSpace(u.PronounPossessive)
	if possessive == "" {
		possessive = "their"
	}
	return name, possessive
}

func (s *service) postRoomActionMessage(ctx context.Context, roomID, actorID uuid.UUID, body string) {
	actionBody := strings.TrimSpace(body)
	if actionBody == "" {
		return
	}

	if timedOut, _ := s.chatRepo.HasActiveMemberTimeout(ctx, roomID, actorID); timedOut {
		return
	}

	messageID := uuid.New()
	if err := s.chatRepo.InsertSystemMessage(ctx, messageID, roomID, actorID, actionBody); err != nil {
		return
	}

	row, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return
	}
	if row == nil {
		return
	}

	vanityRows, _ := s.vanityRoleRepo.GetRolesForUser(ctx, actorID)
	msg := s.messageRowToResponse(*row, nil, nil, s.toVanityRoleResponses(vanityRows))

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return
	}
	event := ws.Message{Type: "chat_message", Data: msg}
	for i := 0; i < len(members); i++ {
		s.hub.SendToUser(members[i], event)
	}
}

func (s *service) messageRowToResponse(row repository.ChatMessageRow, media []dto.PostMediaResponse, reactions []repository.ReactionGroup, vanityRoles []dto.VanityRoleResponse) dto.ChatMessageResponse {
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
	mediaBatch, _ := s.chatRepo.GetMessageMediaBatch(ctx, messageIDs)
	reactionBatch, _ := s.chatRepo.GetReactionsBatch(ctx, messageIDs, userID)
	vanityMap, _ := s.vanityRoleRepo.GetRolesForUsersBatch(ctx, senderIDs)

	messages := make([]dto.ChatMessageResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		messages = append(messages, s.messageRowToResponse(row, mediaBatch[row.ID], reactionBatch[row.ID], s.toVanityRoleResponses(vanityMap[row.SenderID])))
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (s *service) GetMessagesBefore(ctx context.Context, userID, roomID uuid.UUID, before string, limit int) (*dto.ChatMessageListResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := s.chatRepo.GetMessagesBefore(ctx, roomID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("get messages before: %w", err)
	}

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
	mediaBatch, _ := s.chatRepo.GetMessageMediaBatch(ctx, messageIDs)
	reactionBatch, _ := s.chatRepo.GetReactionsBatch(ctx, messageIDs, userID)
	vanityMap, _ := s.vanityRoleRepo.GetRolesForUsersBatch(ctx, senderIDs)

	messages := make([]dto.ChatMessageResponse, 0, len(rows))
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		messages = append(messages, s.messageRowToResponse(row, mediaBatch[row.ID], reactionBatch[row.ID], s.toVanityRoleResponses(vanityMap[row.SenderID])))
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    -1,
		Limit:    limit,
	}, nil
}

func (s *service) SendMessage(ctx context.Context, senderID, roomID uuid.UUID, req dto.SendMessageRequest, files []FileUpload) (*dto.ChatMessageResponse, error) {
	if req.Body == "" && len(files) == 0 {
		return nil, ErrMissingFields
	}
	if req.Body != "" {
		if err := s.filterTexts(ctx, req.Body); err != nil {
			return nil, err
		}
	}

	isMember, err := s.chatRepo.IsMember(ctx, roomID, senderID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}
	activeTimeout, timeoutUntil, _, err := s.chatRepo.GetMemberTimeoutState(ctx, roomID, senderID)
	if err != nil {
		return nil, fmt.Errorf("get timeout state: %w", err)
	}
	if activeTimeout {
		if timeoutUntil != "" {
			return nil, fmt.Errorf("%w until %s", ErrTimedOut, formatTimeoutUntilForUser(timeoutUntil))
		}
		return nil, ErrTimedOut
	}

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}
	for _, memberID := range members {
		if memberID != senderID {
			if blocked, _ := s.blockSvc.IsBlockedEither(ctx, senderID, memberID); blocked {
				return nil, ErrUserBlocked
			}
		}
	}

	for i := 0; i < len(files); i++ {
		if err := s.validateMediaFile(ctx, files[i]); err != nil {
			return nil, err
		}
	}

	sender, err := s.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrUserNotFound
	}

	var replyToID *uuid.UUID
	var replyToPreview *dto.ChatMessageReplyPreview
	var replyToAuthor uuid.UUID
	if req.ReplyToID != nil {
		parent, perr := s.chatRepo.GetMessageByID(ctx, *req.ReplyToID)
		if perr == nil && parent != nil && parent.RoomID == roomID {
			replyToID = req.ReplyToID
			replyToAuthor = parent.SenderID
			preview := parent.Body
			if len(preview) > 140 {
				preview = preview[:140] + "..."
			}
			replyToPreview = &dto.ChatMessageReplyPreview{
				ID:          parent.ID,
				SenderID:    parent.SenderID,
				SenderName:  parent.SenderDisplayName,
				BodyPreview: preview,
			}
		}
	}

	msgID := uuid.New()
	if err := s.chatRepo.InsertMessage(ctx, msgID, roomID, senderID, req.Body, replyToID); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	mediaResponses, err := s.saveMessageMedia(ctx, msgID, files)
	if err != nil {
		if delErr := s.chatRepo.DeleteMessage(ctx, msgID); delErr != nil {
			logger.Log.Error().Err(delErr).Str("message_id", msgID.String()).Msg("failed to roll back message after media save failure")
		}
		return nil, err
	}

	if err := s.chatRepo.MarkRoomRead(ctx, roomID, senderID); err != nil {
		return nil, fmt.Errorf("mark sender read: %w", err)
	}

	displayName := sender.DisplayName
	avatarURL := sender.AvatarURL
	memberRows, _ := s.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	for _, mr := range memberRows {
		if mr.UserID == senderID {
			if mr.Nickname != "" {
				displayName = mr.Nickname
			}
			if mr.MemberAvatarURL != "" {
				avatarURL = mr.MemberAvatarURL
			}
			break
		}
	}

	senderVanity, _ := s.vanityRoleRepo.GetRolesForUser(ctx, senderID)

	resp := &dto.ChatMessageResponse{
		ID:     msgID,
		RoomID: roomID,
		Sender: dto.UserResponse{
			ID:          sender.ID,
			Username:    sender.Username,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			Role:        role.Role(sender.Role),
			VanityRoles: s.toVanityRoleResponses(senderVanity),
		},
		Body:      req.Body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Media:     mediaResponses,
		ReplyTo:   replyToPreview,
		Reactions: []dto.ReactionGroup{},
	}

	roomRow, _ := s.chatRepo.GetRoomByID(ctx, roomID, senderID)
	isGroup := roomRow != nil && roomRow.Type == "group"

	var mentionedIDs map[uuid.UUID]struct{}
	if isGroup {
		mentionedIDs = s.resolveMentions(ctx, req.Body, roomID, senderID)
	}

	members, err = s.chatRepo.GetRoomMembers(ctx, roomID)
	if err == nil {
		msg := ws.Message{
			Type: "chat_message",
			Data: resp,
		}
		for i := 0; i < len(members); i++ {
			memberID := members[i]
			if memberID == senderID {
				continue
			}
			s.hub.SendToUser(memberID, msg)

			inRoom := s.hub.IsUserViewing(roomID, memberID)

			if !inRoom {
				if isGroup {
					_, isMentioned := mentionedIDs[memberID]
					isReplyTarget := replyToAuthor != uuid.Nil && memberID == replyToAuthor

					if isMentioned {
						_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
							RecipientID:   memberID,
							ActorID:       senderID,
							Type:          dto.NotifChatMention,
							ReferenceID:   roomID,
							ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
						})
					} else if isReplyTarget {
						_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
							RecipientID:   memberID,
							ActorID:       senderID,
							Type:          dto.NotifChatReply,
							ReferenceID:   roomID,
							ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
						})
					} else {
						muted, _ := s.chatRepo.IsMuted(ctx, roomID, memberID)
						if !muted {
							roomName := ""
							if roomRow != nil {
								roomName = roomRow.Name
							}
							_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
								RecipientID:   memberID,
								ActorID:       senderID,
								Type:          dto.NotifChatRoomMessage,
								ReferenceID:   roomID,
								ReferenceType: fmt.Sprintf("chat_message:%s", msgID),
								Message:       fmt.Sprintf("sent a message in %s", roomName),
							})
						}
					}
				} else {
					_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
						RecipientID:   memberID,
						ActorID:       senderID,
						Type:          dto.NotifChatMessage,
						ReferenceID:   roomID,
						ReferenceType: "chat",
					})
				}
			}

			if !inRoom {
				total, countErr := s.chatRepo.CountUnreadRoomsForUser(ctx, memberID)
				if countErr == nil {
					s.hub.SendToUser(memberID, ws.Message{
						Type: "chat_unread_bumped",
						Data: map[string]interface{}{
							"room_id": roomID,
							"total":   total,
						},
					})
				}
			}
		}
	}

	return resp, nil
}

func (s *service) resolveMentions(ctx context.Context, body string, roomID, senderID uuid.UUID) map[uuid.UUID]struct{} {
	matches := mentionRegex.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	mentioned := make(map[uuid.UUID]struct{})
	for _, m := range matches {
		username := m[1]
		if _, dup := seen[username]; dup {
			continue
		}
		seen[username] = struct{}{}
		u, err := s.userRepo.GetByUsername(ctx, username)
		if err != nil || u == nil {
			continue
		}
		if u.ID == senderID {
			continue
		}
		isMember, err := s.chatRepo.IsMember(ctx, roomID, u.ID)
		if err != nil || !isMember {
			continue
		}
		if blocked, _ := s.blockSvc.IsBlockedEither(ctx, senderID, u.ID); blocked {
			continue
		}
		mentioned[u.ID] = struct{}{}
	}
	return mentioned
}

func (s *service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.chatRepo.CountUnreadRoomsForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

func (s *service) MarkRead(ctx context.Context, roomID, userID uuid.UUID) error {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := s.chatRepo.MarkRoomRead(ctx, roomID, userID); err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}

	readAt := time.Now().UTC().Format(time.RFC3339)

	total, _ := s.chatRepo.CountUnreadRoomsForUser(ctx, userID)
	s.hub.SendToUser(userID, ws.Message{
		Type: "chat_read",
		Data: map[string]interface{}{
			"room_id": roomID,
			"total":   total,
		},
	})

	members, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err == nil {
		receipt := ws.Message{
			Type: "chat_read_receipt",
			Data: map[string]interface{}{
				"room_id": roomID,
				"user_id": userID,
				"read_at": readAt,
			},
		}
		for i := 0; i < len(members); i++ {
			memberID := members[i]
			if memberID == userID {
				continue
			}
			s.hub.SendToUser(memberID, receipt)
		}
	}

	return nil
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
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, userID)
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
		mod, modErr := s.canModerateRoom(ctx, roomID, userID)
		if modErr != nil {
			return modErr
		}
		canMod = mod
	}
	if !row.IsMember && !canMod {
		return ErrNotMember
	}

	if row.Type == "group" && canMod {
		members, _ := s.chatRepo.GetRoomMembers(ctx, roomID)
		if err := s.chatRepo.DeleteMessages(ctx, roomID); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := s.chatRepo.DeleteRoom(ctx, roomID); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}
		event := ws.Message{
			Type: "chat_room_deleted",
			Data: map[string]interface{}{
				"room_id": roomID,
			},
		}
		for _, mid := range members {
			s.hub.SendToUser(mid, event)
		}
		return nil
	}

	if err := s.chatRepo.RemoveMember(ctx, roomID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	remaining, err := s.chatRepo.CountRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("count remaining members: %w", err)
	}
	if remaining == 0 {
		if err := s.chatRepo.DeleteMessages(ctx, roomID); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := s.chatRepo.DeleteRoom(ctx, roomID); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}
	}

	return nil
}

func (s *service) validateMediaFile(ctx context.Context, f FileUpload) error {
	isVideo := strings.HasPrefix(f.ContentType, "video/")
	var maxSize int64
	var allowed map[string]string
	var typeErr error
	if isVideo {
		maxSize = int64(s.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		allowed = upload.AllowedVideoTypes
		typeErr = upload.ErrInvalidVideoType
	} else {
		maxSize = int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		allowed = upload.AllowedImageTypes
		typeErr = upload.ErrInvalidFileType
	}
	if f.Size > maxSize {
		return fmt.Errorf("file size %dMB exceeds maximum %dMB", f.Size/(1024*1024), maxSize/(1024*1024))
	}
	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open media: %w", err)
	}
	defer r.Close()
	sniffed, _, err := upload.DetectContentType(r)
	if err != nil {
		return err
	}
	if _, ok := allowed[sniffed]; !ok {
		return typeErr
	}
	return nil
}

func (s *service) saveMessageMedia(ctx context.Context, messageID uuid.UUID, files []FileUpload) ([]dto.PostMediaResponse, error) {
	if len(files) == 0 {
		return nil, nil
	}
	results := make([]dto.PostMediaResponse, 0, len(files))
	for i := 0; i < len(files); i++ {
		f := files[i]
		r, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open media: %w", err)
		}
		saved, saveErr := s.uploader.SaveAndRecord(ctx, "chat", f.ContentType, f.Size, r,
			func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error) {
				return s.chatRepo.AddMessageMedia(ctx, messageID, mediaURL, mediaType, thumbURL, sortOrder)
			},
			s.chatRepo.UpdateMessageMediaURL,
			s.chatRepo.UpdateMessageMediaThumbnail,
		)
		r.Close()
		if saveErr != nil {
			return nil, saveErr
		}
		results = append(results, *saved)
	}
	return results, nil
}

func (s *service) buildRoomResponse(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatRoomResponse, error) {
	row, err := s.chatRepo.GetRoomByID(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if row == nil {
		return nil, ErrNotMember
	}

	members, count, err := s.getRoomMemberResponses(ctx, roomID, viewerID)
	if err != nil {
		return nil, err
	}

	resp := s.rowToResponse(*row)
	resp.Members = members
	resp.MemberCount = count
	return &resp, nil
}

func (s *service) SetRoomNickname(ctx context.Context, roomID, userID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error) {
	if err := s.filterTexts(ctx, nickname); err != nil {
		return nil, err
	}
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	locked, err := s.effectiveLocked(ctx, roomID, userID)
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

	if err := s.chatRepo.SetMemberNickname(ctx, roomID, userID, nickname); err != nil {
		return nil, fmt.Errorf("set member nickname: %w", err)
	}

	name, possessive := s.nameAndPossessive(ctx, userID)
	if name != "" {
		if nickname == "" {
			s.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s cleared %s alias.", name, possessive))
		} else {
			s.postRoomActionMessage(ctx, roomID, userID, fmt.Sprintf("%s changed %s alias.", name, possessive))
		}
	}

	return s.broadcastAndBuildMember(ctx, roomID, userID)
}

func (s *service) SetRoomAvatar(ctx context.Context, roomID, userID uuid.UUID, contentType string, fileSize int64, reader io.Reader) (*dto.ChatRoomMemberResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	locked, err := s.effectiveLocked(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, ErrNicknameLocked
	}

	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
	subDir := fmt.Sprintf("chat-avatars/%s", roomID.String())
	avatarURL, err := s.uploadSvc.SaveImage(ctx, subDir, userID, fileSize, maxSize, reader)
	if err != nil {
		return nil, err
	}

	if err := s.chatRepo.SetMemberAvatar(ctx, roomID, userID, avatarURL); err != nil {
		return nil, fmt.Errorf("set member avatar: %w", err)
	}

	return s.broadcastAndBuildMember(ctx, roomID, userID)
}

func (s *service) ClearRoomAvatar(ctx context.Context, roomID, userID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	locked, err := s.effectiveLocked(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, ErrNicknameLocked
	}

	rows, err := s.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	if err == nil {
		for _, r := range rows {
			if r.UserID == userID && r.MemberAvatarURL != "" {
				_ = s.uploadSvc.Delete(r.MemberAvatarURL)
				break
			}
		}
	}

	if err := s.chatRepo.SetMemberAvatar(ctx, roomID, userID, ""); err != nil {
		return nil, fmt.Errorf("clear member avatar: %w", err)
	}

	return s.broadcastAndBuildMember(ctx, roomID, userID)
}

func (s *service) SetMemberNicknameAsMod(ctx context.Context, roomID, actorID, targetID uuid.UUID, nickname string) (*dto.ChatRoomMemberResponse, error) {
	if err := s.requireSiteMod(ctx, actorID); err != nil {
		return nil, err
	}

	if err := s.assertTargetEditable(ctx, roomID, targetID); err != nil {
		return nil, err
	}

	nickname = strings.TrimSpace(nickname)
	if len(nickname) > 32 {
		nickname = nickname[:32]
	}

	locked := nickname != ""
	if err := s.chatRepo.SetMemberNicknameWithLock(ctx, roomID, targetID, nickname, locked); err != nil {
		return nil, fmt.Errorf("set member nickname as mod: %w", err)
	}

	targetName, targetPoss := s.nameAndPossessive(ctx, targetID)
	actorName, _ := s.nameAndPossessive(ctx, actorID)
	if targetName != "" && actorName != "" {
		s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s has had %s alias locked by %s.", targetName, targetPoss, actorName))
	}

	return s.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (s *service) UnlockMemberNickname(ctx context.Context, roomID, actorID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	if err := s.requireSiteMod(ctx, actorID); err != nil {
		return nil, err
	}

	if err := s.assertTargetEditable(ctx, roomID, targetID); err != nil {
		return nil, err
	}

	if err := s.chatRepo.SetMemberNicknameWithLock(ctx, roomID, targetID, "", false); err != nil {
		return nil, fmt.Errorf("unlock nickname: %w", err)
	}

	targetName, targetPoss := s.nameAndPossessive(ctx, targetID)
	actorName, _ := s.nameAndPossessive(ctx, actorID)
	if targetName != "" && actorName != "" {
		s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s has had %s alias reset by %s.", targetName, targetPoss, actorName))
	}

	return s.broadcastAndBuildMember(ctx, roomID, targetID)
}

func (s *service) effectiveLocked(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	siteRole, err := s.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get site role: %w", err)
	}
	if siteRole.IsSiteStaff() {
		return false, nil
	}
	locked, err := s.chatRepo.IsMemberNicknameLocked(ctx, roomID, userID)
	if err != nil {
		return false, fmt.Errorf("check nickname locked: %w", err)
	}
	return locked, nil
}

func (s *service) requireSiteMod(ctx context.Context, userID uuid.UUID) error {
	siteRole, err := s.authzSvc.GetRole(ctx, userID)
	if err != nil {
		return fmt.Errorf("get site role: %w", err)
	}
	if !siteRole.IsSiteStaff() {
		return ErrModRoleRequired
	}
	return nil
}

func (s *service) assertTargetEditable(ctx context.Context, roomID, targetID uuid.UUID) error {
	targetRole, err := s.chatRepo.GetMemberRole(ctx, roomID, targetID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return ErrNotMember
	}
	siteRole, err := s.authzSvc.GetRole(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get target site role: %w", err)
	}
	if siteRole.IsSiteStaff() {
		return ErrTargetImmune
	}
	return nil
}

func (s *service) broadcastAndBuildMember(ctx context.Context, roomID, targetID uuid.UUID) (*dto.ChatRoomMemberResponse, error) {
	rows, err := s.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	vanityMap, _ := s.vanityRoleRepo.GetRolesForUsersBatch(ctx, []uuid.UUID{targetID})

	var resp *dto.ChatRoomMemberResponse
	for _, m := range rows {
		if m.UserID != targetID {
			continue
		}
		locked := m.NicknameLocked && !m.AuthorRoleTyped.IsSiteStaff()
		resp = &dto.ChatRoomMemberResponse{
			User: dto.UserResponse{
				ID:          m.UserID,
				Username:    m.Username,
				DisplayName: m.DisplayName,
				AvatarURL:   m.AvatarURL,
				Role:        m.AuthorRoleTyped,
				VanityRoles: s.toVanityRoleResponses(vanityMap[m.UserID]),
			},
			Role:            m.Role,
			JoinedAt:        m.JoinedAt,
			Nickname:        m.Nickname,
			NicknameLocked:  locked,
			MemberAvatarURL: m.MemberAvatarURL,
			TimeoutUntil:    m.TimeoutUntil,
			TimeoutByStaff:  m.TimeoutByStaff,
		}
		break
	}

	event := ws.Message{
		Type: "chat_member_updated",
		Data: map[string]interface{}{
			"room_id":              roomID,
			"user_id":              targetID,
			"nickname":             stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.Nickname }),
			"member_avatar_url":    stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.MemberAvatarURL }),
			"nickname_locked":      resp != nil && resp.NicknameLocked,
			"timeout_until":        stringOrEmpty(resp, func(r *dto.ChatRoomMemberResponse) string { return r.TimeoutUntil }),
			"timeout_set_by_staff": resp != nil && resp.TimeoutByStaff,
		},
	}
	for _, r := range rows {
		s.hub.SendToUser(r.UserID, event)
	}

	if resp == nil {
		return nil, ErrNotMember
	}
	return resp, nil
}

func stringOrEmpty(resp *dto.ChatRoomMemberResponse, get func(*dto.ChatRoomMemberResponse) string) string {
	if resp == nil {
		return ""
	}
	return get(resp)
}

func (s *service) PinMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	msg, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	canMod, err := s.canModerateRoom(ctx, msg.RoomID, userID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}

	if err := s.chatRepo.PinMessage(ctx, messageID, userID); err != nil {
		return fmt.Errorf("pin message: %w", err)
	}

	pinnedAt := time.Now().UTC().Format(time.RFC3339)
	members, _ := s.chatRepo.GetRoomMembers(ctx, msg.RoomID)
	event := ws.Message{
		Type: "chat_message_pinned",
		Data: map[string]interface{}{
			"room_id":    msg.RoomID,
			"message_id": messageID,
			"pinned_at":  pinnedAt,
			"pinned_by":  userID,
		},
	}
	for _, mid := range members {
		s.hub.SendToUser(mid, event)
	}
	return nil
}

func (s *service) UnpinMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	msg, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}
	if msg.PinnedAt == nil {
		return ErrMessageNotPinned
	}

	canMod, err := s.canModerateRoom(ctx, msg.RoomID, userID)
	if err != nil {
		return err
	}
	if !canMod {
		return ErrNotHost
	}

	if err := s.chatRepo.UnpinMessage(ctx, messageID); err != nil {
		return fmt.Errorf("unpin message: %w", err)
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, msg.RoomID)
	event := ws.Message{
		Type: "chat_message_unpinned",
		Data: map[string]interface{}{
			"room_id":    msg.RoomID,
			"message_id": messageID,
		},
	}
	for _, mid := range members {
		s.hub.SendToUser(mid, event)
	}
	return nil
}

func (s *service) ListPinnedMessages(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.ChatMessageListResponse, error) {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	rows, err := s.chatRepo.ListPinnedMessages(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list pinned messages: %w", err)
	}

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
	mediaBatch, _ := s.chatRepo.GetMessageMediaBatch(ctx, messageIDs)
	reactionBatch, _ := s.chatRepo.GetReactionsBatch(ctx, messageIDs, viewerID)
	vanityMap, _ := s.vanityRoleRepo.GetRolesForUsersBatch(ctx, senderIDs)

	messages := make([]dto.ChatMessageResponse, 0, len(rows))
	for i := range rows {
		row := rows[i]
		messages = append(messages, s.messageRowToResponse(row, mediaBatch[row.ID], reactionBatch[row.ID], s.toVanityRoleResponses(vanityMap[row.SenderID])))
	}

	return &dto.ChatMessageListResponse{
		Messages: messages,
		Total:    len(messages),
	}, nil
}

func validateEmoji(emoji string) error {
	if emoji == "" || len(emoji) > 16 {
		return ErrInvalidEmoji
	}
	return nil
}

func (s *service) resolveMemberDisplayName(ctx context.Context, roomID, userID uuid.UUID) string {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return ""
	}
	name := user.DisplayName
	if name == "" {
		name = user.Username
	}
	rows, _ := s.chatRepo.GetRoomMembersDetailed(ctx, roomID)
	for _, mr := range rows {
		if mr.UserID == userID {
			if mr.Nickname != "" {
				name = mr.Nickname
			}
			break
		}
	}
	return name
}

func (s *service) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if err := validateEmoji(emoji); err != nil {
		return err
	}

	msg, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	isMember, err := s.chatRepo.IsMember(ctx, msg.RoomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	activeTimeout, timeoutUntil, _, err := s.chatRepo.GetMemberTimeoutState(ctx, msg.RoomID, userID)
	if err != nil {
		return fmt.Errorf("get timeout state: %w", err)
	}
	if activeTimeout {
		if timeoutUntil != "" {
			return fmt.Errorf("%w until %s", ErrTimedOut, formatTimeoutUntilForUser(timeoutUntil))
		}
		return ErrTimedOut
	}

	if err := s.chatRepo.AddReaction(ctx, messageID, userID, emoji); err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}

	displayName := s.resolveMemberDisplayName(ctx, msg.RoomID, userID)
	members, _ := s.chatRepo.GetRoomMembers(ctx, msg.RoomID)
	event := ws.Message{
		Type: "chat_reaction_added",
		Data: map[string]interface{}{
			"room_id":      msg.RoomID,
			"message_id":   messageID,
			"emoji":        emoji,
			"user_id":      userID,
			"display_name": displayName,
		},
	}
	for _, mid := range members {
		s.hub.SendToUser(mid, event)
	}
	return nil
}

func (s *service) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if err := validateEmoji(emoji); err != nil {
		return err
	}

	msg, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	isMember, err := s.chatRepo.IsMember(ctx, msg.RoomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	if err := s.chatRepo.RemoveReaction(ctx, messageID, userID, emoji); err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}

	displayName := s.resolveMemberDisplayName(ctx, msg.RoomID, userID)
	members, _ := s.chatRepo.GetRoomMembers(ctx, msg.RoomID)
	event := ws.Message{
		Type: "chat_reaction_removed",
		Data: map[string]interface{}{
			"room_id":      msg.RoomID,
			"message_id":   messageID,
			"emoji":        emoji,
			"user_id":      userID,
			"display_name": displayName,
		},
	}
	for _, mid := range members {
		s.hub.SendToUser(mid, event)
	}
	return nil
}

func (s *service) EditMessage(ctx context.Context, messageID, actorID uuid.UUID, body string) (*dto.ChatMessageResponse, error) {
	if body == "" {
		return nil, ErrMissingFields
	}
	if err := s.filterTexts(ctx, body); err != nil {
		return nil, err
	}

	msg, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return nil, ErrRoomNotFound
	}
	if msg.IsSystem {
		return nil, ErrCannotEditSystemMessage
	}
	if msg.SenderID != actorID {
		return nil, ErrMessageEditPermission
	}

	activeTimeout, timeoutUntil, _, err := s.chatRepo.GetMemberTimeoutState(ctx, msg.RoomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("get timeout state: %w", err)
	}
	if activeTimeout {
		if timeoutUntil != "" {
			return nil, fmt.Errorf("%w until %s", ErrTimedOut, formatTimeoutUntilForUser(timeoutUntil))
		}
		return nil, ErrTimedOut
	}

	if err := s.chatRepo.EditMessage(ctx, messageID, body); err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}

	updated, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil || updated == nil {
		return nil, fmt.Errorf("reload message: %w", err)
	}

	mediaBatch, _ := s.chatRepo.GetMessageMediaBatch(ctx, []uuid.UUID{messageID})
	reactionBatch, _ := s.chatRepo.GetReactionsBatch(ctx, []uuid.UUID{messageID}, actorID)
	vanityRows, _ := s.vanityRoleRepo.GetRolesForUser(ctx, updated.SenderID)
	resp := s.messageRowToResponse(*updated, mediaBatch[messageID], reactionBatch[messageID], s.toVanityRoleResponses(vanityRows))

	members, _ := s.chatRepo.GetRoomMembers(ctx, msg.RoomID)
	event := ws.Message{
		Type: "chat_message_edited",
		Data: resp,
	}
	for _, mid := range members {
		s.hub.SendToUser(mid, event)
	}

	return &resp, nil
}

func (s *service) DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error {
	msg, err := s.chatRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if msg == nil {
		return ErrRoomNotFound
	}

	if msg.SenderRoleTyped.IsSiteStaff() && msg.SenderID != actorID {
		return ErrCannotDeleteStaffMessage
	}

	if msg.SenderID != actorID {
		canMod, err := s.canModerateRoom(ctx, msg.RoomID, actorID)
		if err != nil {
			return err
		}
		if !canMod {
			return ErrMessageDeletePermission
		}
	}

	if err := s.chatRepo.DeleteMessage(ctx, messageID); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	members, _ := s.chatRepo.GetRoomMembers(ctx, msg.RoomID)
	event := ws.Message{
		Type: "chat_message_deleted",
		Data: map[string]interface{}{
			"room_id":    msg.RoomID,
			"message_id": messageID,
		},
	}
	for _, mid := range members {
		s.hub.SendToUser(mid, event)
	}
	return nil
}

func (s *service) getRoomMemberResponses(ctx context.Context, roomID, viewerID uuid.UUID) ([]dto.UserResponse, int, error) {
	memberIDs, err := s.chatRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, 0, fmt.Errorf("get room members: %w", err)
	}

	hasGhost, _ := s.chatRepo.HasGhostMembers(ctx, roomID)
	var viewerIsStaff bool
	if hasGhost {
		r, _ := s.authzSvc.GetRole(ctx, viewerID)
		viewerIsStaff = r.IsSiteStaff()
	}

	members := make([]dto.UserResponse, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if hasGhost && !viewerIsStaff {
			ghost, _ := s.chatRepo.IsGhostMember(ctx, roomID, memberID)
			if ghost {
				continue
			}
		}
		user, err := s.userRepo.GetByID(ctx, memberID)
		if err != nil || user == nil {
			continue
		}
		members = append(members, *user.ToResponse())
	}
	return members, len(members), nil
}
