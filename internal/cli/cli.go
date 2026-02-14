package cli

import (
	"github.com/alecthomas/kong"
	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
)

// FormatterProvider wraps the formatter interface for Kong binding
type FormatterProvider struct {
	Formatter output.Formatter
}

// CLI is the root command structure
type CLI struct {
	Globals

	Auth    AuthCmd    `cmd:"" help:"Authentication commands"`
	Config  ConfigCmd  `cmd:"" help:"Configuration commands"`
	Version VersionCmd `cmd:"" help:"Show version information"`
}

// BeforeApply hook runs before any command execution
// It loads config, resolves region, creates formatter, and binds dependencies
func (c *CLI) BeforeApply(ctx *kong.Context) error {
	// Load config from XDG path (returns defaults if missing)
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Resolve region: CLI flag > config > "us" default
	region := c.Region
	if region == "" && cfg.Region != "" {
		region = cfg.Region
	}
	if region == "" {
		region = "us"
	}
	cfg.Region = region

	// Create output formatter
	formatter := &FormatterProvider{
		Formatter: output.New(c.ResolvedOutput()),
	}

	// Bind dependencies to kong context
	ctx.Bind(cfg)
	ctx.Bind(formatter)
	ctx.Bind(&c.Globals)

	return nil
}

// AuthCmd holds authentication subcommands
type AuthCmd struct {
	Login  LoginCmd  `cmd:"" help:"Log in to Zoho (placeholder for Plan 02)"`
	Logout LogoutCmd `cmd:"" help:"Log out and remove credentials (placeholder for Plan 02)"`
	List   ListCmd   `cmd:"" help:"List saved accounts (placeholder for Plan 02)"`
}

// LoginCmd placeholder
type LoginCmd struct{}

func (cmd *LoginCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("auth login not yet implemented (coming in Plan 02)")
	return nil
}

// LogoutCmd placeholder
type LogoutCmd struct{}

func (cmd *LogoutCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("auth logout not yet implemented (coming in Plan 02)")
	return nil
}

// ListCmd placeholder
type ListCmd struct{}

func (cmd *ListCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("auth list not yet implemented (coming in Plan 02)")
	return nil
}

// ConfigCmd holds configuration subcommands
type ConfigCmd struct {
	Get   GetCmd   `cmd:"" help:"Get config value (placeholder for Plan 03)"`
	Set   SetCmd   `cmd:"" help:"Set config value (placeholder for Plan 03)"`
	Unset UnsetCmd `cmd:"" help:"Unset config value (placeholder for Plan 03)"`
	List  ListConfigCmd `cmd:"" name:"list" help:"List all config values (placeholder for Plan 03)"`
	Path  PathCmd  `cmd:"" help:"Show config file path (placeholder for Plan 03)"`
}

// GetCmd placeholder
type GetCmd struct {
	Key string `arg:"" help:"Config key to get"`
}

func (cmd *GetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("config get not yet implemented (coming in Plan 03)")
	return nil
}

// SetCmd placeholder
type SetCmd struct {
	Key   string `arg:"" help:"Config key to set"`
	Value string `arg:"" help:"Value to set"`
}

func (cmd *SetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("config set not yet implemented (coming in Plan 03)")
	return nil
}

// UnsetCmd placeholder
type UnsetCmd struct {
	Key string `arg:"" help:"Config key to unset"`
}

func (cmd *UnsetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("config unset not yet implemented (coming in Plan 03)")
	return nil
}

// ListConfigCmd placeholder
type ListConfigCmd struct{}

func (cmd *ListConfigCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("config list not yet implemented (coming in Plan 03)")
	return nil
}

// PathCmd placeholder
type PathCmd struct{}

func (cmd *PathCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	fp.Formatter.PrintHint("config path not yet implemented (coming in Plan 03)")
	return nil
}

// VersionCmd shows version information
type VersionCmd struct{}

func (cmd *VersionCmd) Run(ctx *kong.Context) error {
	version := ctx.Model.Vars()["version"]
	println("zoh version " + version)
	return nil
}
