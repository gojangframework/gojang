# Gojang Framework

v0.1.1 - Initial push from dev repo. Work in progress!

A modern, batteries-included web framework for Go with HTMX. Build dynamic web applications with minimal JavaScript and maximum productivity.

## 🌟 Why Gojang?

- **Batteries Included:** Authentication, admin panel, ORM, security - ready to go
- **HTMX First:** Modern interactions without heavy JavaScript frameworks
- **Developer Joy:** Minimal boilerplate, maximum productivity
- **Type Safe:** Ent ORM catches errors at compile time
- **Production Ready:** Security, logging, and best practices built-in
- **Easy to Learn:** Clear documentation and simple patterns

## ✨ Features

- 🔐 **Authentication & Authorization** - Built-in user system with sessions
- 👥 **User Management** - Complete user CRUD with permissions
- 🎛️ **Auto-Generated Admin Panel** - Automatic CRUD interface for any model
- 📊 **Type-Safe ORM** - Powered by Ent with reflection-based queries
- 🎨 **HTML Templates** - Go templates with layouts and partials
- ⚡ **HTMX Integration** - Dynamic interactions without heavy JavaScript
- � **Security First** - CSRF protection, rate limiting, password hashing
- 🎯 **Simple & Clean** - Minimal boilerplate, maximum productivity
- 🚀 **Production Ready** - Audit logging, middleware, error handling

## 🛠️ Technology Stack

| Technology | Purpose |
|------------|---------|
| **Go 1.21+** | Backend language |
| **HTMX** | Dynamic interactions |
| **Ent** | Type-safe ORM |
| **Chi** | HTTP router |
| **Custom CSS** | Clean, semantic styling |
| **SQLite / PostgreSQL** | Database |

## 📁 Project Structure

```
gojang/
├── admin/             # Auto-generated admin panel
├── cmd/web/           # Application entry point
├── config/            # Configuration management
├── http/
│   ├── handlers/      # Request handlers
│   ├── middleware/    # Auth, security, sessions
│   └── routes/        # Route definitions
├── models/
│   └── schema/        # Database models (define here)
├── views/
│   ├── forms/         # Form validation structs
│   ├── renderers/     # View renderer 
│   ├── templates/     # HTML templates
└── └── static/        # CSS, images
```

## 🚀 Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/gojangframework/gojang
   cd gojang
   ```

2. **Copy environment file:**
   ```bash
   cp .env.example .env
   ```

3. **Install dependencies:**
   ```bash
   go mod download
   ```

4. **Run the application:**
   ```bash
   go run ./gojang/cmd/web
   ```

5. **Visit:** http://localhost:8080

That's it! The database is automatically created and migrated on first run.

## 🌱 First Admin Login (Seed)

You need to run seed program to insert the first admin account
   ```bash
   go run ./gojang/cmd/seed
   ```

## ⚒️ Installation

This project uses [Task](https://taskfile.dev/) for task automation (cross-platform alternative to Make).

### Install Task:

**macOS/Linux:**
```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Or using Homebrew:
```bash
brew install go-task
```

**Windows:**
```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Or using Chocolatey:
```bash
choco install go-task
```

For other installation methods, see the [official Task installation guide](https://taskfile.dev/installation/).

### Install Air (Optional - for live reload):

Air provides automatic reload when code changes, making development faster.

**All platforms:**
```bash
go install github.com/air-verse/air@latest
```

After installation, you can use `task dev` to run the server with live reload.

## 🔧 Development Commands

Run `task --list` to see all available tasks:

```bash
task dev              # Run server with live reload
task build            # Build the application
task test             # Run tests
task migrate          # Run database migrations
task seed             # Seed database with initial data
task schema-gen       # Generate Ent code after schema changes
task addpage          # Create a new static page interactively
task addmodel         # Create a new data model interactively
```

Or use plain Go commands:

```bash
go run ./gojang/cmd/web              # Run server
go build -o app ./gojang/cmd/web     # Build binary
go test ./...                         # Run tests
cd gojang/models && go generate ./... # Generate code
```

## 🤖 Automation Commands

Gojang includes powerful code generators to speed up development:

### Add a New Data Model

Automatically generate a complete CRUD model with handlers, routes, templates, and admin integration:

```bash
task addmodel
# or
go run ./gojang/cmd/addmodel
```

This interactive tool will:
- ✅ Create Ent schema with fields
- ✅ Generate database code
- ✅ Add form validation
- ✅ Create CRUD handlers
- ✅ Set up routes
- ✅ Generate HTML templates
- ✅ Register with admin panel

**Example:**
```bash
$ task addmodel
Model name: Product
Icon: 📦
Fields: name:string, price:float, stock:int
# Creates complete CRUD in seconds!
```

See [Add Model Documentation](./gojang/cmd/addmodel/README.md) for details.

### Add a Static Page

Quickly add a new static page:

```bash
task addpage
# or
go run ./gojang/cmd/addpage
```

Creates template, handler, and route for simple pages like About, Contact, etc.

## 📚 Documentation

Ready to start building? Check out our comprehensive guides:

- **[Creating Static Pages](./docs/creating-static-pages.md)** - Add simple pages like About, Contact (~5 minutes)
- **[Creating Data Models](./docs/creating-data-models.md)** - Full CRUD with database models (~20 minutes)
- **[HTMX Integration Patterns](./docs/htmx-patterns.md)** - Master dynamic interactions with HTMX (~15 minutes)
- **[Documentation Index](./docs/README.md)** - Complete guide with all tutorials

### Quick Examples

**Add a simple page (Automated):**
```bash
go run ./gojang/cmd/addpage
# Interactive prompt creates everything!
```

**Add a data model (Automated):**
```bash
go run ./gojang/cmd/addmodel
# Interactive prompt creates complete CRUD!
```

**Manual approach:**
```go
// 1. Create schema: gojang/models/schema/product.go
// 2. Generate: go generate ./...
// 3. Register admin: registry.RegisterModel(...)
```

See the [documentation](./docs/) for detailed step-by-step guides!

## 🎯 Key Features

### Auto-Generated Admin Panel

Register any model and get a full admin interface automatically:

```go
registry.RegisterModel(ModelRegistration{
    ModelType:      &models.Product{},
    Icon:           "📦",
    NamePlural:     "Products",
    ListFields:     []string{"ID", "Name", "Price"},
    ReadonlyFields: []string{"ID", "CreatedAt"},
})
```

Includes:
- ✅ List view with sorting
- ✅ Create/Edit forms with validation
- ✅ Delete with confirmation
- ✅ Relationship handling
- ✅ Search and filters *(coming soon)*

### HTMX Integration

Dynamic interactions without writing JavaScript:

```html
<button hx-get="/products/load" 
        hx-target="#product-list"
        hx-swap="innerHTML">
    Load Products
</button>
```

### Type-Safe Database

Define schemas once, use everywhere:

```go
// Define schema
field.String("name").NotEmpty()
field.Float("price").Positive()

// Use with type safety
product := client.Product.Create().
    SetName("Widget").
    SetPrice(19.99).
    Save(ctx)
```

## 🤝 Contributing

Contributions are welcome! 

Please feel free to submit a Pull Request or email gojangframework@gmail.com

## 📝 License

BSD 3-Clause "New" or "Revised" License

---
