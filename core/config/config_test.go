package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/AvengeMedia/dankcalendar/core/config"
)

var configEnvVars = []string{
	"DANKCAL_API_ADDR",
	"DANKCAL_OAUTH_ADDR",
	"DANKCAL_DB_PATH",
	"DANKCAL_GOOGLE_CLIENT_ID",
	"DANKCAL_GOOGLE_CLIENT_SECRET",
	"DANKCAL_MICROSOFT_CLIENT_ID",
	"DANKCAL_DISABLE_HTTP",
	"DANKCAL_DISABLE_IPC",
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvVars {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestConfigDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg := config.New()

	assert.Equal(t, "127.0.0.1:0", cfg.APIAddr)
	assert.Equal(t, "127.0.0.1:0", cfg.OAuthBindAddr)
	assert.Empty(t, cfg.DatabasePath)
	assert.False(t, cfg.DisableHTTP)
	assert.False(t, cfg.DisableIPC)
}

func TestConfigReadsEnvironment(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DANKCAL_API_ADDR", "127.0.0.1:8765")
	t.Setenv("DANKCAL_DB_PATH", "/tmp/dankcal.db")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("DANKCAL_DISABLE_HTTP", "true")

	cfg := config.New()

	assert.Equal(t, "127.0.0.1:8765", cfg.APIAddr)
	assert.Equal(t, "/tmp/dankcal.db", cfg.DatabasePath)
	assert.Equal(t, "client-id", cfg.GoogleClientID)
	assert.True(t, cfg.DisableHTTP)
	assert.False(t, cfg.DisableIPC)
}
