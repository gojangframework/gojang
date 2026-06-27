# Admin and User Site Separation

This document explains how the Gojang framework maintains a clean separation between the admin panel and the user-facing site.

## Overview

Gojang follows a strict architectural separation between:
- **User Site**: Public-facing pages at `/` routes
- **Admin Panel**: Staff-only management interface at `/admin` routes

## Directory Structure

```
app/
├── gojang/                     # Framework-owned code
│   ├── admin/                  # Admin panel, renderer, routes, templates
│   ├── http/                   # Core auth/user handlers and middleware
│   ├── models/                 # Generated Ent code and migrations
│   └── views/                  # Public renderer and embedded FS wiring
├── pages/                      # App-owned page handlers and routes
├── schema/                     # Ent schema definitions
├── posts/                      # App-owned post handlers, routes, templates
└── views/                      # Shared public templates, static files, i18n, forms
```

## Key Principles

### 1. Separate Renderers

**User Site Renderer** (`renderers.Renderer`):
- Located in `app/gojang/views/renderers/renderer.go`
- Uses `base.html` as layout
- Handles user-facing templates only

**Admin Renderer** (`admin.AdminRenderer`):
- Located in `app/gojang/admin/admin_renderer.go`
- Uses `admin_base.html` as layout
- Handles admin templates only

### 2. Separate Route Namespaces

**User Routes**:
```go
// Mounted at root level
r.Get("/", pageHandler.Home)
r.Get("/posts", postHandler.Index)
r.Get("/dashboard", pageHandler.Dashboard)
```

**Admin Routes**:
```go
// Mounted under /admin prefix with auth, staff, and audit middleware.
r.Mount("/admin", admin.AdminRoutes(adminHandler, sessionManager, client))

// Canonical workspace routes are under /admin/t/{resource}.
// Legacy /admin/{model} routes redirect into the workspace.
```

### 3. No Cross-References

**❌ Don't do this**:
```go
// User handler checking if admin request
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    isAdmin := strings.HasPrefix(r.URL.Path, "/admin/")
    if isAdmin {
        // Use admin template
    } else {
        // Use user template
    }
}
```

**✅ Do this**:
```go
// User handler - user site only
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    h.Renderer.Render(w, r, "posts/list.partial.html", data)
}

// Admin uses generic CRUD automatically
// No model-specific admin handlers needed!
```

### 4. Generic Admin Panel

The admin panel uses **reflection-based CRUD** that works for ALL Ent models
exposed on `*models.Client`. A plain generated model needs no explicit admin
registration. Use `RegisterModel` only to override the discovered resource:

```go
// Optional override for display fields, hooks, or eager loading.
registry.RegisterModel(admin.ModelRegistration{
    ModelType:      &models.Post{},
    Icon:           "✎",
    NamePlural:     "Posts",
    ListFields:     []string{"ID", "Subject", "Author", "CreatedAt"},
    ReadonlyFields: []string{"ID", "CreatedAt", "UpdatedAt"},
    QueryModifier:  preloadPostAuthor, // Optional hook
})
```

No need for model-specific admin handlers or templates!

## Migration from Mixed Architecture

If you have existing code that mixes admin and user concerns:

### Step 1: Remove Admin Functions from User Handlers

**Before**:
```go
// handlers/posts.go
func (h *PostHandler) AdminIndex(w http.ResponseWriter, r *http.Request) {
    // Admin-specific list view
}
```

**After**:
```go
// Just delete it - admin panel handles this generically
```

### Step 2: Remove Path-Based Admin Detection

**Before**:
```go
isAdmin := user.IsStaff && strings.HasPrefix(r.URL.Path, "/admin/")
templateName := "posts/list.partial.html"
if isAdmin {
    templateName = "posts/admin_list.partial.html"
}
h.Renderer.Render(w, r, templateName, data)
```

**After**:
```go
// User handler always uses user template
h.Renderer.Render(w, r, "posts/list.partial.html", data)
// Admin panel automatically uses admin templates
```

### Step 3: Remove Admin Templates from User Renderer

**Before**:
```go
// renderers/renderer.go
pages := []string{
    "posts/index.html",
    "posts/admin_index.html", // ❌ Don't include admin templates
}
```

**After**:
```go
pages := []string{
    "posts/index.html", // ✅ User templates only
}
```

### Step 4: Delete Unused Admin Templates

```powershell
# These are handled by generic admin templates now
Remove-Item app/posts/templates/admin_index.html
Remove-Item app/posts/templates/admin_list.partial.html
```

## Benefits of Separation

1. **Simplicity**: User handlers focus only on user experience
2. **Maintainability**: Changes to admin don't affect user site
3. **Generic Admin**: One set of admin templates/handlers for ALL models
4. **Clear Ownership**: Easy to see which code affects which site
5. **Security**: Admin middleware applied consistently to all admin routes
6. **Testing**: Can test user and admin functionality independently

## Adding a New Model

When you add a new model (e.g., `Product`):

### User Site (if needed)
```go
// app/products/products.handler.go - custom user experience
func (h *ProductHandler) ShowProduct(w http.ResponseWriter, r *http.Request) {
    // Custom user-facing product display
    h.Renderer.Render(w, r, "products/detail.html", data)
}
```

### Admin Panel (automatic)

After `go generate ./app/gojang/models`, the generated `Product` client appears
on `*models.Client`, and the admin registry discovers it automatically. The
admin panel provides full CRUD in the workspace at `/admin/t/product`.

Add an override in `app/gojang/admin/models.go` only when the default resource
metadata is not enough, such as custom list fields, hidden fields, hooks, or
relationship eager loading.

## Verification

To verify proper separation:

1. **Check user handlers** - no references to:
   - `admin_*.html` templates
   - Path checks for `/admin/`
   - AdminRenderer
   - Staff-specific logic

2. **Check admin package** - completely isolated:
   - Own renderer
   - Own templates
   - Own routes (under `/admin`)
   - Generic handlers (no model-specific code)

3. **Test both sites independently**:
   ```
   # User site
   http://localhost:8080/posts
   
   # Admin panel
   http://localhost:8080/admin/t/post
   ```

Both should work completely independently!

## Summary

The Gojang framework maintains strict separation between admin and user sites through:

- ✅ Separate renderers and template directories
- ✅ Separate route namespaces (`/` vs `/admin`)
- ✅ No cross-references or path-based detection
- ✅ Generic reflection-based admin panel
- ✅ Model-specific handlers only in user site (when needed)

This architecture keeps your codebase clean, maintainable, and easy to extend!
