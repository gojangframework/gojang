package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gojangframework/gojang/gojang/utils"
)

// AuditLogger logs important admin actions for security and compliance
type AuditLogger struct{}

// AuditLog represents a structured audit log entry
type AuditLog struct {
	Timestamp time.Time
	UserID    int
	Username  string
	Action    string
	Resource  string
	IP        string
	UserAgent string
	Success   bool
	Details   string
}

// LogAction logs an admin action
func (a *AuditLogger) LogAction(userID int, username, action, resource, ip, userAgent string, success bool, details string) {
	// Log to standard output (in production, send to logging service or database)
	if success {
		utils.Infow("audit.success",
			"user", username,
			"user_id", userID,
			"action", action,
			"resource", resource,
			"ip", ip,
			"details", details,
		)
	} else {
		utils.Errorw("audit.failure",
			"user", username,
			"user_id", userID,
			"action", action,
			"resource", resource,
			"ip", ip,
			"details", details,
		)
	}
}

// AuditMiddleware logs all requests to admin endpoints
func AuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get user from context
		user := GetUserFromRequest(r)
		var userID int
		var userEmail string
		if user != nil {
			userID = user.ID
			userEmail = user.Email
		}

		// Get real client IP (using shared function from ratelimit.go)
		ip := getIP(r)

		// Log the request
		utils.Infow("admin.access",
			"user", userEmail,
			"user_id", userID,
			"method", r.Method,
			"path", r.URL.Path,
			"ip", ip,
		)

		// Create response writer wrapper to capture status code
		ww := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		// Log completion
		duration := time.Since(start)
		utils.Infow("admin.complete",
			"user", userEmail,
			"user_id", userID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration", duration,
		)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// NewAuditLogger creates a new audit logger instance
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// Helper functions for common audit log actions

// LogUserCreated logs when a user is created by an admin
func LogUserCreated(r *http.Request, targetUserID int, targetUsername string) {
	user := GetUserFromRequest(r)
	if user == nil {
		return
	}

	logger := NewAuditLogger()
	ip := getIP(r)
	logger.LogAction(
		user.ID,
		user.Email,
		"CREATE_USER",
		"users",
		ip,
		r.UserAgent(),
		true,
		"Created user: "+targetUsername,
	)
}

// LogUserUpdated logs when a user is updated
func LogUserUpdated(r *http.Request, targetUserID int, targetUsername string) {
	user := GetUserFromRequest(r)
	if user == nil {
		return
	}

	logger := NewAuditLogger()
	ip := getIP(r)
	logger.LogAction(
		user.ID,
		user.Email,
		"UPDATE_USER",
		"users",
		ip,
		r.UserAgent(),
		true,
		"Updated user: "+targetUsername,
	)
}

// LogUserDeleted logs when a user is deleted
func LogUserDeleted(r *http.Request, targetUserID int, targetUsername string) {
	user := GetUserFromRequest(r)
	if user == nil {
		return
	}

	logger := NewAuditLogger()
	ip := getIP(r)
	logger.LogAction(
		user.ID,
		user.Email,
		"DELETE_USER",
		"users",
		ip,
		r.UserAgent(),
		true,
		"Deleted user: "+targetUsername,
	)
}

// LogPostDeleted logs when a post is deleted by an admin
func LogPostDeleted(r *http.Request, postID int, postTitle string) {
	user := GetUserFromRequest(r)
	if user == nil {
		return
	}

	logger := NewAuditLogger()
	ip := getIP(r)
	logger.LogAction(
		user.ID,
		user.Email,
		"DELETE_POST",
		"posts",
		ip,
		r.UserAgent(),
		true,
		"Deleted post: "+postTitle,
	)
}

// LogPermissionDenied logs when a user is denied access
func LogPermissionDenied(r *http.Request, action, resource string) {
	user := GetUserFromRequest(r)
	if user == nil {
		return
	}

	logger := NewAuditLogger()
	ip := getIP(r)
	logger.LogAction(
		user.ID,
		user.Email,
		action,
		resource,
		ip,
		r.UserAgent(),
		false,
		"Permission denied",
	)
}

// getIP extracts the real client IP from the request
// It properly handles X-Forwarded-For by taking the first (leftmost) IP
func getIP(r *http.Request) string {
	// This uses the same logic as getRealIP in ratelimit.go
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
