package secrets

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// NewStore creates a Store instance using platform-appropriate backend.
// Tries OS keyring first, falls back to encrypted file if unavailable.
// Automatically detects WSL and headless environments that need file fallback.
func NewStore() (Store, error) {
	// WSL and headless environments can't use keyring reliably
	if IsWSL() || IsHeadless() {
		fmt.Fprintln(os.Stderr, "Detected WSL/headless environment, using encrypted file storage")
		return NewFileStore("")
	}

	// Try keyring first
	store, err := NewKeyringStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Keyring unavailable (%v), falling back to encrypted file\n", err)
		return NewFileStore("")
	}

	return store, nil
}

// IsWSL returns true if running under Windows Subsystem for Linux.
func IsWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}

	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

// IsHeadless returns true if running in a headless environment (no display server).
// Only applicable on Linux; macOS and Windows are assumed to have GUI.
func IsHeadless() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check for X11 or Wayland display
	return os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == ""
}
