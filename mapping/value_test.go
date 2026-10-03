package mapping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		in      any
		want    time.Duration
		wantErr bool
	}{
		{"30s", 30 * time.Second, false},
		{"1m30s", 90 * time.Second, false},
		{" 2 ", 2 * time.Second, false},
		{"1.5", 1500 * time.Millisecond, false},
		{2, 2 * time.Second, false},
		{int64(3), 3 * time.Second, false},
		{0.25, 250 * time.Millisecond, false},
		{10 * time.Second, 10 * time.Second, false},
		{"soon", 0, true},
		{0, 0, true},
		{"-1s", 0, true},
		{true, 0, true},
		{20_000_000_000, 0, true}, // would wrap to about 49 years
		{int64(maxTimeoutSeconds) + 1, 0, true},
		{1e20, 0, true},
		{"1e20", 0, true},
		{"NaN", 0, true},
		{-5, 0, true},
		{time.Duration(0), 0, true},
		{int64(maxTimeoutSeconds), time.Duration(maxTimeoutSeconds) * time.Second, false},
	}
	for _, tt := range tests {
		got, err := ParseTimeout(tt.in)
		if tt.wantErr {
			assert.Error(t, err, "%#v", tt.in)
			continue
		}
		require.NoError(t, err, "%#v", tt.in)
		assert.Equal(t, tt.want, got, "%#v", tt.in)
	}
}
