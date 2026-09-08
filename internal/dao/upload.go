package dao

import (
	"database/sql"
)

type (
	UploadDAO interface {
		GetAllReferencedFiles(tx ...*sql.Tx) ([]string, error)
	}
)
