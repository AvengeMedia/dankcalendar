package config

import (
	"github.com/AvengeMedia/dankgo/log"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	APIAddr        string `env:"DANKCAL_API_ADDR" envDefault:"127.0.0.1:0"`
	OAuthBindAddr  string `env:"DANKCAL_OAUTH_ADDR" envDefault:"127.0.0.1:0"`
	DatabasePath   string `env:"DANKCAL_DB_PATH"`
	GoogleClientID string `env:"DANKCAL_GOOGLE_CLIENT_ID"`
	GoogleSecret   string `env:"DANKCAL_GOOGLE_CLIENT_SECRET"`
	DisableHTTP    bool   `env:"DANKCAL_DISABLE_HTTP" envDefault:"false"`
	DisableIPC     bool   `env:"DANKCAL_DISABLE_IPC" envDefault:"false"`
}

func New() *Config {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("error parsing config: %v", err)
	}
	return &cfg
}
