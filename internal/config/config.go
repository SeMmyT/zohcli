package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// Config holds the CLI configuration
type Config struct {
	Region        string `json:"region"`
	ClientID      string `json:"client_id,omitempty"`
	ClientSecret  string `json:"client_secret,omitempty"`
	OrgID         string `json:"org_id,omitempty"`
	AccountID     string `json:"account_id,omitempty"`
	DefaultOutput string `json:"default_output,omitempty"`
}

// Load reads config from XDG path, returns defaults if file doesn't exist
func Load() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults when no config file exists
			return &Config{
				Region: "", // Empty means "not set" - will be resolved to "us" in cli.BeforeApply
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json5.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to the XDG config path
func (c *Config) Save() error {
	path := ConfigPath()

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON (not JSON5 for writing - JSON is valid JSON5)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with secure permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Get retrieves a config value by key name
func (c *Config) Get(key string) (string, error) {
	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == key || jsonTag == key+",omitempty" {
			return fmt.Sprintf("%v", v.Field(i).Interface()), nil
		}
	}

	return "", fmt.Errorf("unknown config key: %s", key)
}

// Set sets a config value by key name and saves
func (c *Config) Set(key, value string) error {
	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == key || jsonTag == key+",omitempty" {
			v.Field(i).SetString(value)
			return c.Save()
		}
	}

	return fmt.Errorf("unknown config key: %s", key)
}

// Unset sets a config value to its zero value and saves
func (c *Config) Unset(key string) error {
	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == key || jsonTag == key+",omitempty" {
			v.Field(i).SetString("")
			return c.Save()
		}
	}

	return fmt.Errorf("unknown config key: %s", key)
}

// GetRegionConfig returns the RegionConfig for the configured region
func (c *Config) GetRegionConfig() (RegionConfig, error) {
	return GetRegion(c.Region)
}
