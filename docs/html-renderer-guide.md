# HTML Renderer Guide

This guide explains how the template rendering system works in Gojang, including template loading, caching, HTMX integration, and thread safety.

## Overview

The Gojang renderer handles:
- 🎨 Template parsing and caching
- 🔄 HTMX partial rendering
- 🔒 Thread-safe concurrent access
- 🐛 Hot-reload in debug mode
- 📦 Base template inheritance

**Key Files:**
- `gojang/views/renderers/renderer.go` - Public site renderer
- `gojang/admin/admin_renderer.go` - Admin panel renderer
- `gojang/views/templates/` - Public templates
- `gojang/admin/views/` - Admin templates

---

## How It Works

### 1. Template Loading on Startup

When your app starts, the renderer walks the template directory and parses all `.html` files:

```go
renderer, err := renderers.NewRenderer(debug)
// Automatically parses all templates in gojang/views/templates/
```

**What happens:**
1. Finds all `.html` files recursively
2. Identifies partials (files with `.partial.html`)
3. Parses full pages with `base.html` wrapper
4. Parses partials standalone (no wrapper)
5. Stores templates in memory map

```
templates/
├── base.html              ← Base layout (not cached directly)
├── home.html              ← Parsed with base.html
├── posts/
│   ├── index.html         ← Parsed with base.html
│   ├── list.partial.html  ← Parsed standalone
│   └── new.partial.html   ← Parsed standalone
```

---

## 2. Template Types

### Full Page Templates

Standard pages that wrap in `base.html`:

```html
<!-- posts/index.html -->
{{define "title"}}Posts{{end}}

{{define "content"}}
<h1>All Posts</h1>
<div id="posts-list">
    <!-- Content here -->
</div>
{{end}}
```

**Rendered as:**
- Browser request → Full HTML with header, footer, navigation
- HTMX request → Just the `content` block

### Partial Templates

Fragments for HTMX (end with `.partial.html`):

```html
<!-- posts/list.partial.html -->
{{range .Data.Posts}}
<div class="post">
    <h2>{{.Subject}}</h2>
    <p>{{.Body}}</p>
</div>
{{end}}
```

**Rendered as:**
- Always just the fragment content
- No `base.html` wrapper
- Perfect for HTMX swaps

---

## 3. Rendering Flow

### Standard Browser Request

```
User → /posts → Handler
                   ↓
         renderer.Render(w, r, "posts/index.html", data)
                   ↓
         Template parsed with base.html
                   ↓
         Full HTML page returned
```

**Result:** Complete page with header, nav, footer

### HTMX Request

```
User clicks button → hx-get="/posts/new"
                          ↓
                   Handler detects HX-Request header
                          ↓
         renderer.Render(w, r, "posts/new.partial.html", data)
                          ↓
         Just the partial content returned
                          ↓
         HTMX swaps it into target element
```

**Result:** Only the requested fragment, no full page reload

---

## 4. Thread Safety with Mutex

### Why Mutex is Needed

Go's HTTP server runs each request in its own **goroutine** (lightweight thread):

```go
// Multiple requests happening simultaneously:
goroutine 1: Rendering home.html     (reading templates)
goroutine 2: Rendering posts/list    (reading templates)
goroutine 3: Debug mode hot-reload   (writing templates)
```

Without synchronization → **race condition** → **crash** 💥

### RWMutex (Read-Write Mutex)

The renderer uses `sync.RWMutex` for efficient concurrent access:

```go
type Renderer struct {
    templates map[string]*template.Template
    mu        sync.RWMutex  // Protects templates map
    debug     bool
}
```

**Two types of locks:**

| Lock Type | Usage | Behavior |
|-----------|-------|----------|
| `RLock()` | Reading templates | Multiple reads can happen simultaneously |
| `Lock()` | Writing templates | Exclusive access, blocks all reads/writes |

### In Practice

**Reading (most common - 99% of operations):**
```go
r.mu.RLock()              // Multiple goroutines can read in parallel
tmpl, ok := r.templates[name]
r.mu.RUnlock()
```

**Writing (only in debug mode):**
```go
r.mu.Lock()               // Exclusive lock - blocks everything
r.templates = newTemplates
r.mu.Unlock()
```

### Why RWMutex vs Regular Mutex?

| Scenario | Regular Mutex | RWMutex |
|----------|---------------|---------|
| 100 reads | Serialized (slow) | Parallel (fast ⚡) |
| 1 write during reads | Safe but slow | Safe and optimized |
| Typical workload | Overkill | Perfect fit ✅ |

**RWMutex is ideal because:**
- Reads are frequent (every request)
- Writes are rare (only debug hot-reload)
- Reads don't conflict with each other

---

## 5. Debug Mode Hot-Reload

### How It Works

```go
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, name string, data *TemplateData) error {
    // In debug mode, reload templates on every request
    if r.debug {
        tmpl, err := parseTemplates()  // Re-parse from disk
        if err == nil {
            r.mu.Lock()                 // Exclusive lock
            r.templates = tmpl          // Replace cached templates
            r.mu.Unlock()
        }
    }
    
    // Continue rendering...
}
```

**Benefits:**
- ✅ Edit HTML files and see changes immediately
- ✅ No server restart needed
- ✅ Great for development

**Performance:**
- 🐌 Slower (parses files on every request)
- 🚫 **Never use in production**

### Enabling Debug Mode

```bash
# Set in environment
export DEBUG=true

# Or in code
renderer, err := renderers.NewRenderer(true)  // debug = true
```

---

## 6. Template Data Structure

### TemplateData Fields

```go
type TemplateData struct {
    Title       string                      // Page title
    Data        map[string]interface{}      // Your custom data
    User        *models.User                // Current authenticated user
    CSRFToken   string                      // CSRF protection token
    IsHX        bool                        // Is this an HTMX request?
    Errors      map[string]string           // Form validation errors
    CurrentPath string                      // Current URL path
    Flash       string                      // Flash message text
    FlashType   string                      // Flash type (success, error, info)
}
```

### Usage Example

```go
func (h *PostHandler) Index(w http.ResponseWriter, r *http.Request) {
    posts, _ := h.Client.Post.Query().All(r.Context())
    
    h.Renderer.Render(w, r, "posts/index.html", &renderers.TemplateData{
        Title: "All Posts",
        Data: map[string]interface{}{
            "Posts": posts,
            "Count": len(posts),
        },
    })
}
```

### Automatic Fields

These are set automatically by the renderer:

```go
// Automatically added - you don't set these
data.CSRFToken = nosurf.Token(req)             // CSRF token
data.User = middleware.GetUser(req.Context())  // Current user
data.IsHX = req.Header.Get("HX-Request") == "true"
data.CurrentPath = req.URL.Path
```

---

## 7. HTMX Detection

### How the Renderer Detects HTMX

HTMX adds a header to all requests:

```
HX-Request: true
```

The renderer checks this header:

```go
data.IsHX = req.Header.Get("HX-Request") == "true"
```

### Smart Rendering Logic

```go
// 1. HTMX request for a partial?
if data.IsHX && strings.Contains(name, ".partial.html") {
    // Render just the partial
    return tmpl.Execute(w, data)
}

// 2. HTMX request for full page?
if data.IsHX {
    // Render just the "content" block (no base.html wrapper)
    return tmpl.ExecuteTemplate(w, "content", data)
}

// 3. Regular browser request
// Render full page with base.html
return tmpl.ExecuteTemplate(w, "base.html", data)
```

### Example Flow

```go
// Browser visits /posts
// → Returns full HTML with header, nav, footer

// User clicks "New Post" button with hx-get="/posts/new"
// → Returns just the form partial (injected into modal)

// User clicks nav link with hx-boost="true"
// → Returns just the content block (swapped into main area)
```

---

## 8. Template Functions

Custom functions available in all templates:

### Built-in Functions

```go
funcMap := template.FuncMap{
    "add":      func(a, b int) int { return a + b },
    "sub":      func(a, b int) int { return a - b },
    "mul":      func(a, b int) int { return a * b },
    "div":      func(a, b int) int { return a / b },
    "lower":    func(s string) string { return strings.ToLower(s) },
    "contains": func(slice []string, item string) bool { ... },
}
```

### Usage in Templates

```html
<!-- Math operations -->
<p>Total: {{add .Data.Price .Data.Tax}}</p>
<p>Page {{add .CurrentPage 1}} of {{.TotalPages}}</p>

<!-- String operations -->
<p>Email: {{lower .User.Email}}</p>

<!-- Conditionals -->
{{if contains .Data.Tags "featured"}}
    <span class="badge">Featured</span>
{{end}}
```

---

## 9. Error Handling

### RenderError Helper

Built-in method for error pages:

```go
func (r *Renderer) RenderError(w http.ResponseWriter, req *http.Request, status int, message string) {
    w.WriteHeader(status)
    data := &TemplateData{
        Title: fmt.Sprintf("Error %d", status),
        Data: map[string]interface{}{
            "Status":  status,
            "Message": message,
        },
    }
    _ = r.Render(w, req, "error.html", data)
}
```

### Usage

```go
// 404 Not Found
h.Renderer.RenderError(w, r, http.StatusNotFound, "Post not found")

// 500 Internal Server Error
h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Database error")

// 403 Forbidden
h.Renderer.RenderError(w, r, http.StatusForbidden, "Access denied")
```

---

## 10. Complete Example

### Handler

```go
func (h *PostHandler) Index(w http.ResponseWriter, r *http.Request) {
    // Query posts from database
    posts, err := h.Client.Post.Query().
        WithAuthor().
        Order(models.Desc(post.FieldCreatedAt)).
        All(r.Context())
    
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load posts")
        return
    }

    // Render template
    h.Renderer.Render(w, r, "posts/index.html", &renderers.TemplateData{
        Title: "All Posts",
        Data: map[string]interface{}{
            "Posts": posts,
        },
    })
}
```

### Full Page Template

```html
<!-- posts/index.html -->
{{define "title"}}{{.Title}}{{end}}

{{define "content"}}
<div class="container">
    <div class="header-actions">
        <h1>Posts</h1>
        <button hx-get="/posts/new" 
                hx-target="#modal" 
                hx-swap="innerHTML" 
                class="btn btn-primary">
            New Post
        </button>
    </div>

    <div id="posts-list">
        {{template "posts/list.partial.html" .}}
    </div>
</div>

<div id="modal"></div>
{{end}}
```

### Partial Template

```html
<!-- posts/list.partial.html -->
{{range .Data.Posts}}
<div class="card">
    <h2>{{.Subject}}</h2>
    <p>{{.Body}}</p>
    <small>by {{.Edges.Author.Email}}</small>
    
    <button hx-get="/posts/{{.ID}}/edit" 
            hx-target="#modal"
            class="btn btn-sm">Edit</button>
</div>
{{else}}
<p class="text-muted">No posts yet.</p>
{{end}}
```

---

## 11. Performance Tips

### Production Setup

```go
// ❌ Don't do this in production
renderer := renderers.NewRenderer(true)  // debug = true

// ✅ Production configuration
renderer := renderers.NewRenderer(false)  // debug = false
```

**Why?**
- Debug mode parses templates on every request (slow)
- Production mode caches templates in memory (fast ⚡)

### Template Caching

```
First Request:
- Parse all templates from disk
- Store in memory map
- Serve request

Subsequent Requests (production):
- Read from memory cache (instant ⚡)
- No disk I/O

Subsequent Requests (debug):
- Re-parse from disk every time (slow 🐌)
- Good for development only
```

### Benchmark Comparison

| Mode | Request Time | Disk I/O |
|------|-------------|----------|
| Production | ~100μs | None ✅ |
| Debug | ~10ms | Every request ❌ |

**100x faster in production!**

---

## 12. Admin Panel Renderer

The admin panel has its own renderer with slight differences:

### Key Differences

| Feature | Public Renderer | Admin Renderer |
|---------|----------------|----------------|
| Base template | `base.html` | `admin_base.html` |
| Template dir | `views/templates/` | `admin/views/` |
| Partial handling | Standard | Always fragments |
| Extra functions | Basic | `fieldValue`, `getID`, `formatDateTime` |

### Admin-Specific Functions

```go
funcMap := template.FuncMap{
    // Standard functions
    "add", "sub", "mul", "div", "lower", "contains",
    
    // Admin-specific
    "fieldValue":     extractFieldValue,      // Get field from struct
    "getID":          getIDValue,             // Get ID field
    "formatDateTime": formatDateTimeField,    // Format time fields
}
```

---

## Common Patterns

### Pattern 1: Modal Forms

```go
// Handler returns partial
func (h *PostHandler) New(w http.ResponseWriter, r *http.Request) {
    h.Renderer.Render(w, r, "posts/new.partial.html", nil)
}
```

```html
<!-- Template with HTMX -->
<button hx-get="/posts/new" hx-target="#modal">New Post</button>
<div id="modal"></div>
```

### Pattern 2: List Updates

```go
// Handler returns updated list
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Create post...
    
    w.Header().Set("HX-Trigger", "closeModal")
    w.Header().Set("HX-Retarget", "#posts-list")
    
    h.Renderer.Render(w, r, "posts/list.partial.html", data)
}
```

### Pattern 3: Form Validation Errors

```go
errors := forms.Validate(form)
if len(errors) > 0 {
    h.Renderer.Render(w, r, "posts/new.partial.html", &renderers.TemplateData{
        Errors: errors,  // Show errors in form
    })
    return
}
```

---

## Quick Reference

### Creating a New Page

1. Create template: `views/templates/mypage.html`
2. Define blocks: `{{define "title"}}` and `{{define "content"}}`
3. Create handler: `func (h *PageHandler) MyPage(w, r) { ... }`
4. Render: `h.Renderer.Render(w, r, "mypage.html", data)`

### Creating a Partial

1. Create template: `views/templates/mypartial.partial.html`
2. No blocks needed - just raw HTML
3. Create handler that renders it
4. Use HTMX: `hx-get="/endpoint" hx-target="#somewhere"`

### Debug vs Production

| Environment | Debug Mode | Result |
|------------|-----------|---------|
| Development | `true` | Hot-reload, slower |
| Production | `false` | Cached, faster ⚡ |

---

## Troubleshooting

### Template Not Found

```
Error: template mypage.html not found
```

**Solutions:**
1. Check file exists in `views/templates/`
2. Check file name matches exactly (case-sensitive)
3. Check file has `.html` extension
4. Restart server to reload templates (if not in debug mode)

### Partial Not Rendering

```
Button clicks but nothing happens
```

**Solutions:**
1. Check partial ends with `.partial.html`
2. Check HTMX attributes: `hx-get`, `hx-target`, `hx-swap`
3. Check handler is registered in routes
4. Check browser console for errors
5. Verify HTMX is loaded (check Network tab)

### Race Condition Errors

```
fatal error: concurrent map read and map write
```

**Solution:**
- This means the mutex isn't being used properly
- Check all template map accesses use `RLock/RUnlock` or `Lock/Unlock`
- File a bug report - this shouldn't happen!

---

## Summary

The Gojang renderer provides:

✅ **Template caching** - Fast rendering in production  
✅ **HTMX support** - Smart partial vs full page detection  
✅ **Thread safety** - RWMutex for concurrent requests  
✅ **Hot reload** - Instant feedback during development  
✅ **Base layouts** - DRY template inheritance  
✅ **Type safety** - Structured data passing

**Remember:**
- Use `debug=false` in production
- Partials end with `.partial.html`
- HTMX requests automatically get optimized responses
- Thread safety is handled automatically via RWMutex

Happy rendering! 🎨
