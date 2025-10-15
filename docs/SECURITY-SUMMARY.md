# Security Features

**Gojang Framework Security Implementation Summary**

This document outlines the security features currently implemented in the Gojang framework.

---

## 🔐 Authentication & Password Security

### Password Hashing
- **Argon2id algorithm** - Industry-standard password hashing (superior to bcrypt)
- Parameters: 64MB memory, 3 iterations, 2 parallelism, 16-byte salt, 32-byte key
- Location: `gojang/utils/password.go`

### Features
- ✅ Constant-time password comparison (prevents timing attacks)
- ✅ Generic error messages (prevents user enumeration)
- ✅ Password field marked as sensitive in database schema
- ✅ Comprehensive test coverage

---

## 🎫 Session Management

### Configuration
- **HttpOnly cookies** - Prevents XSS attacks from stealing session tokens
- **Secure flag in production** - Ensures cookies only sent over HTTPS
- **SameSite: Lax** - CSRF protection for navigation requests
- **Idle timeout: 30 minutes** - Auto-logout after inactivity
- **Session lifetime: 12 hours** (configurable)
- Location: `gojang/http/middleware/session.go`

### Features
- ✅ Session token renewal after login (prevents session fixation)
- ✅ Session destruction on logout
- ✅ User active status validation on every request
- ✅ Session data cleared on inactive/deleted accounts

---

## 🛡️ CSRF Protection

### Implementation
- **Library:** github.com/justinas/nosurf
- **Coverage:** All authentication, post, user, and admin routes
- **Method:** Double-submit cookie pattern

### Protected Routes
- ✅ Login and registration forms
- ✅ All POST/PUT/DELETE requests
- ✅ Admin panel operations
- ✅ User management endpoints

### Locations
- `gojang/cmd/web/main.go:109` (auth routes)
- `gojang/http/routes/posts.go:14` (post routes)
- `gojang/http/routes/users.go:14` (user routes)
- `gojang/admin/admin_routes.go:13` (admin routes)

---

## ⏱️ Rate Limiting

### Features
- **Per-IP rate limiting** - Prevents brute force attacks
- **Authentication endpoints:** 5 requests per minute, burst of 10
- **Proper IP extraction** - Handles X-Forwarded-For securely
- **Memory cleanup** - Periodic cleanup of inactive limiters

### Implementation
- Location: `gojang/http/middleware/ratelimit.go`
- Applied to: Login and registration endpoints
- IP validation: Extracts real client IP from X-Forwarded-For (first/leftmost IP)

---

## 🔒 Security Headers

### Headers Configured

1. **Content-Security-Policy (CSP)**
   - `default-src 'self'` - Only load resources from same origin
   - `script-src 'self' 'unsafe-inline' https://unpkg.com` - Script sources
   - `style-src 'self' 'unsafe-inline'` - Style sources
   - `img-src 'self' data: https:` - Image sources
   - `frame-ancestors 'none'` - Clickjacking protection

2. **X-Frame-Options: DENY**
   - Prevents clickjacking attacks

3. **X-Content-Type-Options: nosniff**
   - Prevents MIME type sniffing attacks

4. **Referrer-Policy: strict-origin-when-cross-origin**
   - Controls referrer information leakage

5. **Permissions-Policy**
   - Restricts geolocation, microphone, and camera access

6. **Strict-Transport-Security (HSTS)**
   - Enforces HTTPS connections (production only)
   - `max-age=31536000; includeSubDomains`

### Location
- `gojang/http/middleware/security.go`

---

## 🌐 HTTPS Enforcement

### Features
- **Automatic HTTPS redirect** in production
- **X-Forwarded-Proto support** - Works with reverse proxies
- **Debug mode bypass** - Development remains on HTTP

### Implementation
- Location: `gojang/http/middleware/security.go` (EnforceHTTPS)
- Applied: Globally before all other middleware

---

## ✅ Input Validation

### Server-Side Validation
- **Library:** github.com/go-playground/validator/v10
- **Location:** `gojang/views/forms/forms.go`

### Validated Fields
- ✅ Email format validation
- ✅ Password minimum length (8 characters)
- ✅ Required field validation
- ✅ Password confirmation matching
- ✅ Field length limits

### Features
- User-friendly error messages
- Form-specific validation structs
- Type-safe validation

---

## 💾 Database Security

### ORM Protection
- **Ent ORM** - Prevents SQL injection through parameterized queries
- **Location:** `gojang/models/`
- **No raw SQL queries** - All database access through ORM

### Features
- ✅ Automatic query parameterization
- ✅ Type-safe database operations
- ✅ Foreign key constraints enabled
- ✅ Schema migrations managed automatically

---

## 👤 Authorization & Access Control

### Role-Based Access Control (RBAC)
- **Roles:** regular user, staff, superuser
- **Fields:** `is_active`, `is_staff`, `is_superuser`
- **Location:** `gojang/models/schema/user.go`

### Middleware Protection
1. **RequireAuth** - Ensures user is authenticated
2. **RequireStaff** - Ensures user has staff role
3. **LoadUser** - Loads user from session (optional auth)

### Features
- ✅ Ownership checks for resource access
- ✅ Admin action audit logging
- ✅ Active user validation
- ✅ Graceful handling of deleted/inactive users

---

## 📊 Audit Logging

### Features
- **Structured logging** - Using Zap logger (JSON output)
- **Admin actions tracked** - All admin panel operations logged
- **IP address logging** - Real client IP (properly extracted)
- **Request/response tracking** - Duration, status codes, user info

### Logged Information
- User ID and email
- Action performed
- Resource accessed
- Client IP address
- Timestamp and duration
- Response status code

### Location
- `gojang/http/middleware/audit.go`

---

## 🔍 Security Disclosure

### security.txt
- **Location:** `/.well-known/security.txt`
- **Contact:** security@gojangframework.org
- **Response time:** Within 48 hours
- **Safe harbor policy** - Supports responsible disclosure

---

## 🧪 Security Testing & CI/CD

### Automated Scanning
1. **gosec** - Go security checker
   - Scans for common security issues
   - Runs on every pull request

2. **govulncheck** - Vulnerability scanner
   - Checks for known vulnerabilities in dependencies
   - Runs on every pull request

### CI/CD Integration
- Location: `.github/workflows/test.yml`
- Runs automatically on pull requests
- Reports findings in GitHub Actions

---

## 📝 Configuration Security

### Environment Variables
- **Required fields validated** - `DATABASE_URL`, `SESSION_KEY`
- **Debug mode warning** - Logs warning when DEBUG=true
- **No default secrets** - `.env.example` requires manual secret generation

### Secret Generation
```bash
# Generate SESSION_KEY
openssl rand -base64 32
```

### Features
- ✅ Secrets never committed to repository
- ✅ `.env` file gitignored
- ✅ `.env.example` provides template (no actual secrets)

---

## 🔧 Error Handling

### Security Features
- **Generic error messages** - Don't expose internal details
- **Structured logging** - Sensitive data never logged
- **Custom error pages** - 404 handler configured

### Features
- ✅ Stack traces hidden in production
- ✅ Database errors sanitized
- ✅ Authentication failures return generic messages

---

## 📚 Documentation

### Security Documentation
- [Authentication & Authorization Guide](./authentication-authorization.md)
- [Deployment Guide](./deployment-guide.md) - Production security checklist
- [Logging Guide](./logging-guide.md) - Secure logging practices

---

## 🎯 Production Deployment Checklist

Essential security items for production:

- [ ] Set `DEBUG=false` in environment
- [ ] Generate secure random `SESSION_KEY`
- [ ] Enable PostgreSQL SSL (`sslmode=require`)
- [ ] Configure HTTPS with valid SSL certificate
- [ ] Set `ALLOWED_HOSTS` to production domains
- [ ] Review and test all security headers
- [ ] Set up centralized logging
- [ ] Configure monitoring and alerting
- [ ] Test rate limiting is working
- [ ] Verify HTTPS redirect is active
- [ ] Review audit logs are being captured
- [ ] Set up regular security scanning
- [ ] Document incident response procedures

---

## 🔄 Regular Maintenance

### Recommended Practices
- **Dependency updates:** Review monthly
- **Security patches:** Apply immediately when available
- **Penetration testing:** Annually or after major changes
- **Code reviews:** Include security checklist
- **Monitoring:** Review audit logs regularly
- **Backups:** Test restoration procedures quarterly

---

**Last Updated:** 2025-10-14  
**Framework Version:** Gojang v1.0
