package model

import (
	"github.com/google/uuid"
)

type (
	ResponseMeta struct {
		AuthorID uuid.UUID
		TheoryID uuid.UUID
		Side     string
		ParentID *uuid.UUID
	}
)
