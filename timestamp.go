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
	switch {
	case t < 1e10: // Seconds.
		return time.Unix(int64(t), 0)
	case t < 1e13: // Milliseconds.
		return time.UnixMilli(int64(t))
	case t < 1e16: // Microseconds.
		return time.UnixMicro(int64(t))
	default: // Nanoseconds.
		return time.Unix(int64(t)/1e9, int64(t)%1e9)
	}
}

func (t Timestamp) Add(d time.Duration) Timestamp {
	switch {
	case t < 1e10: // Seconds.
		return Timestamp(t.Time().Add(d).Unix())
	case t < 1e13: // Milliseconds.
		return Timestamp(t.Time().Add(d).UnixMilli())
	case t < 1e16: // Microseconds.
		return Timestamp(t.Time().Add(d).UnixMicro())
	default: // Nanoseconds.
		return Timestamp(t.Time().Add(d).UnixNano())
	}
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
