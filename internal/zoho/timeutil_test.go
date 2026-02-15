package zoho

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeConversionRoundTrip(t *testing.T) {
	original := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	ms := ToUnixMillis(original)
	restored := FromUnixMillis(ms)

	// Compare via Unix millis — FromUnixMillis returns local timezone
	assert.Equal(t, original.UnixMilli(), restored.UnixMilli())
	assert.True(t, original.Equal(restored))
}

func TestToUnixMillis(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ms := ToUnixMillis(ts)
	assert.Equal(t, ts.UnixMilli(), ms)
}

func TestFromUnixMillis(t *testing.T) {
	ms := int64(1718444400000) // 2024-06-15T09:00:00Z
	result := FromUnixMillis(ms)
	assert.Equal(t, ms, result.UnixMilli())
}

func TestFormatMillisTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		ms       int64
		expected string
	}{
		{
			name:     "zero returns empty string",
			ms:       0,
			expected: "",
		},
		{
			name:     "non-zero returns RFC3339",
			ms:       1705320000000, // 2024-01-15T12:00:00Z
			expected: time.UnixMilli(1705320000000).Format(time.RFC3339),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatMillisTimestamp(tt.ms))
		})
	}
}
