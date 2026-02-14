package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/semmy-space/zoh/internal/cli"
	"github.com/semmy-space/zoh/internal/output"
)

var (
	version = "dev"
)

func main() {
	// Parse CLI
	cliInstance := &cli.CLI{}
	ctx := kong.Parse(cliInstance,
		kong.Name("zoh"),
		kong.Description("Zoho CLI for Admin and Mail operations"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version,
		},
	)

	// Run command with bound dependencies
	err := ctx.Run()
	if err != nil {
		// Handle error with proper exit code
		if cliErr, ok := err.(*output.CLIError); ok {
			// We need a formatter instance, create a basic one for error output
			formatter := output.New("plain")
			formatter.PrintError(err)
			if cliErr.Hint != "" {
				formatter.PrintHint(cliErr.Hint)
			}
			os.Exit(cliErr.ExitCode)
		}
		// Unknown error
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(output.ExitGeneral)
	}
}
