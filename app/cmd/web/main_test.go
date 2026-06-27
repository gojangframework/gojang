package main

import (
	"strings"
	"testing"

	"github.com/gojangframework/gojang/app/gojang/config"
)

func TestValidateRecaptchaConfigAllowsBothKeys(t *testing.T) {
	cfg := &config.Config{
		RecaptchaSiteKey:   "site-key",
		RecaptchaSecretKey: "secret-key",
	}

	if err := validateRecaptchaConfig(cfg); err != nil {
		t.Fatalf("validateRecaptchaConfig() error = %v, want nil", err)
	}
}

func TestValidateRecaptchaConfigAllowsEmptyKeys(t *testing.T) {
	if err := validateRecaptchaConfig(&config.Config{}); err != nil {
		t.Fatalf("validateRecaptchaConfig() error = %v, want nil", err)
	}
}

func TestValidateRecaptchaConfigRejectsPartialProductionConfig(t *testing.T) {
	for _, cfg := range []*config.Config{
		{RecaptchaSiteKey: "site-key"},
		{RecaptchaSecretKey: "secret-key"},
	} {
		err := validateRecaptchaConfig(cfg)
		if err == nil {
			t.Fatal("validateRecaptchaConfig() error = nil, want partial config error")
		}
		if !strings.Contains(err.Error(), "RECAPTCHA_SITE_KEY") || !strings.Contains(err.Error(), "RECAPTCHA_SECRET_KEY") {
			t.Fatalf("validateRecaptchaConfig() error = %q, want env names", err.Error())
		}
	}
}

func TestValidateRecaptchaConfigAllowsPartialDebugConfig(t *testing.T) {
	cfg := &config.Config{
		Debug:            true,
		RecaptchaSiteKey: "site-key",
	}

	if err := validateRecaptchaConfig(cfg); err != nil {
		t.Fatalf("validateRecaptchaConfig() error = %v, want nil in debug mode", err)
	}
}
