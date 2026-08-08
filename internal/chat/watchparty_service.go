package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

	watchPartyQuality = "smooth"

	watchPartyReconcileEvery     = 5 * time.Minute
	watchPartyReconcileIdleAfter = 6 * time.Minute

	wsWatchPartyStarted         = "watch_party_started"
	wsWatchPartyEnded           = "watch_party_ended"
	wsWatchPartyParticipantJoin = "watch_party_participant_joined"
	wsWatchPartyParticipantLeft = "watch_party_participant_left"
	wsWatchPartyControlChanged  = "watch_party_control_changed"
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

func (s *watchPartyService) StartWatchParty(ctx context.Context, roomID, actorID uuid.UUID, startURL, region, title, sessionType string, light bool) (*dto.StartWatchPartyResponse, error) {
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
			selectedRegion = strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingHyperbeamRegion))
		}

		vm, err := s.hyperbeamSvc.CreateVM(ctx, hyperbeam.CreateVMOptions{
			StartURL: startURL,
			Region:   selectedRegion,
			Timeout: &hyperbeam.VMTimeoutOpts{
				Offline:  defaultOfflineTimeout,
				Absolute: defaultSessionTimeout,
			},
			Adblock: true,
			WebGL:   true,
			Dark:    !light,
			Quality: &hyperbeam.VMQualityOpts{Mode: watchPartyQuality},
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

	abandonSession := func(reason string, roomCreated bool) {
		if sessionRow.HyperbeamSessionID != "" {
			s.terminateHyperbeam(sessionRow.HyperbeamSessionID)
		}
		if roomCreated {
			if err := s.deleteRoomWithMedia(ctx, sessionID); err != nil {
				logger.Log.Warn().Err(err).Str("session_id", sessionID.String()).Msg("roll back watch party chat room failed")
			}
		}
		_ = s.watchPartyRepo.MarkAllParticipantsLeft(ctx, sessionID)
		_ = s.watchPartyRepo.EndSession(ctx, sessionID, reason)
	}

	if err := s.createWatchPartyRoom(ctx, sessionID, actorID, trimmedTitle); err != nil {
		abandonSession("chat_room_setup_failed", false)
		return nil, err
	}

	if err := s.watchPartyRepo.UpsertParticipant(ctx, sessionID, actorID, true, ""); err != nil {
		abandonSession("controller_setup_failed", true)
		return nil, err
	}

	details := mustJSON(map[string]any{"room_id": roomID, "start_url": startURL, "title": trimmedTitle, "type": partyType})
	if err := s.auditRepo.Create(ctx, actorID, "watch_party.start", "chat_watch_party_session", sessionID.String(), details); err != nil {
		logger.Log.Warn().Err(err).Msg("audit watch_party.start failed")
	}

	stored, err := s.watchPartyRepo.GetByID(ctx, sessionID)
	if err != nil || stored == nil {
		abandonSession("session_reload_failed", true)
		return nil, fmt.Errorf("reload watch party session: %w", err)
	}

	sessionDTO, err := s.buildWatchPartySessionDTO(ctx, stored, actorID, embedURL, true)
	if err != nil {
		abandonSession("session_build_failed", true)
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

	s.admitToWatchPartyRoom(ctx, session.ID, actorID)

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
		return s.endWatchParty(ctx, roomID, sessionID, actorID, "owner_left")
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

	s.evictFromWatchPartyRoom(ctx, session.ID, actorID)

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

	if err := s.assertActiveRoomMember(ctx, roomID, callerID); err != nil {
		return err
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

	body := fmt.Sprintf("%s was kicked by %s.", s.displayNameFor(ctx, targetID, roomID), s.displayNameFor(ctx, callerID, roomID))
	s.postRoomActionMessage(ctx, session.ID, callerID, body)

	if err := s.watchPartyRepo.MarkParticipantLeft(ctx, session.ID, targetID); err != nil {
		return err
	}

	s.evictFromWatchPartyRoom(ctx, session.ID, targetID)

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

	details := mustJSON(map[string]any{"room_id": roomID, "target_user_id": targetID})
	if err := s.auditRepo.CreateForSubject(ctx, callerID, "watch_party.kick", "chat_watch_party_session", session.ID.String(), details, targetID); err != nil {
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
				_ = s.endWatchParty(ctx, roomID, sess.ID, userID, "owner_disconnected")
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

			s.evictFromWatchPartyRoom(ctx, sess.ID, userID)
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

	if err := s.assertActiveRoomMember(ctx, roomID, callerID); err != nil {
		return err
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
	if err := s.auditRepo.CreateForSubject(ctx, callerID, "watch_party.grant_control", "chat_watch_party_session", session.ID.String(), details, targetID); err != nil {
		logger.Log.Warn().Err(err).Msg("audit watch_party.grant_control failed")
	}

	return nil
}

func (s *watchPartyService) transferControlTo(ctx context.Context, roomID uuid.UUID, session *repository.ChatWatchPartySessionRow, targetID uuid.UUID) error {
	if targetID == uuid.Nil {
		return nil
	}

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
			if err := s.hyperbeamSvc.SetControlRole(ctx, session.VMBaseURL, session.HyperbeamAdminToken, p.HyperbeamIdentifier, false); err != nil {
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
		if err := s.hyperbeamSvc.SetControlRole(ctx, session.VMBaseURL, session.HyperbeamAdminToken, targetIdentifier, true); err != nil {
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
	if actorID != uuid.Nil {
		if err := s.assertActiveRoomMember(ctx, roomID, actorID); err != nil {
			return err
		}
	}

	return s.endWatchParty(ctx, roomID, sessionID, actorID, reason)
}

func (s *watchPartyService) endWatchParty(ctx context.Context, roomID, sessionID, actorID uuid.UUID, reason string) error {
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
	if err := s.assertActiveRoomMember(ctx, roomID, userID); err != nil {
		return err
	}

	if identifier == "" {
		return ErrWatchPartyNoIdentifier
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
		if err := s.hyperbeamSvc.SetControlRole(ctx, session.VMBaseURL, session.HyperbeamAdminToken, identifier, participant.HasControl); err != nil {
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
	if err = s.assertActiveRoomMember(ctx, roomID, userID); err != nil {
		return "", "", err
	}

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

	forceMuted, err := s.chatRepo.IsVoiceForceMuted(ctx, session.ID, userID)
	if err != nil {
		return "", "", fmt.Errorf("check voice force mute: %w", err)
	}

	token, err = s.livekitSvc.MintToken(roomName, userID.String(), displayName, !forceMuted, allowScreenShare)
	if err != nil {
		return "", "", err
	}

	return token, s.livekitSvc.URL(), nil
}

func (s *watchPartyService) ForceMuteSessionVoice(ctx context.Context, roomID, sessionID, actorID, targetID uuid.UUID, muted bool) error {
	if !s.screenShareEnabled() {
		return ErrVoiceDisabled
	}

	if err := s.assertActiveRoomMember(ctx, roomID, actorID); err != nil {
		return err
	}

	session, err := s.loadActiveSession(ctx, roomID, sessionID)
	if err != nil {
		return err
	}

	caller, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, actorID)
	if err != nil {
		return err
	}
	if caller == nil || caller.LeftAt.Valid {
		return ErrWatchPartyNotParticipant
	}

	target, err := s.watchPartyRepo.GetParticipant(ctx, session.ID, targetID)
	if err != nil {
		return err
	}
	if target == nil || target.LeftAt.Valid {
		return ErrWatchPartyNotParticipant
	}

	callerRank := s.watchPartyRankOf(ctx, session, actorID)
	targetRank := s.watchPartyRankOf(ctx, session, targetID)
	if callerRank <= targetRank {
		return ErrVoiceMuteForbidden
	}

	roomName := voiceSessionRoomPrefix + session.ID.String()
	allowScreenShare := session.Type == watchPartyTypeScreenShare && session.StartedBy == targetID

	if err := s.chatRepo.SetVoiceForceMuted(ctx, session.ID, targetID, actorID, muted); err != nil {
		return fmt.Errorf("set voice force mute: %w", err)
	}

	return s.livekitSvc.SetCanPublish(ctx, roomName, targetID.String(), !muted, allowScreenShare)
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

func (s *watchPartyService) createWatchPartyRoom(ctx context.Context, sessionID, hostID uuid.UUID, title string) error {
	name := title
	if name == "" {
		name = "Watch party"
	}

	if err := s.chatRepo.CreateSystemRoom(ctx, sessionID, name, "", SystemKindWatchParty, hostID); err != nil {
		return fmt.Errorf("create watch party chat room: %w", err)
	}

	if err := s.chatRepo.AddMemberWithRole(ctx, sessionID, hostID, "host", false); err != nil {
		return fmt.Errorf("add host to watch party chat room: %w", err)
	}

	s.hub.JoinRoom(sessionID, hostID)

	return nil
}

func (s *watchPartyService) admitToWatchPartyRoom(ctx context.Context, sessionID, userID uuid.UUID) {
	alreadyMember, err := s.chatRepo.IsMember(ctx, sessionID, userID)
	if err != nil {
		logger.Log.Warn().Err(err).Str("session_id", sessionID.String()).Msg("check watch party chat membership failed")
		return
	}

	if !alreadyMember {
		if err := s.chatRepo.AddMemberWithRole(ctx, sessionID, userID, "member", false); err != nil {
			logger.Log.Warn().Err(err).Str("session_id", sessionID.String()).Msg("add participant to watch party chat room failed")
			return
		}
	}

	s.hub.JoinRoom(sessionID, userID)
}

func (s *watchPartyService) evictFromWatchPartyRoom(ctx context.Context, sessionID, userID uuid.UUID) {
	if err := s.chatRepo.RemoveMember(ctx, sessionID, userID); err != nil {
		logger.Log.Warn().Err(err).Str("session_id", sessionID.String()).Msg("remove participant from watch party chat room failed")
	}

	s.hub.LeaveRoom(sessionID, userID)
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
