package auth

import "strings"

// DefaultScopes defines the OAuth2 scopes required for v1 functionality.
// Zoho uses comma-separated scopes (not space-separated like standard OAuth2).
// Scope reference: https://www.zoho.com/mail/help/api/
var DefaultScopes = []string{
	// Mail
	"ZohoMail.messages.ALL",
	"ZohoMail.folders.ALL",
	"ZohoMail.tags.ALL",
	"ZohoMail.accounts.ALL",
	// Organization admin
	"ZohoMail.organization.accounts.ALL",
	"ZohoMail.organization.domains.ALL",
	"ZohoMail.organization.groups.ALL",
	"ZohoMail.organization.spam.ALL",
	"ZohoMail.organization.policy.ALL",
	"ZohoMail.organization.audit.READ",
}

// ScopeString returns scopes as a comma-separated string.
// Zoho requires comma separator, not the space separator used by standard OAuth2.
func ScopeString() string {
	return strings.Join(DefaultScopes, ",")
}
