---
name: gojang-auth-security
description: Apply or review authentication, authorization, sessions, password handling, CSRF, rate limiting, audit logging, and security middleware in a Gojang app. Use when protecting routes, checking resource ownership, changing login/register/logout behavior, or preparing security-sensitive code.
---

# Gojang Auth Security

Use this skill for authentication, authorization, and security-sensitive work.

## Workflow

1. Read `docs/authentication-authorization.md` for auth flows and `docs/SECURITY-SUMMARY.md` for implemented security features.
2. Inspect the relevant middleware and handlers before editing:
   - `app/gojang/http/middleware/auth.go`
   - `app/gojang/http/middleware/permissions.go`
   - `app/gojang/http/middleware/session.go`
   - `app/gojang/http/middleware/security.go`
   - `app/gojang/http/middleware/ratelimit.go`
   - `app/gojang/http/handlers/auth.go`
3. Protect private route groups with `middleware.RequireAuth(sm, client)`.
4. Protect staff-only app routes with both `RequireAuth` and `RequireStaff`.
5. Use `middleware.GetUser(r.Context())` to read the authenticated user and `middleware.OwnsResource(r, ownerID)` for ownership checks.
6. Run relevant middleware tests, then all tests:
   - `go test ./app/gojang/http/middleware`
   - `go test ./...`

## Security Rules

- Do not store or log plaintext passwords, password hashes, session IDs, CSRF tokens, or SMTP secrets.
- Use `utils.HashPassword`, `utils.CheckPassword`, and `utils.ValidatePasswordComplexity` instead of custom password logic.
- Keep logout as POST-only.
- Use generic login failure messages to avoid user enumeration.
- Renew session tokens after login or registration.
- Preserve CSRF protection from `nosurf.NewPure` and include CSRF tokens in forms.
- For HTMX auth failures, prefer the existing middleware behavior such as `HX-Redirect`.
- Keep admin mounted under `/admin`; admin routes must remain staff-only and audited.

## Common Route Patterns

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sm, client))
    r.Get("/dashboard", handler.Dashboard)
})
```

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sm, client))
    r.Use(middleware.RequireStaff)
    r.Get("/staff/reports", handler.Reports)
})
```

## Useful References

- Auth guide: `docs/authentication-authorization.md`
- Security summary: `docs/SECURITY-SUMMARY.md`
- Password utilities: `app/gojang/utils/password.go`
- Existing protected feature: `app/posts/posts.route.go`
