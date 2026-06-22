package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gojangframework/gojang/app/gojang/config"
)

func TestSecurityHeaders_DefaultCSP(t *testing.T) {
	handler := SecurityHeadersWithOptions(SecurityOptions{Debug: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' https://unpkg.com",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, want) {
			t.Fatalf("CSP %q does not contain %q", csp, want)
		}
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("Strict-Transport-Security = %q, want empty in debug", got)
	}
}

func TestSecurityHeaders_ConfigOverrides(t *testing.T) {
	cfg := &config.Config{
		Debug:                       false,
		CSPScriptSrc:                []string{"'self'", "https://cdn.example.com"},
		CSPConnectSrc:               []string{"'self'", "https://api.example.com"},
		CSPSameOriginFrameAncestors: []string{"/embed/"},
	}
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/regular", nil)
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{
		"script-src 'self' https://cdn.example.com",
		"connect-src 'self' https://api.example.com",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, want) {
			t.Fatalf("CSP %q does not contain %q", csp, want)
		}
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("Strict-Transport-Security is empty in production mode")
	}
}

func TestSecurityHeaders_SameOriginFramePath(t *testing.T) {
	handler := SecurityHeadersWithOptions(SecurityOptions{
		Debug:                        true,
		SameOriginFrameAncestorPaths: []string{"/embed/"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/embed/report", nil)
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors 'self'") {
		t.Fatalf("CSP %q does not contain same-origin frame ancestors", csp)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %q, want SAMEORIGIN", got)
	}
}
