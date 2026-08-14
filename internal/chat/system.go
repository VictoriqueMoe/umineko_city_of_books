package chat

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
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

type systemService struct {
	*core
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

func (s *systemService) EnsureSystemRooms(ctx context.Context) error {
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

	rooms := make([]repository.NewChatSystemRoom, 0, 2)
	if modsID == uuid.Nil {
		rooms = append(rooms, repository.NewChatSystemRoom{
			ID:          uuid.New(),
			Name:        systemModsName,
			Description: systemModsDesc,
			SystemKind:  SystemKindMods,
			CreatedBy:   creator,
		})
	}
	if adminsID == uuid.Nil {
		rooms = append(rooms, repository.NewChatSystemRoom{
			ID:          uuid.New(),
			Name:        systemAdminsName,
			Description: systemAdminsDesc,
			SystemKind:  SystemKindAdmins,
			CreatedBy:   creator,
		})
	}

	if err := s.chatRepo.CreateSystemRooms(ctx, rooms); err != nil {
		return err
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

func (s *systemService) SyncSystemRoomMembership(ctx context.Context, userID uuid.UUID, newRole role.Role) error {
	modsID, err := s.chatRepo.GetSystemRoomID(ctx, SystemKindMods)
	if err != nil {
		return fmt.Errorf("get mods room: %w", err)
	}
	adminsID, err := s.chatRepo.GetSystemRoomID(ctx, SystemKindAdmins)
	if err != nil {
		return fmt.Errorf("get admins room: %w", err)
	}

	desired := memberRoleForSystem(newRole)

	changes, err := s.chatRepo.SyncSystemRoomMembership(ctx, []repository.SystemRoomMembership{
		{RoomID: modsID, UserID: userID, ShouldBeMember: eligibleForMods(newRole), DesiredRole: desired},
		{RoomID: adminsID, UserID: userID, ShouldBeMember: eligibleForAdmins(newRole), DesiredRole: desired},
	})
	if err != nil {
		return err
	}

	for i := range changes {
		if changes[i].Joined {
			s.hub.JoinRoom(changes[i].RoomID, userID)
			s.hub.SendToUser(userID, ws.Message{
				Type: "chat_room_invited",
				Data: map[string]any{"room_id": changes[i].RoomID},
			})

			continue
		}

		s.hub.LeaveRoom(changes[i].RoomID, userID)
		s.hub.SendToUser(userID, ws.Message{
			Type: "chat_kicked",
			Data: map[string]any{"room_id": changes[i].RoomID},
		})
	}

	return nil
}
