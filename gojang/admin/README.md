# Admin Package

Self-contained admin panel for Gojang framework.

## Structure

```
gojang/admin/
├── admin_renderer.go      # Admin template renderer (independent from public site)
├── admin_routes.go        # Admin route definitions
├── handler.go             # Admin workspace and CRUD HTTP handlers
├── models.go              # Model-specific admin overrides
├── registry.go            # Ent model discovery and reflection-based field metadata
└── views/
    ├── admin_base.html           # Admin base layout
    ├── admin_main.html           # Empty workspace fallback
    ├── workspace.html            # Airtable-style resource workspace
    ├── workspace_grid.partial.html
    ├── grid_cell.partial.html
    └── record_drawer.partial.html
```

## Key Features

- **Separate from public site**: Independent templates, renderer, and routes
- **Generic CRUD**: Automatic admin interface for discovered Ent models
- **Reflection-based**: Auto-discovers model fields and types
- **Smart field detection**: Automatically detects email, password, text, bool, int, time fields
- **HTMX-powered**: Inline grid edits and drawer saves without page reloads
- **Type-safe**: Uses Ent's generated code for database operations

## Usage

### 1. Configure Overrides

Models are discovered from `*models.Client`. Use `models.go` only for admin-specific overrides:

```go
func RegisterModels(registry *Registry) {
    // Optional: override fields, hooks, icons, or list columns.
}
```

### 2. Mount Admin Routes

In `main.go`:

```go
// Setup admin
adminRenderer, _ := admin.NewAdminRenderer(cfg.Debug)
adminRegistry := admin.NewRegistry(client)
admin.RegisterModels(adminRegistry)
adminHandler := admin.NewHandler(adminRegistry, adminRenderer, client)

// Mount admin routes
r.Mount("/admin", admin.AdminRoutes(adminHandler, sessionManager, client))
```

### 3. Access Admin Panel

Navigate to `http://localhost:8080/admin` (requires staff user).

## File Descriptions

### `admin_renderer.go`
- Template renderer specifically for admin panel
- Renders full workspace pages and HTMX fragments
- Uses embedded files from `gojang/admin/views`
- Includes workspace template helpers for cells, drawer inputs, and field locking

### `admin_routes.go`
- Route definitions using chi router
- Applies auth, staff, and audit middleware
- Workspace CRUD routes under `/t/{resource}`
- Legacy `/{model}` admin routes redirect into `/t/{resource}`

### `handler.go`
- HTTP handlers for workspace rendering, inline grid editing, drawer editing, and CRUD operations
- Context-aware database operations
- Smart HTMX response handling
- Error handling and validation

### `registry.go`
- Ent client discovery and model override system
- Reflection-based field discovery from Ent structs
- Field type detection (email, password, int, bool, time, text)
- Automatic readonly field marking (ID, CreatedAt, UpdatedAt)
- AdminOverrides for customization

### `models.go`
- Optional per-model admin overrides
- User password handling and Post author assignment hooks
- Auto-discovery remains the default for all Ent models

## Template System

### Base Template (`admin_base.html`)
- Separate from public site's `base.html`
- Admin-specific header with navigation
- Record drawer target and HTMX CSRF setup
- Custom admin styling

### Full Pages
- `admin_main.html`: Empty workspace fallback
- `workspace.html`: Resource sidebar, toolbar, and grid workspace

### Fragments
- `workspace_grid.partial.html`: Resource grid
- `grid_cell.partial.html`: Inline editable cell
- `record_drawer.partial.html`: Create/edit drawer

## Customization

### Override Model Display

```go
registry.RegisterModel(admin.ModelRegistration{
    ModelType: &models.Post{},
    Icon: "✎",
    NamePlural: "Blog Posts",
    HiddenFields: []string{"Slug"},
})
```

### Add Custom Fields

Add virtual or admin-only fields in a model override:

```go
registry.RegisterModel(admin.ModelRegistration{
    ModelType: &models.User{},
    CustomFields: []admin.FieldConfig{
        {Name: "Password", Type: admin.FieldTypePassword, Editable: true, Virtual: true},
    },
})
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
