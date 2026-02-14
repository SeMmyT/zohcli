package cli

import (
	"os"

	"golang.org/x/term"
)

// Globals holds global flags available to all commands
type Globals struct {
	Region      string `help:"Zoho region" default:"" enum:"us,eu,in,au,jp,ca,sa,uk," env:"ZOH_REGION"`
	Output      string `help:"Output format" default:"auto" enum:"json,plain,rich,auto" short:"o" env:"ZOH_OUTPUT"`
	Verbose     bool   `help:"Verbose output" short:"v" env:"ZOH_VERBOSE"`
	ResultsOnly bool   `help:"Strip JSON envelope, return data array only" env:"ZOH_RESULTS_ONLY"`
	NoInput     bool   `help:"Disable interactive prompts (fail instead)" env:"ZOH_NO_INPUT"`
	Force       bool   `help:"Skip confirmation prompts for destructive operations" env:"ZOH_FORCE"`
	DryRun      bool   `help:"Preview operation without executing" name:"dry-run" env:"ZOH_DRY_RUN"`
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
