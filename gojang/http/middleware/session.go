package middleware

import (
	"time"

	"github.com/gojangframework/gojang/gojang/config"

	"github.com/alexedwards/scs/v2"
)

// NewSessionManager creates a configured session manager
func NewSessionManager(cfg *config.Config) *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Lifetime = cfg.SessionLifetime
	sessionManager.Cookie.Name = "session_id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = !cfg.Debug // true in production
	sessionManager.Cookie.SameSite = 2        // Lax mode (allows navigation)
	sessionManager.Cookie.Path = "/"          // Cookie available for entire site
	sessionManager.IdleTimeout = 30 * time.Minute

	return sessionManager
}
