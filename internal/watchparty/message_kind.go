package watchparty

import (
	"database/sql/driver"
	"fmt"
)

type MessageKind string

const (
	MessageKindUser   MessageKind = "user"
	MessageKindSystem MessageKind = "system"
)

func (k *MessageKind) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*k = ""
	case string:
		*k = MessageKind(v)
	case []byte:
		*k = MessageKind(v)
	default:
		return fmt.Errorf("scan watchparty.MessageKind: unsupported type %T", src)
	}
	return nil
}

func (k *MessageKind) Value() (driver.Value, error) {
	if k == nil {
		return nil, nil
	}
	return string(*k), nil
}
