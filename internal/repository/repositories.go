package repository

import "database/sql"

type (
	Repositories struct {
		Session      SessionRepository
		User         UserRepository
		Theory       TheoryRepository
		Notification NotificationRepository
		Role         RoleRepository
		Settings     SettingsRepository
		AuditLog     AuditLogRepository
		Stats        StatsRepository
		Invite       InviteRepository
		Chat         ChatRepository
		Report       ReportRepository
		Post         PostRepository
		Follow       FollowRepository
	}
)

func New(db *sql.DB) *Repositories {
	return &Repositories{
		Session:      &sessionRepository{db: db},
		User:         &userRepository{db: db},
		Theory:       &theoryRepository{db: db},
		Notification: &notificationRepository{db: db},
		Role:         &roleRepository{db: db},
		Settings:     &settingsRepository{db: db},
		AuditLog:     &auditLogRepository{db: db},
		Stats:        &statsRepository{db: db},
		Invite:       &inviteRepository{db: db},
		Chat:         &chatRepository{db: db},
		Report:       &reportRepository{db: db},
		Post:         &postRepository{db: db},
		Follow:       &followRepository{db: db},
	}
}
