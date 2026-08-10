package dto

import (
	"database/sql/driver"
	"fmt"
)

type TheoryStatus string

const (
	TheoryStatusOpen      TheoryStatus = "open"
	TheoryStatusContested TheoryStatus = "contested"
	TheoryStatusRefuted   TheoryStatus = "refuted"
)

func (t *TheoryStatus) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*t = ""
	case string:
		*t = TheoryStatus(v)
	case []byte:
		*t = TheoryStatus(v)
	default:
		return fmt.Errorf("scan dto.TheoryStatus: unsupported type %T", src)
	}
	return nil
}

func (t TheoryStatus) Value() (driver.Value, error) {
	return string(t), nil
}
