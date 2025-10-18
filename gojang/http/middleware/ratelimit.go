package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gojangframework/gojang/gojang/utils"
	"golang.org/x/time/rate"
)

// IPRateLimiter tracks rate limiters per IP address
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       *sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new IP-based rate limiter
// r is requests per second, b is burst size
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		mu:       &sync.RWMutex{},
		rate:     r,
		burst:    b,
	}
}

// GetLimiter returns the rate limiter for a given IP address
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[ip] = limiter
	}

	return limiter
}

// CleanupOldLimiters removes inactive rate limiters (call periodically)
func (i *IPRateLimiter) CleanupOldLimiters() {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Clear all limiters (they'll be recreated as needed)
	i.limiters = make(map[string]*rate.Limiter)
}

// getRealIP extracts the real client IP from the request
// It properly handles X-Forwarded-For by taking the first (leftmost) IP
func getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header (standard for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// Take the first one (leftmost) which is the original client IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			// Validate it's a proper IP
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
	}

	// Fallback to X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Strip port from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, return as-is (might be just IP without port)
		return r.RemoteAddr
	}
	return ip
}

// logRateLimitViolation logs when a rate limit is exceeded
func logRateLimitViolation(r *http.Request, ip string) {
	utils.Warnw("rate_limit_exceeded",
		"ip", ip,
		"method", r.Method,
		"path", r.URL.Path,
		"user_agent", r.UserAgent(),
	)
}

// RateLimit middleware applies rate limiting per IP address
func RateLimit(limiter *IPRateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get real client IP
			ip := getRealIP(r)

			limiterForIP := limiter.GetLimiter(ip)
			if !limiterForIP.Allow() {
				// Log rate limit violation
				logRateLimitViolation(r, ip)

				// Set retry-after header (suggest waiting 60 seconds)
				w.Header().Set("Retry-After", "60")

				// Check if it's an HTMX request
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Reswap", "innerHTML")
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte(`<div class="alert alert-error">Too many requests. Please wait a moment and try again.</div>`))
					return
				}

				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// StartCleanupRoutine starts a background goroutine to cleanup old limiters
func (i *IPRateLimiter) StartCleanupRoutine(interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			i.CleanupOldLimiters()
		case <-done:
			return
		}
	}
}

// AuthRateLimiter creates a rate limiter specifically for auth endpoints
// Allows 5 requests per minute with burst of 10
func AuthRateLimiter() *IPRateLimiter {
	return NewIPRateLimiter(rate.Every(12*time.Second), 10) // 5 req/min average, 10 burst
}
