package model

import (
	"github.com/google/uuid"
)

type (
	ReportRow struct {
		ID             int
		ReporterID     uuid.UUID
		ReporterName   string
		ReporterAvatar string
		TargetType     string
		TargetID       string
		ContextID      string
		Reason         string
		Status         string
		ResolvedByID   *uuid.UUID
		ResolvedByName string
		CreatedAt      string
	}
)
