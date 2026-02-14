package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
)

// warningShown checks if the file-store warning has already been shown.
// Uses a marker file in the data directory to avoid repeating on every command.
func warningShown() bool {
	return fileExists(warningMarkerPath())
}

// markWarningShown creates the marker file so the warning isn't repeated.
func markWarningShown() {
	_ = os.WriteFile(warningMarkerPath(), []byte("1"), 0600)
}

func warningMarkerPath() string {
	return filepath.Join(xdg.DataHome, "zoh", ".file-store-warning-shown")
}

// quietMode returns true if the user has suppressed warnings via ZOH_QUIET.
func quietMode() bool {
	return os.Getenv("ZOH_QUIET") == "1" || os.Getenv("ZOH_QUIET") == "true"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// warnOnce prints a message to stderr, but only the first time.
// Subsequent invocations are suppressed via a marker file.
// Set ZOH_QUIET=1 to suppress entirely.
func warnOnce(msg string) {
	if quietMode() || warningShown() {
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

// markWarningsDone persists the marker so future commands stay quiet.
func markWarningsDone() {
	if !warningShown() {
		markWarningShown()
	}
}

// NewStore creates a Store instance using platform-appropriate backend.
// Tries OS keyring first, falls back to encrypted file if unavailable.
// Automatically detects WSL and headless environments that need file fallback.
func NewStore() (Store, error) {
	// WSL and headless environments can't use keyring reliably
	if IsWSL() || IsHeadless() {
		warnOnce("Detected WSL/headless environment, using encrypted file storage")
		store, err := NewFileStore("")
		if err != nil {
			return nil, err
		}
		markWarningsDone()
		return store, nil
	}

	// Try keyring first
	store, err := NewKeyringStore()
	if err != nil {
		warnOnce(fmt.Sprintf("Keyring unavailable (%v), falling back to encrypted file", err))
		fstore, ferr := NewFileStore("")
		if ferr != nil {
			return nil, ferr
		}
		markWarningsDone()
		return fstore, nil
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
