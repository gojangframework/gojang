# HTMX Integration Guide

This guide covers HTMX integration patterns used in Gojang to build dynamic, modern web applications without heavy JavaScript frameworks.

## Overview

HTMX allows you to access modern browser features directly from HTML, using attributes instead of JavaScript. In Gojang, HTMX is used to:

- Submit forms asynchronously
- Load content dynamically
- Update parts of the page without full reloads
- Create modal dialogs
- Implement CRUD operations seamlessly

**Key Benefits:**
- 📦 No build step required
- 🎯 Progressive enhancement
- 🚀 Fast, responsive interactions
- 🔧 Server-side rendering
- ✨ Clean, maintainable code

---

## Getting Started

### HTMX Setup

HTMX is already included in `base.html`:

```html
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
```

Every page automatically inherits HTMX capabilities through the base template.

### CSRF Token Integration

HTMX requests automatically include CSRF tokens (configured in `base.html`):

```javascript
document.addEventListener('htmx:configRequest', function(evt) {
    const token = document.querySelector('meta[name="csrf-token"]');
    if (token) {
        evt.detail.headers['X-CSRF-Token'] = token.content;
    }
});
```

**You don't need to configure this yourself** - it's already set up! ✅

---

## Core HTMX Attributes

### Basic Attributes

| Attribute | Purpose | Example |
|-----------|---------|---------|
| `hx-get` | Make a GET request | `hx-get="/posts/new"` |
| `hx-post` | Make a POST request | `hx-post="/posts"` |
| `hx-put` | Make a PUT request | `hx-put="/posts/123"` |
| `hx-delete` | Make a DELETE request | `hx-delete="/posts/123"` |
| `hx-target` | Where to put the response | `hx-target="#modal"` |
| `hx-swap` | How to swap the content | `hx-swap="innerHTML"` |

### Targeting & Swapping

**`hx-target`** - Specifies where the response goes:
```html
<!-- Replace content inside #modal -->
<button hx-get="/posts/new" hx-target="#modal">

<!-- Replace the button itself -->
<button hx-delete="/posts/1" hx-target="this">
```

**`hx-swap`** - Controls how content is inserted:

| Value | Behavior |
|-------|----------|
| `innerHTML` | Replace inner content (default) |
| `outerHTML` | Replace entire element |
| `afterbegin` | Insert before first child |
| `beforeend` | Insert after last child |
| `beforebegin` | Insert before element |
| `afterend` | Insert after element |
| `none` | Don't swap, just make request |

---

## Pattern 1: Modal Dialogs

**The most common pattern in Gojang** - used for forms, confirmations, and detail views.

### Template Setup

Create a modal container in your page:

```html
{{define "content"}}
<div class="container">
    <!-- Your page content -->
    <button hx-get="/posts/new" hx-target="#modal" hx-swap="innerHTML" class="btn btn-primary">
        + New Post
    </button>
</div>

<!-- Modal container (empty until triggered) -->
<div id="modal" class="modal"></div>
{{end}}
```

### Modal Content Template

Create the modal form (e.g., `posts/new.partial.html`):

```html
<div class="modal-backdrop" onclick="this.parentElement.innerHTML = ''">
    <div class="modal-content" onclick="event.stopPropagation()">
        <div class="modal-header">
            <h3>Create New Post</h3>
            <button onclick="this.closest('.modal-backdrop').parentElement.innerHTML = ''" 
                    class="modal-close">&times;</button>
        </div>

        <form hx-post="/posts" 
              hx-target=".posts-container" 
              hx-swap="afterbegin"
              hx-on::after-request="if(event.detail.successful) document.getElementById('modal').innerHTML = ''"
              class="form">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            
            <div class="form-group">
                <label for="subject">Subject</label>
                <input type="text" id="subject" name="subject" required>
            </div>

            <div class="modal-footer">
                <button type="button" 
                        onclick="this.closest('.modal-backdrop').parentElement.innerHTML = ''" 
                        class="btn btn-secondary">
                    Cancel
                </button>
                <button type="submit" class="btn btn-primary">Create Post</button>
            </div>
        </form>
    </div>
</div>
```

### Handler Pattern

```go
func (h *PostHandler) New(w http.ResponseWriter, r *http.Request) {
    // Simply render the modal template
    h.Renderer.Render(w, r, "posts/new.partial.html", nil)
}

func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Parse and validate form
    if err := r.ParseForm(); err != nil {
        h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
        return
    }

    // ... validation and save logic ...

    // Close modal with HX-Trigger header
    w.Header().Set("HX-Trigger", "closeModal")
    
    // Return updated content
    h.Renderer.Render(w, r, "posts/list.partial.html", data)
}
```

**Key Points:**
- ✅ Click outside modal (backdrop) to close
- ✅ Close button in header
- ✅ Auto-close on successful form submission
- ✅ Errors keep modal open for correction

---

## Pattern 2: Form Submissions

### Basic Form Submission

```html
<form hx-post="/posts" hx-target="#posts-list" hx-swap="innerHTML">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    
    <div class="form-group">
        <label for="subject">Subject</label>
        <input type="text" id="subject" name="subject" required>
        {{if index .Errors "Subject"}}
            <span class="error">{{index .Errors "Subject"}}</span>
        {{end}}
    </div>
    
    <button type="submit" class="btn btn-primary">Submit</button>
</form>
```

### Handler Response Pattern

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    // Parse form
    if err := r.ParseForm(); err != nil {
        h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
        return
    }

    // Validate
    errors := forms.Validate(form)
    if len(errors) > 0 {
        // Re-render form with errors
        h.Renderer.Render(w, r, "posts/new.partial.html", &renderers.TemplateData{
            Errors: errors,
        })
        return
    }

    // Save to database
    // ...

    // Return success response (partial or full page)
    h.Renderer.Render(w, r, "posts/list.partial.html", data)
}
```

**Validation Pattern:**
- ❌ Errors: Re-render form with error messages
- ✅ Success: Return updated content

---

## Pattern 3: CRUD Operations

### Create - Add New Items

**Button to open form:**
```html
<button hx-get="/posts/new" 
        hx-target="#modal" 
        hx-swap="innerHTML" 
        class="btn btn-primary">
    + New Post
</button>
```

**Form submits to list:**
```html
<form hx-post="/posts" 
      hx-target=".posts-container" 
      hx-swap="afterbegin">
    <!-- form fields -->
</form>
```

**Handler adds item and returns list:**
```go
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    // ... create post ...
    
    // Close modal
    w.Header().Set("HX-Trigger", "closeModal")
    
    // Return updated list
    posts, _ := h.Client.Post.Query().All(r.Context())
    h.Renderer.Render(w, r, "posts/list.partial.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Posts": posts},
    })
}
```

### Read/Edit - Update Items

**Edit button on each item:**
```html
<button hx-get="/posts/{{.ID}}/edit" 
        hx-target="#modal" 
        hx-swap="innerHTML"
        class="btn-sm btn-secondary">
    Edit
</button>
```

**Edit form replaces specific item:**
```html
<form hx-put="/posts/{{.Data.Post.ID}}" 
      hx-target="#post-{{.Data.Post.ID}}" 
      hx-swap="outerHTML"
      hx-on::after-request="if(event.detail.successful) document.getElementById('modal').innerHTML = ''">
    <!-- form fields with values -->
</form>
```

**Handler returns updated item card:**
```go
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.Atoi(chi.URLParam(r, "id"))
    
    // ... update post ...
    
    // Close modal
    w.Header().Set("HX-Trigger", "closeModal")
    
    // Return updated post card
    h.Renderer.Render(w, r, "posts/card.partial.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Post": updatedPost},
    })
}
```

### Delete - Remove Items

**Two approaches:**

#### 1. Direct Delete (with confirmation)
```html
<button hx-delete="/posts/{{.ID}}" 
        hx-target="#post-{{.ID}}" 
        hx-swap="outerHTML"
        hx-confirm="Are you sure you want to delete this post?"
        class="btn-sm btn-danger">
    Delete
</button>
```

Handler returns empty response (element is removed):
```go
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.Atoi(chi.URLParam(r, "id"))
    
    // Delete post
    h.Client.Post.DeleteOneID(id).ExecX(r.Context())
    
    // Return nothing (hx-swap="outerHTML" removes the element)
    w.WriteHeader(http.StatusOK)
}
```

#### 2. Confirmation Modal
```html
<button hx-get="/posts/{{.ID}}/delete" 
        hx-target="#modal" 
        hx-swap="innerHTML"
        class="btn-sm btn-danger">
    Delete
</button>
```

Delete confirmation template:
```html
<div class="modal-backdrop">
    <div class="modal-content">
        <h3>Confirm Delete</h3>
        <p>Are you sure you want to delete "{{.Data.Post.Subject}}"?</p>
        
        <div class="modal-footer">
            <button onclick="this.closest('.modal-backdrop').parentElement.innerHTML = ''" 
                    class="btn btn-secondary">
                Cancel
            </button>
            <button hx-delete="/posts/{{.Data.Post.ID}}" 
                    hx-target="#post-{{.Data.Post.ID}}" 
                    hx-swap="outerHTML"
                    class="btn btn-danger">
                Delete
            </button>
        </div>
    </div>
</div>
```

---

## Pattern 4: Partial Templates

Partial templates are HTML fragments returned by HTMX requests. They don't include the full page structure.

### Creating Partials

**File naming convention:** Use `.partial.html` suffix (e.g., `posts/list.partial.html`):

```html
<!-- posts/list.partial.html -->
{{range .Data.Posts}}
<div class="card post-card" id="post-{{.ID}}">
    <h3>{{.Subject}}</h3>
    <div class="post-body">{{.Body}}</div>
</div>
{{else}}
<div class="card">
    <p>No posts yet.</p>
</div>
{{end}}
```

### Using Partials

```html
<!-- Full page template -->
<div id="posts-list" class="posts-container">
    {{range .Data.Posts}}
        <!-- Initial render -->
    {{end}}
</div>

<!-- Button that reloads the list -->
<button hx-get="/posts/reload" hx-target="#posts-list" hx-swap="innerHTML">
    Refresh Posts
</button>
```

### Handler for Partials

```go
func (h *PostHandler) Reload(w http.ResponseWriter, r *http.Request) {
    posts, _ := h.Client.Post.Query().All(r.Context())
    
    // Return only the partial
    h.Renderer.Render(w, r, "posts/list.partial.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Posts": posts},
    })
}
```

---

## Pattern 5: Dynamic Content Loading

### Load on Button Click

```html
<div class="container">
    <h1>About Us</h1>
    
    <button hx-get="/about/team" 
            hx-target="#team-section" 
            hx-swap="innerHTML"
            class="btn btn-primary">
        Load Team Members
    </button>
    
    <div id="team-section" style="margin-top: 1.5rem;">
        <!-- Content loads here -->
    </div>
</div>
```

Handler returns HTML fragment:
```go
func (h *PageHandler) AboutTeam(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`
        <div class="team-grid">
            <div class="card">John Doe - CEO</div>
            <div class="card">Jane Smith - CTO</div>
        </div>
    `))
}
```

### Load on Page Load

Use `hx-trigger="load"` to fetch content automatically:

```html
<div hx-get="/dashboard/stats" 
     hx-trigger="load" 
     hx-swap="innerHTML">
    Loading stats...
</div>
```

### Lazy Loading Images/Content

```html
<div hx-get="/posts/{{.ID}}/details" 
     hx-trigger="intersect once"
     hx-swap="innerHTML">
    <div class="loading">Loading...</div>
</div>
```

Loads content when element scrolls into view (infinite scroll pattern).

---

## Pattern 6: Advanced Swap Strategies

### Prepend New Items

Add new items to the top of a list:

```html
<form hx-post="/posts" 
      hx-target="#posts-list" 
      hx-swap="afterbegin">
```

### Append to List

Add items to the bottom:

```html
<button hx-get="/posts/more" 
        hx-target="#posts-list" 
        hx-swap="beforeend">
    Load More
</button>
```

### Replace Specific Element

Update individual items:

```html
<form hx-put="/posts/{{.ID}}" 
      hx-target="#post-{{.ID}}" 
      hx-swap="outerHTML">
```

### Out-of-Band Swaps

Update multiple page areas from one response:

```html
<!-- In handler response -->
<div id="post-123" hx-swap-oob="true">
    <!-- Updated post content -->
</div>
<div id="post-count" hx-swap-oob="true">
    <span>Total: 42 posts</span>
</div>
```

The main target receives normal content, other divs with `hx-swap-oob="true"` update their matching IDs.

---

## Pattern 7: Event Handling

### Built-in Events

```html
<!-- Submit on Enter key -->
<input hx-post="/search" 
       hx-trigger="keyup changed delay:500ms" 
       hx-target="#results">

<!-- Submit on change -->
<select hx-get="/filter" 
        hx-trigger="change" 
        hx-target="#products">

<!-- Multiple triggers -->
<button hx-get="/data" 
        hx-trigger="click, every 30s">
```

### Custom Events

Trigger actions after request completes:

```html
<form hx-post="/posts"
      hx-on::after-request="if(event.detail.successful) showSuccessMessage()">
```

### HTMX Events

```html
<!-- Show loading indicator -->
<form hx-post="/save"
      hx-indicator="#spinner">
    <button type="submit">Save</button>
</form>
<div id="spinner" class="htmx-indicator">Loading...</div>
```

### Server-Triggered Events

Send custom headers to trigger client-side events:

```go
// In handler
w.Header().Set("HX-Trigger", "closeModal")
w.Header().Set("HX-Trigger", `{"showMessage": {"text": "Post created!"}}`)
```

Listen in JavaScript:

```javascript
document.addEventListener('showMessage', function(evt) {
    alert(evt.detail.text);
});
```

---

## Pattern 8: Response Headers

HTMX recognizes special response headers to control behavior:

### Common Response Headers

```go
// Close modal after successful action
w.Header().Set("HX-Trigger", "closeModal")

// Change swap target
w.Header().Set("HX-Retarget", "#different-element")

// Change swap strategy
w.Header().Set("HX-Reswap", "innerHTML")

// Client-side redirect
w.Header().Set("HX-Redirect", "/dashboard")

// Refresh the page
w.Header().Set("HX-Refresh", "true")
```

### Example: Dynamic Retargeting

```go
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    // ... create post ...
    
    // Change where response goes
    w.Header().Set("HX-Retarget", "#posts-list")
    w.Header().Set("HX-Reswap", "innerHTML")
    
    // Close modal
    w.Header().Set("HX-Trigger", "closeModal")
    
    h.Renderer.Render(w, r, "posts/list.partial.html", data)
}
```

---

## Authentication & Authorization

### Checking for HTMX Requests

```go
func (h *Handler) Action(w http.ResponseWriter, r *http.Request) {
    // Check if this is an HTMX request
    if r.Header.Get("HX-Request") == "true" {
        // Return partial
        h.Renderer.Render(w, r, "partial.html", data)
    } else {
        // Return full page
        h.Renderer.Render(w, r, "full.html", data)
    }
}
```

### Protected Routes

HTMX works seamlessly with authentication middleware:

```go
// In routes
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth)
    
    r.Get("/posts/new", handler.New)
    r.Post("/posts", handler.Create)
    r.Put("/posts/{id}", handler.Update)
    r.Delete("/posts/{id}", handler.Delete)
})
```

If not authenticated, middleware redirects or returns error - HTMX displays it in the target.

### Handling Auth Errors in HTMX

Return proper error templates:

```go
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !authenticated {
            if r.Header.Get("HX-Request") == "true" {
                // Return error fragment
                w.WriteHeader(http.StatusUnauthorized)
                w.Write([]byte(`<div class="alert alert-error">Please log in</div>`))
                return
            }
            // Regular request - redirect to login
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## Best Practices

### ✅ DO

1. **Use semantic IDs** for targets
   ```html
   <div id="post-123">  <!-- ✅ Descriptive -->
   <div id="item-1">     <!-- ❌ Generic -->
   ```

2. **Include CSRF tokens** in all forms
   ```html
   <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
   ```

3. **Use partial templates** for HTMX responses
   ```
   posts/index.html        # Full page
   posts/list.partial.html # Just the list
   posts/card.partial.html # Single item
   ```

4. **Validate on the server** - Never trust client input
   ```go
   errors := forms.Validate(form)
   if len(errors) > 0 {
       h.Renderer.Render(w, r, "form.html", &renderers.TemplateData{
           Errors: errors,
       })
       return
   }
   ```

5. **Return appropriate HTTP status codes**
   ```go
   w.WriteHeader(http.StatusBadRequest)  // 400 for validation errors
   w.WriteHeader(http.StatusNotFound)    // 404 for missing resources
   ```

6. **Use loading indicators** for slow operations
   ```html
   <button hx-post="/save" hx-indicator="#spinner">Save</button>
   <div id="spinner" class="htmx-indicator">Saving...</div>
   ```

### ❌ DON'T

1. **Don't forget error handling** - Always handle form validation errors
2. **Don't skip CSRF protection** - It's required for security
3. **Don't return full pages** for HTMX requests - Use partials
4. **Don't use inline JavaScript** when HTMX attributes work
5. **Don't put business logic** in templates - Keep it in handlers

---

## Common Patterns Summary

### Modal Form → Create Item → Update List

```html
<!-- 1. Button opens modal -->
<button hx-get="/posts/new" hx-target="#modal">New</button>

<!-- 2. Modal form submits -->
<form hx-post="/posts" hx-target="#posts-list" hx-swap="innerHTML">

<!-- 3. Handler returns updated list -->
```

### Edit in Modal → Update Specific Item

```html
<!-- 1. Edit button opens modal -->
<button hx-get="/posts/123/edit" hx-target="#modal">Edit</button>

<!-- 2. Form updates specific item -->
<form hx-put="/posts/123" hx-target="#post-123" hx-swap="outerHTML">
```

### Delete with Confirmation

```html
<button hx-delete="/posts/123" 
        hx-target="#post-123" 
        hx-swap="outerHTML"
        hx-confirm="Delete this post?">
    Delete
</button>
```

### Dynamic Content Loading

```html
<div hx-get="/content" hx-trigger="load" hx-swap="innerHTML">
    Loading...
</div>
```

---

## Troubleshooting

### Content Not Updating

**Problem:** HTMX request succeeds but page doesn't update

**Solutions:**
- ✅ Check `hx-target` selector is correct and element exists
- ✅ Verify handler returns HTML (not JSON or empty response)
- ✅ Check `hx-swap` strategy matches your intent
- ✅ Use browser DevTools Network tab to inspect response

### Form Not Submitting

**Problem:** Form submit doesn't trigger HTMX request

**Solutions:**
- ✅ Ensure HTMX is loaded (check browser console)
- ✅ Check form has `hx-post`, `hx-put`, etc.
- ✅ Verify CSRF token is included
- ✅ Check for JavaScript errors in console

### Modal Not Closing

**Problem:** Modal stays open after successful submission

**Solutions:**
- ✅ Add `hx-on::after-request` to form
- ✅ Set `HX-Trigger: closeModal` header in handler
- ✅ Check modal event listener is registered (in `base.html`)

### 403 Forbidden Errors

**Problem:** HTMX requests return 403

**Solutions:**
- ✅ Include CSRF token in form: `<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">`
- ✅ Check CSRF header configuration in `base.html`
- ✅ Ensure session is valid

### Partial Returns Full Page

**Problem:** HTMX request gets full HTML page instead of fragment

**Solutions:**
- ✅ Use correct template name (e.g., `posts/list.partial.html`)
- ✅ Don't render with base template for partials
- ✅ Check handler is using correct template

---

## Complete Example: Todo List

Here's a complete CRUD example combining all patterns:

### Template: `todos/index.html`

```html
{{define "title"}}My Todos{{end}}

{{define "content"}}
<div class="container">
    <div class="page-header">
        <h2>My Todo List</h2>
        <button hx-get="/todos/new" hx-target="#modal" hx-swap="innerHTML" class="btn btn-primary">
            + Add Todo
        </button>
    </div>

    <div id="todos-list" class="todos-container">
        {{range .Data.Todos}}
        <div class="card todo-card" id="todo-{{.ID}}">
            <div class="todo-content">
                <input type="checkbox" 
                       {{if .Completed}}checked{{end}}
                       hx-put="/todos/{{.ID}}/toggle"
                       hx-target="#todo-{{.ID}}"
                       hx-swap="outerHTML">
                <span class="{{if .Completed}}completed{{end}}">{{.Title}}</span>
            </div>
            <div class="todo-actions">
                <button hx-get="/todos/{{.ID}}/edit" 
                        hx-target="#modal" 
                        hx-swap="innerHTML"
                        class="btn-sm btn-secondary">
                    Edit
                </button>
                <button hx-delete="/todos/{{.ID}}" 
                        hx-target="#todo-{{.ID}}" 
                        hx-swap="outerHTML"
                        hx-confirm="Delete this todo?"
                        class="btn-sm btn-danger">
                    Delete
                </button>
            </div>
        </div>
        {{else}}
        <div class="card">
            <p class="text-center">No todos yet. Add one to get started!</p>
        </div>
        {{end}}
    </div>
</div>

<div id="modal" class="modal"></div>
{{end}}
```

### Template: `todos/new.partial.html`

```html
<div class="modal-backdrop" onclick="this.parentElement.innerHTML = ''">
    <div class="modal-content" onclick="event.stopPropagation()">
        <div class="modal-header">
            <h3>New Todo</h3>
            <button onclick="this.closest('.modal-backdrop').parentElement.innerHTML = ''" 
                    class="modal-close">&times;</button>
        </div>

        <form hx-post="/todos" 
              hx-target="#todos-list" 
              hx-swap="innerHTML"
              hx-on::after-request="if(event.detail.successful) document.getElementById('modal').innerHTML = ''"
              class="form">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            
            <div class="form-group">
                <label for="title">Title</label>
                <input type="text" id="title" name="title" required>
                {{if index .Errors "Title"}}
                    <span class="error">{{index .Errors "Title"}}</span>
                {{end}}
            </div>

            <div class="modal-footer">
                <button type="button" 
                        onclick="this.closest('.modal-backdrop').parentElement.innerHTML = ''" 
                        class="btn btn-secondary">
                    Cancel
                </button>
                <button type="submit" class="btn btn-primary">Create</button>
            </div>
        </form>
    </div>
</div>
```

### Template: `todos/card.partial.html`

```html
<div class="card todo-card" id="todo-{{.Data.Todo.ID}}">
    <div class="todo-content">
        <input type="checkbox" 
               {{if .Data.Todo.Completed}}checked{{end}}
               hx-put="/todos/{{.Data.Todo.ID}}/toggle"
               hx-target="#todo-{{.Data.Todo.ID}}"
               hx-swap="outerHTML">
        <span class="{{if .Data.Todo.Completed}}completed{{end}}">{{.Data.Todo.Title}}</span>
    </div>
    <div class="todo-actions">
        <button hx-get="/todos/{{.Data.Todo.ID}}/edit" 
                hx-target="#modal" 
                hx-swap="innerHTML"
                class="btn-sm btn-secondary">
            Edit
        </button>
        <button hx-delete="/todos/{{.Data.Todo.ID}}" 
                hx-target="#todo-{{.Data.Todo.ID}}" 
                hx-swap="outerHTML"
                hx-confirm="Delete this todo?"
                class="btn-sm btn-danger">
            Delete
        </button>
    </div>
</div>
```

### Handler: `todos.go`

```go
package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/go-chi/chi/v5"
    "github.com/gojangframework/gojang/gojang/models"
    "github.com/gojangframework/gojang/gojang/views/renderers"
)

type TodoHandler struct {
    Client   *models.Client
    Renderer *renderers.Renderer
}

// Index - List all todos
func (h *TodoHandler) Index(w http.ResponseWriter, r *http.Request) {
    todos, err := h.Client.Todo.Query().All(r.Context())
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load todos")
        return
    }

    h.Renderer.Render(w, r, "todos/index.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Todos": todos},
    })
}

// New - Show create form
func (h *TodoHandler) New(w http.ResponseWriter, r *http.Request) {
    h.Renderer.Render(w, r, "todos/new.partial.html", nil)
}

// Create - Create new todo
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
        return
    }

    title := r.Form.Get("title")
    if title == "" {
        h.Renderer.Render(w, r, "todos/new.partial.html", &renderers.TemplateData{
            Errors: map[string]string{"Title": "Title is required"},
        })
        return
    }

    // Create todo
    _, err := h.Client.Todo.Create().
        SetTitle(title).
        SetCompleted(false).
        Save(r.Context())
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create todo")
        return
    }

    // Close modal
    w.Header().Set("HX-Trigger", "closeModal")

    // Return updated list
    todos, _ := h.Client.Todo.Query().All(r.Context())
    h.Renderer.Render(w, r, "todos/list.partial.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Todos": todos},
    })
}

// Toggle - Toggle todo completion
func (h *TodoHandler) Toggle(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.Atoi(chi.URLParam(r, "id"))
    
    todo, err := h.Client.Todo.Get(r.Context(), id)
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusNotFound, "Todo not found")
        return
    }

    // Toggle completed status
    todo, err = h.Client.Todo.UpdateOne(todo).
        SetCompleted(!todo.Completed).
        Save(r.Context())
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update todo")
        return
    }

    // Return updated card
    h.Renderer.Render(w, r, "todos/card.partial.html", &renderers.TemplateData{
        Data: map[string]interface{}{"Todo": todo},
    })
}

// Delete - Delete todo
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.Atoi(chi.URLParam(r, "id"))
    
    if err := h.Client.Todo.DeleteOneID(id).Exec(r.Context()); err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete todo")
        return
    }

    // Return empty (outerHTML swap removes the element)
    w.WriteHeader(http.StatusOK)
}
```

---

## Next Steps

- ✅ **Practice:** Try adding HTMX to your existing pages
- ✅ **Read:** [HTMX Documentation](https://htmx.org/docs/)
- ✅ **Explore:** Check `gojang/views/templates/posts/` for more examples
- ✅ **Build:** Create your own CRUD feature using these patterns

---

## Quick Reference

### Essential Attributes

```html
hx-get="/path"           <!-- GET request -->
hx-post="/path"          <!-- POST request -->
hx-put="/path"           <!-- PUT request -->
hx-delete="/path"        <!-- DELETE request -->
hx-target="#id"          <!-- Where response goes -->
hx-swap="innerHTML"      <!-- How to insert response -->
hx-trigger="click"       <!-- When to trigger -->
hx-confirm="message"     <!-- Confirmation dialog -->
```

### Common Patterns

| Pattern | Attributes |
|---------|-----------|
| Modal Form | `hx-get="/form" hx-target="#modal" hx-swap="innerHTML"` |
| Create Item | `hx-post="/items" hx-target="#list" hx-swap="afterbegin"` |
| Update Item | `hx-put="/items/1" hx-target="#item-1" hx-swap="outerHTML"` |
| Delete Item | `hx-delete="/items/1" hx-target="#item-1" hx-swap="outerHTML"` |
| Load on Page Load | `hx-get="/data" hx-trigger="load"` |
| Search with Delay | `hx-get="/search" hx-trigger="keyup changed delay:500ms"` |

---

Happy coding with HTMX! 🚀
