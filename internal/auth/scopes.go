package auth

import "strings"

// DefaultScopes defines the OAuth2 scopes required for v1 functionality.
// Zoho uses comma-separated scopes (not space-separated like standard OAuth2).
var DefaultScopes = []string{
	"ZohoMail.messages.ALL",
	"ZohoMail.folders.ALL",
	"ZohoMail.accounts.READ",
	"ZohoMail.organization.accounts.READ",
	"ZohoAdmin.orgs.ALL",
}

// ScopeString returns scopes as a comma-separated string.
// Zoho requires comma separator, not the space separator used by standard OAuth2.
func ScopeString() string {
	return strings.Join(DefaultScopes, ",")
}
