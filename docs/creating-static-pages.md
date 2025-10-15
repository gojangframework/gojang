# Creating Static Pages (No Data Model)

This guide shows you how to add simple static pages to your Gojang application, such as an About page, Contact page, or Terms of Service page.

## Overview

Creating a static page involves three simple steps:
1. Create the HTML template
2. Create the handler function
3. Register the route

**Estimated time:** 5 minutes per page

---

## Step 1: Create the Template

Templates are located in `gojang/views/templates/`.

### Create `gojang/views/templates/about.html`

```html
{{define "title"}}About Us{{end}}

{{define "content"}}
<div class="container">
    <h1>About Us</h1>
    
    <div class="card">
        <p>
            Welcome to our application! We're building amazing things with Go and HTMX.
        </p>
        
        <h2>Our Mission</h2>
        <p>
            To provide the best web framework experience using modern technologies
            while maintaining simplicity and performance.
        </p>
        
        <h2>Technology Stack</h2>
        <ul>
            <li>Go - Fast, reliable backend</li>
            <li>HTMX - Dynamic interactions without heavy JavaScript</li>
            <li>Ent - Type-safe ORM</li>
            <li>Custom CSS - Clean, semantic styling</li>
        </ul>
        
        <div class="alert alert-info" style="margin-top: 2rem;">
            <p>
                <strong>Fun Fact:</strong> This entire page was created in less than 5 minutes!
            </p>
        </div>
    </div>
</div>
{{end}}
```

### Template Structure Explained

Your template must define two blocks:

- **`{{define "title"}}`** - Page title shown in browser tab and `<h1>` tags
- **`{{define "content"}}`** - Main page content

The template automatically inherits from `base.html`, which includes:
- ✅ Header with navigation
- ✅ Footer
- ✅ All CSS/JS dependencies
- ✅ Flash messages
- ✅ HTMX integration

---

## Step 2: Create the Handler

Handlers are located in `gojang/http/handlers/pages.go`.

### Add to `gojang/http/handlers/pages.go`

```go
// About renders the about page
func (h *PageHandler) About(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "about.html", &renderers.TemplateData{
		Title: "About Us",
		Data:  map[string]interface{}{},
	})
}
```

### Handler Patterns

#### Basic Static Page
```go
func (h *PageHandler) PageName(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "template.html", &renderers.TemplateData{
		Title: "Page Title",
		Data:  map[string]interface{}{},
	})
}
```

#### Page with Data
```go
func (h *PageHandler) Contact(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "contact.html", &renderers.TemplateData{
		Title: "Contact Us",
		Data: map[string]interface{}{
			"Email":   "support@example.com",
			"Phone":   "+1-234-567-8900",
			"Address": "123 Main St, City, State",
		},
	})
}
```

Then access in template:
```html
{{define "content"}}
<p>Email: {{.Data.Email}}</p>
<p>Phone: {{.Data.Phone}}</p>
{{end}}
```

#### Page with Current User
```go
func (h *PageHandler) Profile(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.GetUserFromContext(r.Context())
	
	h.Renderer.Render(w, r, "profile.html", &renderers.TemplateData{
		Title: "My Profile",
		Data: map[string]interface{}{
			"User": user,
		},
	})
}
```

---

## Step 3: Register the Route

Routes are located in `gojang/http/routes/pages.go`.

### Add to `gojang/http/routes/pages.go`

Find the `RegisterPageRoutes` function and add your route:

```go
func RegisterPageRoutes(r chi.Router, handler *handlers.PageHandler) {
	// Existing routes
	r.Get("/", handler.Home)
	
	// Add your new route
	r.Get("/about", handler.About)
}
```

### Route Patterns

#### Simple Route
```go
r.Get("/about", handler.About)
r.Get("/contact", handler.Contact)
r.Get("/terms", handler.Terms)
```

#### Route with Authentication Required
```go
r.Group(func(r chi.Router) {
	r.Use(middleware.RequireAuth)
	r.Get("/dashboard", handler.Dashboard)
	r.Get("/settings", handler.Settings)
})
```

#### Route with Admin Permission
```go
r.Group(func(r chi.Router) {
	r.Use(middleware.RequireAuth)
	r.Use(middleware.RequirePermission("is_staff"))
	r.Get("/admin-panel", handler.AdminPanel)
})
```

#### Route with URL Parameters
```go
r.Get("/blog/{slug}", handler.BlogPost)

// In handler:
func (h *PageHandler) BlogPost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	// ... fetch blog post by slug
}
```

---

## Complete Example: Contact Page

Let's create a complete contact page from scratch.

### 1. Create `gojang/views/templates/contact.html`

```html
{{define "title"}}Contact Us{{end}}

{{define "content"}}
<div class="container">
    <h1>Contact Us</h1>
    
    <div class="card">
        <h2>Get in Touch</h2>
        
        <div class="contact-info">
            <div class="contact-item">
                <strong>Email:</strong>
                <p>{{.Data.Email}}</p>
            </div>
            
            <div class="contact-item">
                <strong>Phone:</strong>
                <p>{{.Data.Phone}}</p>
            </div>
            
            <div class="contact-item">
                <strong>Address:</strong>
                <p>{{.Data.Address}}</p>
            </div>
        </div>
    </div>
    
    <div class="alert alert-info">
        <p>
            💡 <strong>Tip:</strong> You can extend this page with an HTMX-powered contact form!
        </p>
    </div>
</div>

<style>
    .contact-info {
        display: grid;
        gap: 1.5rem;
        margin-top: 1rem;
    }
    .contact-item strong {
        display: block;
        margin-bottom: 0.25rem;
    }
</style>
{{end}}
```

### 2. Add Handler to `gojang/http/handlers/pages.go`

```go
// Contact renders the contact page
func (h *PageHandler) Contact(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "contact.html", &renderers.TemplateData{
		Title: "Contact Us",
		Data: map[string]interface{}{
			"Email":   "support@gojang.dev",
			"Phone":   "+1 (555) 123-4567",
			"Address": "123 Framework Street, Go City, GC 12345",
		},
	})
}
```

### 3. Register Route in `gojang/http/routes/pages.go`

```go
func RegisterPageRoutes(r chi.Router, handler *handlers.PageHandler) {
	r.Get("/", handler.Home)
	r.Get("/about", handler.About)
	r.Get("/contact", handler.Contact)  // ✅ Add this line
}
```

### 4. Test Your Page

1. Restart the server: `go run ./gojang/cmd/web`
2. Visit: http://localhost:8080/contact
3. ✅ Done!

---

## Adding Navigation Links

To add your new page to the navigation menu:

### Edit `gojang/views/templates/base.html`

Find the navigation section and add your link:

```html
<nav class="flex space-x-4">
    <a href="/" class="text-white hover:text-blue-200">Home</a>
    <a href="/posts" class="text-white hover:text-blue-200">Posts</a>
    <a href="/about" class="text-white hover:text-blue-200">About</a>
    <a href="/contact" class="text-white hover:text-blue-200">Contact</a>
</nav>
```

---

## Adding HTMX Interactions (Optional)

Want to make your static page interactive? Add HTMX attributes!

### Example: Dynamic Content Loading

```html
{{define "content"}}
<div class="container">
    <h1>About Us</h1>
    
    <div class="card">
        <!-- Button that loads content via HTMX -->
        <button 
            class="btn btn-primary"
            hx-get="/about/team"
            hx-target="#team-section"
            hx-swap="innerHTML">
            Load Team Members
        </button>
        
        <!-- Content will appear here -->
        <div id="team-section" style="margin-top: 1.5rem;">
            <!-- Team data loads here -->
        </div>
    </div>
</div>
{{end}}
```

Then create a handler that returns just the HTML fragment:

```go
func (h *PageHandler) AboutTeam(w http.ResponseWriter, r *http.Request) {
	// Return just the fragment, not the full page
	w.Write([]byte(`
		<div class="team-grid">
			<div class="card">John Doe - CEO</div>
			<div class="card">Jane Smith - CTO</div>
			<div class="card">Bob Johnson - Designer</div>
		</div>
		<style>
			.team-grid {
				display: grid;
				grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
				gap: 1rem;
			}
		</style>
	`))
}
```

---

## Common Patterns

### FAQ Page with Collapsible Sections

```html
{{define "content"}}
<div class="container">
    <h1>Frequently Asked Questions</h1>
    
    <div class="faq-list">
        {{range .Data.FAQs}}
        <details class="card">
            <summary style="cursor: pointer; font-weight: 600;">{{.Question}}</summary>
            <p style="margin-top: 0.5rem;">{{.Answer}}</p>
        </details>
        {{end}}
    </div>
</div>

<style>
    .faq-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }
</style>
{{end}}
```

### Privacy Policy with Table of Contents

```html
{{define "content"}}
<div class="container">
    <h1>Privacy Policy</h1>
    
    <div class="card">
        <h2>Table of Contents</h2>
        <ul>
            <li><a href="#section-1">Information We Collect</a></li>
            <li><a href="#section-2">How We Use Your Data</a></li>
            <li><a href="#section-3">Your Rights</a></li>
        </ul>
    </div>
    
    <section id="section-1" class="card">
        <h2>1. Information We Collect</h2>
        <p>...</p>
    </section>
</div>
{{end}}
```

---

## Troubleshooting

### Page Not Found (404)
- ✅ Check route is registered in `routes/pages.go`
- ✅ Check handler method exists in `handlers/pages.go`
- ✅ Restart the server after changes

### Template Not Rendering
- ✅ Check template file exists in `gojang/views/templates/`
- ✅ Check `{{define "title"}}` and `{{define "content"}}` are present
- ✅ Check for syntax errors in template

### Styles Not Applied
- ✅ Check Tailwind CSS classes are correct
- ✅ Ensure `base.html` is loading CSS properly
- ✅ Clear browser cache

### Data Not Showing
- ✅ Check `Data` map in handler contains the key
- ✅ Check template uses correct syntax: `{{.Data.KeyName}}`
- ✅ Check for nil values

---

## Next Steps

- ✅ **Read:** [Creating Pages with Data Models](./creating-data-models.md)
- ✅ **Learn:** [HTMX Integration Patterns](./htmx-patterns.md)
- ✅ **Explore:** Check existing templates in `gojang/views/templates/` for more examples

---

## Quick Reference

| Task | File | Function |
|------|------|----------|
| Create template | `gojang/views/templates/page.html` | Define `title` and `content` blocks |
| Add handler | `gojang/http/handlers/pages.go` | Create `func (h *PageHandler) PageName()` |
| Register route | `gojang/http/routes/pages.go` | Add `r.Get("/path", handler.Method)` |
| Add navigation | `gojang/views/templates/base.html` | Add link in `<nav>` section |

---
