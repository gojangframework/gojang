package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetLanguageSetsCookieAndRedirects(t *testing.T) {
	handler := &PageHandler{}
	req := httptest.NewRequest(http.MethodGet, "/set-language?lang=es", nil)
	req.Header.Set("Referer", "/posts?lang=ko&page=2")
	rec := httptest.NewRecorder()

	handler.SetLanguage(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusSeeOther)
	}
	if got := res.Header.Get("Location"); got != "/posts?page=2" {
		t.Fatalf("Location = %q, want %q", got, "/posts?page=2")
	}

	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "lang" || cookies[0].Value != "es" {
		t.Fatalf("cookie = %s=%s, want lang=es", cookies[0].Name, cookies[0].Value)
	}
	if cookies[0].Path != "/" {
		t.Fatalf("cookie path = %q, want /", cookies[0].Path)
	}
	if cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want Lax", cookies[0].SameSite)
	}
}

func TestSetLanguageInvalidFallsBackToEnglish(t *testing.T) {
	handler := &PageHandler{}
	req := httptest.NewRequest(http.MethodGet, "/set-language?lang=fr", nil)
	rec := httptest.NewRecorder()

	handler.SetLanguage(rec, req)

	res := rec.Result()
	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Value != "en" {
		t.Fatalf("cookie value = %q, want en", cookies[0].Value)
	}
	if location := res.Header.Get("Location"); location != "/" {
		t.Fatalf("Location = %q, want /", location)
	}
}

func TestSetLanguageAcceptsCanonicalLowercaseChinese(t *testing.T) {
	handler := &PageHandler{}
	req := httptest.NewRequest(http.MethodGet, "/set-language?lang=zh-hant", nil)
	rec := httptest.NewRecorder()

	handler.SetLanguage(rec, req)

	cookieHeader := rec.Result().Header.Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "lang=zh-hant") {
		t.Fatalf("Set-Cookie = %q, want lang=zh-hant", cookieHeader)
	}
}
