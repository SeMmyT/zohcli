package zoho

import (
	"strings"
	"time"
)

// SearchQuery builds Zoho Mail search syntax
type SearchQuery struct {
	parts []string
}

// NewSearchQuery creates an empty search query builder
func NewSearchQuery() *SearchQuery {
	return &SearchQuery{
		parts: []string{},
	}
}

// From adds a sender email filter
func (sq *SearchQuery) From(email string) *SearchQuery {
	sq.parts = append(sq.parts, "from:"+email)
	return sq
}

// To adds a recipient email filter
func (sq *SearchQuery) To(email string) *SearchQuery {
	sq.parts = append(sq.parts, "to:"+email)
	return sq
}

// Subject adds a subject text filter
func (sq *SearchQuery) Subject(text string) *SearchQuery {
	sq.parts = append(sq.parts, "subject:"+text)
	return sq
}

// DateAfter adds a filter for messages after a date
func (sq *SearchQuery) DateAfter(date time.Time) *SearchQuery {
	sq.parts = append(sq.parts, "after:"+date.Format("2006/01/02"))
	return sq
}

// DateBefore adds a filter for messages before a date
func (sq *SearchQuery) DateBefore(date time.Time) *SearchQuery {
	sq.parts = append(sq.parts, "before:"+date.Format("2006/01/02"))
	return sq
}

// HasAttachment adds a filter for messages with attachments
func (sq *SearchQuery) HasAttachment() *SearchQuery {
	sq.parts = append(sq.parts, "has:attachment")
	return sq
}

// IsUnread adds a filter for unread messages
func (sq *SearchQuery) IsUnread() *SearchQuery {
	sq.parts = append(sq.parts, "is:unread")
	return sq
}

// Text adds a free-text search query
func (sq *SearchQuery) Text(query string) *SearchQuery {
	sq.parts = append(sq.parts, query)
	return sq
}

// Build returns the complete search query string
func (sq *SearchQuery) Build() string {
	return strings.Join(sq.parts, " ")
}

// IsEmpty returns true if no search criteria have been added
func (sq *SearchQuery) IsEmpty() bool {
	return len(sq.parts) == 0
}
