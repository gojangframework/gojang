# Gojang Documentation

Welcome to the Gojang framework documentation! This guide will help you build modern web applications with Go and HTMX.

## Core Concepts

### [Architecture: Admin & User Site Separation](./architecture-separation.md)

**Understanding Gojang's clean architecture**

Learn about:
- Why admin and user sites are completely separated
- How the generic admin panel works
- Best practices for maintaining separation
- Migration guide from mixed architectures

**Read this first** to understand the framework's design philosophy!

---

## Tutorials

### 1. [Creating Static Pages](./creating-static-pages.md)

**Learn how to add simple pages without data models**

Perfect for:
- About pages
- Contact pages
- Terms of Service
- FAQ pages

**Topics covered:**
- Template structure
- Handler creation
- Route registration
- Custom CSS usage
- HTMX basics

**Time:** ~5 minutes per page

---

### 2. [Quick Start: Adding a Data Model](./quick-start-data-model.md)

**The simplest way to add a new data model - perfect for beginners!**

Perfect for:
- Your first model
- Learning the basics
- Quick prototyping
- Simple CRUD apps

**Topics covered:**
- 🆕 **Automated generation** with `addmodel` command
- Minimal 4-property Product example
- Step-by-step with code snippets
- Handler, routes, and views
- Admin panel integration
- Testing your model

**Time:** ~10 minutes (or instant with `addmodel` command!)

---

### 3. [Creating Pages with Data Models](./creating-data-models.md)

**Comprehensive guide for advanced data model features**

Perfect for:
- Complex models with many fields
- Advanced relationships
- Custom validation
- Production applications

**Topics covered:**
- 🆕 **Automated generation** with enhanced `addmodel` command
- 🆕 **Dry-run mode** for safe preview
- 🆕 **Command-line automation** for CI/CD
- Advanced Ent schema features
- Code generation
- Database migrations
- Form validation
- CRUD handlers
- Template patterns
- Admin panel integration
- Troubleshooting

**Time:** ~20-30 minutes per model (or instant with automation!)

---

### 4. [HTMX Integration Patterns](./htmx-patterns.md)

**Master HTMX patterns for building dynamic, interactive user interfaces**

Perfect for:
- Modal dialogs
- Form submissions
- CRUD operations
- Dynamic content loading
- Real-time updates

**Topics covered:**
- Core HTMX attributes
- Modal dialog patterns
- Form handling & validation
- CRUD operations (Create, Read, Update, Delete)
- Partial templates
- Event handling
- Response headers
- Authentication with HTMX
- Best practices

**Time:** ~15-20 minutes to read, lifetime to master

---

---

## Advanced Guides

### 5. [Authentication & Authorization Deep Dive](./authentication-authorization.md)

**Master user authentication and authorization in Gojang**

Perfect for:
- User registration and login
- Session management
- Password security
- Access control middleware
- Role-based permissions
- Resource ownership checks

**Topics covered:**
- User model and password hashing
- Session-based authentication
- Login/logout flows
- Authentication middleware (RequireAuth, RequireStaff)
- Authorization patterns
- Security best practices
- Testing authentication

**Time:** ~30 minutes to read

---

### 6. [Deployment Guide](./deployment-guide.md)

**Deploy your Gojang application to production**

Perfect for:
- Docker containerization
- VPS deployments (DigitalOcean, Linode)
- Cloud platforms (AWS, GCP, Azure)
- PaaS providers (Fly.io, Railway, Heroku)

**Topics covered:**
- Building production binaries
- Docker and Docker Compose setup
- VPS server configuration
- Nginx reverse proxy setup
- SSL/TLS with Let's Encrypt
- Database setup (PostgreSQL)
- Process management (systemd)
- CI/CD pipelines
- Monitoring and backups

**Time:** ~45 minutes to read, 1-2 hours to deploy

---

### 7. [Distributed Deployment & Load Balancing](./distributed-deployment.md)

**Scale your Gojang application with load balancing and distributed deployment**

Perfect for:
- High-traffic applications
- High availability requirements
- Horizontal scaling
- Zero-downtime deployments
- Multi-server architectures

**Topics covered:**
- Distributed deployment capabilities
- PostgreSQL for shared state
- Redis for session management
- Load balancer configuration (Nginx, HAProxy, cloud LBs)
- Multiple deployment architectures
- Health checks and monitoring
- Rolling deployments
- Best practices and troubleshooting

**Time:** ~30 minutes to read

---

### 8. [Logging Guide](./logging-guide.md)

**Master structured logging with Zap for production-ready observability**

Perfect for:
- Understanding the logging system
- Production debugging and monitoring
- Performance optimization
- Security audit trails

**Topics covered:**
- Zap-based structured logging
- Formatted vs structured logging
- Log levels (DEBUG, INFO, WARN, ERROR)
- Best practices and patterns
- Handler and middleware logging
- Production considerations
- Anti-patterns to avoid

**Time:** ~10 minutes to read

---

### 9. [Testing Best Practices](./testing-best-practices.md)

**Write effective tests for your Gojang application**

Perfect for:
- Unit testing
- Integration testing
- Handler testing
- Database testing
- Test-driven development (TDD)

**Topics covered:**
- Testing structure and conventions
- Unit test patterns
- Table-driven tests
- Testing handlers and middleware
- Database test setup
- Mock objects and test doubles
- Test fixtures and helpers
- Benchmarking
- Code coverage
- CI/CD integration

**Time:** ~25 minutes to read

---

### 10. [Security Features](./SECURITY-SUMMARY.md) 🔒

**Comprehensive overview of implemented security features**

Perfect for:
- Understanding security implementations
- Production deployment preparation
- Security compliance verification
- Security feature reference

**Topics covered:**
- Authentication & password security (Argon2id)
- Session management & CSRF protection
- Rate limiting with proper IP handling
- Security headers (including HSTS)
- Input validation & database security
- HTTPS enforcement
- Audit logging
- Security disclosure (security.txt)
- CI/CD security scanning

**Time:** ~10 minutes to read

---

### 11. [Taskfile Commands Guide](./taskfile-guide.md)

**Master the Task automation commands for database migrations and development**

Perfect for:
- Database migrations
- Development workflow automation
- Understanding available commands
- Migration best practices

**Topics covered:**
- Installing Task
- Database migration commands (migrate, migrate-down, migrate-create)
- Other development commands (build, test, dev, etc.)
- Migration workflow
- Best practices and troubleshooting
- Common migration patterns

**Time:** ~15 minutes to read

---

## Documentation Structure

```
docs/
├── README.md                           # This file - Documentation index
├── architecture-separation.md          # Guide: Admin & User site separation
├── creating-static-pages.md            # Tutorial: Static pages
├── quick-start-data-model.md           # Tutorial: Simple data model (START HERE!)
├── creating-data-models.md             # Tutorial: Data models & CRUD (comprehensive)
├── htmx-patterns.md                    # Guide: HTMX integration patterns
├── authentication-authorization.md     # Guide: Auth & authorization
├── deployment-guide.md                 # Guide: Production deployment
├── distributed-deployment.md           # Guide: Load balancing & distributed setup
├── logging-guide.md                    # Guide: Logging Guide
├── testing-best-practices.md           # Guide: Testing strategies
├── SECURITY-SUMMARY.md                 # Guide: Security features overview
└── taskfile-guide.md                   # Guide: Task commands & migrations
```

---

## Framework Architecture

### Project Structure

```
gojang/
├── admin/                    # Admin panel (auto-CRUD interface)
│   ├── models.go            # Model registration
│   ├── registry.go          # Generic CRUD operations
│   ├── handler.go           # Admin handlers
│   ├── admin_renderer.go    # Admin template renderer
│   └── views/               # Admin templates
├── cmd/
│   └── web/
│       └── main.go          # Application entry point
├── config/
│   └── config.go            # Configuration management
├── http/
│   ├── handlers/            # Request handlers
│   ├── middleware/          # Auth, sessions, security
│   ├── routes/              # Route definitions
│   └── security/            # Password hashing
├── models/
│   ├── schema/              # Ent schemas (YOUR models go here)
│   └── *.go                 # Generated Ent code
└── views/
    ├── forms/               # Form validation structs
    ├── renderers/           # Template rendering
    ├── templates/           # HTML templates
    └── static/              # CSS, images
```

## Key Concepts

### Template Rendering

Templates use Go's `html/template` with two required blocks:

```html
{{define "title"}}Page Title{{end}}

{{define "content"}}
<!-- Your content here -->
{{end}}
```

Base template (`base.html`) provides:
- Header with navigation
- Footer
- Flash messages
- HTMX integration

### Handlers

Handlers follow this pattern:

```go
func (h *Handler) Action(w http.ResponseWriter, r *http.Request) {
    // 1. Parse input
    // 2. Validate
    // 3. Process (database operations)
    // 4. Render response
}
```

### Routes

Routes are organized by feature:

```go
func RegisterFeatureRoutes(r chi.Router, handler *handlers.FeatureHandler) {
    r.Get("/feature", handler.List)
    
    r.Group(func(r chi.Router) {
        r.Use(middleware.RequireAuth)
        r.Post("/feature", handler.Create)
    })
}
```

### Admin Panel

The admin panel provides automatic CRUD interface for any Ent model:

```go
registry.RegisterModel(ModelRegistration{
    ModelType:      &models.YourModel{},
    Icon:           "🎯",
    NamePlural:     "Your Models",
    ListFields:     []string{"ID", "Name", "CreatedAt"},
    ReadonlyFields: []string{"ID", "CreatedAt"},
})
```

No custom admin code needed! ✨

---

## Common Patterns

### Authentication Required

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth)
    r.Get("/protected", handler.Protected)
})
```

### Staff/Admin Only

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth)
    r.Use(middleware.RequirePermission("is_staff"))
    r.Get("/admin-only", handler.AdminOnly)
})
```

### Get Current User

```go
user, authenticated := middleware.GetUserFromContext(r.Context())
if !authenticated {
    // Not logged in
}
```

### Form Validation

```go
var form forms.MyForm
if err := forms.Decode(r, &form); err != nil {
    // Handle decode error
}

if err := forms.Validate(r.Context(), &form); err != nil {
    // Handle validation errors
}
```

### HTMX Partial Response

```go
if r.Header.Get("HX-Request") == "true" {
    // Return just the fragment
    h.Renderer.Render(w, r, "partial.html", data)
} else {
    // Return full page
    h.Renderer.Render(w, r, "full.html", data)
}
```

---

## Development Workflow

### 1. Adding a New Feature

1. Create Ent schema (if needed)
2. Generate code: `cd gojang/models && go generate ./...`
3. Create handler
4. Create routes
5. Create templates
6. Register in admin panel (optional)
7. Test!

### 2. Running the App

```bash
# Development (basic)
go run ./gojang/cmd/web

# Development with live reload (recommended)
task dev

# Using task commands
task build            # Build the application
task run              # Build and run
task test             # Run tests

# Build for production
go build -o app ./gojang/cmd/web
./app
```

**To enable live reload (recommended for development):**

Install Air:
```bash
go install github.com/air-verse/air@latest
```

Then run:
```bash
task dev
# or
air
```

Air will automatically restart your server when you save changes to your Go files!

### 3. Database Migrations

Auto-migration runs on startup, but you can also use the migrate commands:

```bash
# Apply all pending migrations
task migrate

# Rollback last migration
task migrate-down

# Create a new migration
task migrate-create name=add_new_table
```

See the [Taskfile Commands Guide](./taskfile-guide.md) for detailed migration documentation.

### 4. Testing

```bash
# Run all tests
go test ./...

# Test specific package
go test ./gojang/http/handlers

# With coverage
go test -cover ./...
```

---

## Useful Commands

```bash
# Generate Ent code after schema changes
cd gojang/models && go generate ./...

# Format code
go fmt ./...

# Check for issues
go vet ./...

# Update dependencies
go mod tidy

# Build
go build -o web.exe ./gojang/cmd/web

# Run
./web.exe
```

---

## Configuration

Environment variables in `.env`:

```bash
# Server
PORT=8080
DEBUG=true

# Database
DATABASE_URL=./dev.db

# Session
SESSION_SECRET=your-secret-key-here

# Security
CSRF_SECRET=your-csrf-secret-here
```

---

## Best Practices

### Security

✅ **DO:**
- Use middleware.RequireAuth for protected routes
- Validate all user input
- Use CSRF protection (built-in)
- Hash passwords (use security.HashPassword)
- Sanitize HTML output (built-in)

❌ **DON'T:**
- Store passwords in plain text
- Trust user input
- Expose sensitive data in logs
- Skip validation

### Code Organization

✅ **DO:**
- One handler per model/feature
- One route file per feature
- Group related templates in folders
- Use meaningful names

❌ **DON'T:**
- Put everything in main.go
- Mix concerns (handlers doing DB + rendering + validation)
- Use generic names (handler1, page2)

### Performance

✅ **DO:**
- Use eager loading for relationships (.WithAuthor())
- Add database indexes on frequently queried fields
- Use pagination for large lists
- Cache static assets

❌ **DON'T:**
- Load all records without limit
- Make N+1 queries
- Skip indexes on foreign keys

---

## Troubleshooting

### Common Issues

**Problem:** Changes not reflecting after edit
- **Solution:** Restart the server (or use `air` for auto-reload)

**Problem:** Template not found
- **Solution:** Check file exists in `gojang/views/templates/` and name matches

**Problem:** Database locked error
- **Solution:** Close other connections to the SQLite database

**Problem:** Build errors after schema change
- **Solution:** Delete generated files and regenerate:
  ```bash
  cd gojang/models
  rm *.go
  go generate ./...
  ```

**Problem:** 404 on route
- **Solution:** Check route is registered in `main.go` and server restarted

---

## Getting Help

1. **Check the tutorials** - Most common tasks are covered
2. **Read existing code** - Look at User and Post models for examples
3. **Check documentation**:
   - [Go Templates](https://pkg.go.dev/html/template)
   - [Ent ORM](https://entgo.io/docs/getting-started)
   - [HTMX](https://htmx.org/docs)
   - [Chi Router](https://github.com/go-chi/chi)
4. **Open an issue** on GitHub

---

## Quick Links

- 📄 [Creating Static Pages](./creating-static-pages.md)
- ⚡ [Quick Start: Adding a Data Model](./quick-start-data-model.md) **← Start here!**
- 📊 [Creating Data Models (Comprehensive)](./creating-data-models.md)
- ⚡ [HTMX Integration Patterns](./htmx-patterns.md)
- 🏗️ [Architecture: Admin & User Separation](./architecture-separation.md)
- 🔑 [Authentication & Authorization Deep Dive](./authentication-authorization.md)
- 🚀 [Deployment Guide](./deployment-guide.md)
- ⚖️ [Distributed Deployment & Load Balancing](./distributed-deployment.md)
- 🧪 [Testing Best Practices](./testing-best-practices.md)
- 🔒 [Security Features](./SECURITY-SUMMARY.md) **← Read before production!**
- 📋 [Taskfile Commands Guide](./taskfile-guide.md)

---
