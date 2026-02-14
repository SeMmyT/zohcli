package cli

// LsCmd provides desire-path shortcuts for listing resources
// These are aliases to full command paths for faster interactive use
type LsCmd struct {
	Users   AdminUsersListCmd  `cmd:"" help:"List users (shortcut for admin users list)"`
	Groups  AdminGroupsListCmd `cmd:"" help:"List groups (shortcut for admin groups list)"`
	Folders MailFoldersListCmd `cmd:"" help:"List folders (shortcut for mail folders list)"`
	Labels  MailLabelsListCmd  `cmd:"" help:"List labels (shortcut for mail labels list)"`
}
