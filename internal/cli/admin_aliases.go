package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/SeMmyT/zohcli/internal/output"
)

// AdminAliasesListCmd lists aliases for a user
type AdminAliasesListCmd struct {
	Identifier string `arg:"" help:"User ID (zuid) or email address"`
}

// Run executes the list aliases command
func (cmd *AdminAliasesListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	type AliasDisplay struct {
		Email     string `json:"email"`
		IsAlias   bool   `json:"isAlias"`
		IsPrimary bool   `json:"isPrimary"`
	}

	var aliases []AliasDisplay
	for _, ea := range user.EmailAddress {
		aliases = append(aliases, AliasDisplay{
			Email:     ea.MailID,
			IsAlias:   ea.IsAlias,
			IsPrimary: ea.IsPrimary,
		})
	}

	columns := []output.Column{
		{Name: "Email", Key: "Email"},
		{Name: "Alias", Key: "IsAlias"},
		{Name: "Primary", Key: "IsPrimary"},
	}

	return fp.Formatter.PrintList(aliases, columns)
}

// AdminAliasesAddCmd adds an alias to a user
type AdminAliasesAddCmd struct {
	Identifier string   `arg:"" help:"User ID (zuid) or email address"`
	Aliases    []string `arg:"" help:"Email alias(es) to add"`
}

// Run executes the add alias command
func (cmd *AdminAliasesAddCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would add aliases to %s: %v\n", cmd.Identifier, cmd.Aliases)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	if err := adminClient.AddAlias(ctx, user.AccountID, cmd.Aliases); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to add aliases: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	for _, alias := range cmd.Aliases {
		fmt.Fprintf(os.Stderr, "Alias added: %s -> %s\n", alias, user.PrimaryEmail())
	}

	return nil
}

// AdminAliasesRemoveCmd removes an alias from a user
type AdminAliasesRemoveCmd struct {
	Identifier string   `arg:"" help:"User ID (zuid) or email address"`
	Aliases    []string `arg:"" help:"Email alias(es) to remove"`
}

// Run executes the remove alias command
func (cmd *AdminAliasesRemoveCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would remove aliases from %s: %v\n", cmd.Identifier, cmd.Aliases)
		return nil
	}

	if !globals.Force {
		return &output.CLIError{
			Message:  "Removing aliases requires --force flag",
			ExitCode: output.ExitUsage,
		}
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	if err := adminClient.RemoveAlias(ctx, user.AccountID, cmd.Aliases); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to remove aliases: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	for _, alias := range cmd.Aliases {
		fmt.Fprintf(os.Stderr, "Alias removed: %s from %s\n", alias, user.PrimaryEmail())
	}

	return nil
}
