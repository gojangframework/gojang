# Creating Pages with Data Models

This comprehensive guide shows you how to add a complete data model to your Gojang application, including database schema, CRUD operations, and admin panel integration.

> **🎉 Updated October 2025:** The admin panel now uses reflection for automatic CRUD operations!
> No more manual switch statements in `registry.go` - just register your model and it works automatically.

## Overview

Adding a new data model involves these steps:
1. Define the Ent schema
2. Generate the database code
3. Run migrations
4. Create handlers and routes
5. Create templates
6. Register with admin panel (**now automatic!**)

**Estimated time:** 10-15 minutes per model

---

## Example: Creating a "Product" Model

Let's build a complete Product catalog feature with full CRUD capabilities.

---

## Step 1: Define the Ent Schema

Ent is the ORM used by Gojang. Schemas are defined in `app/schema/`.

### Create `app/schema/product.go`

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Product holds the schema definition for the Product entity.
type Product struct {
	ent.Schema
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
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

// Edges of the Product (relationships).
func (Product) Edges() []ent.Edge {
	return []ent.Edge{
		// Product belongs to a User (creator)
		edge.From("creator", User.Type).
			Ref("products").
			Unique().
			Required(),
	}
}
```

### Update User Schema to Add Relationship

Edit `app/schema/user.go` and add the products edge:

```go
// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("posts", Post.Type),
		edge.To("products", Product.Type),  // ✅ Add this line
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
go generate ./app/gojang/models
```

This creates:
- ✅ `product.go` - The Product model
- ✅ `product_create.go` - Create builder
- ✅ `product_update.go` - Update builder
- ✅ `product_query.go` - Query builder
- ✅ `product_delete.go` - Delete builder
- ✅ `product/product.go` - Predicates and constants

### Run Auto-Migration

The application automatically migrates on startup, but you can also run migrations manually:

```bash
go run ./app/cmd/web
```

Look for the migration log:
```
✅ Auto-migration completed
```

### Manual Migration (Alternative)

If you need more control, create a migration file:

```bash
# Create migrations directory if it doesn't exist
mkdir -p app/gojang/models/migrations

# Create migration file
cat > app/gojang/models/migrations/000003_create_products.up.sql << EOF
CREATE TABLE products (
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

CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_creator ON products(creator_id);
EOF

# Create down migration
cat > app/gojang/models/migrations/000003_create_products.down.sql << EOF
DROP TABLE products;
EOF
```

---

## Step 3: Create Form Validation Structs

Forms are defined in `app/views/forms/forms.go`.

### Add to `app/views/forms/forms.go`

```go
// ProductForm is used for creating/editing products
type ProductForm struct {
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

Feature handlers live with their feature package. Create a package under `app/products/` for the Product handler, routes, and templates.

### Create `app/products/products.handler.go`

```go
package products

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/app/gojang/http/middleware"
	"github.com/gojangframework/gojang/app/gojang/models"
	"github.com/gojangframework/gojang/app/views/forms"
	"github.com/gojangframework/gojang/app/gojang/views/renderers"
)

type ProductHandler struct {
	Client   *models.Client
	Renderer *renderers.Renderer
}

func NewProductHandler(client *models.Client, renderer *renderers.Renderer) *ProductHandler {
	return &ProductHandler{
		Client:   client,
		Renderer: renderer,
	}
}

// Index lists all products
func (h *ProductHandler) Index(w http.ResponseWriter, r *http.Request) {
	products, err := h.Client.Product.Query().
		WithCreator(). // Eager load creator
		Order(models.Desc("created_at")).
		All(r.Context())

	if err != nil {
		log.Printf("Error loading products: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load products")
		return
	}

	h.Renderer.Render(w, r, "products/index.html", &renderers.TemplateData{
		Title: "Products",
		Data: map[string]interface{}{
			"Products": products,
		},
	})
}

// New shows the create form
func (h *ProductHandler) New(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "products/new.partial.html", &renderers.TemplateData{
		Title: "New Product",
		Data:  map[string]interface{}{},
	})
}

// Create handles product creation
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Parse and validate form
	var form forms.ProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		h.Renderer.Render(w, r, "products/new.partial.html", &renderers.TemplateData{
			Title: "New Product",
			Data: map[string]interface{}{
				"Errors": err,
				"Form":   form,
			},
		})
		return
	}

	// Get current user from context
	user := middleware.GetUser(r.Context())

	// Create product
	product, err := h.Client.Product.Create().
		SetName(form.Name).
		SetDescription(form.Description).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetSKU(form.SKU).
		SetIsActive(form.IsActive).
		SetCreatorID(user.ID).
		Save(r.Context())

	if err != nil {
		log.Printf("Error creating product: %v", err)
		h.Renderer.Render(w, r, "products/new.partial.html", &renderers.TemplateData{
			Title: "New Product",
			Data: map[string]interface{}{
				"Error": "Failed to create product",
				"Form":  form,
			},
		})
		return
	}

	// Redirect to product list with success message
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

// Edit shows the edit form
func (h *ProductHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	product, err := h.Client.Product.Get(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Product not found")
		return
	}

	h.Renderer.Render(w, r, "products/edit.partial.html", &renderers.TemplateData{
		Title: "Edit Product",
		Data: map[string]interface{}{
			"Product": product,
		},
	})
}

// Update handles product updates
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	var form forms.ProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		product, _ := h.Client.Product.Get(r.Context(), id)
		h.Renderer.Render(w, r, "products/edit.partial.html", &renderers.TemplateData{
			Title: "Edit Product",
			Data: map[string]interface{}{
				"Errors":  err,
				"Product": product,
				"Form":    form,
			},
		})
		return
	}

	// Update product
	_, err := h.Client.Product.UpdateOneID(id).
		SetName(form.Name).
		SetDescription(form.Description).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetSKU(form.SKU).
		SetIsActive(form.IsActive).
		Save(r.Context())

	if err != nil {
		log.Printf("Error updating product: %v", err)
		product, _ := h.Client.Product.Get(r.Context(), id)
		h.Renderer.Render(w, r, "products/edit.partial.html", &renderers.TemplateData{
			Title: "Edit Product",
			Data: map[string]interface{}{
				"Error":   "Failed to update product",
				"Product": product,
			},
		})
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

// Delete handles product deletion
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := h.Client.Product.DeleteOneID(id).Exec(r.Context()); err != nil {
		log.Printf("Error deleting product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete product")
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}
```

---

## Step 5: Create Routes

Routes are organized by feature. Create a new route file.

### Create `app/products/products.route.go`

```go
package products

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"

	"github.com/gojangframework/gojang/app/gojang/http/middleware"
	"github.com/gojangframework/gojang/app/gojang/models"
)

func ProductRoutes(handler *ProductHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()

	// Public routes (anyone can view)
	r.Get("/", handler.Index)

	// Protected routes (require authentication)
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))

		auth.Get("/new", handler.New)
		auth.Post("/", handler.Create)
		auth.Get("/{id}/edit", handler.Edit)
		auth.Put("/{id}", handler.Update)
		auth.Delete("/{id}", handler.Delete)
	})

	return r
}
```

### Register Routes in Main Application

Edit `app/cmd/web/main.go` and add your routes:

```go
// Add this import:
// "github.com/gojangframework/gojang/app/products"

// Find the route registration section and add:
productHandler := products.NewProductHandler(client, publicRenderer)
r.Mount("/products", products.ProductRoutes(productHandler, sessionManager, client))
```

---

## Step 6: Create Templates

Templates are located in `app/views/templates/`. Create a new directory for your model.

### Create Directory Structure

```bash
mkdir -p app/products/templates
```

### Create `app/products/templates/index.html`

```html
{{define "title"}}Products{{end}}

{{define "content"}}
<div class="container">
    <div class="page-header">
        <h1>Products</h1>
        {{if .User}}
        <a href="/products/new" class="btn btn-primary">
            + New Product
        </a>
        {{end}}
    </div>

    {{if .Data.Products}}
    <div class="product-grid">
        {{range .Data.Products}}
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
                <a href="/products/{{.ID}}/edit" class="btn btn-primary btn-sm">
                    Edit
                </a>
                <form method="POST" action="/products/{{.ID}}" style="display: inline;">
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
        <p>No products found</p>
        {{if .User}}
        <a href="/products/new" class="btn btn-primary">
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

### Create `app/products/templates/new.partial.html`

```html
{{define "title"}}New Product{{end}}

{{define "content"}}
<div class="container">
    <h1>New Product</h1>

    {{if .Data.Error}}
    <div class="alert alert-error">
        <p>{{.Data.Error}}</p>
    </div>
    {{end}}

    <form method="POST" action="/products" class="form">
        <div class="form-group">
            <label for="name">Product Name *</label>
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
                Create Product
            </button>
            <a href="/products" class="btn btn-secondary">
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

### Create `app/products/templates/edit.partial.html`

```html
{{define "title"}}Edit Product{{end}}

{{define "content"}}
<div class="max-w-2xl mx-auto px-4 py-8">
    <h1 class="text-3xl font-bold mb-6">Edit Product</h1>

    {{if .Data.Error}}
    <div class="bg-red-50 border-l-4 border-red-500 p-4 mb-6">
        <p class="text-red-700">{{.Data.Error}}</p>
    </div>
    {{end}}

    <form method="POST" action="/products/{{.Data.Product.ID}}" class="bg-white shadow-md rounded-lg p-6">
        <input type="hidden" name="_method" value="PUT">

        <!-- Same form fields as new.partial.html, but with Product data -->
        <div class="mb-4">
            <label for="name" class="block text-gray-700 font-bold mb-2">
                Product Name *
            </label>
            <input type="text"
                   id="name"
                   name="name"
                   value="{{if .Data.Form}}{{.Data.Form.Name}}{{else}}{{.Data.Product.Name}}{{end}}"
                   required
                   class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
        </div>

        <!-- ... rest of form fields ... -->
        <!-- (Copy from new.partial.html and replace .Data.Form with .Data.Product) -->

        <div class="flex space-x-4">
            <button type="submit"
                    class="flex-1 bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
                Update Product
            </button>
            <a href="/products"
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

### Add to `app/gojang/admin/models.go`

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

	// ✅ Add Product model
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.Product{},
		Icon:           "📦",
		NamePlural:     "Products",
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
go run ./app/cmd/web
```

### 2. Test Public Routes

- Visit http://localhost:8080/products
- Should see empty state or list of products

### 3. Test CRUD Operations

1. **Create:** http://localhost:8080/products/new
2. **List:** http://localhost:8080/products
3. **Edit:** Click "Edit" button
4. **Delete:** Click "Delete" button

### 4. Test Admin Panel

1. Visit http://localhost:8080/admin
2. Should see "Products 📦" in sidebar
3. Click to manage products via the admin workspace, or visit http://localhost:8080/admin/t/product directly

---

## Advanced Patterns

### Adding Search/Filter

```go
func (h *ProductHandler) Index(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	productQuery := h.Client.Product.Query().WithCreator()

	if query != "" {
		productQuery = productQuery.Where(
			models.Or(
				models.ProductNameContains(query),
				models.ProductSKUContains(query),
			),
		)
	}

	products, err := productQuery.All(r.Context())
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

products, err := h.Client.Product.Query().
	WithCreator().
	Limit(limit).
	Offset(offset).
	All(r.Context())

count, _ := h.Client.Product.Query().Count(r.Context())
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
_, err := h.Client.Product.UpdateOneID(id).
	SetDeletedAt(time.Now()).
	Save(r.Context())
```

---

## Complete Checklist

When adding a new model, use this checklist:

- [ ] Create Ent schema in `app/schema/`
- [ ] Add relationships to related schemas
- [ ] Run `go generate ./app/gojang/models`
- [ ] Create form struct in `app/views/forms/forms.go`
- [ ] Create handler in `app/products/products.handler.go`
- [ ] Create routes in `app/products/products.route.go`
- [ ] Register routes in `app/cmd/web/main.go`
- [ ] Create templates in `app/products/templates/`
- [ ] Register with admin panel in `app/gojang/admin/models.go`
- [ ] ~~Add case statements in `app/gojang/admin/registry.go`~~ ✅ **No longer needed!**
- [ ] Test CRUD operations
- [ ] Test admin panel integration
- [ ] Add navigation links (optional)
- [ ] Add search/filter (optional)
- [ ] Add pagination (optional)

---

## Troubleshooting

### Build Errors After Schema Changes

```bash
go generate ./app/gojang/models
```

### Migration Fails

Check for:
- ✅ Unique constraints violated
- ✅ Foreign key relationships correct
- ✅ Field types compatible with database

### Handler Not Found

- ✅ Check handler is created in `app/products/products.handler.go`
- ✅ Check route is mounted in `app/cmd/web/main.go`
- ✅ Restart server after changes

### Template Not Rendering

- ✅ Check template exists in correct directory
- ✅ Check `{{define "title"}}` and `{{define "content"}}` exist
- ✅ Check field names match model struct

### Admin Panel Not Showing Model

- ✅ Check model registered in `models.go`
- ✅ ~~Check case statements added to all methods in `registry.go`~~ No longer needed!
- ✅ Verify model name matches Ent client field (e.g., `client.Product`)
- ✅ Restart server

---

## Next Steps

- ✅ **Read:** [Creating Static Pages](./creating-static-pages.md)
- ✅ **Learn:** [HTMX Integration Patterns](./htmx-patterns.md)
- ✅ **Explore:** Check existing models in `app/schema/` for more examples
- ✅ **Advanced:** [Ent Documentation](https://entgo.io/docs/getting-started)

---

## Quick Reference

| Step | File | Action |
|------|------|--------|
| 1. Schema | `app/schema/model.go` | Define fields and edges |
| 2. Generate | Terminal | `go generate ./app/gojang/models` |
| 3. Form | `app/views/forms/forms.go` | Add validation struct |
| 4. Handler | `app/products/products.handler.go` | Create CRUD handlers |
| 5. Routes | `app/products/products.route.go` | Define URL patterns |
| 6. Main | `app/cmd/web/main.go` | Register routes |
| 7. Templates | `app/products/templates/` | Create HTML views |
| 8. Admin | `app/gojang/admin/models.go` | Register model (auto CRUD!) |
| ~~9. Registry~~ | ~~`app/gojang/admin/registry.go`~~ | ~~Add case statements~~ ✅ **Removed!** |

---
