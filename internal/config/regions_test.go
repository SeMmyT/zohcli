package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRegion(t *testing.T) {
	validRegions := []string{"us", "eu", "in", "au", "jp", "ca", "sa", "uk"}

	for _, code := range validRegions {
		t.Run("valid_"+code, func(t *testing.T) {
			cfg, err := GetRegion(code)
			require.NoError(t, err)
			assert.NotEmpty(t, cfg.AccountsServer)
			assert.NotEmpty(t, cfg.APIBase)
			assert.NotEmpty(t, cfg.MailBase)
		})
	}

	t.Run("invalid region returns error", func(t *testing.T) {
		_, err := GetRegion("xx")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown region")
	})

	t.Run("empty string returns error", func(t *testing.T) {
		_, err := GetRegion("")
		assert.Error(t, err)
	})

	t.Run("us region has correct URLs", func(t *testing.T) {
		cfg, err := GetRegion("us")
		require.NoError(t, err)
		assert.Equal(t, "https://accounts.zoho.com", cfg.AccountsServer)
		assert.Equal(t, "https://www.zohoapis.com", cfg.APIBase)
		assert.Equal(t, "https://mail.zoho.com", cfg.MailBase)
	})
}

func TestValidRegions(t *testing.T) {
	regions := ValidRegions()

	assert.Len(t, regions, 8)
	assert.Equal(t, []string{"au", "ca", "eu", "in", "jp", "sa", "uk", "us"}, regions)

	// Verify sorted
	for i := 1; i < len(regions); i++ {
		assert.Less(t, regions[i-1], regions[i], "regions should be sorted")
	}
}
