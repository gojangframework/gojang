# Logging Guide

This guide covers the structured logging system in Gojang, best practices, and common patterns.

## Overview

Gojang uses **Zap** for high-performance structured logging with environment-based configuration:

- 🔍 **Development Mode** - Human-readable console output with colors
- 🚀 **Production Mode** - Structured JSON output for log aggregation
- 📊 **Multiple Levels** - DEBUG, INFO, WARN, ERROR
- 🔧 **Zero Allocation** - High-performance structured logging
- 🎯 **Contextual** - Key-value pairs for better observability

**Benefits:**
- ✅ Fast and efficient (zero allocation)
- ✅ Structured output (JSON in production)
- ✅ Environment-aware (auto-configures based on ENV)
- ✅ Easy to parse and query
- ✅ Compatible with log aggregation tools

---

## Quick Start

### 1. Initialization

The logger is initialized automatically in `cmd/web/main.go`:

```go
package main

import (
    "github.com/gojangframework/gojang/gojang/utils"
)

func main() {
    cfg := config.MustLoad()
    
    // Initialize logger (auto-detects environment)
    lvl := ""
    if cfg.Debug {
        lvl = "debug"
    }
    if err := utils.Init(lvl); err != nil {
        log.Fatalf("failed to initialize logger: %v", err)
    }
    
    // Logger is now ready to use throughout your application
    utils.Infof("Server starting on port %s", cfg.Port)
}
```

### 2. Environment Configuration

Set the log level via environment variables:

```bash
# .env file
LOG_LEVEL=info      # Options: debug, info, warn, error
ENV=production      # Options: development, production
DEBUG=false         # true enables debug mode
```

**Priority:** `LOG_LEVEL` → `DEBUG` flag → `ENV` setting → default (info)

---

## Logging Methods

### Formatted Logging (Printf-style)

Use for simple messages with string formatting:

```go
import "github.com/gojangframework/gojang/gojang/utils"

// Debug - Development-only verbose output
utils.Debugf("Processing request for user %d", userID)

// Info - General informational messages
utils.Infof("Server started on port %s", port)

// Warn - Warning conditions that should be reviewed
utils.Warnf("Rate limit approaching: %d/%d requests", current, max)

// Error - Error conditions that need attention
utils.Errorf("Failed to connect to database: %v", err)
```

### Structured Logging (Key-Value pairs)

Use for production logging with rich context:

```go
// Infow - Structured info logging
utils.Infow("user.login",
    "user_id", user.ID,
    "email", user.Email,
    "ip", req.RemoteAddr,
)

// Debugw - Structured debug logging
utils.Debugw("query.executed",
    "query", "SELECT * FROM users",
    "duration_ms", duration.Milliseconds(),
    "rows", count,
)

// Warnw - Structured warning logging
utils.Warnw("cache.miss",
    "key", cacheKey,
    "fallback", "database",
)

// Errorw - Structured error logging
utils.Errorw("database.query_failed",
    "query", "INSERT INTO posts",
    "error", err,
    "user_id", userID,
)
```

**Output in production (JSON):**
```json
{
  "level": "error",
  "ts": "2025-10-09T10:30:45.123Z",
  "msg": "database.query_failed",
  "query": "INSERT INTO posts",
  "error": "connection timeout",
  "user_id": 42
}
```

---

## Best Practices

### ✅ DO: Log Meaningful Events

```go
// Good: Logs important business events
utils.Infow("order.created",
    "order_id", order.ID,
    "user_id", user.ID,
    "total", order.Total,
)

// Good: Logs errors with context
utils.Errorw("payment.failed",
    "order_id", order.ID,
    "payment_method", method,
    "error", err,
)

// Good: Logs security events
utils.Warnw("auth.failed_login",
    "email", email,
    "ip", req.RemoteAddr,
    "attempts", attempts,
)
```

### ❌ DON'T: Over-Log or Under-Log

```go
// Bad: Logging expected behavior
utils.Infof("Rendering template %s", templateName)  // Too noisy

// Bad: Logging before error handling
utils.Errorw("user.not_found", "id", id)
return errors.New("user not found")  // Redundant

// Bad: Not logging critical errors
if err := db.Save(user); err != nil {
    return err  // Missing context!
}

// Good: Log once with full context
if err := db.Save(user); err != nil {
    utils.Errorw("user.save_failed",
        "user_id", user.ID,
        "operation", "update_profile",
        "error", err,
    )
    return err
}
```

### 🎯 When to Log What

**DEBUG** - Development-only verbose information
```go
utils.Debugf("Cache hit for key: %s", key)
utils.Debugw("template.rendered", "name", tmplName, "duration_ms", ms)
```

**INFO** - Important business events and milestones
```go
utils.Infof("Server started on port %s", port)
utils.Infow("user.registered", "user_id", id, "email", email)
utils.Infow("order.completed", "order_id", id, "total", total)
```

**WARN** - Unexpected situations that don't prevent operation
```go
utils.Warnf("Rate limit reached for IP: %s", ip)
utils.Warnw("cache.slow", "operation", "get", "duration_ms", 500)
utils.Warnw("user.inactive", "user_id", id, "last_login", lastLogin)
```

**ERROR** - Failures that require investigation
```go
utils.Errorf("Database connection failed: %v", err)
utils.Errorw("api.request_failed", "endpoint", url, "error", err)
utils.Errorw("payment.charge_failed", "amount", amount, "error", err)
```

---

## Common Patterns

### 1. Handler Logging

```go
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Log the start of important operations (DEBUG level)
    utils.Debugw("handler.user_create", "method", r.Method, "path", r.URL.Path)
    
    // ... parse form ...
    
    // Log business logic failures
    if err := h.Client.User.Create().Save(r.Context()); err != nil {
        utils.Errorw("user.create_failed",
            "email", form.Email,
            "error", err,
        )
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create user")
        return
    }
    
    // Log successful operations (INFO level)
    utils.Infow("user.created",
        "user_id", user.ID,
        "email", user.Email,
    )
}
```

### 2. Audit Logging

```go
func AuditMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := middleware.GetUser(r.Context())
        
        // Log admin actions
        if strings.HasPrefix(r.URL.Path, "/admin/") {
            utils.Infow("admin.access",
                "user_id", user.ID,
                "email", user.Email,
                "method", r.Method,
                "path", r.URL.Path,
                "ip", r.RemoteAddr,
            )
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### 3. Database Operations

```go
func (s *PostService) GetAll(ctx context.Context) ([]*models.Post, error) {
    posts, err := s.client.Post.Query().All(ctx)
    if err != nil {
        // Log query failures with context
        utils.Errorw("database.query_failed",
            "model", "post",
            "operation", "query_all",
            "error", err,
        )
        return nil, err
    }
    
    return posts, nil
}
```

### 4. Background Jobs

```go
func (w *EmailWorker) ProcessQueue(ctx context.Context) {
    utils.Infof("Email worker started")
    
    for {
        select {
        case <-ctx.Done():
            utils.Infof("Email worker stopped")
            return
        default:
            if err := w.processNextEmail(); err != nil {
                utils.Errorw("email.send_failed",
                    "error", err,
                    "retry_count", retries,
                )
            }
        }
    }
}
```

---

## Anti-Patterns to Avoid

### ❌ Don't Log Sensitive Data

```go
// BAD - Exposes sensitive information
utils.Infow("user.login", "password", password)  // Never!
utils.Errorw("payment.failed", "card_number", cardNum)  // Never!

// GOOD - Log safely
utils.Infow("user.login", "user_id", userID)
utils.Errorw("payment.failed", "last_4_digits", last4)
```

### ❌ Don't Log in Hot Paths

```go
// BAD - Creates log spam in production
for _, item := range items {
    utils.Debugf("Processing item: %v", item)  // Avoid loops!
}

// GOOD - Log aggregated results
utils.Debugf("Processed %d items", len(items))
```

### ❌ Don't Mix Logging Libraries

```go
// BAD - Inconsistent logging
log.Printf("User created")  // stdlib log
utils.Infof("Order placed")  // utils logger
fmt.Println("Payment received")  // fmt

// GOOD - Use utils logger consistently
utils.Infof("User created")
utils.Infof("Order placed")
utils.Infof("Payment received")
```

---

## Testing with Logs

Logs are automatically suppressed during tests. To enable logging in tests:

```go
func TestUserCreate(t *testing.T) {
    // Initialize logger for test
    if err := utils.Init("debug"); err != nil {
        t.Fatal(err)
    }
    
    // Your test code...
}
```

---

## Production Considerations

### Log Aggregation

Gojang's JSON output works seamlessly with:
- **Elasticsearch + Kibana** (ELK Stack)
- **Grafana Loki**
- **Datadog**
- **CloudWatch Logs**
- **Splunk**

### Example: Querying Structured Logs

```bash
# Find all failed logins in the last hour
jq 'select(.msg == "auth.failed_login" and .level == "warn")' logs.json

# Find errors for specific user
jq 'select(.user_id == 42 and .level == "error")' logs.json

# Calculate average query duration
jq -s '[.[] | select(.msg == "query.executed")] | map(.duration_ms) | add/length' logs.json
```

### Performance

Zap's structured logging is extremely efficient:
- Zero allocation logging in hot paths
- Faster than stdlib `log` package
- JSON encoding is optimized
- Suitable for high-throughput applications

---

## Summary

**Golden Rules:**
1. ✅ Use structured logging (`Infow`, `Errorw`) in production code
2. ✅ Log business events, not framework behavior
3. ✅ Include context (user_id, request_id, etc.)
4. ✅ Use appropriate log levels
5. ❌ Don't log sensitive data (passwords, tokens, PII)
6. ❌ Don't over-log in hot paths
7. ❌ Don't log before returning errors (pick one)

**Quick Reference:**
- `utils.Debugf()` / `utils.Debugw()` - Development verbose output
- `utils.Infof()` / `utils.Infow()` - Business events
- `utils.Warnf()` / `utils.Warnw()` - Unexpected conditions
- `utils.Errorf()` / `utils.Errorw()` - Errors requiring attention

---

## Next Steps

- Read [Testing Best Practices](./testing-best-practices.md) to learn about testing
- Read [Deployment Guide](./deployment-guide.md) for production setup
- Check [Authentication & Authorization](./authentication-authorization.md) for security patterns

**Happy logging! 📝**
