# Quick Start: Adding a Data Model

**A simplified guide to add a new data model in 6 easy steps.**

This guide shows you how to add a simple SampleProduct model with 4 properties. Perfect for beginners!

**Estimated time:** 10 minutes ⚡

---

## 🚀 Automated Option (NEW!)

**Want to skip manual setup?** Use the `addmodel` command to automatically generate everything:

```bash
# Interactive mode - you'll be prompted for details
task addmodel

# Or use command-line flags for automation
go run ./gojang/cmd/addmodel \
  --model SampleProduct \
  --icon "📦" \
  --fields "name:string:required,price:float:required,stock:int,description:text"
```

This automatically creates all the files shown in the manual steps below. Perfect for rapid prototyping!

**Features:**
- ✅ Dry-run mode to preview changes (`--dry-run`)
- ✅ Reserved keyword validation
- ✅ Command-line automation support
- ✅ Comprehensive examples (`--examples`)

See `gojang/cmd/addmodel/README.md` for full documentation or continue below for manual step-by-step instructions.

---

## Manual Setup (Learn the Details)

Follow these steps to understand how models are structured in Gojang:

---

## What We're Building

A SampleProduct model with:
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

Create a new file `gojang/models/schema/sampleproduct.go`:

```go
package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SampleProduct struct {
	ent.Schema
}

func (SampleProduct) Fields() []ent.Field {
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
cd gojang/models
go generate ./...
cd ../..
```

This creates all the database code automatically.

---

## Step 3: Create Form Validation

Add to `gojang/views/forms/forms.go`:

```go
type SampleProductForm struct {
	Name        string  `form:"name" validate:"required"`
	Price       float64 `form:"price" validate:"required,gt=0"`
	Stock       int     `form:"stock" validate:"gte=0"`
	Description string  `form:"description"`
}
```

---

## Step 4: Create Handler

Create `gojang/http/handlers/sampleproducts.go`:

```go
package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/views/forms"
	"github.com/gojangframework/gojang/gojang/views/renderers"
)

type SampleProductHandler struct {
	Client   *models.Client
	Renderer *renderers.Renderer
}

func NewSampleProductHandler(client *models.Client, renderer *renderers.Renderer) *SampleProductHandler {
	return &SampleProductHandler{
		Client:   client,
		Renderer: renderer,
	}
}

// Index - List all sample products
func (h *SampleProductHandler) Index(w http.ResponseWriter, r *http.Request) {
	sampleproducts, err := h.Client.SampleProduct.Query().
		Order(models.Desc("created_at")).
		All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load sample products")
		return
	}

	h.Renderer.Render(w, r, "sampleproducts/index.html", &renderers.TemplateData{
		Title: "Sample Products",
		Data: map[string]interface{}{
			"SampleProducts": sampleproducts,
		},
	})
}

// New - Show form to create sample product
func (h *SampleProductHandler) New(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "sampleproducts/new.partial.html", &renderers.TemplateData{
		Title: "New Sample Product",
	})
}

// Create - Save new sample product
func (h *SampleProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
		return
	}

	var form forms.SampleProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		h.Renderer.Render(w, r, "sampleproducts/new.partial.html", &renderers.TemplateData{
			Title: "New Sample Product",
			Data: map[string]interface{}{
				"Form":   form,
				"Errors": err,
			},
		})
		return
	}

	_, err := h.Client.SampleProduct.Create().
		SetName(form.Name).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetDescription(form.Description).
		Save(r.Context())

	if err != nil {
		log.Printf("Error creating sample product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create sample product")
		return
	}

	http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
}

// Edit - Show form to edit sample product
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

// Update - Save changes to sample product
func (h *SampleProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
		return
	}

	var form forms.SampleProductForm
	if err := forms.Decode(r, &form); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := forms.Validate(r.Context(), &form); err != nil {
		sampleproduct, _ := h.Client.SampleProduct.Get(r.Context(), id)
		h.Renderer.Render(w, r, "sampleproducts/edit.partial.html", &renderers.TemplateData{
			Title: "Edit Sample Product",
			Data: map[string]interface{}{
				"SampleProduct": sampleproduct,
				"Form":          form,
				"Errors":        err,
			},
		})
		return
	}

	_, err := h.Client.SampleProduct.UpdateOneID(id).
		SetName(form.Name).
		SetPrice(form.Price).
		SetStock(form.Stock).
		SetDescription(form.Description).
		Save(r.Context())

	if err != nil {
		log.Printf("Error updating sample product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update sample product")
		return
	}

	http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
}

// Delete - Remove sample product
func (h *SampleProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := h.Client.SampleProduct.DeleteOneID(id).Exec(r.Context()); err != nil {
		log.Printf("Error deleting sample product: %v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete sample product")
		return
	}

	http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
}
```

---

## Step 5: Create Routes

Create `gojang/http/routes/sampleproducts.go`:

```go
package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func SampleProductRoutes(handler *handlers.SampleProductHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)

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

Edit `gojang/cmd/web/main.go` and add these lines where other routes are registered:

```go
// Find this section (around line 150):
sampleProductHandler := handlers.NewSampleProductHandler(client, renderer)
router.Mount("/sampleproducts", routes.SampleProductRoutes(sampleProductHandler, sessionManager, client))
```

---

## Step 6: Create Templates

### Create directory:

```bash
mkdir -p gojang/views/templates/sampleproducts
```

### Create `gojang/views/templates/sampleproducts/index.html`:

```html
{{define "title"}}Sample Products{{end}}

{{define "content"}}
<div class="page-header">
    <h1>Sample Products</h1>
    {{if .User}}
    <a href="/sampleproducts/new" 
       hx-get="/sampleproducts/new" 
       hx-target="#modal-content"
       hx-swap="innerHTML"
       class="btn btn-primary">
        Add Sample Product
    </a>
    {{end}}
</div>

{{if .Data.SampleProducts}}
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
            {{range .Data.SampleProducts}}
            <tr>
                <td>{{.Name}}</td>
                <td>${{printf "%.2f" .Price}}</td>
                <td>{{.Stock}}</td>
                <td>{{.Description}}</td>
                {{if $.User}}
                <td class="actions">
                    <a href="/sampleproducts/{{.ID}}/edit"
                       hx-get="/sampleproducts/{{.ID}}/edit"
                       hx-target="#modal-content"
                       hx-swap="innerHTML"
                       class="btn btn-sm btn-primary">
                        Edit
                    </a>
                    <form method="POST" action="/sampleproducts/{{.ID}}" style="display: inline;">
                        <input type="hidden" name="_method" value="DELETE">
                        <button type="submit"
                                onclick="return confirm('Delete this sample product?')"
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
<p>No sample products found.</p>
{{end}}
{{end}}
```

### Create `gojang/views/templates/sampleproducts/new.partial.html`:

```html
{{define "title"}}New Sample Product{{end}}

{{define "content"}}
<div class="modal-header">
    <h2>New Sample Product</h2>
</div>

<form method="POST" action="/sampleproducts" hx-post="/sampleproducts" hx-swap="none">
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
        <button type="submit" class="btn btn-primary">Create Sample Product</button>
        <button type="button" onclick="closeModal()" class="btn btn-secondary">Cancel</button>
    </div>
</form>
{{end}}
```

### Create `gojang/views/templates/sampleproducts/edit.partial.html`:

```html
{{define "title"}}Edit Sample Product{{end}}

{{define "content"}}
<div class="modal-header">
    <h2>Edit Sample Product</h2>
</div>

<form method="POST" action="/sampleproducts/{{.Data.SampleProduct.ID}}" hx-put="/sampleproducts/{{.Data.SampleProduct.ID}}" hx-swap="none">
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
               value="{{if .Data.Form}}{{.Data.Form.Name}}{{else}}{{.Data.SampleProduct.Name}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="price">Price</label>
        <input type="number" 
               id="price" 
               name="price" 
               step="0.01"
               value="{{if .Data.Form}}{{.Data.Form.Price}}{{else}}{{.Data.SampleProduct.Price}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="stock">Stock</label>
        <input type="number" 
               id="stock" 
               name="stock" 
               value="{{if .Data.Form}}{{.Data.Form.Stock}}{{else}}{{.Data.SampleProduct.Stock}}{{end}}"
               required
               class="form-control">
    </div>

    <div class="form-group">
        <label for="description">Description</label>
        <textarea id="description" 
                  name="description" 
                  rows="3"
                  class="form-control">{{if .Data.Form}}{{.Data.Form.Description}}{{else}}{{.Data.SampleProduct.Description}}{{end}}</textarea>
    </div>

    <div class="form-actions">
        <button type="submit" class="btn btn-primary">Update Sample Product</button>
        <button type="button" onclick="closeModal()" class="btn btn-secondary">Cancel</button>
    </div>
</form>
{{end}}
```

---

## Step 7: Register with Admin Panel

Add to `gojang/admin/models.go`:

```go
func RegisterModels(registry *Registry) {
	// ... existing models ...

	// Add SampleProduct model
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.SampleProduct{},
		Icon:           "📦",
		NamePlural:     "Sample Products",
		ListFields:     []string{"ID", "Name", "Price", "Stock"},
		ReadonlyFields: []string{"ID", "CreatedAt"},
	})
}
```

That's it! The admin panel automatically handles CRUD operations.

---

## Step 8: Test It!

1. **Restart the server:**
   ```bash
   go run ./gojang/cmd/web
   ```

2. **Visit the sample products page:**
   - Public: http://localhost:8080/sampleproducts
   - Admin: http://localhost:8080/admin/sampleproducts

3. **Create a sample product:**
   - Log in
   - Click "Add Sample Product"
   - Fill the form
   - Click "Create Sample Product"

---

## Next Steps

Want to learn more?

- **Add relationships** - Connect sample products to categories or users
- **Add images** - Upload product photos
- **Add pagination** - Handle large lists
- **Add search** - Filter sample products
- **Add validation** - Custom validation rules

See the [comprehensive guide](./creating-data-models.md) for advanced features.

---

## Quick Checklist

When adding a new model:

- [ ] Create schema in `gojang/models/schema/`
- [ ] Run `go generate ./...` in models directory
- [ ] Add form struct to `gojang/views/forms/forms.go`
- [ ] Create handler in `gojang/http/handlers/`
- [ ] Create routes in `gojang/http/routes/`
- [ ] Register routes in `gojang/cmd/web/main.go`
- [ ] Create templates in `gojang/views/templates/[model]/`
- [ ] Register in `gojang/admin/models.go`
- [ ] Test CRUD operations

---

## Troubleshooting

**Schema changes not applied?**
```bash
cd gojang/models && go generate ./... && cd ../..
```

**Templates not found?**
- Check files exist in `gojang/views/templates/sampleproducts/`
- Restart the server

**404 error?**
- Check routes are registered in `main.go`
- Restart the server

**Form validation not working?**
- Check form struct has correct tags
- Check field names match HTML inputs

---

**Need help?** See the [full documentation](./creating-data-models.md) or [README](./README.md).
