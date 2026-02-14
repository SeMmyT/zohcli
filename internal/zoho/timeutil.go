package zoho

import "time"

// ToUnixMillis converts time.Time to Unix milliseconds (int64)
func ToUnixMillis(t time.Time) int64 {
	return t.UnixMilli()
}

// FromUnixMillis converts Unix milliseconds (int64) to time.Time
func FromUnixMillis(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// FormatMillisTimestamp formats a millisecond timestamp for display in RFC3339
func FormatMillisTimestamp(ms int64) string {
	if ms == 0 {
		return ""
	}
	return FromUnixMillis(ms).Format(time.RFC3339)
}
