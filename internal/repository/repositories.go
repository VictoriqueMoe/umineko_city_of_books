package repository

import "database/sql"

type (
	Repositories struct {
		db           *sql.DB
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
		Art          ArtRepository
		Upload       UploadRepository
		Block        BlockRepository
		Announcement AnnouncementRepository
		Mystery      MysteryRepository
		Ship         ShipRepository
		Fanfic       FanficRepository
		Journal      JournalRepository
	}
)

func (r *Repositories) DB() *sql.DB {
	return r.db
}

func New(db *sql.DB) *Repositories {
	return &Repositories{
		db:           db,
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
		Art:          &artRepository{db: db},
		Upload:       &uploadRepository{db: db},
		Block:        &blockRepository{db: db},
		Announcement: &announcementRepository{db: db},
		Mystery:      &mysteryRepository{db: db},
		Ship:         &shipRepository{db: db},
		Fanfic:       &fanficRepository{db: db},
		Journal:      &journalRepository{db: db},
	}
}
