package output

import "fmt"

// Exit codes following sysexits.h convention
const (
	ExitOK           = 0  // Success
	ExitGeneral      = 1  // General error
	ExitUsage        = 2  // Invalid usage / bad arguments
	ExitAuth         = 3  // Authentication failure
	ExitNotFound     = 4  // Resource not found
	ExitConflict     = 5  // Conflict (resource already exists)
	ExitForbidden    = 6  // Permission denied
	ExitRateLimit    = 75 // Rate limited (EX_TEMPFAIL from sysexits.h)
	ExitTimeout      = 8  // Request timeout
	ExitAPIError     = 9  // Zoho API error (non-specific)
	ExitConfigError  = 10 // Configuration error
	ExitNetworkError = 11 // Network connectivity error
)

// CLIError represents a structured error with exit code and optional hint
type CLIError struct {
	ExitCode int
	Message  string
	Hint     string
}

// Error implements the error interface
func (e *CLIError) Error() string {
	return e.Message
}

// NewCLIError creates a new CLIError
func NewCLIError(code int, msg string) *CLIError {
	return &CLIError{
		ExitCode: code,
		Message:  msg,
	}
}

// WithHint adds a user-facing hint to the error
func (e *CLIError) WithHint(hint string) *CLIError {
	e.Hint = hint
	return e
}

// ExitWithError prints the error via the formatter and exits with the correct code
func ExitWithError(formatter Formatter, err error) {
	if cliErr, ok := err.(*CLIError); ok {
		formatter.PrintError(err)
		if cliErr.Hint != "" {
			formatter.PrintHint(cliErr.Hint)
		}
		// Note: Actual os.Exit call should be in main.go, not here
		// This function is just a helper
		return
	}

	// Unknown error - print as general error
	formatter.PrintError(fmt.Errorf("error: %v", err))
}
