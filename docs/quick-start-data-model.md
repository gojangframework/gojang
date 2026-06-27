# Quick Start: Adding a Data Model

**A simplified guide to add a new data model in 6 easy steps.**

This guide shows you how to add a simple Product model with 4 properties. Perfect for beginners!

**Estimated time:** 10 minutes ⚡

---

## Manual Setup (Learn the Details)

Follow these steps to understand how models are structured in Gojang:

---

## What We're Building

A Product model with:
- **Name** - product name
- **Price** - product price
- **Stock** - quantity available
- **Description** - product details

We'll create:
- ✅ Database schema
- ✅ CRUD handlers (Create, Read, Update, Delete)
- ✅ Routes
- ✅ Templates (views)
- ✅ Admin panel integration

---

## Step 1: Create the Schema

Create a new file `app/schema/product.go`:

```go
package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Product struct {
	ent.Schema
}

func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),

		field.Float("price").
			Positive(),

		field.Int("stock").
			Default(0),

		field.Text("description").
			Optional(),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}
```

---

## Step 2: Generate Database Code

Run these commands:

```bash
go generate ./app/gojang/models
```

This creates all the database code automatically.

---

## Step 3: Create Form Validation

Add to `app/views/forms/forms.go`:

```go
type ProductForm struct {
	Name        string  `form:"name" validate:"required"`
	Price       float64 `form:"price" validate:"required,gt=0"`
	Stock       int     `form:"stock" validate:"gte=0"`
	Description string  `form:"description"`
}
```

---

## Step 4: Create Handler

Create `app/products/products.handler.go`:

```go
package products

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

// Index - List all products
func (h *ProductHandler) Index(w http.ResponseWriter, r *http.Request) {
	products, err := h.Client.Product.Query().
		Order(models.Desc("created_at")).
		All(r.Context())
	if err != nil {
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

// New - Show form to create product
func (h *ProductHandler) New(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "products/new.partial.html", &renderers.TemplateData{
		Title: "New Product",
	})
}

// Create - Save new product
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
		return
	}

	var form forms.ProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		h.Renderer.Render(w, r, "products/new.partial.html", &renderers.TemplateData{
			Title: "New Product",
			Data: map[string]interface{}{
				"Form":   form,
				"Errors": err,
			},
		})
		return
	}

	_, err := h.Client.Product.Create().
		SetName(form.Name).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetDescription(form.Description).
		Save(r.Context())

	if err != nil {
		log.Printf("Error creating product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create product")
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

// Edit - Show form to edit product
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

// Update - Save changes to product
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
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
				"Product": product,
				"Form":          form,
				"Errors":        err,
			},
		})
		return
	}

	_, err := h.Client.Product.UpdateOneID(id).
		SetName(form.Name).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetDescription(form.Description).
		Save(r.Context())

	if err != nil {
		log.Printf("Error updating product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update product")
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

// Delete - Remove product
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

Create `app/products/products.route.go`:

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

	// Public routes
	r.Get("/", handler.Index)

	// Protected routes (require login)
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

### Register in Main

Edit `app/cmd/web/main.go` and add these lines where other routes are registered:

```go
// Add this import:
// "github.com/gojangframework/gojang/app/products"

// Find this section (around line 150):
productHandler := products.NewProductHandler(client, publicRenderer)
r.Mount("/products", products.ProductRoutes(productHandler, sessionManager, client))
```

---

## Step 6: Create Templates

### Create directory:

```bash
mkdir -p app/products/templates
```

### Create `app/products/templates/index.html`:

```html
{{define "title"}}Products{{end}}

{{define "content"}}
<div class="page-header">
    <h1>Products</h1>
    {{if .User}}
    <a href="/products/new"
       hx-get="/products/new"
       hx-target="#modal-content"
       hx-swap="innerHTML"
       class="btn btn-primary">
        Add Product
    </a>
    {{end}}
</div>

{{if .Data.Products}}
<div class="table-container">
    <table class="table">
        <thead>
            <tr>
                <th>Name</th>
                <th>Price</th>
                <th>Stock</th>
                <th>Description</th>
                {{if .User}}<th>Actions</th>{{end}}
            </tr>
        </thead>
        <tbody>
            {{range .Data.Products}}
            <tr>
                <td>{{.Name}}</td>
                <td>${{printf "%.2f" .Price}}</td>
                <td>{{.Stock}}</td>
                <td>{{.Description}}</td>
                {{if $.User}}
                <td class="actions">
                    <a href="/products/{{.ID}}/edit"
                       hx-get="/products/{{.ID}}/edit"
                       hx-target="#modal-content"
                       hx-swap="innerHTML"
                       class="btn btn-sm btn-primary">
                        Edit
                    </a>
                    <form method="POST" action="/products/{{.ID}}" style="display: inline;">
                        <input type="hidden" name="_method" value="DELETE">
                        <button type="submit"
                                onclick="return confirm('Delete this product?')"
                                class="btn btn-sm btn-danger">
                            Delete
                        </button>
                    </form>
                </td>
                {{end}}
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{else}}
<p>No products found.</p>
{{end}}
{{end}}
```

### Create `app/products/templates/new.partial.html`:

```html
{{define "title"}}New Product{{end}}

{{define "content"}}
<div class="modal-header">
    <h2>New Product</h2>
</div>

<form method="POST" action="/products" hx-post="/products" hx-swap="none">
    {{if .Data.Errors}}
    <div class="alert alert-danger">
        {{range .Data.Errors}}
        <p>{{.}}</p>
        {{end}}
    </div>
    {{end}}

    <div class="form-group">
        <label for="name">Name</label>
        <input type="text"
               id="name"
               name="name"
               value="{{if .Data.Form}}{{.Data.Form.Name}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="price">Price</label>
        <input type="number"
               id="price"
               name="price"
               step="0.01"
               value="{{if .Data.Form}}{{.Data.Form.Price}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="stock">Stock</label>
        <input type="number"
               id="stock"
               name="stock"
               value="{{if .Data.Form}}{{.Data.Form.Stock}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="description">Description</label>
        <textarea id="description"
                  name="description"
                  rows="3"
                  class="form-control">{{if .Data.Form}}{{.Data.Form.Description}}{{end}}</textarea>
    </div>

    <div class="form-actions">
        <button type="submit" class="btn btn-primary">Create Product</button>
        <button type="button" onclick="closeModal()" class="btn btn-secondary">Cancel</button>
    </div>
</form>
{{end}}
```

### Create `app/products/templates/edit.partial.html`:

```html
{{define "title"}}Edit Product{{end}}

{{define "content"}}
<div class="modal-header">
    <h2>Edit Product</h2>
</div>

<form method="POST" action="/products/{{.Data.Product.ID}}" hx-put="/products/{{.Data.Product.ID}}" hx-swap="none">
    {{if .Data.Errors}}
    <div class="alert alert-danger">
        {{range .Data.Errors}}
        <p>{{.}}</p>
        {{end}}
    </div>
    {{end}}

    <div class="form-group">
        <label for="name">Name</label>
        <input type="text"
               id="name"
               name="name"
               value="{{if .Data.Form}}{{.Data.Form.Name}}{{else}}{{.Data.Product.Name}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="price">Price</label>
        <input type="number"
               id="price"
               name="price"
               step="0.01"
               value="{{if .Data.Form}}{{.Data.Form.Price}}{{else}}{{.Data.Product.Price}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="stock">Stock</label>
        <input type="number"
               id="stock"
               name="stock"
               value="{{if .Data.Form}}{{.Data.Form.Stock}}{{else}}{{.Data.Product.Stock}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="description">Description</label>
        <textarea id="description"
                  name="description"
                  rows="3"
                  class="form-control">{{if .Data.Form}}{{.Data.Form.Description}}{{else}}{{.Data.Product.Description}}{{end}}</textarea>
    </div>

    <div class="form-actions">
        <button type="submit" class="btn btn-primary">Update Product</button>
        <button type="button" onclick="closeModal()" class="btn btn-secondary">Cancel</button>
    </div>
</form>
{{end}}
```

---

## Step 7: Confirm Admin Auto-Discovery

After `go generate ./app/gojang/models`, the generated `Product` client is
available on `*models.Client`, so the admin panel discovers it automatically.
No `app/gojang/admin/models.go` change is required for a plain resource.

Add a `registry.RegisterModel(ModelRegistration{...})` override only when you
need custom admin behavior, such as a different icon/name, curated list fields,
hidden or readonly fields, hooks, custom fields, or eager-loaded relationships.

That's it! The admin panel automatically handles CRUD operations at
`/admin/t/product`.

---

## Step 8: Test It!

1. **Restart the server:**
   ```bash
   go run ./app/cmd/web
   ```

2. **Visit the products page:**
   - Public: http://localhost:8080/products
   - Admin: http://localhost:8080/admin/t/product

3. **Create a product:**
   - Log in
   - Click "Add Product"
   - Fill the form
   - Click "Create Product"

---

## Next Steps

Want to learn more?

- **Add relationships** - Connect products to categories or users
- **Add images** - Upload product photos
- **Add pagination** - Handle large lists
- **Add search** - Filter products
- **Add validation** - Custom validation rules

See the [comprehensive guide](./creating-data-models.md) for advanced features.

---

## Quick Checklist

When adding a new model:

- [ ] Create schema in `app/schema/`
- [ ] Run `go generate ./app/gojang/models`
- [ ] Add form struct to `app/views/forms/forms.go`
- [ ] Create handler in `app/products/products.handler.go`
- [ ] Create routes in `app/products/products.route.go`
- [ ] Register routes in `app/cmd/web/main.go`
- [ ] Create templates in `app/products/templates/`
- [ ] Add an admin override in `app/gojang/admin/models.go` only if needed
- [ ] Test CRUD operations

---

## Troubleshooting

**Schema changes not applied?**
```bash
go generate ./app/gojang/models
```

**Templates not found?**
- Check files exist in `app/products/templates/`
- Restart the server

**404 error?**
- Check routes are registered in `main.go`
- Restart the server

**Form validation not working?**
- Check form struct has correct tags
- Check field names match HTML inputs

---

**Need help?** See the [full documentation](./creating-data-models.md) or [README](./README.md).
