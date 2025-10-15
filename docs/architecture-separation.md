# Admin and User Site Separation

This document explains how the Gojang framework maintains a clean separation between the admin panel and the user-facing site.

## Overview

Gojang follows a strict architectural separation between:
- **User Site**: Public-facing pages at `/` routes
- **Admin Panel**: Staff-only management interface at `/admin` routes

## Directory Structure

```
gojang/
├── admin/                      # Admin panel (isolated)
│   ├── handler.go              # Generic CRUD handlers
│   ├── models.go               # Model registration
│   ├── registry.go             # Reflection-based operations
│   ├── admin_renderer.go       # Admin-specific renderer
│   ├── admin_routes.go         # Admin route definitions
│   └── views/                  # Admin templates
│       ├── admin_base.html
│       ├── admin_main.html
│       ├── model_index.html
│       ├── model_list.partial.html
│       ├── model_form.partial.html
│       └── model_delete.partial.html
│
├── http/
│   ├── handlers/               # User site handlers (isolated)
│   │   ├── auth.go
│   │   ├── pages.go
│   │   ├── posts.go
│   │   └── users.go
│   │
│   └── routes/                 # User site routes
│       ├── pages.go
│       ├── posts.go
│       └── users.go
│
└── views/
    ├── renderers/
    │   └── renderer.go         # User site renderer
    │
    └── templates/              # User site templates
        ├── base.html
        ├── home.html
        ├── posts/
        └── users/
```

## Key Principles

### 1. Separate Renderers

**User Site Renderer** (`renderers.Renderer`):
- Located in `gojang/views/renderers/renderer.go`
- Uses `base.html` as layout
- Handles user-facing templates only

**Admin Renderer** (`admin.AdminRenderer`):
- Located in `gojang/admin/admin_renderer.go`
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
// Mounted under /admin prefix with staff-only middleware
r.Route("/admin", func(r chi.Router) {
    r.Use(middleware.RequireStaff)
    r.Get("/", adminHandler.Dashboard)
    r.Get("/{model}", adminHandler.Index)
    r.Post("/{model}", adminHandler.Create)
    // ... generic CRUD for all models
})
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

The admin panel uses **reflection-based CRUD** that works for ALL models:

```go
// Register a model - that's it!
admin.RegisterModel(admin.ModelRegistration{
    Name:         "post",
    Model:        &models.Post{},
    ListFields:   []string{"ID", "Subject", "CreatedAt"},
    FormFields:   []string{"Subject", "Body"},
    SearchFields: []string{"Subject", "Body"},
    BeforeSave:   beforeSavePost, // Optional hook
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
Remove-Item gojang/views/templates/posts/admin_index.html
Remove-Item gojang/views/templates/posts/admin_list.partial.html
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
// http/handlers/products.go - custom user experience
func (h *ProductHandler) ShowProduct(w http.ResponseWriter, r *http.Request) {
    // Custom user-facing product display
    h.Renderer.Render(w, r, "products/detail.html", data)
}
```

### Admin Panel (always)
```go
// gojang/admin/models.go - just register it!
admin.RegisterModel(admin.ModelRegistration{
    Name:       "product",
    Model:      &models.Product{},
    ListFields: []string{"ID", "Name", "Price"},
    FormFields: []string{"Name", "Description", "Price"},
})
```

That's it! The admin panel automatically provides full CRUD at `/admin/product`.

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
   http://localhost:8080/admin/post
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
