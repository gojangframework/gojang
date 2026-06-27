package utils

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRecaptchaVerifierDisabledWithoutBothKeys(t *testing.T) {
	verifier := NewRecaptchaVerifier(RecaptchaConfig{
		SiteKey: "site-key",
	})

	if verifier.Enabled() {
		t.Fatal("verifier should be disabled without both keys")
	}
	if got := verifier.SiteKey(); got != "" {
		t.Fatalf("SiteKey() = %q, want empty", got)
	}
	if got := verifier.ScriptURL(); got != "" {
		t.Fatalf("ScriptURL() = %q, want empty", got)
	}
	if err := verifier.Verify(context.Background(), "", "register", ""); err != nil {
		t.Fatalf("Verify() error = %v, want nil when disabled", err)
	}
}

func TestRecaptchaVerifierSuccess(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		gotForm = r.PostForm
		w.Write([]byte(`{"success":true,"score":0.9,"action":"register","hostname":"example.com"}`))
	}))
	t.Cleanup(server.Close)

	verifier := NewRecaptchaVerifier(RecaptchaConfig{
		SiteKey:          "site-key",
		SecretKey:        "secret-key",
		VerifyURL:        server.URL,
		AllowedHostnames: []string{"example.com"},
	})

	if err := verifier.Verify(context.Background(), "token", "register", "203.0.113.10"); err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
	if gotForm.Get("secret") != "secret-key" || gotForm.Get("response") != "token" || gotForm.Get("remoteip") != "203.0.113.10" {
		t.Fatalf("posted form = %#v", gotForm)
	}
	if got := verifier.ScriptURL(); got != "https://www.google.com/recaptcha/api.js?render=site-key" {
		t.Fatalf("ScriptURL() = %q", got)
	}
}

func TestRecaptchaVerifierRejectsLowScore(t *testing.T) {
	verifier := newRecaptchaVerifierWithResponse(t, `{"success":true,"score":0.3,"action":"register"}`)

	err := verifier.Verify(context.Background(), "token", "register", "")
	if !errors.Is(err, ErrRecaptchaVerificationFailed) {
		t.Fatalf("Verify() error = %v, want ErrRecaptchaVerificationFailed", err)
	}
	var recaptchaErr *RecaptchaVerificationError
	if !errors.As(err, &recaptchaErr) {
		t.Fatalf("Verify() error = %T, want RecaptchaVerificationError", err)
	}
	if recaptchaErr.Reason != "score below threshold" {
		t.Fatalf("Reason = %q, want score below threshold", recaptchaErr.Reason)
	}
	if recaptchaErr.Score != 0.3 || recaptchaErr.MinScore != 0.5 {
		t.Fatalf("score details = score %f min %f, want 0.3/0.5", recaptchaErr.Score, recaptchaErr.MinScore)
	}
}

func TestRecaptchaVerifierRejectsHostnameMismatch(t *testing.T) {
	verifier := newRecaptchaVerifierWithResponse(t, `{"success":true,"score":0.9,"action":"register","hostname":"evil.example"}`)
	verifier.allowedHostnames = []string{"example.com"}

	err := verifier.Verify(context.Background(), "token", "register", "")
	var recaptchaErr *RecaptchaVerificationError
	if !errors.As(err, &recaptchaErr) {
		t.Fatalf("Verify() error = %T, want RecaptchaVerificationError", err)
	}
	if recaptchaErr.Reason != "hostname mismatch" {
		t.Fatalf("Reason = %q, want hostname mismatch", recaptchaErr.Reason)
	}
	if recaptchaErr.Hostname != "evil.example" {
		t.Fatalf("Hostname = %q, want evil.example", recaptchaErr.Hostname)
	}
}

func TestRecaptchaVerifierAllowsWildcardHostname(t *testing.T) {
	verifier := newRecaptchaVerifierWithResponse(t, `{"success":true,"score":0.9,"action":"register","hostname":"app.example.com"}`)
	verifier.allowedHostnames = []string{"*.example.com"}

	if err := verifier.Verify(context.Background(), "token", "register", ""); err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
}

func TestRecaptchaVerifierRejectsWrongAction(t *testing.T) {
	verifier := newRecaptchaVerifierWithResponse(t, `{"success":true,"score":0.9,"action":"login"}`)

	err := verifier.Verify(context.Background(), "token", "register", "")
	var recaptchaErr *RecaptchaVerificationError
	if !errors.As(err, &recaptchaErr) {
		t.Fatalf("Verify() error = %T, want RecaptchaVerificationError", err)
	}
	if recaptchaErr.Reason != "action mismatch" {
		t.Fatalf("Reason = %q, want action mismatch", recaptchaErr.Reason)
	}
	if recaptchaErr.Action != "login" || recaptchaErr.ExpectedAction != "register" {
		t.Fatalf("action details = %q/%q, want login/register", recaptchaErr.Action, recaptchaErr.ExpectedAction)
	}
}

func TestRecaptchaVerifierRejectsMissingToken(t *testing.T) {
	verifier := NewRecaptchaVerifier(RecaptchaConfig{
		SiteKey:   "site-key",
		SecretKey: "secret-key",
	})

	err := verifier.Verify(context.Background(), " ", "register", "")
	var recaptchaErr *RecaptchaVerificationError
	if !errors.As(err, &recaptchaErr) {
		t.Fatalf("Verify() error = %T, want RecaptchaVerificationError", err)
	}
	if recaptchaErr.Reason != "missing token" {
		t.Fatalf("Reason = %q, want missing token", recaptchaErr.Reason)
	}
}

func TestRecaptchaVerifierIncludesGoogleErrorCodes(t *testing.T) {
	verifier := newRecaptchaVerifierWithResponse(t, `{"success":false,"error-codes":["invalid-input-secret"]}`)

	err := verifier.Verify(context.Background(), "token", "register", "")
	var recaptchaErr *RecaptchaVerificationError
	if !errors.As(err, &recaptchaErr) {
		t.Fatalf("Verify() error = %T, want RecaptchaVerificationError", err)
	}
	if recaptchaErr.Reason != "google rejected token" {
		t.Fatalf("Reason = %q, want google rejected token", recaptchaErr.Reason)
	}
	if len(recaptchaErr.ErrorCodes) != 1 || recaptchaErr.ErrorCodes[0] != "invalid-input-secret" {
		t.Fatalf("ErrorCodes = %#v, want invalid-input-secret", recaptchaErr.ErrorCodes)
	}
}

func newRecaptchaVerifierWithResponse(t *testing.T, response string) *RecaptchaVerifier {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if contentType := r.Header.Get("Content-Type"); !strings.Contains(contentType, "application/x-www-form-urlencoded") {
			t.Fatalf("Content-Type = %q, want form encoded", contentType)
		}
		w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	return NewRecaptchaVerifier(RecaptchaConfig{
		SiteKey:   "site-key",
		SecretKey: "secret-key",
		VerifyURL: server.URL,
	})
}
