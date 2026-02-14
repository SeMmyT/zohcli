package config

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

// ConfigDir returns the XDG-compliant config directory for zoh
// Typically ~/.config/zoh/ on Linux
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, "zoh")
}

// ConfigPath returns the full path to the config file
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json5")
}

// CacheDir returns the XDG-compliant cache directory for zoh
// Typically ~/.cache/zoh/ on Linux (for token cache in Plan 02)
func CacheDir() string {
	return filepath.Join(xdg.CacheHome, "zoh")
}

// DataDir returns the XDG-compliant data directory for zoh
// Typically ~/.local/share/zoh/ on Linux (for future use)
func DataDir() string {
	return filepath.Join(xdg.DataHome, "zoh")
}
