package cli

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/zoho"
)

// MessageListRow is a display struct for message list output with formatted fields
type MessageListRow struct {
	Status        string
	FromAddress   string
	Subject       string
	Date          string
	Attachment    string
	MessageID     string
}

// MessageDetail is a display struct combining metadata and content
type MessageDetail struct {
	Subject       string
	From          string
	To            string
	Cc            string
	Date          string
	Size          string
	Status        string
	Priority      string
	HasAttachment string
	MessageID     string
	ThreadID      string
	FolderID      string
	Body          string
}

// MailMessagesListCmd lists messages in a folder
type MailMessagesListCmd struct {
	Folder string `help:"Folder name or ID" default:"Inbox" short:"f"`
	Limit  int    `help:"Maximum messages to show" short:"l" default:"50"`
	All    bool   `help:"Fetch all messages (no pagination limit)" short:"a"`
}

// Run executes the list messages command
func (cmd *MailMessagesListCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	mailClient, err := newMailClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve folder name to folder ID
	var folderID string
	folder, err := mailClient.GetFolderByName(ctx, cmd.Folder)
	if err != nil {
		// If GetFolderByName fails, treat cmd.Folder as a folder ID
		folderID = cmd.Folder
	} else {
		folderID = folder.FolderID
	}

	var messages []zoho.MessageSummary

	if cmd.All {
		// Use PageIterator to fetch all messages
		iterator := zoho.NewPageIterator(func(start, limit int) ([]zoho.MessageSummary, error) {
			return mailClient.ListMessages(ctx, folderID, start, limit)
		}, 50)

		messages, err = iterator.FetchAll()
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch messages: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	} else {
		// Fetch single page
		messages, err = mailClient.ListMessages(ctx, folderID, 0, cmd.Limit)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch messages: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	}

	// Convert to display rows with formatted fields
	rows := make([]MessageListRow, len(messages))
	for i, msg := range messages {
		// Format timestamp (unix ms to human readable)
		timestamp := time.UnixMilli(msg.ReceivedTime).Format("2006-01-02 15:04")

		// Format attachment indicator
		attachment := ""
		if msg.HasAttachment {
			attachment = "Y"
		}

		rows[i] = MessageListRow{
			Status:      msg.Status,
			FromAddress: msg.FromAddress,
			Subject:     msg.Subject,
			Date:        timestamp,
			Attachment:  attachment,
			MessageID:   msg.MessageID,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Status", Key: "Status"},
		{Name: "From", Key: "FromAddress"},
		{Name: "Subject", Key: "Subject"},
		{Name: "Date", Key: "Date"},
		{Name: "Attachment", Key: "Attachment"},
		{Name: "ID", Key: "MessageID"},
	}

	return fp.Formatter.PrintList(rows, columns)
}

// MailMessagesGetCmd gets full details for a specific message
type MailMessagesGetCmd struct {
	MessageID string `arg:"" help:"Message ID to retrieve"`
	Folder    string `help:"Folder name or ID (required)" short:"f" required:""`
}

// Run executes the get message command
func (cmd *MailMessagesGetCmd) Run(cfg *config.Config, fp *FormatterProvider, globals *Globals) error {
	mailClient, err := newMailClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve folder name to folder ID
	var folderID string
	folder, err := mailClient.GetFolderByName(ctx, cmd.Folder)
	if err != nil {
		// If GetFolderByName fails, treat cmd.Folder as a folder ID
		folderID = cmd.Folder
	} else {
		folderID = folder.FolderID
	}

	// Fetch metadata and content (two API calls)
	metadata, err := mailClient.GetMessageMetadata(ctx, folderID, cmd.MessageID)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch message metadata: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	content, err := mailClient.GetMessageContent(ctx, folderID, cmd.MessageID)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch message content: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Build display struct
	detail := MessageDetail{
		Subject:       metadata.Subject,
		From:          metadata.FromAddress,
		To:            metadata.ToAddress,
		Cc:            metadata.CcAddress,
		Date:          time.UnixMilli(metadata.SentDateInGMT).Format("2006-01-02 15:04:05 MST"),
		Size:          formatBytes(metadata.MessageSize),
		Status:        metadata.Status,
		Priority:      formatPriority(metadata.Priority),
		HasAttachment: formatBool(metadata.HasAttachment),
		MessageID:     metadata.MessageID,
		ThreadID:      metadata.ThreadID,
		FolderID:      metadata.FolderID,
		Body:          formatBody(content.Content, globals.ResolvedOutput()),
	}

	return fp.Formatter.Print(detail)
}

// formatBytes converts bytes to human-readable size
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatPriority converts priority int to string
func formatPriority(priority int) string {
	switch priority {
	case 0:
		return "Normal"
	case 1:
		return "High"
	default:
		return fmt.Sprintf("%d", priority)
	}
}

// formatBool converts bool to string
func formatBool(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

// formatBody processes HTML content based on output mode
func formatBody(htmlContent, outputMode string) string {
	if outputMode == "json" {
		// JSON mode returns raw HTML
		return htmlContent
	}

	// Plain and rich modes: strip HTML tags
	re := regexp.MustCompile("<[^>]*>")
	stripped := re.ReplaceAllString(htmlContent, "")
	return strings.TrimSpace(stripped)
}
