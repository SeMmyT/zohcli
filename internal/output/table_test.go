package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{name: "shorter than max", s: "hello", maxLen: 10, expected: "hello"},
		{name: "equal to max", s: "hello", maxLen: 5, expected: "hello"},
		{name: "longer than max", s: "hello world", maxLen: 8, expected: "hello..."},
		{name: "maxLen less than 3", s: "hello", maxLen: 2, expected: "he"},
		{name: "maxLen exactly 3", s: "hello", maxLen: 3, expected: "..."},
		{name: "empty string", s: "", maxLen: 5, expected: ""},
		{name: "maxLen zero", s: "hello", maxLen: 0, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, TruncateString(tt.s, tt.maxLen))
		})
	}
}

func TestPadString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		width    int
		expected string
	}{
		{name: "shorter than width", s: "hi", width: 5, expected: "hi   "},
		{name: "equal to width", s: "hello", width: 5, expected: "hello"},
		{name: "longer than width", s: "hello!", width: 5, expected: "hello!"},
		{name: "empty string", s: "", width: 3, expected: "   "},
		{name: "width zero", s: "hi", width: 0, expected: "hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, PadString(tt.s, tt.width))
		})
	}
}
