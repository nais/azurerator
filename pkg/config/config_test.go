package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nais/azureator/pkg/config"
)

func TestValidateSecretRotationMaxAge(t *testing.T) {
	for _, test := range []struct {
		name   string
		maxAge time.Duration
		valid  bool
	}{
		{name: "zero", maxAge: 0},
		{name: "negative", maxAge: -time.Hour},
		{name: "364 days", maxAge: 364 * 24 * time.Hour, valid: true},
		{name: "365 days", maxAge: 365 * 24 * time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Config{
				Azure:          config.AzureConfig{Delay: config.AzureDelay{CredentialGracePeriod: time.Minute}},
				SecretRotation: config.SecretRotation{MaxAge: test.maxAge},
			}
			err := cfg.Validate(nil)
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), config.SecretRotationMaxAge)
			}
		})
	}
}
