package config

import (
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/gojangframework/gojang/gojang/utils"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string   `env:"DATABASE_URL,required"`
	SessionKey   string   `env:"SESSION_KEY,required"`
	Debug        bool     `env:"DEBUG" envDefault:"false"`
	Port         string   `env:"PORT" envDefault:"8080"`
	AllowedHosts []string `env:"ALLOWED_HOSTS" envSeparator:","`

	// Session settings
	SessionLifetime time.Duration `env:"SESSION_LIFETIME" envDefault:"12h"`

	// SMTP
	SMTPHost string `env:"SMTP_HOST"`
	SMTPPort int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser string `env:"SMTP_USER"`
	SMTPPass string `env:"SMTP_PASS"`
	SMTPFrom string `env:"SMTP_FROM" envDefault:"noreply@localhost"`
}

func Load() (*Config, error) {
	// Try to load .env file, fall back to .env.example for development
	if err := godotenv.Load(); err != nil {
		// If .env doesn't exist, try .env.example
		if err := godotenv.Load(".env.example"); err != nil {
			// Silently use environment variables only
		}
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if cfg.Debug {
		utils.Warnf("Running in DEBUG mode")
	}

	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		utils.Errorf("Failed to load config: %v", err)
		panic(err)
	}
	return cfg
}
