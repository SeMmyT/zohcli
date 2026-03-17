package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/SeMmyT/zohcli/internal/output"
)

// AdminAntispamCmd holds antispam policy subcommands
type AdminAntispamCmd struct {
	Get AdminAntispamGetCmd `cmd:"" help:"Show all antispam settings"`
	Set AdminAntispamSetCmd `cmd:"" help:"Update an antispam check action"`
}

// AdminAntispamGetCmd shows all antispam settings as a table
type AdminAntispamGetCmd struct{}

// Run executes the antispam get command
func (cmd *AdminAntispamGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	checks := []struct {
		Name string
		Key  string
	}{
		{"SPF", "spf"},
		{"DKIM", "dkim"},
		{"DMARC", "dmarc"},
		{"Cousin Domain", "cousin-domain"},
	}

	type displayRow struct {
		Check  string
		Action string
	}

	rows := make([]displayRow, 0, len(checks))
	for _, c := range checks {
		val, err := adminClient.GetAntispamOption(ctx, c.Key)
		if err != nil {
			val = fmt.Sprintf("error: %v", err)
		}
		rows = append(rows, displayRow{
			Check:  c.Name,
			Action: val,
		})
	}

	columns := []output.Column{
		{Name: "Check", Key: "Check"},
		{Name: "Action", Key: "Action"},
	}

	return fp.Formatter.PrintList(rows, columns)
}

// AdminAntispamSetCmd updates a single antispam check
type AdminAntispamSetCmd struct {
	Check  string `help:"Antispam check to update" required:"" enum:"spf,dkim,dmarc,cousin-domain"`
	Action string `help:"Action to take on failure: quarantine, reject, temp-reject, allow, spam" required:"" enum:"quarantine,reject,temp-reject,allow,spam"`
}

// Run executes the antispam set command
func (cmd *AdminAntispamSetCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Map user-friendly action names to API values
	actionMap := map[string]string{
		"quarantine":  "moveToQuarantine",
		"reject":      "permanentReject",
		"temp-reject": "temporaryReject",
		"allow":       "allowMail",
		"spam":        "moveToSpam",
	}

	apiAction := actionMap[cmd.Action]

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would set %s antispam action to %s (%s)\n", cmd.Check, cmd.Action, apiAction)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	err = adminClient.SetAntispamOption(ctx, cmd.Check, apiAction)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to update antispam setting: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Antispam %s action updated to %s\n", cmd.Check, cmd.Action)
	return nil
}
