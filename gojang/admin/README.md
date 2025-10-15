# Admin Package

Self-contained admin panel for Gojang framework.

## Structure

```
gojang/admin/
├── admin_renderer.go      # Admin template renderer (independent from public site)
├── admin_routes.go        # Admin route definitions
├── handler.go             # Admin HTTP handlers (CRUD operations)
├── models.go              # Model registration (User, Post, etc.)
├── registry.go            # Model registry with reflection-based field discovery
└── views/
    ├── admin_base.html           # Admin base layout
    ├── admin_main.html           # Admin dashboard (renamed from dashboard.html)
    ├── model_index.html          # Model list page
    ├── model_list.partial.html   # Model list partial (HTMX)
    ├── model_form.html           # Create/Edit form modal
    └── model_delete.html         # Delete confirmation modal
```

## Key Features

- **Separate from public site**: Independent templates, renderer, and routes
- **Generic CRUD**: Automatic admin interface for any Ent model
- **Reflection-based**: Auto-discovers model fields and types
- **Smart field detection**: Automatically detects email, password, text, bool, int, time fields
- **HTMX-powered**: Modal forms and instant updates without page reloads
- **Type-safe**: Uses Ent's generated code for database operations

## Usage

### 1. Register Models

In `models.go`, register your models:

```go
func RegisterModels(registry *Registry) {
    registerUserModel(registry)
    registerPostModel(registry)
    // Add more models here
}
```

### 2. Mount Admin Routes

In `main.go`:

```go
// Setup admin
adminRenderer, _ := admin.NewAdminRenderer(cfg.Debug)
adminRegistry := admin.NewRegistry(client)
admin.RegisterModels(adminRegistry)
adminHandler := admin.NewHandler(adminRegistry, adminRenderer)

// Mount admin routes
r.Mount("/admin", admin.AdminRoutes(adminHandler, sessionManager, client))
```

### 3. Access Admin Panel

Navigate to `http://localhost:8080/admin` (requires staff user).

## File Descriptions

### `admin_renderer.go`
- Template renderer specifically for admin panel
- No base layout wrapper for modals
- Uses `./gojang/admin/views/templates` for template files
- Includes template functions: `fieldValue`, `getID`, `formatDateTime`

### `admin_routes.go`
- Route definitions using chi router
- Applies auth, staff, and audit middleware
- Generic CRUD routes: `/{model}`, `/{model}/new`, `/{model}/{id}/edit`, etc.

### `handler.go`
- HTTP handlers for all CRUD operations
- Context-aware database operations
- Smart HTMX response handling
- Error handling and validation

### `registry.go`
- Model registration system
- Reflection-based field discovery from Ent models
- Field type detection (email, password, int, bool, time, text)
- Automatic readonly field marking (ID, CreatedAt, UpdatedAt)
- AdminOverrides for customization

### `models.go`
- Model registration functions
- Minimal configuration needed (~30 lines per model)
- Auto-discovers fields from Ent schemas

## Template System

### Base Template (`admin_base.html`)
- Separate from public site's `base.html`
- Admin-specific header with navigation
- Modal containers for forms and confirmations
- Custom admin styling

### Full Pages
- `admin_main.html`: Shows all registered models
- `model_index.html`: Lists all records for a model

### Modals/Fragments
- `model_form.html`: Create/edit form (rendered as modal)
- `model_delete.html`: Delete confirmation (rendered as modal)
- `model_list.partial.html`: Table of records (HTMX partial)

## Customization

### Override Model Display

```go
admin.RegisterAdmin(registry, "Post", &admin.AdminOverrides{
    Icon: "📝",
    NamePlural: "Blog Posts",
    HiddenFields: []string{"Slug"},
})
```

### Add Custom Fields

Extend the field detection in `registry.go`:

```go
func detectFieldType(fieldName string, fieldType string) string {
    if strings.Contains(lower, "url") {
        return "url"
    }
    // ... existing detection
}
```

## Security

- **Requires authentication**: `RequireAuth` middleware
- **Requires staff status**: `RequireStaff` middleware
- **Audit logging**: All admin actions are logged
- **CSRF protection**: `nosurf` middleware on all forms

## Dependencies

- `github.com/go-chi/chi/v5` - Router
- `github.com/justinas/nosurf` - CSRF protection
- Gojang's `http/middleware` - Auth, audit, security
- Gojang's `models` - Ent database client
