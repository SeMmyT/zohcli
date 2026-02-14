package browser

import (
	"os/exec"
	"runtime"
)

// Open opens the specified URL in the default browser.
func Open(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return nil // Unsupported platform, silently fail
	}

	return cmd.Start()
}
