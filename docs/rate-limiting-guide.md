# Rate Limiting Guide

## Overview

Rate limiting is implemented to protect authentication endpoints from brute force attacks. The system uses per-IP rate limiting with token bucket algorithm via Go's `golang.org/x/time/rate` package.

## Features

- **Per-IP tracking**: Each IP address has its own rate limiter
- **Token bucket algorithm**: Allows bursts while maintaining average rate
- **Automatic cleanup**: Periodic cleanup of inactive limiters to prevent memory leaks
- **Real IP extraction**: Properly handles proxies and load balancers
- **HTMX support**: Custom error responses for HTMX requests
- **Logging**: Violations are logged with IP, path, and user agent

## Configuration

### Authentication Endpoints

Default settings for login and registration:
- **Rate**: 5 requests per minute (average)
- **Burst**: 10 requests
- **Calculation**: One request every 12 seconds on average

```go
// Located in: gojang/http/middleware/ratelimit.go
func AuthRateLimiter() *IPRateLimiter {
    return NewIPRateLimiter(rate.Every(12*time.Second), 10)
}
```

### How It Works

1. **Initial burst**: A new IP can make up to 10 requests immediately (burst size)
2. **Token regeneration**: After burst is used, tokens regenerate at 1 per 12 seconds
3. **Sustained rate**: Over time, allows ~5 requests per minute

### Example Scenarios

**Scenario 1: Normal User**
- User tries to login 3 times quickly → All allowed (within burst)
- User waits 1 minute → Can try 5 more times

**Scenario 2: Attack Attempt**
- Attacker makes 10 rapid requests → All initially allowed (burst)
- Next request → Rate limited (429 error)
- Must wait ~12 seconds per additional attempt

## IP Address Extraction

The system correctly extracts the real client IP, even behind proxies:

```go
Priority order:
1. X-Forwarded-For (first/leftmost IP)
2. X-Real-IP
3. RemoteAddr
```

### Security Considerations

- ✅ Takes **first** IP from X-Forwarded-For (original client)
- ✅ Validates IP format before using
- ✅ Prevents header spoofing by proper ordering
- ✅ Falls back to RemoteAddr if headers invalid

## Usage

### Applying to Routes

Rate limiting is applied to specific routes using middleware:

```go
authLimiter := middleware.AuthRateLimiter()

// Apply to specific POST endpoints
r.With(middleware.RateLimit(authLimiter)).Post("/login", authHandler.LoginPOST)
r.With(middleware.RateLimit(authLimiter)).Post("/register", authHandler.RegisterPOST)
```

### Starting Cleanup Routine

The cleanup routine prevents memory leaks by periodically removing inactive limiters:

```go
cleanupDone := make(chan struct{})
defer close(cleanupDone)

// Cleanup every 5 minutes
go authLimiter.StartCleanupRoutine(5*time.Minute, cleanupDone)
```

## Custom Rate Limiters

You can create custom rate limiters for different endpoints:

```go
// API rate limiter: 100 requests per minute
apiLimiter := middleware.NewIPRateLimiter(rate.Every(600*time.Millisecond), 20)

// Strict rate limiter: 1 request per minute
strictLimiter := middleware.NewIPRateLimiter(rate.Every(60*time.Second), 1)

// Apply to routes
r.With(middleware.RateLimit(apiLimiter)).Get("/api/data", handler)
```

## Response Behavior

### Standard Requests

When rate limit is exceeded:
- **Status Code**: 429 Too Many Requests
- **Header**: `Retry-After: 60` (seconds)
- **Body**: "Too many requests. Please try again later."

### HTMX Requests

For HTMX-enhanced forms:
- **Status Code**: 429 Too Many Requests
- **Header**: `HX-Reswap: innerHTML`
- **Header**: `Retry-After: 60`
- **Body**: HTML alert div with user-friendly message

```html
<div class="alert alert-error">
    Too many requests. Please wait a moment and try again.
</div>
```

## Logging

Rate limit violations are automatically logged:

```go
[WARN] rate_limit_exceeded [ip 203.0.113.1 method POST path /login user_agent Mozilla/5.0...]
```

Log fields:
- `ip`: Client IP address
- `method`: HTTP method
- `path`: Request path
- `user_agent`: User agent string

## Testing

Comprehensive tests are available in `gojang/http/middleware/ratelimit_test.go`:

```bash
# Run all rate limit tests
go test ./gojang/http/middleware -v -run RateLimit

# Run specific test
go test ./gojang/http/middleware -v -run TestRateLimit_BlocksExcessRequests
```

Test coverage includes:
- ✅ Creating rate limiters
- ✅ Per-IP tracking
- ✅ Request allowing/blocking
- ✅ IP extraction from headers
- ✅ HTMX request handling
- ✅ Cleanup routine
- ✅ Independent IP tracking

## Monitoring

### Checking Rate Limit Violations

Monitor your logs for `rate_limit_exceeded` warnings:

```bash
# Search for rate limit violations
grep "rate_limit_exceeded" logs/*.log

# Count violations per IP
grep "rate_limit_exceeded" logs/*.log | grep -oP 'ip \K[0-9.]+' | sort | uniq -c | sort -nr
```

### Signs of Attack

Watch for:
- Multiple `rate_limit_exceeded` warnings from same IP
- Many different IPs hitting rate limit simultaneously
- Rate limits during off-peak hours

## Production Considerations

### Behind a Reverse Proxy

Ensure your proxy (Nginx, Caddy, etc.) forwards real IP:

**Nginx:**
```nginx
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Real-IP $remote_addr;
```

**Caddy:**
```
reverse_proxy localhost:8080
```
(Caddy automatically sets X-Forwarded-For)

### Adjusting Limits

For stricter security:
```go
// 3 attempts per minute
authLimiter := NewIPRateLimiter(rate.Every(20*time.Second), 5)
```

For more lenient limits:
```go
// 10 attempts per minute
authLimiter := NewIPRateLimiter(rate.Every(6*time.Second), 15)
```

### Whitelisting IPs

To whitelist specific IPs (e.g., monitoring systems):

```go
func RateLimit(limiter *IPRateLimiter, whitelist []string) func(next http.Handler) http.Handler {
    whitelistMap := make(map[string]bool)
    for _, ip := range whitelist {
        whitelistMap[ip] = true
    }
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := getRealIP(r)
            
            // Skip rate limiting for whitelisted IPs
            if whitelistMap[ip] {
                next.ServeHTTP(w, r)
                return
            }
            
            // Normal rate limiting...
        })
    }
}
```

## Troubleshooting

### Issue: Legitimate users being rate limited

**Causes:**
- Office/building sharing single public IP
- VPN exit nodes
- Mobile carrier NAT

**Solutions:**
- Increase burst size
- Increase rate limit
- Implement user-based rate limiting (after authentication)

### Issue: Rate limits not working

**Checks:**
1. Verify middleware is applied to route
2. Check IP extraction in logs
3. Verify cleanup routine is running
4. Test with curl/httpie

### Issue: All requests from same IP

**Causes:**
- Not behind proxy (using RemoteAddr of proxy)
- Proxy not sending X-Forwarded-For
- X-Forwarded-For validation failing

**Solution:**
- Configure proxy to forward real IP
- Check proxy logs
- Add debug logging to `getRealIP()`

## References

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [HTTP 429 Status Code](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/429)
- [X-Forwarded-For Header](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For)
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)

## See Also

- [Security Summary](SECURITY-SUMMARY.md)
- [Authentication & Authorization](authentication-authorization.md)
- [Deployment Guide](deployment-guide.md)
