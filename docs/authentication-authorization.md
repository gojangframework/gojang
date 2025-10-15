# Authentication & Authorization Deep Dive

This comprehensive guide covers authentication and authorization in Gojang, from basic login flows to advanced permission systems.

## Overview

Gojang includes a complete authentication and authorization system with:
- 🔐 Session-based authentication
- 👥 User management
- 🔒 Password hashing with bcrypt
- 🛡️ CSRF protection
- 🎫 Token-based sessions
- 🚪 Middleware-based access control
- 👮 Role-based permissions

**Key Components:**
- `gojang/http/handlers/auth.go` - Authentication handlers
- `gojang/http/middleware/auth.go` - Authentication middleware
- `gojang/http/security/password.go` - Password hashing utilities
- `gojang/models/schema/user.go` - User model schema

---

## Authentication System

### User Model

The User model includes all necessary fields for authentication:

```go
// gojang/models/schema/user.go
type User struct {
    ent.Schema
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        field.String("email").
            Unique().
            NotEmpty(),
        
        field.String("password_hash").
            Sensitive(),  // Not exposed in API responses
        
        field.Bool("is_active").
            Default(true),
        
        field.Bool("is_staff").
            Default(false),
        
        field.Time("last_login").
            Optional(),
        
        field.Time("created_at").
            Default(time.Now).
            Immutable(),
    }
}
```

**Field Explanations:**
- `email` - Unique identifier for login
- `password_hash` - Bcrypt hashed password (never store plaintext!)
- `is_active` - Allows disabling accounts without deletion
- `is_staff` - Admin/staff permission flag
- `last_login` - Track user activity
- `created_at` - Account creation timestamp

---

## Password Security

### Hashing Passwords

Gojang uses bcrypt for password hashing with appropriate cost factor:

```go
import "github.com/gojangframework/gojang/gojang/http/security"

// Hash a password
hash, err := security.HashPassword("user-password")
if err != nil {
    // Handle error
}

// Store hash in database
user := client.User.Create().
    SetEmail("user@example.com").
    SetPasswordHash(hash).
    Save(ctx)
```

**Implementation details:**
```go
// gojang/http/security/password.go
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword(
        []byte(password), 
        bcrypt.DefaultCost,  // Cost factor 10
    )
    return string(bytes), err
}
```

### Verifying Passwords

```go
// Check if password matches hash
match, err := security.CheckPassword(user.PasswordHash, "user-password")
if err != nil {
    // Handle error
}

if !match {
    // Invalid password
    return errors.New("invalid credentials")
}

// Password is valid, proceed with login
```

**Implementation:**
```go
func CheckPassword(hash, password string) (bool, error) {
    err := bcrypt.CompareHashAndPassword(
        []byte(hash), 
        []byte(password),
    )
    if err != nil {
        if err == bcrypt.ErrMismatchedHashAndPassword {
            return false, nil  // Wrong password
        }
        return false, err  // Other error
    }
    return true, nil  // Password matches
}
```

**Security Best Practices:**
- ✅ Never store plaintext passwords
- ✅ Use bcrypt with cost factor 10-12
- ✅ Mark password_hash field as `Sensitive()` in schema
- ✅ Never log or expose password hashes
- ✅ Always use constant-time comparison (bcrypt handles this)

---

## Session Management

### Session Configuration

Sessions are managed using [alexedwards/scs](https://github.com/alexedwards/scs):

```go
// gojang/http/middleware/session.go
func NewSessionManager(cfg *config.Config) *scs.SessionManager {
    sessionManager := scs.New()
    sessionManager.Lifetime = cfg.SessionLifetime  // Default: 24 hours
    sessionManager.Cookie.Name = "session_id"
    sessionManager.Cookie.HttpOnly = true          // Prevent XSS access
    sessionManager.Cookie.Secure = !cfg.Debug      // HTTPS only in production
    sessionManager.Cookie.SameSite = 2             // Lax mode
    sessionManager.Cookie.Path = "/"
    sessionManager.IdleTimeout = 30 * time.Minute  // Auto-logout after inactivity
    
    return sessionManager
}
```

**Cookie Settings Explained:**
- `HttpOnly: true` - Prevents JavaScript access (XSS protection)
- `Secure: true` - Only send over HTTPS in production
- `SameSite: Lax` - CSRF protection while allowing navigation
- `IdleTimeout` - Automatic logout after inactivity

### Storing Session Data

```go
// Store user ID in session after successful login
sessionManager.Put(ctx, "user_id", userID)

// Renew token to prevent session fixation attacks
sessionManager.RenewToken(ctx)

// Retrieve user ID from session
userID := sessionManager.GetInt(ctx, "user_id")

// Remove session data (logout)
sessionManager.Destroy(ctx)
```

**Session Lifecycle:**
1. **Login** - Store user_id, renew token
2. **Request** - Load user from session
3. **Idle timeout** - Auto-destroy after 30 minutes inactivity
4. **Logout** - Explicitly destroy session
5. **Expired** - Auto-cleanup after 24 hours

---

## Authentication Flow

### Registration

```go
// gojang/http/handlers/auth.go
func (h *AuthHandler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
    // 1. Parse and validate form
    if err := r.ParseForm(); err != nil {
        h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
        return
    }
    
    email := r.Form.Get("email")
    password := r.Form.Get("password")
    
    // 2. Validate input
    if email == "" || password == "" {
        h.Renderer.Render(w, r, "auth/register.html", &renderers.TemplateData{
            Errors: map[string]string{
                "general": "Email and password required",
            },
        })
        return
    }
    
    // 3. Check if user already exists
    exists, _ := h.Client.User.Query().
        Where(user.EmailEQ(email)).
        Exist(r.Context())
    
    if exists {
        h.Renderer.Render(w, r, "auth/register.html", &renderers.TemplateData{
            Errors: map[string]string{
                "email": "Email already registered",
            },
        })
        return
    }
    
    // 4. Hash password
    hash, err := security.HashPassword(password)
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, 
            "Failed to hash password")
        return
    }
    
    // 5. Create user
    u, err := h.Client.User.Create().
        SetEmail(email).
        SetPasswordHash(hash).
        Save(r.Context())
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, 
            "Failed to create user")
        return
    }
    
    // 6. Auto-login after registration
    h.Sessions.Put(r.Context(), "user_id", u.ID)
    h.Sessions.RenewToken(r.Context())
    
    // 7. Redirect to dashboard
    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
```

**Registration Steps:**
1. Parse and validate form data
2. Check if email already exists
3. Hash password with bcrypt
4. Create user in database
5. Auto-login (store user_id in session)
6. Renew session token (security)
7. Redirect to dashboard

### Login

```go
func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
    // 1. Parse form
    if err := r.ParseForm(); err != nil {
        h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
        return
    }
    
    email := r.Form.Get("email")
    password := r.Form.Get("password")
    
    // 2. Find user by email
    u, err := h.Client.User.Query().
        Where(user.EmailEQ(email)).
        Only(r.Context())
    
    if err != nil {
        // User not found - return generic error
        h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
            Errors: map[string]string{
                "general": "Invalid email or password",
            },
        })
        return
    }
    
    // 3. Verify password
    ok, err := security.CheckPassword(u.PasswordHash, password)
    if err != nil || !ok {
        // Wrong password - return generic error
        h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
            Errors: map[string]string{
                "general": "Invalid email or password",
            },
        })
        return
    }
    
    // 4. Check if account is active
    if !u.IsActive {
        h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
            Errors: map[string]string{
                "general": "Your account is inactive",
            },
        })
        return
    }
    
    // 5. Update last login timestamp
    _, err = h.Client.User.UpdateOneID(u.ID).
        SetLastLogin(time.Now()).
        Save(r.Context())
    if err != nil {
        log.Printf("WARNING: Failed to update last login for user %d: %v", 
            u.ID, err)
        // Don't fail login for this
    }
    
    // 6. Create session
    h.Sessions.Put(r.Context(), "user_id", u.ID)
    h.Sessions.RenewToken(r.Context())
    
    // 7. Handle redirect
    redirectURL := r.Form.Get("next")
    if redirectURL == "" {
        redirectURL = "/dashboard"
    }
    
    // 8. Handle HTMX requests
    if r.Header.Get("HX-Request") == "true" {
        w.Header().Set("HX-Redirect", redirectURL)
        w.WriteHeader(http.StatusOK)
        return
    }
    
    http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
```

**Login Steps:**
1. Parse form data
2. Find user by email
3. Verify password with bcrypt
4. Check if account is active
5. Update last_login timestamp
6. Create session and renew token
7. Redirect to next URL or dashboard
8. Handle HTMX requests differently

**Security Notes:**
- ✅ Use generic error messages ("Invalid email or password") to prevent user enumeration
- ✅ Don't reveal if email exists or password is wrong
- ✅ Check is_active flag before allowing login
- ✅ Renew session token after login (prevents session fixation)
- ✅ Support "next" parameter for redirecting after login

### Logout

```go
func (h *AuthHandler) LogoutPOST(w http.ResponseWriter, r *http.Request) {
    // Destroy session
    h.Sessions.Destroy(r.Context())
    
    // Redirect to home page
    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

**Important:** Logout should be POST only (not GET) to prevent CSRF attacks via image tags or links.

---

## Authorization Middleware

### RequireAuth - Basic Authentication

Ensures user is logged in:

```go
// gojang/http/middleware/auth.go
func RequireAuth(sm *scs.SessionManager, client *models.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Get user ID from session
            userID := sm.GetInt(r.Context(), "user_id")
            if userID == 0 {
                // Not logged in
                if r.Header.Get("HX-Request") == "true" {
                    // HTMX request - send redirect header
                    w.Header().Set("HX-Redirect", "/login")
                    w.WriteHeader(http.StatusUnauthorized)
                    return
                }
                // Regular request - redirect with next parameter
                http.Redirect(w, r, "/login?next="+r.URL.Path, http.StatusSeeOther)
                return
            }
            
            // 2. Load user from database
            user, err := client.User.Get(r.Context(), userID)
            if err != nil || !user.IsActive {
                // User deleted or deactivated - destroy session
                sm.Destroy(r.Context())
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }
            
            // 3. Add user to context
            ctx := context.WithValue(r.Context(), userContextKey, user)
            
            // 4. Continue to next handler
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Usage:**
```go
// Protect a single route
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sessionManager, dbClient))
    r.Get("/dashboard", handler.Dashboard)
})

// Protect multiple routes
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sessionManager, dbClient))
    
    r.Get("/profile", handler.Profile)
    r.Get("/settings", handler.Settings)
    r.Post("/settings", handler.SettingsUpdate)
})
```

### LoadUser - Optional Authentication

Loads user if logged in, but doesn't require it:

```go
func LoadUser(sm *scs.SessionManager, client *models.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := sm.GetInt(r.Context(), "user_id")
            if userID != 0 {
                // Try to load user
                user, err := client.User.Get(r.Context(), userID)
                if err == nil && user.IsActive {
                    // Add to context
                    ctx := context.WithValue(r.Context(), userContextKey, user)
                    r = r.WithContext(ctx)
                } else {
                    // Invalid session - clean it up
                    sm.Destroy(r.Context())
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Usage:**
```go
// Use on public pages that show different content for logged-in users
r.Group(func(r chi.Router) {
    r.Use(middleware.LoadUser(sessionManager, dbClient))
    
    r.Get("/", handler.Home)  // Shows login/register or user menu
    r.Get("/about", handler.About)
})
```

### RequireStaff - Admin Authorization

Ensures user is staff/admin:

```go
func RequireStaff(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get user from context (RequireAuth must be used first)
        user := GetUser(r.Context())
        if user == nil || !user.IsStaff {
            // Not staff - forbidden
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

**Usage:**
```go
// Chain with RequireAuth
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sessionManager, dbClient))
    r.Use(middleware.RequireStaff)
    
    r.Get("/admin", handler.AdminPanel)
    r.Get("/admin/users", handler.AdminUsers)
})
```

### GetUser - Retrieve Current User

```go
func GetUser(ctx context.Context) *models.User {
    user, _ := ctx.Value(userContextKey).(*models.User)
    return user
}
```

**Usage in handlers:**
```go
func (h *Handler) MyProfile(w http.ResponseWriter, r *http.Request) {
    // Get current user from context
    user := middleware.GetUser(r.Context())
    
    // Use user data
    h.Renderer.Render(w, r, "profile.html", &renderers.TemplateData{
        Data: map[string]interface{}{
            "User": user,
        },
    })
}
```

**Usage in templates:**
```html
{{if .User}}
    <p>Welcome, {{.User.Email}}!</p>
    {{if .User.IsStaff}}
        <a href="/admin">Admin Panel</a>
    {{end}}
{{else}}
    <a href="/login">Login</a>
{{end}}
```

---

## Route Protection Patterns

### Public Routes
No authentication required:

```go
// Public routes
r.Get("/", handler.Home)
r.Get("/about", handler.About)

// Auth routes (public)
r.Get("/login", authHandler.LoginGET)
r.Post("/login", authHandler.LoginPOST)
r.Get("/register", authHandler.RegisterGET)
r.Post("/register", authHandler.RegisterPOST)
```

### Protected Routes
Authentication required:

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sessionManager, dbClient))
    
    // All routes in this group require login
    r.Get("/dashboard", handler.Dashboard)
    r.Get("/profile", handler.Profile)
    r.Post("/profile", handler.ProfileUpdate)
    
    // Logout must be POST
    r.Post("/logout", authHandler.LogoutPOST)
})
```

### Admin Routes
Staff/admin permission required:

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sessionManager, dbClient))
    r.Use(middleware.RequireStaff)
    
    // Admin-only routes
    r.Get("/admin", handler.AdminPanel)
    r.Get("/admin/users", handler.UserList)
    r.Post("/admin/users/{id}/deactivate", handler.UserDeactivate)
})
```

### Mixed Routes
Different permissions for different methods:

```go
r.Group(func(r chi.Router) {
    // Anyone can view posts
    r.Get("/posts", handler.PostList)
    r.Get("/posts/{id}", handler.PostView)
    
    r.Group(func(r chi.Router) {
        // Must be logged in to create/edit/delete
        r.Use(middleware.RequireAuth(sessionManager, dbClient))
        
        r.Get("/posts/new", handler.PostNew)
        r.Post("/posts", handler.PostCreate)
        r.Get("/posts/{id}/edit", handler.PostEdit)
        r.Put("/posts/{id}", handler.PostUpdate)
        r.Delete("/posts/{id}", handler.PostDelete)
    })
})
```

---

## Advanced Patterns

### Custom Permissions

Create your own permission middleware:

```go
// RequirePermission checks a custom permission
func RequirePermission(permissionCheck func(*models.User) bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := GetUser(r.Context())
            if user == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            if !permissionCheck(user) {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
r.Use(RequirePermission(func(u *models.User) bool {
    return u.IsStaff && u.Email == "admin@example.com"
}))
```

### Resource-Based Authorization

Check if user owns a resource:

```go
func (h *PostHandler) Edit(w http.ResponseWriter, r *http.Request) {
    postID, _ := strconv.Atoi(chi.URLParam(r, "id"))
    user := middleware.GetUser(r.Context())
    
    // Load post with author
    post, err := h.Client.Post.Query().
        Where(post.IDEQ(postID)).
        WithAuthor().
        Only(r.Context())
    
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusNotFound, "Post not found")
        return
    }
    
    // Check ownership
    if post.Edges.Author.ID != user.ID && !user.IsStaff {
        h.Renderer.RenderError(w, r, http.StatusForbidden, 
            "You can only edit your own posts")
        return
    }
    
    // User is authorized - show edit form
    h.Renderer.Render(w, r, "posts/edit.partial.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Post": post},
    })
}
```

### Role-Based Access Control (RBAC)

Extend the User model with roles:

```go
// gojang/models/schema/user.go
func (User) Fields() []ent.Field {
    return []ent.Field{
        // ... existing fields ...
        
        field.Enum("role").
            Values("user", "moderator", "admin").
            Default("user"),
    }
}
```

Create role middleware:

```go
func RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := GetUser(r.Context())
            if user == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // Check if user has one of the required roles
            hasRole := false
            for _, role := range roles {
                if user.Role == role {
                    hasRole = true
                    break
                }
            }
            
            if !hasRole {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
r.Use(RequireRole("admin", "moderator"))
```

---

## Security Best Practices

### ✅ DO

1. **Always hash passwords** with bcrypt
   ```go
   hash, _ := security.HashPassword(password)
   ```

2. **Use HTTPS in production** (cookie Secure flag)
   ```go
   sessionManager.Cookie.Secure = !cfg.Debug
   ```

3. **Renew session tokens** after login
   ```go
   sessionManager.RenewToken(ctx)
   ```

4. **Use HttpOnly cookies** (prevent XSS)
   ```go
   sessionManager.Cookie.HttpOnly = true
   ```

5. **Implement idle timeout** (auto-logout)
   ```go
   sessionManager.IdleTimeout = 30 * time.Minute
   ```

6. **Check is_active flag** before allowing login
   ```go
   if !user.IsActive {
       return errors.New("account inactive")
   }
   ```

7. **Use generic error messages** (prevent user enumeration)
   ```go
   errors := map[string]string{
       "general": "Invalid email or password",
   }
   ```

8. **Protect logout with POST** (not GET)
   ```go
   r.Post("/logout", handler.LogoutPOST)
   ```

9. **Validate on server** (never trust client)
   ```go
   if email == "" || password == "" {
       return errors.New("validation failed")
   }
   ```

10. **Load user on each request** (check if still active/exists)
    ```go
    user, err := client.User.Get(ctx, userID)
    if err != nil || !user.IsActive {
        sm.Destroy(ctx)
        // Redirect to login
    }
    ```

### ❌ DON'T

1. **Don't store plaintext passwords** - Always hash
2. **Don't expose password hashes** - Mark fields as Sensitive()
3. **Don't reveal user existence** - Use generic error messages
4. **Don't skip session renewal** - Prevents session fixation
5. **Don't trust client data** - Always validate server-side
6. **Don't use GET for logout** - Use POST to prevent CSRF
7. **Don't skip CSRF protection** - Use built-in middleware
8. **Don't log sensitive data** - Passwords, tokens, etc.
9. **Don't allow weak passwords** - Implement password requirements
10. **Don't forget rate limiting** - Prevent brute force attacks

---

## Testing Authentication

### Unit Tests

Test password hashing:

```go
func TestHashPassword(t *testing.T) {
    password := "testpassword123"
    
    hash, err := security.HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }
    
    if hash == password {
        t.Error("Hash should not equal plaintext")
    }
    
    match, err := security.CheckPassword(hash, password)
    if err != nil || !match {
        t.Error("Password verification failed")
    }
}
```

Test middleware:

```go
func TestRequireAuth_NotLoggedIn(t *testing.T) {
    // Create test request without session
    req := httptest.NewRequest("GET", "/protected", nil)
    rec := httptest.NewRecorder()
    
    // Create handler with middleware
    handler := middleware.RequireAuth(sessionManager, client)(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
        }),
    )
    
    handler.ServeHTTP(rec, req)
    
    // Should redirect to login
    if rec.Code != http.StatusSeeOther {
        t.Errorf("Expected redirect, got %d", rec.Code)
    }
}
```

### Integration Tests

Test complete login flow:

```go
func TestLoginFlow(t *testing.T) {
    // 1. Register user
    password := "testpass123"
    hash, _ := security.HashPassword(password)
    
    user, err := client.User.Create().
        SetEmail("test@example.com").
        SetPasswordHash(hash).
        Save(ctx)
    
    if err != nil {
        t.Fatal(err)
    }
    
    // 2. Attempt login
    form := url.Values{}
    form.Set("email", "test@example.com")
    form.Set("password", password)
    
    req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    rec := httptest.NewRecorder()
    
    authHandler.LoginPOST(rec, req)
    
    // 3. Verify redirect
    if rec.Code != http.StatusSeeOther {
        t.Errorf("Expected redirect, got %d", rec.Code)
    }
    
    // 4. Verify session created
    // Check session cookie, etc.
}
```

---

## Common Issues & Solutions

### Issue: Session Lost After Server Restart

**Problem:** Users are logged out when server restarts

**Solution:** Use persistent session store (Redis, database)

```go
import "github.com/alexedwards/scs/mysqlstore"

// Use MySQL for session storage
sessionManager.Store = mysqlstore.New(db)
```

### Issue: "Invalid CSRF Token"

**Problem:** Forms return CSRF errors

**Solution:** Ensure CSRF token is included in forms

```html
<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
```

And CSRF middleware is enabled:

```go
r.Use(middleware.CSRF(csrfSecret))
```

### Issue: User Enumeration

**Problem:** Different errors reveal if email exists

**Solution:** Use generic error messages

```go
// ❌ DON'T
if !userExists {
    return "User not found"
}
if !passwordMatch {
    return "Wrong password"
}

// ✅ DO
return "Invalid email or password"
```

### Issue: Session Fixation

**Problem:** Attacker can steal sessions

**Solution:** Always renew token after login

```go
sessionManager.RenewToken(ctx)
```

---

## Next Steps

- ✅ **Implement:** Add authentication to your custom features
- ✅ **Secure:** Review security best practices
- ✅ **Test:** Write tests for authentication flows
- ✅ **Read:** [Deployment Guide](./deployment-guide.md) for production setup
- ✅ **Explore:** Check `gojang/http/middleware/auth.go` for more details

---

## Quick Reference

### Common Functions

```go
// Password hashing
hash, err := security.HashPassword(password)
match, err := security.CheckPassword(hash, password)

// Session management
sessionManager.Put(ctx, "user_id", id)
userID := sessionManager.GetInt(ctx, "user_id")
sessionManager.RenewToken(ctx)
sessionManager.Destroy(ctx)

// Get current user
user := middleware.GetUser(ctx)
```

### Middleware Chain

```go
// Public route
r.Get("/public", handler.Public)

// Protected route
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sm, client))
    r.Get("/protected", handler.Protected)
})

// Admin route
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(sm, client))
    r.Use(middleware.RequireStaff)
    r.Get("/admin", handler.Admin)
})
```

---

Happy coding securely! 🔐
