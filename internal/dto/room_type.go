package dto

import (
	"database/sql/driver"
	"fmt"
)

type RoomType string

const (
	RoomTypeDM    RoomType = "dm"
	RoomTypeGroup RoomType = "group"
)

func (t *RoomType) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*t = ""
	case string:
		*t = RoomType(v)
	case []byte:
		*t = RoomType(v)
	default:
		return fmt.Errorf("scan dto.RoomType: unsupported type %T", src)
	}
	return nil
}

func (t RoomType) Value() (driver.Value, error) {
	return string(t), nil
}
