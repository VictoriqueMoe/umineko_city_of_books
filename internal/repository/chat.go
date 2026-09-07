package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	ChatRepository interface {
		dao.ChatDAO

		CreateDMRoomAtomic(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		CreateGroupRoom(ctx context.Context, s spec.NewChatGroupRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		UpdateGroupRoom(ctx context.Context, s spec.UpdateChatRoom, tx ...*sql.Tx) error
		CreateSystemRoomWithHost(ctx context.Context, s spec.NewChatSystemRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error)
		CreateSystemRooms(ctx context.Context, rooms []spec.NewChatSystemRoom, tx ...*sql.Tx) error
		SyncSystemRoomMembership(ctx context.Context, targets []spec.SystemRoomMembership, tx ...*sql.Tx) ([]model.SystemRoomMembershipChange, error)
		AddMemberWithSystemMessage(ctx context.Context, s spec.ChatMemberJoinAnnouncement, tx ...*sql.Tx) (*model.ChatMessageRow, error)
		DeleteRoomWithMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		DeleteMessageWithMedia(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		InsertMessageAndMarkRead(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error)
		InsertSystemMessage(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error)
	}

	chatRepository struct {
		db *sql.DB
		dao.ChatDAO
		audit AuditLogRepository
	}
)

func NewChatRepo(database *sql.DB, chatDAO dao.ChatDAO, auditLog AuditLogRepository) ChatRepository {
	return &chatRepository{db: database, ChatDAO: chatDAO, audit: auditLog}
}

func (r *chatRepository) CreateGroupRoom(ctx context.Context, s spec.NewChatGroupRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	var created *model.ChatRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.ChatDAO.CreateRoom(ctx, spec.NewChatRoom{
			Name:        s.Name,
			Description: s.Description,
			Type:        string(dto.RoomTypeGroup),
			IsPublic:    s.IsPublic,
			IsRP:        s.IsRP,
			CreatedBy:   s.CreatedBy,
		}, tx)
		if err != nil {
			return fmt.Errorf("create group room: %w", err)
		}

		if len(s.Tags) > 0 {
			if err := r.ChatDAO.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: created.ID, Tags: s.Tags}, tx); err != nil {
				return fmt.Errorf("add room tags: %w", err)
			}
		}

		if err := r.ChatDAO.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: created.ID, UserID: s.CreatedBy, Role: "host"}, tx); err != nil {
			return fmt.Errorf("add creator to group: %w", err)
		}

		for _, memberID := range s.MemberIDs {
			if err := r.ChatDAO.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: created.ID, UserID: memberID, Role: "member"}, tx); err != nil {
				return fmt.Errorf("add member to group: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatRepository) UpdateGroupRoom(ctx context.Context, s spec.UpdateChatRoom, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ChatDAO.UpdateRoom(ctx, s, tx); err != nil {
			return fmt.Errorf("update group room: %w", err)
		}

		if err := r.ChatDAO.ReplaceRoomTags(ctx, spec.ChatRoomTags{RoomID: s.RoomID, Tags: s.Tags}, tx); err != nil {
			return fmt.Errorf("replace room tags: %w", err)
		}

		return nil
	})
}

func (r *chatRepository) CreateSystemRoomWithHost(ctx context.Context, s spec.NewChatSystemRoom, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	var created *model.ChatRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		created, err = r.ChatDAO.CreateSystemRoom(ctx, s, tx)
		if err != nil {
			return err
		}

		return r.ChatDAO.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: created.ID, UserID: s.CreatedBy, Role: "host"}, tx)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatRepository) CreateSystemRooms(ctx context.Context, rooms []spec.NewChatSystemRoom, tx ...*sql.Tx) error {
	if len(rooms) == 0 {
		return nil
	}

	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		for _, room := range rooms {
			if _, err := r.ChatDAO.CreateSystemRoom(ctx, room, tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *chatRepository) SyncSystemRoomMembership(ctx context.Context, targets []spec.SystemRoomMembership, tx ...*sql.Tx) ([]model.SystemRoomMembershipChange, error) {
	var changes []model.SystemRoomMembershipChange

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		changes = nil

		for _, target := range targets {
			if target.RoomID == uuid.Nil {
				continue
			}

			currentRole, err := r.ChatDAO.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: target.RoomID, UserID: target.UserID}, tx)
			if err != nil {
				return fmt.Errorf("get current role: %w", err)
			}

			wasMember := currentRole != ""

			switch {
			case target.ShouldBeMember && !wasMember:
				if err := r.ChatDAO.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: target.RoomID, UserID: target.UserID, Role: target.DesiredRole}, tx); err != nil {
					return err
				}

				changes = append(changes, model.SystemRoomMembershipChange{RoomID: target.RoomID, Joined: true})

			case !target.ShouldBeMember && wasMember:
				if err := r.ChatDAO.RemoveMember(ctx, spec.ChatMemberRef{RoomID: target.RoomID, UserID: target.UserID}, tx); err != nil {
					return err
				}

				changes = append(changes, model.SystemRoomMembershipChange{RoomID: target.RoomID, Left: true})

			case target.ShouldBeMember && wasMember && currentRole != target.DesiredRole:
				if err := r.ChatDAO.SetMemberRole(ctx, spec.ChatMemberRoleUpdate{RoomID: target.RoomID, UserID: target.UserID, Role: target.DesiredRole}, tx); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return changes, nil
}

func (r *chatRepository) AddMemberWithSystemMessage(ctx context.Context, s spec.ChatMemberJoinAnnouncement, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	var created *model.ChatMessageRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.ChatDAO.AddMemberWithRole(ctx, s.Member, tx); err != nil {
			return err
		}

		if s.Message.Body == "" {
			return nil
		}

		msg, err := r.ChatDAO.InsertMessageRow(ctx, s.Message, tx)
		if err != nil {
			return err
		}

		if err := r.ChatDAO.TouchRoomActivityForMessage(ctx, spec.ChatRoomActivityTouch{RoomID: s.Message.RoomID, IsSystem: s.Message.IsSystem}, tx); err != nil {
			return err
		}

		created = msg

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *chatRepository) DeleteRoomWithMessages(ctx context.Context, roomID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		paths = nil

		mediaURLs, err := r.ChatDAO.ListRoomMediaURLs(ctx, roomID, tx)
		if err != nil {
			return fmt.Errorf("list room media urls: %w", err)
		}

		avatarURLs, err := r.ChatDAO.ListRoomMemberAvatarURLs(ctx, roomID, tx)
		if err != nil {
			return fmt.Errorf("list room member avatar urls: %w", err)
		}

		paths = append(paths, mediaURLs...)
		paths = append(paths, avatarURLs...)

		if err := r.ChatDAO.DeleteMessages(ctx, roomID, tx); err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}

		if err := r.ChatDAO.DeleteRoom(ctx, roomID, tx); err != nil {
			return fmt.Errorf("delete room: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *chatRepository) DeleteMessageWithMedia(ctx context.Context, messageID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	var paths []string

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		var err error

		paths, err = r.ChatDAO.ListMessageMediaURLs(ctx, messageID, tx)
		if err != nil {
			return fmt.Errorf("list message media urls: %w", err)
		}

		if err := r.ChatDAO.DeleteMessage(ctx, messageID, tx); err != nil {
			return fmt.Errorf("delete message: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (r *chatRepository) CreateDMRoomAtomic(ctx context.Context, s spec.ChatDMPair, tx ...*sql.Tx) (*model.ChatRoomRow, error) {
	var result *model.ChatRoomRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		existing, err := r.ChatDAO.FindDMRoomByPair(ctx, s, tx)
		if err != nil {
			return err
		}

		if existing != nil {
			if err := r.ChatDAO.RejoinDMMembers(ctx, spec.ChatDMMembers{RoomID: existing.ID, UserA: s.UserA, UserB: s.UserB}, tx); err != nil {
				return err
			}

			result = existing

			return nil
		}

		created, err := r.ChatDAO.CreateDMRoom(ctx, s, tx)
		if err != nil {
			return err
		}

		if err := r.ChatDAO.AddDMMembers(ctx, spec.ChatDMMembers{RoomID: created.ID, UserA: s.UserA, UserB: s.UserB}, tx); err != nil {
			return err
		}

		result = created

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *chatRepository) InsertMessageAndMarkRead(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	var result *model.ChatMessageRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		msg, err := r.insertMessage(ctx, s, tx)
		if err != nil {
			return fmt.Errorf("insert message: %w", err)
		}

		if err := r.ChatDAO.MarkRoomRead(ctx, spec.ChatMemberRef{RoomID: s.RoomID, UserID: s.SenderID}, tx); err != nil {
			return fmt.Errorf("mark sender read: %w", err)
		}

		result = msg

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *chatRepository) InsertSystemMessage(ctx context.Context, s spec.NewChatMessage, tx ...*sql.Tx) (*model.ChatMessageRow, error) {
	s.IsSystem = true

	var result *model.ChatMessageRow

	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		msg, err := r.insertMessage(ctx, s, tx)
		if err != nil {
			return err
		}

		result = msg

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *chatRepository) insertMessage(ctx context.Context, message spec.NewChatMessage, tx *sql.Tx) (*model.ChatMessageRow, error) {
	msg, err := r.ChatDAO.InsertMessageRow(ctx, message, tx)
	if err != nil {
		return nil, err
	}

	if err := r.ChatDAO.TouchRoomActivityForMessage(ctx, spec.ChatRoomActivityTouch{RoomID: message.RoomID, IsSystem: message.IsSystem}, tx); err != nil {
		return nil, err
	}

	return msg, nil
}
