---
name: gojang-auth-security
description: Apply or review authentication, authorization, sessions, email verification, password reset, password handling, CSRF, rate limiting, audit logging, and security middleware in a Gojang app. Use when protecting routes, checking resource ownership, changing login/register/logout/forgot-password/verify-email behavior, or preparing security-sensitive code.
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
3. Inspect auth email support when changing verification or reset flows:
   - `app/gojang/utils/email.go`
   - `app/views/templates/auth/`
   - `app/views/forms/forms.go`
   - `app/schema/user.go`
4. Protect private route groups with `middleware.RequireAuth(sm, client)`.
5. Protect staff-only app routes with both `RequireAuth` and `RequireStaff`.
6. Use `middleware.GetUser(r.Context())` to read the authenticated user and `middleware.OwnsResource(r, ownerID)` for ownership checks.
7. Run relevant handler and middleware tests, then all tests:
   - `go test ./app/gojang/http/handlers`
   - `go test ./app/gojang/http/middleware`
   - `go test ./...`

## Security Rules

- Do not store or log plaintext passwords, password hashes, session IDs, CSRF tokens, email verification tokens, password reset tokens, SMTP secrets, or AWS secrets.
- Use `utils.HashPassword`, `utils.CheckPassword`, and `utils.ValidatePasswordComplexity` instead of custom password logic.
- Keep logout as POST-only.
- Use generic login failure messages to avoid user enumeration.
- Use generic forgot-password success responses; only active matching users should receive reset email.
- Renew session tokens after login or registration.
- Renew session tokens after successful email verification.
- Preserve CSRF protection from `nosurf.NewPure` and include CSRF tokens in forms.
- For HTMX auth failures, prefer the existing middleware behavior such as `HX-Redirect`.
- Keep admin mounted under `/admin`; admin routes must remain staff-only and audited.

## Email Verification And Reset

- Keep new registrations unverified, create the session, send or queue the verification email, then redirect to `/register-verify-email`.
- Let unverified login create a session and redirect to `/register-verify-email`; do not auto-send a new verification email on every login.
- Keep `RequireAuth` redirecting unverified users to `/register-verify-email`, except `/register-verify-email`, `/register-send-verification-email`, `/verify-email`, and `/logout`.
- Reuse an active verification token on resend; expired verification links may request a new token without an active session.
- Mark email verified and clear verification token fields after a valid `/verify-email` token.
- Reset password only through a valid, unexpired reset token. On success, update the password hash, clear reset token fields, mark email verified, and clear verification token fields.
- Build auth links from `APP_BASE_URL`; expose generated links in rendered pages only in debug mode.
- Prefer Amazon SES when `AWS_SES_ACCESS_KEY_ID`, `AWS_SES_SECRET_ACCESS_KEY`, and `AWS_SES_FROM_EMAIL_ADDRESS` are configured; fall back to SMTP when SES is incomplete and `SMTP_HOST` is configured.

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
- Email service: `app/gojang/utils/email.go`
- Existing protected feature: `app/posts/posts.route.go`
