package nostr

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	_ sql.Scanner = (*Timestamp)(nil)
)

type Timestamp int64

func Now() Timestamp {
	return Timestamp(time.Now().Unix())
}

func (t Timestamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp(int64(t) + int64(d.Seconds()))
}

func (t *Timestamp) Scan(src any) error {
	switch v := src.(type) {
	case int:
		*t = Timestamp(v)
	case int64:
		*t = Timestamp(v)
	case time.Time:
		*t = Timestamp(v.Unix())
	case *time.Time:
		if v != nil {
			*t = Timestamp(v.Unix())
		} else {
			*t = 0
		}
	default:
		return fmt.Errorf("%w: %T: %v", ErrInvalidTimestamp, src, src)
	}
	return nil
}
