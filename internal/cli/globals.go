package cli

import (
	"os"

	"golang.org/x/term"
)

// Globals holds global flags available to all commands
type Globals struct {
	Region  string `help:"Zoho region" default:"" enum:"us,eu,in,au,jp,ca,sa,uk," env:"ZOH_REGION"`
	Output  string `help:"Output format" default:"auto" enum:"json,plain,rich,auto" short:"o" env:"ZOH_OUTPUT"`
	Verbose bool   `help:"Verbose output" short:"v" env:"ZOH_VERBOSE"`
}

// ResolvedOutput returns the effective output mode
// "auto" detects TTY: if stdout is TTY -> rich, else -> plain
func (g *Globals) ResolvedOutput() string {
	if g.Output != "auto" {
		return g.Output
	}

	// Detect if stdout is a TTY
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return "rich"
	}

	return "plain"
}
