package config

import (
	"testing"
	"time"

	"github.com/caarlos0/env/v9"
)

func TestConfig_DefaultValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlite://test.db")
	t.Setenv("SESSION_KEY", "test-session-key-32-chars-long!")

	cfg := &Config{}
	cfg.Port = "8080"
	cfg.Debug = false
	cfg.AppBaseURL = "http://localhost:8080"
	cfg.SessionLifetime = 12 * time.Hour
	cfg.SMTPPort = 587
	cfg.SMTPFrom = "noreply@localhost"
	cfg.AWSSESRegion = "us-east-1"
	cfg.AWSSESFromEmail = "noreply@localhost"
	cfg.AWSSESFromDisplayName = "Gojang"
	cfg.RecaptchaMinScore = 0.5

	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}

	if cfg.Debug != false {
		t.Errorf("Expected debug false by default, got %v", cfg.Debug)
	}

	if cfg.SessionLifetime != 12*time.Hour {
		t.Errorf("Expected session lifetime 12h, got %v", cfg.SessionLifetime)
	}

	if cfg.AppBaseURL != "http://localhost:8080" {
		t.Errorf("Expected app base URL http://localhost:8080, got %s", cfg.AppBaseURL)
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("Expected SMTP port 587, got %d", cfg.SMTPPort)
	}

	if cfg.SMTPFrom != "noreply@localhost" {
		t.Errorf("Expected SMTP from noreply@localhost, got %s", cfg.SMTPFrom)
	}

	if cfg.AWSSESRegion != "us-east-1" {
		t.Errorf("Expected AWS SES region us-east-1, got %s", cfg.AWSSESRegion)
	}

	if cfg.RecaptchaMinScore != 0.5 {
		t.Errorf("Expected recaptcha min score 0.5, got %f", cfg.RecaptchaMinScore)
	}
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := &Config{
		DatabaseURL:                  "postgresql://localhost/testdb",
		SessionKey:                   "custom-session-key",
		Debug:                        true,
		Port:                         "3000",
		AppBaseURL:                   "https://app.example.com",
		SessionLifetime:              24 * time.Hour,
		SMTPHost:                     "smtp.example.com",
		SMTPPort:                     465,
		SMTPUser:                     "user@example.com",
		SMTPPass:                     "password",
		SMTPFrom:                     "custom@example.com",
		AWSSESAccessKeyID:            "AKIA_TEST",
		AWSSESRegion:                 "us-west-2",
		GoogleAnalyticsMeasurementID: "G-1234567890",
		RecaptchaSiteKey:             "site-key",
		RecaptchaSecretKey:           "secret-key",
		RecaptchaMinScore:            0.7,
		RecaptchaAllowedHostnames:    []string{"example.com", "*.example.com"},
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

	if cfg.AppBaseURL != "https://app.example.com" {
		t.Errorf("Expected app base URL https://app.example.com, got %s", cfg.AppBaseURL)
	}

	if cfg.SessionLifetime != 24*time.Hour {
		t.Errorf("Expected session lifetime 24h, got %v", cfg.SessionLifetime)
	}

	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("Expected SMTP host smtp.example.com, got %s", cfg.SMTPHost)
	}

	if cfg.AWSSESAccessKeyID != "AKIA_TEST" {
		t.Errorf("Expected AWS SES access key to be configured")
	}

	if cfg.GoogleAnalyticsMeasurementID != "G-1234567890" {
		t.Errorf("Expected Google Analytics measurement ID to be configured")
	}

	if cfg.RecaptchaSiteKey != "site-key" {
		t.Errorf("Expected recaptcha site key site-key, got %s", cfg.RecaptchaSiteKey)
	}

	if cfg.RecaptchaSecretKey != "secret-key" {
		t.Errorf("Expected recaptcha secret key secret-key, got %s", cfg.RecaptchaSecretKey)
	}

	if cfg.RecaptchaMinScore != 0.7 {
		t.Errorf("Expected recaptcha min score 0.7, got %f", cfg.RecaptchaMinScore)
	}

	if len(cfg.RecaptchaAllowedHostnames) != 2 || cfg.RecaptchaAllowedHostnames[0] != "example.com" {
		t.Errorf("Expected recaptcha allowed hostnames to be populated, got %#v", cfg.RecaptchaAllowedHostnames)
	}
}

func TestConfig_SESEnvNames(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlite://test.db")
	t.Setenv("SESSION_KEY", "test-session-key-32-chars-long!")
	t.Setenv("AWS_SES_ACCESS_KEY_ID", "AKIA_SES_TEST")
	t.Setenv("AWS_SES_SECRET_ACCESS_KEY", "ses-secret")
	t.Setenv("AWS_SES_REGION", "us-west-2")
	t.Setenv("AWS_SES_FROM_EMAIL_ADDRESS", "ses@example.com")
	t.Setenv("AWS_SES_FROM_DISPLAY_NAME", "SES Sender")

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		t.Fatalf("env.Parse() error = %v", err)
	}

	if cfg.AWSSESAccessKeyID != "AKIA_SES_TEST" {
		t.Fatalf("AWSSESAccessKeyID = %q", cfg.AWSSESAccessKeyID)
	}
	if cfg.AWSSESSecretAccessKey != "ses-secret" {
		t.Fatalf("AWSSESSecretAccessKey was not parsed")
	}
	if cfg.AWSSESRegion != "us-west-2" {
		t.Fatalf("AWSSESRegion = %q", cfg.AWSSESRegion)
	}
	if cfg.AWSSESFromEmail != "ses@example.com" {
		t.Fatalf("AWSSESFromEmail = %q", cfg.AWSSESFromEmail)
	}
	if cfg.AWSSESFromDisplayName != "SES Sender" {
		t.Fatalf("AWSSESFromDisplayName = %q", cfg.AWSSESFromDisplayName)
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
