package zoho

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPrimaryEmail(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected string
	}{
		{
			name:     "returns PrimaryEmailID when set",
			user:     User{PrimaryEmailID: "primary@example.com"},
			expected: "primary@example.com",
		},
		{
			name: "falls back to IsPrimary email",
			user: User{
				EmailAddress: []EmailAddress{
					{MailID: "alias@example.com", IsPrimary: false},
					{MailID: "primary@example.com", IsPrimary: true},
				},
			},
			expected: "primary@example.com",
		},
		{
			name: "falls back to first email",
			user: User{
				EmailAddress: []EmailAddress{
					{MailID: "first@example.com", IsPrimary: false},
					{MailID: "second@example.com", IsPrimary: false},
				},
			},
			expected: "first@example.com",
		},
		{
			name:     "returns empty when no emails",
			user:     User{},
			expected: "",
		},
		{
			name: "PrimaryEmailID takes precedence over IsPrimary",
			user: User{
				PrimaryEmailID: "override@example.com",
				EmailAddress: []EmailAddress{
					{MailID: "primary@example.com", IsPrimary: true},
				},
			},
			expected: "override@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.PrimaryEmail())
		})
	}
}

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      APIError
		expected string
	}{
		{
			name: "returns MoreInfo when set",
			err: APIError{
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
				}{Code: 400, Description: "Bad Request"},
				Data: struct {
					MoreInfo string `json:"moreInfo"`
				}{MoreInfo: "Email address already exists"},
			},
			expected: "Email address already exists",
		},
		{
			name: "falls back to Description",
			err: APIError{
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
				}{Code: 500, Description: "Internal Server Error"},
			},
			expected: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestUserJSONUnmarshal(t *testing.T) {
	t.Run("accountId is a quoted string, zuid is a number", func(t *testing.T) {
		// This is the actual Zoho API response format — accountId is returned
		// as a quoted JSON string while all other IDs are raw numbers.
		raw := `{
			"accountId": "123456789",
			"zuid": 987654321,
			"displayName": "Test User",
			"firstName": "Test",
			"lastName": "User",
			"role": "admin",
			"usedStorage": 1048576,
			"planStorage": 5368709120,
			"emailAddress": [
				{"mailId": "test@example.com", "isAlias": false, "isPrimary": true}
			],
			"primaryEmailAddress": "test@example.com"
		}`

		var user User
		err := json.Unmarshal([]byte(raw), &user)
		require.NoError(t, err)

		assert.Equal(t, "123456789", user.AccountID)
		assert.Equal(t, int64(987654321), user.ZUID)
		assert.Equal(t, "Test User", user.DisplayName)
		assert.Equal(t, "test@example.com", user.PrimaryEmailID)
		assert.Len(t, user.EmailAddress, 1)
		assert.True(t, user.EmailAddress[0].IsPrimary)
	})

	t.Run("handles missing optional fields", func(t *testing.T) {
		raw := `{"accountId": "1", "zuid": 2}`

		var user User
		err := json.Unmarshal([]byte(raw), &user)
		require.NoError(t, err)

		assert.Equal(t, "1", user.AccountID)
		assert.Equal(t, int64(2), user.ZUID)
		assert.Empty(t, user.PrimaryEmailID)
		assert.Empty(t, user.EmailAddress)
	})
}

func TestAPIErrorJSONUnmarshal(t *testing.T) {
	raw := `{
		"status": {"code": 400, "description": "Bad Request"},
		"data": {"moreInfo": "Invalid email format"}
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(raw), &apiErr)
	require.NoError(t, err)

	assert.Equal(t, 400, apiErr.Status.Code)
	assert.Equal(t, "Invalid email format", apiErr.Error())
}
