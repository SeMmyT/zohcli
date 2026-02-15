package zoho

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearchQueryIsEmpty(t *testing.T) {
	sq := NewSearchQuery()
	assert.True(t, sq.IsEmpty())

	sq.From("test@example.com")
	assert.False(t, sq.IsEmpty())
}

func TestSearchQueryBuild(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *SearchQuery
		expected string
	}{
		{
			name:     "empty query",
			build:    func() *SearchQuery { return NewSearchQuery() },
			expected: "",
		},
		{
			name: "from filter",
			build: func() *SearchQuery {
				return NewSearchQuery().From("alice@example.com")
			},
			expected: "from:alice@example.com",
		},
		{
			name: "to filter",
			build: func() *SearchQuery {
				return NewSearchQuery().To("bob@example.com")
			},
			expected: "to:bob@example.com",
		},
		{
			name: "subject filter",
			build: func() *SearchQuery {
				return NewSearchQuery().Subject("meeting")
			},
			expected: "subject:meeting",
		},
		{
			name: "has attachment",
			build: func() *SearchQuery {
				return NewSearchQuery().HasAttachment()
			},
			expected: "has:attachment",
		},
		{
			name: "is unread",
			build: func() *SearchQuery {
				return NewSearchQuery().IsUnread()
			},
			expected: "is:unread",
		},
		{
			name: "free text",
			build: func() *SearchQuery {
				return NewSearchQuery().Text("invoice")
			},
			expected: "invoice",
		},
		{
			name: "date after",
			build: func() *SearchQuery {
				return NewSearchQuery().DateAfter(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
			},
			expected: "after:2024/06/01",
		},
		{
			name: "date before",
			build: func() *SearchQuery {
				return NewSearchQuery().DateBefore(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))
			},
			expected: "before:2024/12/31",
		},
		{
			name: "chained filters joined with spaces",
			build: func() *SearchQuery {
				return NewSearchQuery().
					From("alice@example.com").
					To("bob@example.com").
					Subject("report").
					HasAttachment()
			},
			expected: "from:alice@example.com to:bob@example.com subject:report has:attachment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.build().Build())
		})
	}
}
