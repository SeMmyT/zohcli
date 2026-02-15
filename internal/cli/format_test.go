package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{name: "zero bytes", bytes: 0, expected: "0 B"},
		{name: "small bytes", bytes: 512, expected: "512 B"},
		{name: "just under 1KB", bytes: 1023, expected: "1023 B"},
		{name: "exactly 1KB", bytes: 1024, expected: "1.0 KB"},
		{name: "1.5KB", bytes: 1536, expected: "1.5 KB"},
		{name: "exactly 1MB", bytes: 1024 * 1024, expected: "1.0 MB"},
		{name: "exactly 1GB", bytes: 1024 * 1024 * 1024, expected: "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBytes(tt.bytes))
		})
	}
}

func TestFormatPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		expected string
	}{
		{name: "normal", priority: 0, expected: "Normal"},
		{name: "high", priority: 1, expected: "High"},
		{name: "unknown", priority: 5, expected: "5"},
		{name: "negative", priority: -1, expected: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatPriority(tt.priority))
		})
	}
}

func TestFormatBool(t *testing.T) {
	assert.Equal(t, "Yes", formatBool(true))
	assert.Equal(t, "No", formatBool(false))
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "empty string", value: "", expected: ""},
		{name: "1 char", value: "a", expected: "****"},
		{name: "4 chars", value: "abcd", expected: "****"},
		{name: "5 chars", value: "abcde", expected: "****bcde"},
		{name: "long string", value: "secret-key-12345", expected: "****2345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskSecret(tt.value))
		})
	}
}
