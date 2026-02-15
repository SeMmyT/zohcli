package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopeString(t *testing.T) {
	result := ScopeString()

	// Zoho uses comma-separated scopes (not space-separated)
	assert.NotContains(t, result, " ")
	assert.Contains(t, result, ",")

	// All default scopes are present
	for _, scope := range DefaultScopes {
		assert.Contains(t, result, scope)
	}

	// No leading or trailing commas
	assert.False(t, strings.HasPrefix(result, ","))
	assert.False(t, strings.HasSuffix(result, ","))

	// Number of commas = len(scopes) - 1
	assert.Equal(t, len(DefaultScopes)-1, strings.Count(result, ","))
}
