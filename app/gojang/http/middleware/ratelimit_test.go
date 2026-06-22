package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Every(time.Second), 5)

	if limiter == nil {
		t.Fatal("Expected limiter to be created")
	}

	if limiter.rate != rate.Every(time.Second) {
		t.Errorf("Expected rate %v, got %v", rate.Every(time.Second), limiter.rate)
	}

	if limiter.burst != 5 {
		t.Errorf("Expected burst 5, got %d", limiter.burst)
	}
}

func TestGetLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Every(time.Second), 5)

	ip := "192.168.1.1"
	l1 := limiter.GetLimiter(ip)
	l2 := limiter.GetLimiter(ip)

	// Should return same limiter for same IP
	if l1 != l2 {
		t.Error("Expected same limiter for same IP")
	}

	// Should create different limiter for different IP
	ip2 := "192.168.1.2"
	l3 := limiter.GetLimiter(ip2)
	if l1 == l3 {
		t.Error("Expected different limiter for different IP")
	}
}

func TestCleanupOldLimiters(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Every(time.Second), 5)

	// Add some limiters
	limiter.GetLimiter("192.168.1.1")
	limiter.GetLimiter("192.168.1.2")
	limiter.GetLimiter("192.168.1.3")

	if len(limiter.limiters) != 3 {
		t.Errorf("Expected 3 limiters, got %d", len(limiter.limiters))
	}

	// Cleanup
	limiter.CleanupOldLimiters()

	if len(limiter.limiters) != 0 {
		t.Errorf("Expected 0 limiters after cleanup, got %d", len(limiter.limiters))
	}
}

func TestGetRealIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getRealIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", ip)
	}
}

func TestGetRealIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")

	ip := getRealIP(req)
	if ip != "203.0.113.1" {
		t.Errorf("Expected IP 203.0.113.1 (first in XFF), got %s", ip)
	}
}

func TestGetRealIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.1")

	ip := getRealIP(req)
	if ip != "203.0.113.1" {
		t.Errorf("Expected IP 203.0.113.1, got %s", ip)
	}
}

func TestGetRealIP_Priority(t *testing.T) {
	// X-Forwarded-For should take priority over X-Real-IP
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("X-Real-IP", "203.0.113.2")

	ip := getRealIP(req)
	if ip != "203.0.113.1" {
		t.Errorf("Expected IP from X-Forwarded-For (203.0.113.1), got %s", ip)
	}
}

func TestRateLimit_AllowsRequests(t *testing.T) {
	// Create limiter that allows 10 requests per second
	limiter := NewIPRateLimiter(rate.Every(100*time.Millisecond), 10)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := RateLimit(limiter)(handler)

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRateLimit_BlocksExcessRequests(t *testing.T) {
	// Create very strict limiter: 1 request per 10 seconds, burst of 2
	limiter := NewIPRateLimiter(rate.Every(10*time.Second), 2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := RateLimit(limiter)(handler)

	ip := "192.168.1.1:12345"

	// First 2 requests should succeed (burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	// Check Retry-After header
	if w.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	// Create strict limiter
	limiter := NewIPRateLimiter(rate.Every(10*time.Second), 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimit(limiter)(handler)

	// IP 1 makes request (uses burst)
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("IP1 request: Expected status 200, got %d", w1.Code)
	}

	// IP 2 should still be able to make request (independent limiter)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("IP2 request: Expected status 200, got %d", w2.Code)
	}
}

func TestRateLimit_HTMXRequest(t *testing.T) {
	// Create strict limiter
	limiter := NewIPRateLimiter(rate.Every(10*time.Second), 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimit(limiter)(handler)

	ip := "192.168.1.1:12345"

	// First request (uses burst)
	req1 := httptest.NewRequest("POST", "/login", nil)
	req1.RemoteAddr = ip
	req1.Header.Set("HX-Request", "true")
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	// Second request should be rate limited
	req2 := httptest.NewRequest("POST", "/login", nil)
	req2.RemoteAddr = ip
	req2.Header.Set("HX-Request", "true")
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w2.Code)
	}

	// Check HTMX-specific headers
	if w2.Header().Get("HX-Reswap") != "innerHTML" {
		t.Error("Expected HX-Reswap header for HTMX request")
	}

	// Check response body contains alert
	body := w2.Body.String()
	if body == "" {
		t.Error("Expected non-empty response body for HTMX request")
	}
}

func TestAuthRateLimiter(t *testing.T) {
	limiter := AuthRateLimiter()

	if limiter == nil {
		t.Fatal("Expected AuthRateLimiter to return a limiter")
	}

	// Verify it has the expected rate (5 per minute = every 12 seconds)
	if limiter.rate != rate.Every(12*time.Second) {
		t.Errorf("Expected rate of 1 per 12s, got %v", limiter.rate)
	}

	// Verify burst of 10
	if limiter.burst != 10 {
		t.Errorf("Expected burst of 10, got %d", limiter.burst)
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Every(time.Second), 5)

	// Add some limiters
	limiter.GetLimiter("192.168.1.1")
	limiter.GetLimiter("192.168.1.2")

	if len(limiter.limiters) != 2 {
		t.Fatalf("Expected 2 limiters, got %d", len(limiter.limiters))
	}

	// Start cleanup routine with very short interval
	done := make(chan struct{})
	go limiter.StartCleanupRoutine(50*time.Millisecond, done)

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// Stop cleanup
	close(done)

	// Check limiters were cleaned up
	limiter.mu.RLock()
	count := len(limiter.limiters)
	limiter.mu.RUnlock()

	if count != 0 {
		t.Errorf("Expected 0 limiters after cleanup, got %d", count)
	}
}
