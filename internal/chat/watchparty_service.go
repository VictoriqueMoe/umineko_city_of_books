package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"umineko_city_of_books/internal/watchparty"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/hyperbeam"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	defaultOfflineTimeout = 300
	defaultSessionTimeout = 14400
	maxWatchPartyTitleLen = 80
	maxWatchPartyBodyLen  = 2000

	watchPartyReconcileEvery     = 5 * time.Minute
	watchPartyReconcileIdleAfter = 6 * time.Minute

	wsWatchPartyStarted         = "watch_party_started"
	wsWatchPartyEnded           = "watch_party_ended"
	wsWatchPartyParticipantJoin = "watch_party_participant_joined"
	wsWatchPartyParticipantLeft = "watch_party_participant_left"
	wsWatchPartyControlChanged  = "watch_party_control_changed"
	wsWatchPartyMessage         = "watch_party_message"
	wsWatchPartyKicked          = "watch_party_kicked"

	watchPartyTypeHyperbeam   = "hyperbeam"
	watchPartyTypeScreenShare = "screenshare"
)

type watchPartyService struct {
	*core
}

func normaliseWatchPartyType(t string) (string, error) {
	switch t {
	case "", watchPartyTypeHyperbeam:
		return watchPartyTypeHyperbeam, nil

	case watchPartyTypeScreenShare:
		return watchPartyTypeScreenShare, nil

	default:
		return "", ErrWatchPartyInvalidType
	}
}

func (s *watchPartyService) screenShareEnabled() bool {
	return s.livekitSvc != nil && s.livekitSvc.Enabled()
}

func (s *watchPartyService) StartWatchParty(ctx context.Context, roomID, actorID uuid.UUID, startURL, region, title, sessionType string) (*dto.StartWatchPartyResponse, error) {
	partyType, err := normaliseWatchPartyType(sessionType)
	if err != nil {
		return nil, err
	}

	if partyType == watchPartyTypeScreenShare {
		if !s.screenShareEnabled() {
			return nil, ErrWatchPartyDisabled
		}
	} else if s.hyperbeamSvc == nil || !s.hyperbeamSvc.Enabled() {
		return nil, ErrWatchPartyDisabled
	}

	if err := s.assertActiveRoomMember(ctx, roomID, actorID); err != nil {
		return nil, err
	}

	room, err := s.chatRepo.GetRoomByID(ctx, roomID, actorID)
	if err != nil {
		return nil, fmt.Errorf("load room for watch party: %w", err)
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	if room.Type != "group" || room.IsSystem {
		return nil, ErrWatchPartyWrongRoomType
	}

	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) > maxWatchPartyTitleLen {
		trimmedTitle = trimmedTitle[:maxWatchPartyTitleLen]
	}

	sessionRow := repository.ChatWatchPartySessionRow{
		RoomID:       roomID,
		StartedBy:    actorID,
		ControllerID: actorID,
		Title:        trimmedTitle,
		Type:         partyType,
	}

	embedURL := ""
	if partyType == watchPartyTypeHyperbeam {
		selectedRegion := region
		if selectedRegion == "" {
			selectedRegion = config.Cfg.HyperbeamRegion
		}

		vm, err := s.hyperbeamSvc.CreateVM(ctx, hyperbeam.CreateVMOptions{
			StartURL: startURL,
			Region:   selectedRegion,
			Timeout: &hyperbeam.VMTimeoutOpts{
				Offline:  defaultOfflineTimeout,
				Absolute: defaultSessionTimeout,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("create hyperbeam vm: %w", err)
		}

		vmBaseURL, baseErr := hyperbeam.ExtractVMBaseURL(vm.EmbedURL)
		if baseErr != nil {
			s.terminateHyperbeam(vm.SessionID)
			return nil, fmt.Errorf("extract vm base url: %w", baseErr)
		}

		sessionRow.HyperbeamSessionID = vm.SessionID
		sessionRow.HyperbeamAdminToken = vm.AdminToken
		sessionRow.EmbedURL = vm.EmbedURL
		sessionRow.VMBaseURL = vmBaseURL
		sessionRow.StartURL = stringToNull(startURL)
		sessionRow.Region = stringToNull(selectedRegion)
		embedURL = vm.EmbedURL
	}

	sessionID, err := s.watchPartyRepo.CreateSession(ctx, sessionRow)
	if err != nil {
		if sessionRow.HyperbeamSessionID != "" {
			s.terminateHyperbeam(sessionRow.HyperbeamSessionID)
		}
		return nil, err
	}

	if err := s.watchPartyRepo.UpsertParticipant(ctx, sessionID, actorID, true, ""); err != nil {
		if sessionRow.HyperbeamSessionID != "" {
			s.terminateHyperbeam(sessionRow.HyperbeamSessionID)
		}
		_ = s.watchPartyRepo.EndSession(ctx, sessionID, "controller_setup_failed")
		return nil, err
	}

	details := mustJSON(map[string]any{"room_id": roomID, "start_url": startURL, "title": trimmedTitle, "type": partyType})
	if err := s.auditRepo.Create(ctx, actorID, "watch_party.start", "chat_watch_party_session", sessionID.String(), details); err != nil {
		logger.Log.Warn().Err(err).Msg("audit watch_party.start failed")
	}

	stored, err := s.watchPartyRepo.GetByID(ctx, sessionID)
	if err != nil || stored == nil {
		return nil, fmt.Errorf("reload watch party session: %w", err)
	}

	sessionDTO, err := s.buildWatchPartySessionDTO(ctx, stored, actorID, embedURL, true)
	if err != nil {
		return nil, err
	}

	broadcast := s.buildWatchPartySessionDTOForBroadcast(ctx, stored)
	s.hub.BroadcastToRoom(roomID, ws.Message{
		Type: wsWatchPartyStarted,
		Data: dto.WatchPartyStartedEvent{Session: broadcast},
	}, uuid.Nil)

	hostName, _ := s.nameAndPossessive(ctx, actorID)
	if hostName == "" {
		hostName = "Someone"
	}
	partyLabel := trimmedTitle
	if partyLabel == "" {
		partyLabel = "Untitled party"
	}
	s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s is hosting a watch party: %s", hostName, partyLabel))

	return &dto.StartWatchPartyResponse{
		Session:  *sessionDTO,
		EmbedURL: embedURL,
	}, nil
}

func (s *watchPartyService) JoinWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID) (*dto.JoinWatchPartyResponse, error) {
	if err := s.assertActiveRoomMember(ctx, roomID, actorID); err != nil {
		return nil, err
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return nil, err
	}

	if session.Type == watchPartyTypeScreenShare {
		if !s.screenShareEnabled() {
			return nil, ErrWatchPartyDisabled
		}
	} else {
		if s.hyperbeamSvc == nil || !s.hyperbeamSvc.Enabled() {
			return nil, ErrWatchPartyDisabled
		}
		if _, statusErr := s.hyperbeamSvc.GetVMStatus(ctx, session.HyperbeamSessionID); statusErr != nil {
			if hyperbeamSessionGone(statusErr) {
				s.cleanupDeadSession(session, "vm_gone")
				return nil, ErrWatchPartyNotActive
			}
			logger.Log.Warn().Err(statusErr).Str("hyperbeam_session_id", session.HyperbeamSessionID).Msg("vm status check failed (continuing)")
		}
	}

	isController := session.ControllerID == actorID
	existing, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, actorID)
	if err != nil {
		return nil, err
	}
	hasControl := isController || (existing != nil && existing.HasControl && !existing.LeftAt.Valid)

	if err := s.watchPartyRepo.UpsertParticipant(ctx, session.ID, actorID, hasControl, ""); err != nil {
		return nil, err
	}

	participantDTO, err := s.buildWatchPartyParticipantDTO(ctx, session.ID, actorID, hasControl)
	if err != nil {
		return nil, err
	}
	s.hub.BroadcastToRoom(roomID, ws.Message{
		Type: wsWatchPartyParticipantJoin,
		Data: dto.WatchPartyParticipantEvent{
			SessionID:   session.ID,
			RoomID:      roomID,
			Participant: *participantDTO,
		},
	}, actorID)

	sessionDTO, err := s.buildWatchPartySessionDTO(ctx, session, actorID, session.EmbedURL, hasControl)
	if err != nil {
		return nil, err
	}

	return &dto.JoinWatchPartyResponse{
		Session:  *sessionDTO,
		EmbedURL: session.EmbedURL,
	}, nil
}

func (s *watchPartyService) LeaveWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID) error {
	session, err := s.watchPartyRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil || session.RoomID != roomID || session.Status != "active" {
		return nil
	}

	participant, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, actorID)
	if err != nil {
		return err
	}
	if participant == nil || participant.LeftAt.Valid {
		return nil
	}

	if session.StartedBy == actorID {
		return s.EndWatchParty(ctx, roomID, sessionID, actorID, "owner_left")
	}

	if participant.HasControl {
		if err := s.transferControlTo(ctx, roomID, session, session.StartedBy); err != nil {
			logger.Log.Warn().Err(err).Msg("auto-return control to owner on leave failed")
		} else {
			s.postControlChangeSystemMessage(ctx, roomID, session.ID, actorID, session.StartedBy, "auto_owner_return")
		}
	}

	if err := s.watchPartyRepo.MarkParticipantLeft(ctx, session.ID, actorID); err != nil {
		return err
	}

	s.hub.BroadcastToRoom(roomID, ws.Message{
		Type: wsWatchPartyParticipantLeft,
		Data: dto.WatchPartyParticipantLeftEvent{
			SessionID: session.ID,
			RoomID:    roomID,
			UserID:    actorID,
		},
	}, uuid.Nil)

	return nil
}

func (s *watchPartyService) KickWatchPartyParticipant(ctx context.Context, roomID, sessionID, callerID, targetID uuid.UUID) error {
	if callerID == targetID {
		return ErrWatchPartyCannotKickSelf
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return err
	}

	target, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, targetID)
	if err != nil {
		return err
	}
	if target == nil || target.LeftAt.Valid {
		return ErrWatchPartyNotParticipant
	}

	callerRank := s.watchPartyRankOf(ctx, session, callerID)
	targetRank := s.watchPartyRankOf(ctx, session, targetID)
	if callerRank <= targetRank {
		return ErrWatchPartyCannotKick
	}

	if target.HasControl {
		if err := s.transferControlTo(ctx, roomID, session, session.StartedBy); err != nil {
			logger.Log.Warn().Err(err).Msg("auto-return control on kick failed")
		} else if session.StartedBy != targetID {
			s.postControlChangeSystemMessage(ctx, roomID, session.ID, callerID, session.StartedBy, "auto_owner_return")
		}
	}

	if err := s.watchPartyRepo.MarkParticipantLeft(ctx, session.ID, targetID); err != nil {
		return err
	}

	s.hub.BroadcastToRoom(roomID, ws.Message{
		Type: wsWatchPartyParticipantLeft,
		Data: dto.WatchPartyParticipantLeftEvent{
			SessionID: session.ID,
			RoomID:    roomID,
			UserID:    targetID,
		},
	}, uuid.Nil)

	s.hub.SendToUser(targetID, ws.Message{
		Type: wsWatchPartyKicked,
		Data: dto.WatchPartyKickedEvent{
			SessionID: session.ID,
			RoomID:    roomID,
			ActorID:   callerID,
		},
	})

	body := fmt.Sprintf("%s was kicked by %s.", s.displayNameFor(ctx, targetID, roomID), s.displayNameFor(ctx, callerID, roomID))
	s.postWatchPartySystemMessage(ctx, roomID, session.ID, body)

	details := mustJSON(map[string]any{"room_id": roomID, "target_user_id": targetID})
	if err := s.auditRepo.Create(ctx, callerID, "watch_party.kick", "chat_watch_party_session", session.ID.String(), details); err != nil {
		logger.Log.Warn().Err(err).Msg("audit watch_party.kick failed")
	}

	return nil
}

func (s *watchPartyService) HandleClientDisconnect(ctx context.Context, userID uuid.UUID, roomIDs []uuid.UUID) {
	if s.watchPartyRepo == nil {
		return
	}
	for _, roomID := range roomIDs {
		sessions, err := s.watchPartyRepo.ListActiveByRoom(ctx, roomID)
		if err != nil {
			logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("disconnect: list active watch parties failed")
			continue
		}
		for i := range sessions {
			sess := sessions[i]
			participant, err := s.watchPartyRepo.GetParticipant(ctx, sess.ID, userID)
			if err != nil || participant == nil || participant.LeftAt.Valid {
				continue
			}
			if sess.StartedBy == userID {
				_ = s.EndWatchParty(ctx, roomID, sess.ID, userID, "owner_disconnected")
				continue
			}
			if participant.HasControl {
				if err := s.transferControlTo(ctx, roomID, &sess, sess.StartedBy); err != nil {
					logger.Log.Warn().Err(err).Msg("disconnect: auto-return control failed")
				} else {
					s.postControlChangeSystemMessage(ctx, roomID, sess.ID, userID, sess.StartedBy, "auto_owner_return")
				}
			}
			if err := s.watchPartyRepo.MarkParticipantLeft(ctx, sess.ID, userID); err != nil {
				logger.Log.Warn().Err(err).Msg("disconnect: mark participant left failed")
				continue
			}
			s.hub.BroadcastToRoom(roomID, ws.Message{
				Type: wsWatchPartyParticipantLeft,
				Data: dto.WatchPartyParticipantLeftEvent{
					SessionID: sess.ID,
					RoomID:    roomID,
					UserID:    userID,
				},
			}, uuid.Nil)
		}
	}
}

func (s *watchPartyService) GrantWatchPartyControl(ctx context.Context, roomID, sessionID, callerID, targetID uuid.UUID) error {
	if s.hyperbeamSvc == nil || !s.hyperbeamSvc.Enabled() {
		return ErrWatchPartyDisabled
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return err
	}

	caller, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, callerID)
	if err != nil {
		return err
	}
	callerIsController := caller != nil && !caller.LeftAt.Valid && caller.HasControl

	if !callerIsController {
		callerRank := s.watchPartyRankOf(ctx, session, callerID)
		controllerRank := s.watchPartyRankOf(ctx, session, session.ControllerID)
		if callerRank <= controllerRank {
			return ErrWatchPartyOutranked
		}
	}

	target, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, targetID)
	if err != nil {
		return err
	}
	if target == nil || target.LeftAt.Valid {
		return ErrWatchPartyNotParticipant
	}
	if target.HasControl {
		return nil
	}

	if err := s.transferControlTo(ctx, roomID, session, targetID); err != nil {
		return err
	}

	reason := "pass"
	if callerID == targetID {
		reason = "reclaim"
	}
	s.postControlChangeSystemMessage(ctx, roomID, session.ID, callerID, targetID, reason)

	details := mustJSON(map[string]any{"room_id": roomID, "target_user_id": targetID})
	if err := s.auditRepo.Create(ctx, callerID, "watch_party.grant_control", "chat_watch_party_session", session.ID.String(), details); err != nil {
		logger.Log.Warn().Err(err).Msg("audit watch_party.grant_control failed")
	}

	return nil
}

func (s *watchPartyService) transferControlTo(ctx context.Context, roomID uuid.UUID, session *repository.ChatWatchPartySessionRow, targetID uuid.UUID) error {
	participants, err := s.watchPartyRepo.GetActiveParticipants(ctx, session.ID)
	if err != nil {
		return err
	}
	for i := range participants {
		p := participants[i]
		if !p.HasControl || p.UserID == targetID {
			continue
		}
		if p.HyperbeamIdentifier != "" && session.VMBaseURL != "" {
			if err := s.hyperbeamSvc.SetControlRole(ctx, session.VMBaseURL, p.HyperbeamIdentifier, false); err != nil {
				logger.Log.Warn().Err(err).Str("user_id", p.UserID.String()).Msg("transfer: demote previous controller failed")
			}
		}
		if err := s.watchPartyRepo.SetParticipantControl(ctx, session.ID, p.UserID, false); err != nil {
			return err
		}
		s.hub.BroadcastToRoom(roomID, ws.Message{
			Type: wsWatchPartyControlChanged,
			Data: dto.WatchPartyControlChangedEvent{
				SessionID:  session.ID,
				RoomID:     roomID,
				UserID:     p.UserID,
				HasControl: false,
			},
		}, uuid.Nil)
	}

	var targetIdentifier string
	for i := range participants {
		if participants[i].UserID == targetID {
			targetIdentifier = participants[i].HyperbeamIdentifier
			break
		}
	}
	if targetIdentifier != "" && session.VMBaseURL != "" {
		if err := s.hyperbeamSvc.SetControlRole(ctx, session.VMBaseURL, targetIdentifier, true); err != nil {
			logger.Log.Warn().Err(err).Str("user_id", targetID.String()).Msg("transfer: promote target permissions failed (continuing)")
		}
	}
	if err := s.watchPartyRepo.SetParticipantControl(ctx, session.ID, targetID, true); err != nil {
		return err
	}
	if err := s.watchPartyRepo.SetControllerID(ctx, session.ID, targetID); err != nil {
		logger.Log.Warn().Err(err).Msg("transfer: update controller_id failed")
	}
	s.hub.BroadcastToRoom(roomID, ws.Message{
		Type: wsWatchPartyControlChanged,
		Data: dto.WatchPartyControlChangedEvent{
			SessionID:  session.ID,
			RoomID:     roomID,
			UserID:     targetID,
			HasControl: true,
		},
	}, uuid.Nil)
	return nil
}

func (s *watchPartyService) EndWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID, reason string) error {
	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return err
	}

	if actorID != uuid.Nil {
		caller, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, actorID)
		if err != nil {
			return err
		}
		if caller == nil || caller.LeftAt.Valid || !caller.HasControl {
			actorRole, _ := s.roleRepo.GetRole(ctx, actorID)
			if !actorRole.IsSiteStaff() {
				return ErrWatchPartyNotController
			}
		}
	}

	if s.hyperbeamSvc != nil && session.Type != watchPartyTypeScreenShare {
		if err := s.hyperbeamSvc.TerminateVM(ctx, session.HyperbeamSessionID); err != nil {
			logger.Log.Warn().Err(err).Str("hyperbeam_session_id", session.HyperbeamSessionID).Msg("terminate hyperbeam vm failed")
		}
	}

	s.cleanupDeadSession(session, reason)

	if actorID != uuid.Nil {
		details := mustJSON(map[string]any{"room_id": roomID, "reason": reason})
		if err := s.auditRepo.Create(ctx, actorID, "watch_party.end", "chat_watch_party_session", session.ID.String(), details); err != nil {
			logger.Log.Warn().Err(err).Msg("audit watch_party.end failed")
		}

		hostName, _ := s.nameAndPossessive(ctx, session.StartedBy)
		if hostName == "" {
			hostName = "Someone"
		}
		partyLabel := strings.TrimSpace(session.Title)
		if partyLabel == "" {
			partyLabel = "Untitled party"
		}
		s.postRoomActionMessage(ctx, roomID, actorID, fmt.Sprintf("%s's watch party ended: %s", hostName, partyLabel))
	}

	return nil
}

func (s *watchPartyService) IdentifyWatchPartyParticipant(ctx context.Context, roomID, sessionID, userID uuid.UUID, identifier string) error {
	if identifier == "" {
		return ErrWatchPartyMessageEmpty
	}
	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return err
	}
	participant, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, userID)
	if err != nil {
		return err
	}
	if participant == nil || participant.LeftAt.Valid {
		return ErrWatchPartyNotParticipant
	}
	if err := s.watchPartyRepo.SetParticipantIdentifier(ctx, session.ID, userID, identifier); err != nil {
		return err
	}
	if s.hyperbeamSvc != nil && s.hyperbeamSvc.Enabled() && session.VMBaseURL != "" {
		if err := s.hyperbeamSvc.SetControlRole(ctx, session.VMBaseURL, identifier, participant.HasControl); err != nil {
			logger.Log.Warn().Err(err).Str("hyperbeam_session_id", session.HyperbeamSessionID).Msg("identify: set user permissions failed")
		}
	}
	return nil
}

func (s *watchPartyService) ListWatchParties(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.WatchPartyListResponse, error) {
	if err := s.assertActiveRoomMember(ctx, roomID, viewerID); err != nil {
		return nil, err
	}
	rows, err := s.watchPartyRepo.ListActiveByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	sessions := make([]dto.WatchPartySession, 0, len(rows))
	for i := range rows {
		row := rows[i]
		if row.Type != watchPartyTypeScreenShare && s.hyperbeamSvc != nil && s.hyperbeamSvc.Enabled() {
			if _, statusErr := s.hyperbeamSvc.GetVMStatus(ctx, row.HyperbeamSessionID); statusErr != nil {
				if hyperbeamSessionGone(statusErr) {
					s.cleanupDeadSession(&row, "vm_gone")
					continue
				}
				logger.Log.Warn().Err(statusErr).Str("hyperbeam_session_id", row.HyperbeamSessionID).Msg("list watch parties: vm status check failed")
			}
		}
		s2, err := s.buildWatchPartySessionDTO(ctx, &row, viewerID, "", false)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *s2)
	}
	return &dto.WatchPartyListResponse{
		Sessions:           sessions,
		Enabled:            s.WatchPartyEnabled(),
		ScreenShareEnabled: s.screenShareEnabled(),
	}, nil
}

func (s *watchPartyService) MintSessionVoiceToken(ctx context.Context, roomID, sessionID, userID uuid.UUID) (token, url string, err error) {
	if !s.screenShareEnabled() {
		return "", "", ErrVoiceDisabled
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return "", "", err
	}

	participant, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, userID)
	if err != nil {
		return "", "", err
	}
	if participant == nil || participant.LeftAt.Valid {
		return "", "", ErrWatchPartyNotParticipant
	}

	allowScreenShare := session.Type == watchPartyTypeScreenShare && session.StartedBy == userID
	roomName := voiceSessionRoomPrefix + session.ID.String()
	displayName := s.displayNameFor(ctx, userID, roomID)
	canMic := !s.isVoiceMuted(roomName, userID)

	token, err = s.livekitSvc.MintToken(roomName, userID.String(), displayName, canMic, allowScreenShare)
	if err != nil {
		return "", "", err
	}

	return token, s.livekitSvc.URL(), nil
}

func (s *watchPartyService) ForceMuteSessionVoice(ctx context.Context, roomID, sessionID, actorID, targetID uuid.UUID, muted bool) error {
	if !s.screenShareEnabled() {
		return ErrVoiceDisabled
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return err
	}

	callerRank := s.watchPartyRankOf(ctx, session, actorID)
	targetRank := s.watchPartyRankOf(ctx, session, targetID)
	if callerRank <= targetRank {
		return ErrVoiceMuteForbidden
	}

	roomName := voiceSessionRoomPrefix + session.ID.String()
	allowScreenShare := session.Type == watchPartyTypeScreenShare && session.StartedBy == targetID

	s.setVoiceMuted(roomName, targetID, muted)

	return s.livekitSvc.SetCanPublish(ctx, roomName, targetID.String(), !muted, allowScreenShare)
}

func (s *watchPartyService) SendWatchPartyMessage(ctx context.Context, roomID, sessionID, senderID uuid.UUID, body string) (*dto.WatchPartyMessage, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil, ErrWatchPartyMessageEmpty
	}
	if len(trimmed) > maxWatchPartyBodyLen {
		return nil, ErrWatchPartyMessageTooLong
	}
	if err := s.filterTexts(ctx, trimmed); err != nil {
		return nil, err
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return nil, err
	}

	participant, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, senderID)
	if err != nil {
		return nil, err
	}
	if participant == nil || participant.LeftAt.Valid {
		return nil, ErrWatchPartyNotParticipant
	}

	msgID := uuid.New()
	if err := s.watchPartyRepo.InsertMessage(ctx, msgID, session.ID, senderID, trimmed); err != nil {
		return nil, err
	}

	row, err := s.watchPartyRepo.GetMessageByID(ctx, msgID)
	if err != nil || row == nil {
		return nil, fmt.Errorf("reload watch party message: %w", err)
	}
	msgDTO := s.buildWatchPartyMessageDTO(ctx, row)

	s.broadcastWatchPartyMessage(ctx, session.ID, roomID, msgDTO)

	return &msgDTO, nil
}

func (s *watchPartyService) GetWatchPartyMessages(ctx context.Context, roomID, sessionID, viewerID uuid.UUID) (*dto.WatchPartyMessagesResponse, error) {
	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return nil, err
	}
	participant, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, viewerID)
	if err != nil {
		return nil, err
	}
	if participant == nil || participant.LeftAt.Valid {
		return nil, ErrWatchPartyNotParticipant
	}
	rows, err := s.watchPartyRepo.ListMessages(ctx, session.ID, 200)
	if err != nil {
		return nil, err
	}
	out := make([]dto.WatchPartyMessage, 0, len(rows))
	for i := range rows {
		out = append(out, s.buildWatchPartyMessageDTO(ctx, &rows[i]))
	}
	return &dto.WatchPartyMessagesResponse{Messages: out}, nil
}

func (s *watchPartyService) WatchPartyEnabled() bool {
	return s.hyperbeamSvc != nil && s.hyperbeamSvc.Enabled()
}

func (s *watchPartyService) ReconcileWatchPartiesOnce(ctx context.Context) {
	if s.watchPartyRepo == nil {
		return
	}
	cutoff := time.Now().UTC().Add(-watchPartyReconcileIdleAfter).Format(time.RFC3339Nano)
	sessions, err := s.watchPartyRepo.ListIdleActiveSessions(ctx, cutoff)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("reconcile watch parties: list failed")
		return
	}
	for i := range sessions {
		session := sessions[i]
		if s.hyperbeamSvc != nil && session.Type != watchPartyTypeScreenShare {
			if err := s.hyperbeamSvc.TerminateVM(ctx, session.HyperbeamSessionID); err != nil {
				logger.Log.Warn().Err(err).Str("hyperbeam_session_id", session.HyperbeamSessionID).Msg("reconcile: terminate vm failed")
			}
		}
		s.cleanupDeadSession(&session, "idle_reconcile")
	}
}

func (s *watchPartyService) StartWatchPartyReconcileLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(watchPartyReconcileEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				{
					return
				}
			case <-ticker.C:
				{
					s.ReconcileWatchPartiesOnce(ctx)
				}
			}
		}
	}()
}

func (s *watchPartyService) assertActiveRoomMember(ctx context.Context, roomID, userID uuid.UUID) error {
	isMember, err := s.chatRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}
	return nil
}

func (s *watchPartyService) loadActiveSession(ctx context.Context, roomID, sessionID uuid.UUID) (*repository.ChatWatchPartySessionRow, error) {
	session, err := s.watchPartyRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.RoomID != roomID || session.Status != "active" {
		return nil, ErrWatchPartyNotActive
	}
	return session, nil
}

func (s *watchPartyService) terminateHyperbeam(sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.hyperbeamSvc.TerminateVM(ctx, sessionID); err != nil {
		logger.Log.Warn().Err(err).Str("hyperbeam_session_id", sessionID).Msg("cleanup terminate vm failed")
	}
}

func hyperbeamSessionGone(err error) bool {
	if apiErr, ok := errors.AsType[*hyperbeam.APIError](err); ok {
		return apiErr.StatusCode == 404 || apiErr.StatusCode == 410
	}
	return false
}

func (s *watchPartyService) cleanupDeadSession(session *repository.ChatWatchPartySessionRow, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.watchPartyRepo.MarkAllParticipantsLeft(ctx, session.ID); err != nil {
		logger.Log.Warn().Err(err).Msg("cleanup dead session: mark participants left failed")
	}
	if err := s.watchPartyRepo.EndSession(ctx, session.ID, reason); err != nil {
		logger.Log.Warn().Err(err).Msg("cleanup dead session: end session failed")
	}
	if err := s.watchPartyRepo.DeleteMessagesForSession(ctx, session.ID); err != nil {
		logger.Log.Warn().Err(err).Msg("cleanup dead session: delete watch party messages failed")
	}

	s.clearVoiceMuted(voiceSessionRoomPrefix + session.ID.String())

	s.hub.BroadcastToRoom(session.RoomID, ws.Message{
		Type: wsWatchPartyEnded,
		Data: dto.WatchPartyEndedEvent{
			SessionID: session.ID,
			RoomID:    session.RoomID,
			Reason:    reason,
		},
	}, uuid.Nil)
}

func (s *watchPartyService) broadcastWatchPartyMessage(ctx context.Context, sessionID, roomID uuid.UUID, msg dto.WatchPartyMessage) {
	participants, err := s.watchPartyRepo.GetActiveParticipants(ctx, sessionID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("broadcast watch party message: list participants failed")
		return
	}
	event := ws.Message{
		Type: wsWatchPartyMessage,
		Data: dto.WatchPartyMessageEvent{
			SessionID: sessionID,
			RoomID:    roomID,
			Message:   msg,
		},
	}
	for i := range participants {
		s.hub.SendToUser(participants[i].UserID, event)
	}
}

func (s *watchPartyService) buildWatchPartySessionDTO(ctx context.Context, session *repository.ChatWatchPartySessionRow, viewerID uuid.UUID, viewerEmbedURL string, viewerHasControl bool) (*dto.WatchPartySession, error) {
	participants, err := s.buildParticipantsDTO(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	out := dto.WatchPartySession{
		ID:           session.ID,
		RoomID:       session.RoomID,
		StartedBy:    session.StartedBy,
		ControllerID: session.ControllerID,
		Title:        session.Title,
		Type:         session.Type,
		StartURL:     nullToString(session.StartURL),
		Region:       nullToString(session.Region),
		Status:       session.Status,
		StartedAt:    session.StartedAt,
		EndedAt:      nullToString(session.EndedAt),
		Participants: participants,
	}
	if viewerID != uuid.Nil {
		participant, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, viewerID)
		if err != nil {
			return nil, err
		}
		isParticipant := participant != nil && !participant.LeftAt.Valid
		hasControl := false
		if isParticipant {
			hasControl = participant.HasControl
		}
		if !hasControl {
			hasControl = viewerHasControl
		}
		out.Viewer = &dto.WatchPartyViewerContext{
			IsParticipant: isParticipant,
			HasControl:    hasControl,
			EmbedURL:      viewerEmbedURL,
		}
	}
	return &out, nil
}

func (s *watchPartyService) buildWatchPartySessionDTOForBroadcast(ctx context.Context, session *repository.ChatWatchPartySessionRow) dto.WatchPartySession {
	participants, _ := s.buildParticipantsDTO(ctx, session.ID)
	return dto.WatchPartySession{
		ID:           session.ID,
		RoomID:       session.RoomID,
		StartedBy:    session.StartedBy,
		ControllerID: session.ControllerID,
		Title:        session.Title,
		Type:         session.Type,
		StartURL:     nullToString(session.StartURL),
		Region:       nullToString(session.Region),
		Status:       session.Status,
		StartedAt:    session.StartedAt,
		Participants: participants,
	}
}

func (s *watchPartyService) buildParticipantsDTO(ctx context.Context, sessionID uuid.UUID) ([]dto.WatchPartyParticipant, error) {
	rows, err := s.watchPartyRepo.GetActiveParticipants(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []dto.WatchPartyParticipant{}, nil
	}
	userIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		userIDs = append(userIDs, rows[i].UserID)
	}
	roleMap, _ := s.roleRepo.GetRoles(ctx, userIDs)
	vanityMap, _ := s.vanityRoleRepo.GetRolesForUsersBatch(ctx, userIDs)

	out := make([]dto.WatchPartyParticipant, 0, len(rows))
	for i := range rows {
		p := rows[i]
		out = append(out, dto.WatchPartyParticipant{
			User: dto.UserResponse{
				ID:          p.UserID,
				Username:    p.Username,
				DisplayName: p.DisplayName,
				AvatarURL:   p.AvatarURL,
				Role:        roleMap[p.UserID],
				VanityRoles: s.toVanityRoleResponses(vanityMap[p.UserID]),
			},
			HasControl: p.HasControl,
			JoinedAt:   p.JoinedAt,
		})
	}
	return out, nil
}

func (s *watchPartyService) buildWatchPartyParticipantDTO(ctx context.Context, sessionID, userID uuid.UUID, hasControl bool) (*dto.WatchPartyParticipant, error) {
	row, err := s.watchPartyRepo.GetParticipant(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrWatchPartyNotParticipant
	}
	userRole, _ := s.roleRepo.GetRole(ctx, userID)
	vanityRows, _ := s.vanityRoleRepo.GetRolesForUser(ctx, userID)
	return &dto.WatchPartyParticipant{
		User: dto.UserResponse{
			ID:          row.UserID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarURL,
			Role:        userRole,
			VanityRoles: s.toVanityRoleResponses(vanityRows),
		},
		HasControl: hasControl,
		JoinedAt:   row.JoinedAt,
	}, nil
}

func (s *watchPartyService) buildWatchPartyMessageDTO(ctx context.Context, row *repository.ChatWatchPartyMessageRow) dto.WatchPartyMessage {
	kind := row.Kind
	if kind == "" {
		kind = watchparty.MessageKindUser
	}
	out := dto.WatchPartyMessage{
		ID:        row.ID,
		SessionID: row.SessionID,
		Kind:      kind,
		Body:      row.Body,
		CreatedAt: row.CreatedAt,
	}
	if kind == watchparty.MessageKindSystem || !row.SenderID.Valid {
		return out
	}
	senderRole, _ := s.roleRepo.GetRole(ctx, row.SenderID.UUID)
	vanity, _ := s.vanityRoleRepo.GetRolesForUser(ctx, row.SenderID.UUID)
	out.Sender = &dto.UserResponse{
		ID:          row.SenderID.UUID,
		Username:    row.SenderUsername.String,
		DisplayName: row.SenderDisplayName.String,
		AvatarURL:   row.SenderAvatarURL.String,
		Role:        senderRole,
		VanityRoles: s.toVanityRoleResponses(vanity),
	}
	return out
}

func stringToNull(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: s}
}

func nullToString(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return n.String
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
