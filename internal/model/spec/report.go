package spec

import (
	"github.com/google/uuid"
)

type (
	NewReport struct {
		ReporterID uuid.UUID
		TargetType string
		TargetID   string
		ContextID  string
		Reason     string
	}

	ReportFilter struct {
		Status string
		Limit  int
		Offset int
	}

	ReportResolution struct {
		ID         int
		ResolvedBy uuid.UUID
		Comment    string
	}
)
