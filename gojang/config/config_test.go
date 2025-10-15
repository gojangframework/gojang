package config

import (
	"os"
	"testing"
	"time"
)

func TestConfig_DefaultValues(t *testing.T) {
	// Set required environment variables
	os.Setenv("DATABASE_URL", "sqlite://test.db")
	os.Setenv("SESSION_KEY", "test-session-key-32-chars-long!")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("SESSION_KEY")
	}()

	cfg := &Config{}
	cfg.Port = "8080"
	cfg.Debug = false
	cfg.SessionLifetime = 12 * time.Hour
	cfg.SMTPPort = 587
	cfg.SMTPFrom = "noreply@localhost"

	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}

	if cfg.Debug != false {
		t.Errorf("Expected debug false by default, got %v", cfg.Debug)
	}

	if cfg.SessionLifetime != 12*time.Hour {
		t.Errorf("Expected session lifetime 12h, got %v", cfg.SessionLifetime)
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("Expected SMTP port 587, got %d", cfg.SMTPPort)
	}

	if cfg.SMTPFrom != "noreply@localhost" {
		t.Errorf("Expected SMTP from noreply@localhost, got %s", cfg.SMTPFrom)
	}
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := &Config{
		DatabaseURL:     "postgresql://localhost/testdb",
		SessionKey:      "custom-session-key",
		Debug:           true,
		Port:            "3000",
		SessionLifetime: 24 * time.Hour,
		SMTPHost:        "smtp.example.com",
		SMTPPort:        465,
		SMTPUser:        "user@example.com",
		SMTPPass:        "password",
		SMTPFrom:        "custom@example.com",
	}

	if cfg.DatabaseURL != "postgresql://localhost/testdb" {
		t.Errorf("Expected custom database URL, got %s", cfg.DatabaseURL)
	}

	if cfg.Debug != true {
		t.Errorf("Expected debug true, got %v", cfg.Debug)
	}

	if cfg.Port != "3000" {
		t.Errorf("Expected port 3000, got %s", cfg.Port)
	}

	if cfg.SessionLifetime != 24*time.Hour {
		t.Errorf("Expected session lifetime 24h, got %v", cfg.SessionLifetime)
	}

	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("Expected SMTP host smtp.example.com, got %s", cfg.SMTPHost)
	}
}

func TestConfig_AllowedHosts(t *testing.T) {
	cfg := &Config{
		AllowedHosts: []string{"localhost", "example.com", "*.example.com"},
	}

	if len(cfg.AllowedHosts) != 3 {
		t.Errorf("Expected 3 allowed hosts, got %d", len(cfg.AllowedHosts))
	}

	if cfg.AllowedHosts[0] != "localhost" {
		t.Errorf("Expected first allowed host localhost, got %s", cfg.AllowedHosts[0])
	}
}
