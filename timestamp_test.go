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
				require.Equal(t, tt.want, ts)
			}
		})
	}
}
