# Creating Pages with Data Models

This comprehensive guide shows you how to add a complete data model to your Gojang application, including database schema, CRUD operations, and admin panel integration.

> **🎉 Updated October 2025:** The admin panel now uses reflection for automatic CRUD operations!  
> No more manual switch statements in `registry.go` - just register your model and it works automatically.

## 🚀 Automated Model Generation (NEW!)

**Skip manual setup!** Use the `addmodel` command to automatically generate complete CRUD functionality:

```bash
# Interactive mode
task addmodel

# Non-interactive mode with flags
go run ./gojang/cmd/addmodel \
  --model Product \
  --icon "📦" \
  --fields "name:string:required,description:text,price:float:required,stock:int"
```

**New Features:**
- ✅ **Dry-Run Mode** - Preview changes before committing (`--dry-run`)
- ✅ **Reserved Keyword Validation** - Prevents Go keyword conflicts
- ✅ **Command-Line Automation** - Perfect for CI/CD pipelines
- ✅ **Built-in Examples** - Run `--examples` for usage guide
- ✅ **Colorized Output** - Easy-to-read console messages
- ✅ **Timestamp Control** - Optional created_at fields

**Example with dry-run:**
```bash
# Preview what would be created
go run ./gojang/cmd/addmodel \
  --model Article \
  --fields "title:string:required,content:text:required" \
  --dry-run
```

See `gojang/cmd/addmodel/README.md` for comprehensive documentation.

---

## Manual Setup (Advanced)

For complete control over your models or to understand the internals, follow the manual steps below.

## Overview

Adding a new data model involves these steps:
1. Define the Ent schema
2. Generate the database code
3. Run migrations
4. Create handlers and routes
5. Create templates
6. Register with admin panel (**now automatic!**)

**Estimated time:** ~~20-30 minutes~~ **→ 10-15 minutes per model** ⚡ (or instant with `addmodel` command!)

---

## Example: Creating a "SampleProduct" Model

Let's build a complete SampleProduct catalog feature with full CRUD capabilities.

---

## Step 1: Define the Ent Schema

Ent is the ORM used by Gojang. Schemas are defined in `gojang/models/schema/`.

### Create `gojang/models/schema/sampleproduct.go`

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SampleProduct holds the schema definition for the SampleProduct entity.
type SampleProduct struct {
	ent.Schema
}

// Fields of the SampleProduct.
func (SampleProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(200),
		
		field.Text("description").
			Optional(),
		
		field.Float("price").
			Positive().
			Comment("Price in USD"),
		
		field.Int("stock").
			Default(0).
			NonNegative().
			Comment("Current inventory count"),
		
		field.String("sku").
			Unique().
			NotEmpty().
			MaxLen(100).
			Comment("Stock Keeping Unit"),
		
		field.Bool("is_active").
			Default(true).
			Comment("Whether product is visible to customers"),
		
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the SampleProduct (relationships).
func (SampleProduct) Edges() []ent.Edge {
	return []ent.Edge{
		// SampleProduct belongs to a User (creator)
		edge.From("creator", User.Type).
			Ref("sample_sampleproducts").
			Unique().
			Required(),
	}
}
```

### Update User Schema to Add Relationship

Edit `gojang/models/schema/user.go` and add the sample_sampleproducts edge:

```go
// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("posts", Post.Type),
		edge.To("sample_sampleproducts", SampleSampleProduct.Type),  // ✅ Add this line
	}
}
```

### Field Type Reference

```go
// String fields
field.String("name").NotEmpty().MaxLen(255)
field.Text("description").Optional()

// Numeric fields
field.Int("quantity").Default(0)
field.Float("price").Positive()
field.Float32("rating").Min(0).Max(5)

// Boolean fields
field.Bool("is_active").Default(true)

// Time fields
field.Time("created_at").Default(time.Now).Immutable()
field.Time("expires_at").Optional()

// Enum fields
field.Enum("status").Values("draft", "published", "archived").Default("draft")

// JSON fields
field.JSON("metadata", map[string]interface{}{}).Optional()

// Unique constraints
field.String("email").Unique()

// Sensitive fields (excluded from JSON)
field.String("password_hash").Sensitive()
```

---

## Step 2: Generate Code and Migrate

### Generate Ent Code

```bash
cd gojang/models
go generate ./...
```

This creates:
- ✅ `sampleproduct.go` - The SampleProduct model
- ✅ `sampleproduct_create.go` - Create builder
- ✅ `sampleproduct_update.go` - Update builder
- ✅ `sampleproduct_query.go` - Query builder
- ✅ `sampleproduct_delete.go` - Delete builder
- ✅ `sampleproduct/sampleproduct.go` - Predicates and constants

### Run Auto-Migration

The application automatically migrates on startup, but you can also run migrations manually:

```bash
go run ./gojang/cmd/web
```

Look for the migration log:
```
✅ Auto-migration completed
```

### Manual Migration (Alternative)

If you need more control, create a migration file:

```bash
# Create migrations directory if it doesn't exist
mkdir -p gojang/models/migrations

# Create migration file
cat > gojang/models/migrations/000003_create_sample_sampleproducts.up.sql << EOF
CREATE TABLE sample_sampleproducts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price REAL NOT NULL,
    stock INTEGER DEFAULT 0 NOT NULL,
    sku VARCHAR(100) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    creator_id INTEGER NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_sample_sampleproducts_sku ON sample_sampleproducts(sku);
CREATE INDEX idx_sample_sampleproducts_creator ON sample_sampleproducts(creator_id);
EOF

# Create down migration
cat > gojang/models/migrations/000003_create_sample_sampleproducts.down.sql << EOF
DROP TABLE sample_sampleproducts;
EOF
```

---

## Step 3: Create Form Validation Structs

Forms are defined in `gojang/views/forms/forms.go`.

### Add to `gojang/views/forms/forms.go`

```go
// SampleProductForm is used for creating/editing sample sampleproducts
type SampleProductForm struct {
	Name        string  `form:"name" validate:"required,max=200"`
	Description string  `form:"description"`
	Price       float64 `form:"price" validate:"required,gt=0"`
	Stock       int     `form:"stock" validate:"gte=0"`
	SKU         string  `form:"sku" validate:"required,max=100"`
	IsActive    bool    `form:"is_active"`
}
```

### Validation Tags Reference

```go
validate:"required"              // Field cannot be empty
validate:"email"                 // Must be valid email
validate:"min=3,max=50"          // String length between 3-50
validate:"gte=0"                 // Number >= 0
validate:"gt=0"                  // Number > 0
validate:"oneof=draft published" // Must be one of these values
validate:"url"                   // Must be valid URL
validate:"alphanum"              // Only letters and numbers
```

---

## Step 4: Create Handlers

Handlers are located in `gojang/http/handlers/`. Create a new file for your model.

### Create `gojang/http/handlers/samplesampleproducts.go`

```go
package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/views/forms"
	"github.com/gojangframework/gojang/gojang/views/renderers"
)

type SampleSampleProductHandler struct {
	Client   *models.Client
	Renderer *renderers.Renderer
}

func NewSampleSampleProductHandler(client *models.Client, renderer *renderers.Renderer) *SampleSampleProductHandler {
	return &SampleSampleProductHandler{
		Client:   client,
		Renderer: renderer,
	}
}

// Index lists all sample sampleproducts
func (h *SampleSampleProductHandler) Index(w http.ResponseWriter, r *http.Request) {
	samplesampleproducts, err := h.Client.SampleProduct.Query().
		WithCreator(). // Eager load creator
		Order(models.Desc("created_at")).
		All(r.Context())
	
	if err != nil {
		log.Printf("Error loading sampleproducts: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load sampleproducts")
		return
	}

	h.Renderer.Render(w, r, "sampleproducts/index.html", &renderers.TemplateData{
		Title: "Sample Products",
		Data: map[string]interface{}{
			"Sample Products": sampleproducts,
		},
	})
}

// New shows the create form
func (h *SampleProductHandler) New(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "sampleproducts/new.partial.html", &renderers.TemplateData{
		Title: "New Sample Product",
		Data:  map[string]interface{}{},
	})
}

// Create handles product creation
func (h *SampleProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Parse and validate form
	var form forms.SampleProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		h.Renderer.Render(w, r, "sampleproducts/new.partial.html", &renderers.TemplateData{
			Title: "New Sample Product",
			Data: map[string]interface{}{
				"Errors": err,
				"Form":   form,
			},
		})
		return
	}

	// Get current user from context
	user, _ := middleware.GetUserFromContext(r.Context())

	// Create sample product
	sampleproduct, err := h.Client.SampleProduct.Create().
		SetName(form.Name).
		SetDescription(form.Description).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetSKU(form.SKU).
		SetIsActive(form.IsActive).
		SetCreatorID(user.ID).
		Save(r.Context())

	if err != nil {
		log.Printf("Error creating sample product: %v", err)
		h.Renderer.Render(w, r, "sampleproducts/new.partial.html", &renderers.TemplateData{
			Title: "New Sample Product",
			Data: map[string]interface{}{
				"Error": "Failed to create sample product",
				"Form":  form,
			},
		})
		return
	}

	// Redirect to sample product list with success message
	http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
}

// Edit shows the edit form
func (h *SampleProductHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	sampleproduct, err := h.Client.SampleProduct.Get(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Sample product not found")
		return
	}

	h.Renderer.Render(w, r, "sampleproducts/edit.partial.html", &renderers.TemplateData{
		Title: "Edit Sample Product",
		Data: map[string]interface{}{
			"SampleProduct": sampleproduct,
		},
	})
}

// Update handles product updates
func (h *SampleProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	var form forms.SampleProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		product, _ := h.Client.SampleProduct.Get(r.Context(), id)
		h.Renderer.Render(w, r, "sampleproducts/edit.partial.html", &renderers.TemplateData{
			Title: "Edit Sample Product",
			Data: map[string]interface{}{
				"Errors":  err,
				"SampleProduct": sampleproduct,
				"Form":    form,
			},
		})
		return
	}

	// Update sample product
	_, err := h.Client.SampleProduct.UpdateOneID(id).
		SetName(form.Name).
		SetDescription(form.Description).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetSKU(form.SKU).
		SetIsActive(form.IsActive).
		Save(r.Context())

	if err != nil {
		log.Printf("Error updating sample product: %v", err)
		product, _ := h.Client.SampleProduct.Get(r.Context(), id)
		h.Renderer.Render(w, r, "sampleproducts/edit.partial.html", &renderers.TemplateData{
			Title: "Edit Sample Product",
			Data: map[string]interface{}{
				"Error":   "Failed to update sample product",
				"SampleProduct": sampleproduct,
			},
		})
		return
	}

	http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
}

// Delete handles product deletion
func (h *SampleProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := h.Client.SampleProduct.DeleteOneID(id).Exec(r.Context()); err != nil {
		log.Printf("Error deleting product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete product")
		return
	}

	http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
}
```

---

## Step 5: Create Routes

Routes are organized by feature. Create a new route file.

### Create `gojang/http/routes/sampleproducts.go`

```go
package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
)

// RegisterSampleProductRoutes registers all product-related routes
func RegisterSampleProductRoutes(r chi.Router, handler *handlers.SampleProductHandler) {
	r.Route("/sampleproducts", func(r chi.Router) {
		// Public routes (anyone can view)
		r.Get("/", handler.Index)

		// Protected routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)

			r.Get("/new", handler.New)
			r.Post("/", handler.Create)
			r.Get("/{id}/edit", handler.Edit)
			r.Put("/{id}", handler.Update)
			r.Delete("/{id}", handler.Delete)
		})
	})
}
```

### Register Routes in Main Application

Edit `gojang/cmd/web/main.go` and add your routes:

```go
// Find the route registration section and add:
sampleProductHandler := handlers.NewSampleProductHandler(client, renderer)
routes.RegisterSampleProductRoutes(router, sampleProductHandler)
```

---

## Step 6: Create Templates

Templates are located in `gojang/views/templates/`. Create a new directory for your model.

### Create Directory Structure

```bash
mkdir -p gojang/views/templates/sampleproducts
```

### Create `gojang/views/templates/sampleproducts/index.html`

```html
{{define "title"}}Sample Products{{end}}

{{define "content"}}
<div class="container">
    <div class="page-header">
        <h1>Sample Products</h1>
        {{if .User}}
        <a href="/sampleproducts/new" class="btn btn-primary">
            + New Sample Product
        </a>
        {{end}}
    </div>

    {{if .Data.SampleProducts}}
    <div class="product-grid">
        {{range .Data.SampleProducts}}
        <div class="card">
            <h2>{{.Name}}</h2>
            <p>{{.Description}}</p>
            
            <div class="product-details">
                <span class="price">${{printf "%.2f" .Price}}</span>
                <span class="badge {{if gt .Stock 0}}badge-success{{else}}badge-danger{{end}}">
                    {{if gt .Stock 0}}
                        {{.Stock}} in stock
                    {{else}}
                        Out of stock
                    {{end}}
                </span>
            </div>
            
            <div class="product-meta">
                SKU: {{.SKU}}<br>
                Created by: {{.Edges.Creator.Email}}<br>
                {{.CreatedAt.Format "Jan 2, 2006"}}
            </div>
            
            {{if $.User}}
            <div class="actions">
                <a href="/sampleproducts/{{.ID}}/edit" class="btn btn-primary btn-sm">
                    Edit
                </a>
                <form method="POST" action="/sampleproducts/{{.ID}}" style="display: inline;">
                    <input type="hidden" name="_method" value="DELETE">
                    <button type="submit" 
                            onclick="return confirm('Are you sure?')"
                            class="btn btn-danger btn-sm">
                        Delete
                    </button>
                </form>
            </div>
            {{end}}
        </div>
        {{end}}
    </div>
    {{else}}
    <div class="card" style="text-align: center;">
        <p>No sampleproducts found</p>
        {{if .User}}
        <a href="/sampleproducts/new" class="btn btn-primary">
            Create your first product →
        </a>
        {{end}}
    </div>
    {{end}}
</div>

<style>
    .product-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 1.5rem;
    }
    .product-details {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin: 1rem 0;
    }
    .price {
        font-size: 1.5rem;
        font-weight: bold;
        color: var(--primary);
    }
    .product-meta {
        font-size: 0.875rem;
        color: var(--secondary);
        margin-bottom: 1rem;
    }
</style>
{{end}}
```

### Create `gojang/views/templates/sampleproducts/new.partial.html`

```html
{{define "title"}}New Sample Product{{end}}

{{define "content"}}
<div class="container">
    <h1>New Sample Product</h1>

    {{if .Data.Error}}
    <div class="alert alert-error">
        <p>{{.Data.Error}}</p>
    </div>
    {{end}}

    <form method="POST" action="/sampleproducts" class="form">
        <div class="form-group">
            <label for="name">Sample Product Name *</label>
            <input type="text" 
                   id="name" 
                   name="name" 
                   value="{{if .Data.Form}}{{.Data.Form.Name}}{{end}}"
                   required>
            {{if .Data.Errors}}
                {{if index .Data.Errors "name"}}
                <span class="error">{{index .Data.Errors "name"}}</span>
                {{end}}
            {{end}}
        </div>

        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" 
                      name="description" 
                      rows="4">{{if .Data.Form}}{{.Data.Form.Description}}{{end}}</textarea>
        </div>

        <div class="form-row">
            <div class="form-group">
                <label for="price">Price ($) *</label>
                <input type="number" 
                       id="price" 
                       name="price" 
                       step="0.01"
                       min="0"
                       value="{{if .Data.Form}}{{.Data.Form.Price}}{{end}}"
                       required>
                {{if .Data.Errors}}
                    {{if index .Data.Errors "price"}}
                    <span class="error">{{index .Data.Errors "price"}}</span>
                    {{end}}
                {{end}}
            </div>

            <div class="form-group">
                <label for="stock">Stock</label>
                <input type="number" 
                       id="stock" 
                       name="stock" 
                       min="0"
                       value="{{if .Data.Form}}{{.Data.Form.Stock}}{{else}}0{{end}}">
            </div>
        </div>

        <div class="form-group">
            <label for="sku">SKU *</label>
            <input type="text" 
                   id="sku" 
                   name="sku" 
                   value="{{if .Data.Form}}{{.Data.Form.SKU}}{{end}}"
                   required>
            {{if .Data.Errors}}
                {{if index .Data.Errors "sku"}}
                <span class="error">{{index .Data.Errors "sku"}}</span>
                {{end}}
            {{end}}
        </div>

        <div class="form-group">
            <label class="checkbox">
                <input type="checkbox" 
                       name="is_active" 
                       value="true"
                       {{if or (not .Data.Form) .Data.Form.IsActive}}checked{{end}}>
                Active (visible to customers)
            </label>
        </div>

        <div class="form-actions">
            <button type="submit" class="btn btn-primary">
                Create Sample Product
            </button>
            <a href="/sampleproducts" class="btn btn-secondary">
                Cancel
            </a>
        </div>
    </form>
</div>

<style>
    .form-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 1rem;
    }
    .form-actions {
        display: flex;
        gap: 1rem;
        margin-top: 1.5rem;
    }
    .form-actions .btn {
        flex: 1;
    }
</style>
{{end}}
```

### Create `gojang/views/templates/sampleproducts/edit.partial.html`

```html
{{define "title"}}Edit Sample Product{{end}}

{{define "content"}}
<div class="max-w-2xl mx-auto px-4 py-8">
    <h1 class="text-3xl font-bold mb-6">Edit Sample Product</h1>

    {{if .Data.Error}}
    <div class="bg-red-50 border-l-4 border-red-500 p-4 mb-6">
        <p class="text-red-700">{{.Data.Error}}</p>
    </div>
    {{end}}

    <form method="POST" action="/sampleproducts/{{.Data.SampleProduct.ID}}" class="bg-white shadow-md rounded-lg p-6">
        <input type="hidden" name="_method" value="PUT">
        
        <!-- Same form fields as new.partial.html, but with SampleProduct data -->
        <div class="mb-4">
            <label for="name" class="block text-gray-700 font-bold mb-2">
                Sample Product Name *
            </label>
            <input type="text" 
                   id="name" 
                   name="name" 
                   value="{{if .Data.Form}}{{.Data.Form.Name}}{{else}}{{.Data.SampleProduct.Name}}{{end}}"
                   required
                   class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
        </div>

        <!-- ... rest of form fields ... -->
        <!-- (Copy from new.partial.html and replace .Data.Form with .Data.SampleProduct) -->

        <div class="flex space-x-4">
            <button type="submit" 
                    class="flex-1 bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
                Update Sample Product
            </button>
            <a href="/sampleproducts" 
               class="flex-1 bg-gray-300 hover:bg-gray-400 text-gray-800 font-bold py-2 px-4 rounded text-center">
                Cancel
            </a>
        </div>
    </form>
</div>
{{end}}
```

---

## Step 7: Register with Admin Panel

The admin panel provides automatic CRUD interface for your models.

### Add to `gojang/admin/models.go`

```go
func RegisterModels(registry *Registry) {
	// Existing User model
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.User{},
		// ... existing config ...
	})

	// Existing Post model
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.Post{},
		// ... existing config ...
	})

	// ✅ Add SampleProduct model
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.SampleProduct{},
		Icon:           "📦",
		NamePlural:     "Sample Products",
		ListFields:     []string{"ID", "Name", "Price", "Stock", "SKU", "IsActive"},
		ReadonlyFields: []string{"ID", "CreatedAt", "UpdatedAt"},
		
		// Eager load creator relationship
		QueryModifier: func(ctx context.Context, query interface{}) interface{} {
			if q, ok := query.(*models.ProductQuery); ok {
				return q.WithCreator()
			}
			return query
		},
	})
}
```

---

## Step 8: Test Your New Model

### 1. Restart the Server

```bash
go run ./gojang/cmd/web
```

### 2. Test Public Routes

- Visit http://localhost:8080/sampleproducts
- Should see empty state or list of sampleproducts

### 3. Test CRUD Operations

1. **Create:** http://localhost:8080/sampleproducts/new
2. **List:** http://localhost:8080/sampleproducts
3. **Edit:** Click "Edit" button
4. **Delete:** Click "Delete" button

### 4. Test Admin Panel

1. Visit http://localhost:8080/admin
2. Should see "Products 📦" in sidebar
3. Click to manage sampleproducts via admin interface

---

## Advanced Patterns

### Adding Search/Filter

```go
func (h *SampleProductHandler) Index(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	
	productQuery := h.Client.SampleProduct.Query().WithCreator()
	
	if query != "" {
		productQuery = productQuery.Where(
			models.Or(
				models.ProductNameContains(query),
				models.ProductSKUContains(query),
			),
		)
	}
	
	sampleproducts, err := productQuery.All(r.Context())
	// ... rest of handler
}
```

### Adding Pagination

```go
page, _ := strconv.Atoi(r.URL.Query().Get("page"))
if page < 1 {
	page = 1
}
limit := 20
offset := (page - 1) * limit

sampleproducts, err := h.Client.SampleProduct.Query().
	WithCreator().
	Limit(limit).
	Offset(offset).
	All(r.Context())

count, _ := h.Client.SampleProduct.Query().Count(r.Context())
totalPages := (count + limit - 1) / limit
```

### Adding Image Upload

1. **Add field to schema:**

```go
field.String("image_url").Optional()
```

2. **Handle file upload in handler:**

```go
file, header, err := r.FormFile("image")
if err == nil {
	defer file.Close()
	// Save file and get URL
	imageURL := saveUploadedFile(file, header)
	builder.SetImageURL(imageURL)
}
```

### Adding Soft Delete

```go
// In schema
field.Time("deleted_at").Optional().Nillable()

// In handler
_, err := h.Client.SampleProduct.UpdateOneID(id).
	SetDeletedAt(time.Now()).
	Save(r.Context())
```

---

## Complete Checklist

When adding a new model, use this checklist:

- [ ] Create Ent schema in `gojang/models/schema/`
- [ ] Add relationships to related schemas
- [ ] Run `go generate ./...` in models directory
- [ ] Create form struct in `gojang/views/forms/forms.go`
- [ ] Create handler in `gojang/http/handlers/`
- [ ] Create routes in `gojang/http/routes/`
- [ ] Register routes in `gojang/cmd/web/main.go`
- [ ] Create templates in `gojang/views/templates/[model]/`
- [ ] Register with admin panel in `gojang/admin/models.go`
- [ ] ~~Add case statements in `gojang/admin/registry.go`~~ ✅ **No longer needed!**
- [ ] Test CRUD operations
- [ ] Test admin panel integration
- [ ] Add navigation links (optional)
- [ ] Add search/filter (optional)
- [ ] Add pagination (optional)

---

## Troubleshooting

### Build Errors After Schema Changes

```bash
cd gojang/models
rm -rf *.go
go generate ./...
```

### Migration Fails

Check for:
- ✅ Unique constraints violated
- ✅ Foreign key relationships correct
- ✅ Field types compatible with database

### Handler Not Found

- ✅ Check handler is created in `handlers/`
- ✅ Check handler is registered in `main.go`
- ✅ Restart server after changes

### Template Not Rendering

- ✅ Check template exists in correct directory
- ✅ Check `{{define "title"}}` and `{{define "content"}}` exist
- ✅ Check field names match model struct

### Admin Panel Not Showing Model

- ✅ Check model registered in `models.go`
- ✅ ~~Check case statements added to all methods in `registry.go`~~ No longer needed!
- ✅ Verify model name matches Ent client field (e.g., `client.SampleProduct`)
- ✅ Restart server

---

## Next Steps

- ✅ **Read:** [Creating Static Pages](./creating-static-pages.md)
- ✅ **Learn:** [HTMX Integration Patterns](./htmx-patterns.md)
- ✅ **Explore:** Check existing models in `gojang/models/schema/` for more examples
- ✅ **Advanced:** [Ent Documentation](https://entgo.io/docs/getting-started)

---

## Quick Reference

| Step | File | Action |
|------|------|--------|
| 1. Schema | `gojang/models/schema/model.go` | Define fields and edges |
| 2. Generate | Terminal | `cd gojang/models && go generate ./...` |
| 3. Form | `gojang/views/forms/forms.go` | Add validation struct |
| 4. Handler | `gojang/http/handlers/model.go` | Create CRUD handlers |
| 5. Routes | `gojang/http/routes/model.go` | Define URL patterns |
| 6. Main | `gojang/cmd/web/main.go` | Register routes |
| 7. Templates | `gojang/views/templates/model/` | Create HTML views |
| 8. Admin | `gojang/admin/models.go` | Register model (auto CRUD!) |
| ~~9. Registry~~ | ~~`gojang/admin/registry.go`~~ | ~~Add case statements~~ ✅ **Removed!** |

---
