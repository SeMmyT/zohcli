package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCLIError(t *testing.T) {
	err := NewCLIError(ExitAuth, "authentication failed")
	assert.Equal(t, ExitAuth, err.ExitCode)
	assert.Equal(t, "authentication failed", err.Message)
	assert.Empty(t, err.Hint)
}

func TestCLIErrorError(t *testing.T) {
	err := &CLIError{Message: "something broke"}
	assert.Equal(t, "something broke", err.Error())
}

func TestCLIErrorWithHint(t *testing.T) {
	err := NewCLIError(ExitAuth, "auth failed")
	result := err.WithHint("Run: zoh auth login")

	// Fluent builder returns same pointer
	assert.Same(t, err, result)
	assert.Equal(t, "Run: zoh auth login", err.Hint)
}

func TestCLIErrorImplementsError(t *testing.T) {
	var err error = NewCLIError(ExitGeneral, "test")
	assert.Equal(t, "test", err.Error())
}
