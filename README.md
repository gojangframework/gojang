# Gojang Framework

v0.3.1 - Email verification for new registration, forgot/reset password process, AWS SES email implemntation

v0.3 - New file structure - separating dev files from Gojang framework files - AI Skills added - Docs updated - addModel and addPage automations removed - Ent schema files moved to more convenient location

v0.2 - Added some changes based on what I learned from another production app that has been built on top of Gojang over the past eight months. I learned a lot about Gojang’s pros and cons, and I hope to make many improvements based on my experience running the app in production.

A modern, batteries-included web framework for Go with HTMX. Build dynamic web applications with minimal JavaScript and maximum productivity.

## 🌟 Why Gojang?

- **AI Skills:** Start AI development from strong boilerplate, save time and tokens - Claude and Codex compatible
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
app/
├── cmd/               # Application commands
│   ├── migrate/
│   ├── seed/
│   └── web/
├── gojang/            # Framework core, admin, auth, models, renderers
├── pages/             # App-owned page handlers, routes, and templates
├── posts/             # App-owned post handlers, routes, and templates
└── views/
    ├── i18n/          # Public translation files
    └── static/        # Public CSS, images
```

## 🚀 Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/gojangframework/gojang
   ```

2. **Copy environment file:**
   ```bash
   cp .env.example .env
   ```

3. **Install dependencies:**
   ```bash
   go mod download
   ```

4. **Run Ent schema generation:**
   ```bash
   task schema-gen
   ```
5. **Run the application in dev mode:**
   ```bash
   task dev
   ```

6. **Visit:** http://localhost:8080

That's it! The database is automatically created and migrated on first run.

## 🌱 First Admin Login (Seed)

You need to run seed program to insert the first admin account
   ```bash
   task seed
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
```

Or use plain Go commands:

```bash
go run ./app/cmd/web                 # Run server
go build -o app ./app/cmd/web        # Build binary
go test ./...                         # Run tests
go generate ./app/gojang/models     # Generate code
```

## 📚 Documentation

Ready to start building? Check out our comprehensive guides:

- **[Creating Static Pages](./docs/creating-static-pages.md)** - Add simple pages like About, Contact (~5 minutes)
- **[Creating Data Models](./docs/creating-data-models.md)** - Full CRUD with database models (~20 minutes)
- **[HTMX Integration Patterns](./docs/htmx-patterns.md)** - Master dynamic interactions with HTMX (~15 minutes)
- **[Documentation Index](./docs/README.md)** - Complete guide with all tutorials

### Quick Examples
```go
// 1. Create schema: app/schema/product.go
// 2. Generate: go generate ./app/gojang/models
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
