package config

import (
	"fmt"
	"sort"
)

// RegionConfig holds the endpoint URLs for a Zoho data center
type RegionConfig struct {
	AccountsServer string
	APIBase        string
	MailBase       string
}

// Regions maps region codes to their endpoint configurations
// Covers all 8 Zoho data centers
var Regions = map[string]RegionConfig{
	"us": {
		AccountsServer: "https://accounts.zoho.com",
		APIBase:        "https://www.zohoapis.com",
		MailBase:       "https://mail.zoho.com",
	},
	"eu": {
		AccountsServer: "https://accounts.zoho.eu",
		APIBase:        "https://www.zohoapis.eu",
		MailBase:       "https://mail.zoho.eu",
	},
	"in": {
		AccountsServer: "https://accounts.zoho.in",
		APIBase:        "https://www.zohoapis.in",
		MailBase:       "https://mail.zoho.in",
	},
	"au": {
		AccountsServer: "https://accounts.zoho.com.au",
		APIBase:        "https://www.zohoapis.com.au",
		MailBase:       "https://mail.zoho.com.au",
	},
	"jp": {
		AccountsServer: "https://accounts.zoho.jp",
		APIBase:        "https://www.zohoapis.jp",
		MailBase:       "https://mail.zoho.jp",
	},
	"ca": {
		AccountsServer: "https://accounts.zohocloud.ca",
		APIBase:        "https://www.zohoapis.ca",
		MailBase:       "https://mail.zohocloud.ca",
	},
	"sa": {
		AccountsServer: "https://accounts.zoho.sa",
		APIBase:        "https://www.zohoapis.sa",
		MailBase:       "https://mail.zoho.sa",
	},
	"uk": {
		AccountsServer: "https://accounts.zoho.uk",
		APIBase:        "https://www.zohoapis.uk",
		MailBase:       "https://mail.zoho.uk",
	},
}

// GetRegion returns the configuration for the specified region
func GetRegion(name string) (RegionConfig, error) {
	cfg, ok := Regions[name]
	if !ok {
		return RegionConfig{}, fmt.Errorf("unknown region: %s", name)
	}
	return cfg, nil
}

// ValidRegions returns a sorted list of valid region codes
func ValidRegions() []string {
	regions := make([]string, 0, len(Regions))
	for code := range Regions {
		regions = append(regions, code)
	}
	sort.Strings(regions)
	return regions
}
