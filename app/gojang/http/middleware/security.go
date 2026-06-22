package middleware

import (
	"net/http"
	"strings"

	"github.com/gojangframework/gojang/app/gojang/config"
)

// EnforceHTTPS redirects HTTP requests to HTTPS in production
func EnforceHTTPS(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only enforce HTTPS in production (when Debug is false)
			if !cfg.Debug && r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
				// Construct HTTPS URL
				httpsURL := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	return SecurityHeadersWithOptions(SecurityOptionsFromConfig(cfg))
}

// SecurityOptions configures security-related HTTP headers.
type SecurityOptions struct {
	Debug                        bool
	DefaultSrc                   []string
	ScriptSrc                    []string
	StyleSrc                     []string
	ImgSrc                       []string
	FontSrc                      []string
	ConnectSrc                   []string
	FrameAncestors               []string
	SameOriginFrameAncestorPaths []string
}

// SecurityOptionsFromConfig converts application config into middleware options.
func SecurityOptionsFromConfig(cfg *config.Config) SecurityOptions {
	if cfg == nil {
		return defaultSecurityOptions()
	}

	opts := defaultSecurityOptions()
	opts.Debug = cfg.Debug
	opts.DefaultSrc = overrideOrDefault(cfg.CSPDefaultSrc, opts.DefaultSrc)
	opts.ScriptSrc = overrideOrDefault(cfg.CSPScriptSrc, opts.ScriptSrc)
	opts.StyleSrc = overrideOrDefault(cfg.CSPStyleSrc, opts.StyleSrc)
	opts.ImgSrc = overrideOrDefault(cfg.CSPImgSrc, opts.ImgSrc)
	opts.FontSrc = overrideOrDefault(cfg.CSPFontSrc, opts.FontSrc)
	opts.ConnectSrc = overrideOrDefault(cfg.CSPConnectSrc, opts.ConnectSrc)
	opts.FrameAncestors = overrideOrDefault(cfg.CSPFrameAncestors, opts.FrameAncestors)
	opts.SameOriginFrameAncestorPaths = append([]string(nil), cfg.CSPSameOriginFrameAncestors...)
	return opts
}

// SecurityHeadersWithOptions adds security-related HTTP headers.
func SecurityHeadersWithOptions(opts SecurityOptions) func(http.Handler) http.Handler {
	opts = normalizeSecurityOptions(opts)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			frameAncestors := opts.FrameAncestors
			frameOptions := "DENY"
			if pathHasPrefix(r.URL.Path, opts.SameOriginFrameAncestorPaths) {
				frameAncestors = []string{"'self'"}
				frameOptions = "SAMEORIGIN"
			}

			w.Header().Set("Content-Security-Policy", buildCSP(opts, frameAncestors))
			w.Header().Set("X-Frame-Options", frameOptions)

			// X-Content-Type-Options
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Referrer-Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions-Policy
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Strict-Transport-Security (HSTS) - only in production
			if !opts.Debug {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func defaultSecurityOptions() SecurityOptions {
	return SecurityOptions{
		DefaultSrc:     []string{"'self'"},
		ScriptSrc:      []string{"'self'", "'unsafe-inline'", "https://unpkg.com"},
		StyleSrc:       []string{"'self'", "'unsafe-inline'"},
		ImgSrc:         []string{"'self'", "data:", "https:"},
		FontSrc:        []string{"'self'"},
		ConnectSrc:     []string{"'self'"},
		FrameAncestors: []string{"'none'"},
	}
}

func normalizeSecurityOptions(opts SecurityOptions) SecurityOptions {
	defaults := defaultSecurityOptions()
	opts.DefaultSrc = overrideOrDefault(opts.DefaultSrc, defaults.DefaultSrc)
	opts.ScriptSrc = overrideOrDefault(opts.ScriptSrc, defaults.ScriptSrc)
	opts.StyleSrc = overrideOrDefault(opts.StyleSrc, defaults.StyleSrc)
	opts.ImgSrc = overrideOrDefault(opts.ImgSrc, defaults.ImgSrc)
	opts.FontSrc = overrideOrDefault(opts.FontSrc, defaults.FontSrc)
	opts.ConnectSrc = overrideOrDefault(opts.ConnectSrc, defaults.ConnectSrc)
	opts.FrameAncestors = overrideOrDefault(opts.FrameAncestors, defaults.FrameAncestors)
	return opts
}

func overrideOrDefault(values, defaults []string) []string {
	cleaned := cleanDirectiveValues(values)
	if len(cleaned) == 0 {
		return append([]string(nil), defaults...)
	}
	return cleaned
}

func cleanDirectiveValues(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func buildCSP(opts SecurityOptions, frameAncestors []string) string {
	directives := []string{
		"default-src " + strings.Join(opts.DefaultSrc, " "),
		"script-src " + strings.Join(opts.ScriptSrc, " "),
		"style-src " + strings.Join(opts.StyleSrc, " "),
		"img-src " + strings.Join(opts.ImgSrc, " "),
		"font-src " + strings.Join(opts.FontSrc, " "),
		"connect-src " + strings.Join(opts.ConnectSrc, " "),
		"frame-ancestors " + strings.Join(cleanDirectiveValues(frameAncestors), " "),
	}

	return strings.Join(directives, "; ") + ";"
}

func pathHasPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
