package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/semmy-space/zoh/internal/auth"
	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/secrets"
	"github.com/semmy-space/zoh/internal/zoho"
)

// newMailClient creates a MailClient from config and stored credentials
func newMailClient(cfg *config.Config) (*zoho.MailClient, error) {
	store, err := secrets.NewStore()
	if err != nil {
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	tokenCache, err := auth.NewTokenCache(cfg, store)
	if err != nil {
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize token cache: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	mailClient, err := zoho.NewMailClient(cfg, tokenCache)
	if err != nil {
		// Check if it's an authentication error
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
			return nil, &output.CLIError{
				Message:  fmt.Sprintf("Authentication failed: %v\n\nRun: zoh auth login", err),
				ExitCode: output.ExitAuth,
			}
		}
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to create mail client: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return mailClient, nil
}

// MailFoldersListCmd lists all mail folders
type MailFoldersListCmd struct{}

// Run executes the list folders command
func (cmd *MailFoldersListCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	mailClient, err := newMailClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	folders, err := mailClient.ListFolders(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch folders: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Name", Key: "FolderName"},
		{Name: "Type", Key: "FolderType"},
		{Name: "Path", Key: "Path"},
		{Name: "Messages", Key: "MessageCount"},
		{Name: "Unread", Key: "UnreadCount"},
		{Name: "ID", Key: "FolderID"},
	}

	return fp.Formatter.PrintList(folders, columns)
}

// MailLabelsListCmd lists all mail labels
type MailLabelsListCmd struct{}

// Run executes the list labels command
func (cmd *MailLabelsListCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	mailClient, err := newMailClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	labels, err := mailClient.ListLabels(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch labels: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Name", Key: "LabelName"},
		{Name: "Color", Key: "LabelColor"},
		{Name: "ID", Key: "LabelID"},
	}

	return fp.Formatter.PrintList(labels, columns)
}
