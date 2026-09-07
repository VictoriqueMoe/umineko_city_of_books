package audit

import (
	"github.com/google/uuid"
)

type (
	Entry struct {
		ID              int
		ActorID         uuid.UUID
		ActorName       string
		Action          Action
		TargetType      TargetType
		TargetID        string
		Details         string
		CreatedAt       string
		SubjectID       *uuid.UUID
		SubjectName     string
		SubjectUsername string
	}

	NewEntry struct {
		ActorID    uuid.UUID
		Action     Action
		TargetType TargetType
		TargetID   string
		Details    string
		SubjectID  uuid.UUID
	}
)
