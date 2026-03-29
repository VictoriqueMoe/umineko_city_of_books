package repository

import "database/sql"

type (
	Repositories struct {
		Session      SessionRepository
		User         UserRepository
		Theory       TheoryRepository
		Notification NotificationRepository
		Role         RoleRepository
	}
)

func New(db *sql.DB) *Repositories {
	return &Repositories{
		Session:      &sessionRepository{db: db},
		User:         &userRepository{db: db},
		Theory:       &theoryRepository{db: db},
		Notification: &notificationRepository{db: db},
		Role:         &roleRepository{db: db},
	}
}
