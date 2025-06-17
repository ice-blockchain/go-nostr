package nostr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimestamp_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    Timestamp
		wantErr bool
	}{
		{
			name:    "scan from int",
			input:   1234567890,
			want:    Timestamp(1234567890),
			wantErr: false,
		},
		{
			name:    "scan from int64",
			input:   int64(1234567890),
			want:    Timestamp(1234567890),
			wantErr: false,
		},
		{
			name:    "scan from time.Time",
			input:   time.Unix(1234567890, 0),
			want:    Timestamp(1234567890),
			wantErr: false,
		},
		{
			name:    "scan from *time.Time (non-nil)",
			input:   func() *time.Time { t := time.Unix(1234567890, 0); return &t }(),
			want:    Timestamp(1234567890),
			wantErr: false,
		},
		{
			name:    "scan from *time.Time (nil)",
			input:   (*time.Time)(nil),
			want:    Timestamp(0),
			wantErr: false,
		},
		{
			name:    "scan from unsupported type",
			input:   "not a timestamp",
			want:    Timestamp(0),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts Timestamp

			err := ts.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, tt.want.Equal(ts), "expected %v, got %v", tt.want, ts)
			}
		})
	}
}
func TestTimestamp_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ts       Timestamp
		duration time.Duration
		want     Timestamp
	}{
		// Seconds scale
		{
			name:     "add to seconds scale (positive)",
			ts:       Timestamp(1234567890), // seconds
			duration: 60 * time.Second,
			want:     Timestamp(1234567950),
		},
		{
			name:     "add to seconds scale (negative)",
			ts:       Timestamp(1234567890), // seconds
			duration: -60 * time.Second,
			want:     Timestamp(1234567830),
		},

		// Milliseconds scale
		{
			name:     "add to milliseconds scale (positive)",
			ts:       Timestamp(1234567890123), // milliseconds
			duration: 60 * time.Second,
			want:     Timestamp(1234567950123),
		},
		{
			name:     "add to milliseconds scale (negative)",
			ts:       Timestamp(1234567890123), // milliseconds
			duration: -60 * time.Second,
			want:     Timestamp(1234567830123),
		},

		// Microseconds scale
		{
			name:     "add to microseconds scale (positive)",
			ts:       Timestamp(1234567890123456), // microseconds
			duration: 60 * time.Second,
			want:     Timestamp(1234567950123456),
		},
		{
			name:     "add to microseconds scale (negative)",
			ts:       Timestamp(1234567890123456), // microseconds
			duration: -60 * time.Second,
			want:     Timestamp(1234567830123456),
		},

		// Nanoseconds scale
		{
			name:     "add to nanoseconds scale (positive)",
			ts:       Timestamp(1234567890123456789), // nanoseconds
			duration: 60 * time.Second,
			want:     Timestamp(1234567950123456789),
		},
		{
			name:     "add to nanoseconds scale (negative)",
			ts:       Timestamp(1234567890123456789), // nanoseconds
			duration: -60 * time.Second,
			want:     Timestamp(1234567830123456789),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.Add(tt.duration)
			require.Equal(t, tt.want, got)
		})
	}
}
func TestTimestamp_Time(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		timestamp Timestamp
		want      time.Time
	}{
		{
			name:      "seconds scale",
			timestamp: Timestamp(1234567890),
			want:      time.Unix(1234567890, 0),
		},
		{
			name:      "milliseconds scale",
			timestamp: Timestamp(1234567890123),
			want:      time.Unix(1234567890, 123000000),
		},
		{
			name:      "microseconds scale",
			timestamp: Timestamp(1234567890123456),
			want:      time.Unix(1234567890, 123456000),
		},
		{
			name:      "nanoseconds scale",
			timestamp: Timestamp(1234567890123456789),
			want:      time.Unix(1234567890, 123456789),
		},
		{
			name:      "zero value",
			timestamp: Timestamp(0),
			want:      time.Unix(0, 0),
		},
		{
			name:      "negative value",
			timestamp: Timestamp(-1234567890),
			want:      time.Unix(-1234567890, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.timestamp.Time()
			require.Equal(t, tt.want, got)
		})
	}
}
func TestTimestamp_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Timestamp
		wantErr bool
	}{
		{
			name:    "valid timestamp",
			input:   "1234567890",
			want:    Timestamp(1234567890),
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "zero timestamp",
			input:   "0",
			want:    Timestamp(0),
			wantErr: false,
		},
		{
			name:    "large valid timestamp",
			input:   "9223372036854775807", // max int64
			want:    Timestamp(9223372036854775807),
			wantErr: false,
		},
		{
			name:    "negative timestamp",
			input:   "-123",
			want:    Timestamp(0),
			wantErr: true,
		},
		{
			name:    "non-integer string",
			input:   "not a number",
			want:    Timestamp(0),
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    Timestamp(0),
			wantErr: true,
		},
		{
			name:    "floating point number",
			input:   "123.456",
			want:    Timestamp(0),
			wantErr: true,
		},
		{
			name:    "too large number",
			input:   "9223372036854775808", // max int64 + 1
			want:    Timestamp(0),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimestamp(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidTimestamp)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
func TestTimestamp_Comparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ts1      Timestamp
		ts2      Timestamp
		isAfter  bool
		isBefore bool
		isEqual  bool
	}{
		// Equal timestamps at different scales
		{
			name:     "equal timestamps (seconds)",
			ts1:      Timestamp(1234567890),
			ts2:      Timestamp(1234567890),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},
		{
			name:     "equal timestamps (milliseconds)",
			ts1:      Timestamp(1234567890123),
			ts2:      Timestamp(1234567890123),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},
		{
			name:     "equal timestamps (microseconds)",
			ts1:      Timestamp(1234567890123456),
			ts2:      Timestamp(1234567890123456),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},
		{
			name:     "equal timestamps (nanoseconds)",
			ts1:      Timestamp(1234567890123456789),
			ts2:      Timestamp(1234567890123456789),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},

		// After/Before relationships at same scale
		{
			name:     "ts1 after ts2 (seconds)",
			ts1:      Timestamp(1234567891),
			ts2:      Timestamp(1234567890),
			isAfter:  true,
			isBefore: false,
			isEqual:  false,
		},
		{
			name:     "ts1 before ts2 (seconds)",
			ts1:      Timestamp(1234567889),
			ts2:      Timestamp(1234567890),
			isAfter:  false,
			isBefore: true,
			isEqual:  false,
		},

		// Comparison across different scales
		{
			name:     "compare seconds and milliseconds (equal)",
			ts1:      Timestamp(1234567890),
			ts2:      Timestamp(1234567890000),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},
		{
			name:     "compare seconds and milliseconds (after)",
			ts1:      Timestamp(1234567891),
			ts2:      Timestamp(1234567890999),
			isAfter:  true,
			isBefore: false,
			isEqual:  false,
		},
		{
			name:     "compare milliseconds and microseconds (equal)",
			ts1:      Timestamp(1234567890123),
			ts2:      Timestamp(1234567890123000),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},

		// Edge cases
		{
			name:     "zero values",
			ts1:      Timestamp(0),
			ts2:      Timestamp(0),
			isAfter:  false,
			isBefore: false,
			isEqual:  true,
		},
		{
			name:     "zero and non-zero",
			ts1:      Timestamp(0),
			ts2:      Timestamp(1),
			isAfter:  false,
			isBefore: true,
			isEqual:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isAfter, tt.ts1.After(tt.ts2), "After() returned incorrect result")
			require.Equal(t, tt.isBefore, tt.ts1.Before(tt.ts2), "Before() returned incorrect result")
			require.Equal(t, tt.isEqual, tt.ts1.Equal(tt.ts2), "Equal() returned incorrect result")
		})
	}
}
