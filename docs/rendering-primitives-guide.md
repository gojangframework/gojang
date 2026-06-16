# Rendering Primitives Guide

This guide explains the reusable rendering and template primitives in Gojang, including shared template functions, direct partial rendering, reusable components, and the generic table component.

## Overview

Rendering primitives help you keep templates consistent and reduce repeated view code across pages, HTMX fragments, and components.

The Gojang renderer now supports:
- Shared template functions for strings, math, ranges, dates, numbers, and JSON
- Component templates loaded from `gojang/views/templates/components/`
- Direct partial rendering without requiring an HTMX request header
- Direct component rendering for reusable UI fragments
- A generic table component with optional pagination

**Key Files:**
- `gojang/views/renderers/renderer.go` - Public renderer and primitive APIs
- `gojang/views/templates/components/` - Shared component templates
- `gojang/views/templates/components/table.html` - Generic table component
- `gojang/views/renderers/renderer_test.go` - Renderer primitive tests

---

## How It Works

### 1. Component Loading on Startup

When the renderer starts, it loads component templates once and parses them into every page and partial template:

```go
renderer, err := renderers.NewRenderer(debug)
```

**What happens:**
1. Reads component templates from `templates/components/`
2. Walks all `.html` templates recursively
3. Skips components as standalone pages
4. Parses full pages with `base.html` and all components
5. Parses partials standalone with all components
6. Stores each renderable template in the renderer cache

```
templates/
├── base.html
├── components/
│   └── table.html              ← Parsed into pages and partials
├── posts/
│   ├── index.html              ← Can use {{template "table" .Data.Table}}
│   └── list.partial.html       ← Can use the same component
└── users/
    └── index.html
```

Components are not rendered directly as pages. They are shared named templates used by pages, partials, or `RenderComponent`.

---

## 2. Shared Template Functions

Gojang exposes one common function map through `renderers.TemplateFuncMap`.

### Math Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `add` | Add two integers | `{{add 2 3}}` |
| `sub` | Subtract two integers | `{{sub 5 2}}` |
| `subtract` | Alias for `sub` | `{{subtract 5 2}}` |
| `mul` | Multiply two integers | `{{mul 4 3}}` |
| `div` | Divide two integers, returns `0` for divide by zero | `{{div 10 2}}` |

### String Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `lower` | Lowercase a string | `{{lower .Name}}` |
| `join` | Join string slices | `{{join .Tags ", "}}` |
| `contains` | Check if a string slice contains an item | `{{contains .Roles "admin"}}` |
| `hasPrefix` | Check string prefix | `{{hasPrefix .CurrentPath "/admin"}}` |

### Range Functions

Use `until` when you need a zero-based range:

```html
{{range until 3}}
    <span>{{.}}</span>
{{end}}
```

Output values:

```text
0 1 2
```

Use `iterate` when you need a start and end:

```html
{{range iterate 2 5}}
    <span>{{.}}</span>
{{end}}
```

Output values:

```text
2 3 4
```

### Formatting Functions

```html
{{formatDate "Jan 02, 2006" .CreatedAt}}
{{formatNumber .Amount}}
```

`formatNumber` prints numeric values with two decimal places and handles pointers safely.

### JSON for JavaScript

Use `toJSON` when embedding server data into a script:

```html
<script>
    const chartData = {{toJSON .Data.ChartData}};
</script>
```

`toJSON` returns safe JavaScript output for script contexts. Avoid using it to display plain visible text.

### Translation Functions

The existing i18n helpers remain available:

```html
{{t . "welcome"}}
{{t . "welcome_user" .User.Email}}
{{range tArray . "feature_list"}}
    <li>{{.}}</li>
{{end}}
```

---

## 3. Direct Partial Rendering

Use `RenderPartial` when a handler needs to return a fragment regardless of whether the request includes `HX-Request: true`.

```go
func (h *PostHandler) NewModal(w http.ResponseWriter, r *http.Request) {
    err := h.Renderer.RenderPartial(w, r, "posts/new", &renderers.TemplateData{
        Title: "New Post",
    })
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to render form")
    }
}
```

Both names work:

```go
renderer.RenderPartial(w, r, "posts/new", data)
renderer.RenderPartial(w, r, "posts/new.partial.html", data)
```

**What `RenderPartial` does:**
- Forces `.IsHX` to `true`
- Adds CSRF token and authenticated user like `Render`
- Detects request language
- Sets `Content-Type: text/html; charset=utf-8`
- Executes the partial directly without `base.html`

This is useful for modal endpoints, preview endpoints, component refreshes, and tests.

---

## 4. Reusable Components

Component templates live in:

```text
gojang/views/templates/components/
```

Each component should define a named template:

```html
{{define "status_badge"}}
    <span class="badge badge-{{lower .Status}}">{{.Status}}</span>
{{end}}
```

Then call it from any page or partial:

```html
{{template "status_badge" .Data.Item}}
```

### Rendering a Component Directly

Handlers can render a component by name:

```go
func (h *PageHandler) TableFragment(w http.ResponseWriter, r *http.Request) {
    table := renderers.TableData{
        Columns: []string{"Name", "Status"},
        Rows: []renderers.TableRow{
            {Cells: []interface{}{"Gojang", "Ready"}},
        },
    }

    err := h.Renderer.RenderComponent(w, r, "table", table)
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to render table")
    }
}
```

`RenderComponent` is helpful when a route exists only to refresh one reusable UI block.

---

## 5. Generic Table Component

The built-in table component is named `table`.

### Data Shape

```go
type TableData struct {
    Columns      []string
    Rows         []TableRow
    EmptyMessage string
    Pagination   *PaginationData
}

type TableRow struct {
    Cells []interface{}
}

type PaginationData struct {
    Page       int
    TotalPages int
    TotalCount int
    HasPrev    bool
    HasNext    bool
    PrevURL    string
    NextURL    string
}
```

### Basic Table

```go
table := renderers.TableData{
    Columns: []string{"Title", "Author"},
    Rows: []renderers.TableRow{
        {Cells: []interface{}{"First Post", "admin@example.com"}},
        {Cells: []interface{}{"Second Post", "staff@example.com"}},
    },
}
```

Render inside a page:

```html
{{template "table" .Data.Table}}
```

### Empty Table

```go
table := renderers.TableData{
    Columns:      []string{"Title", "Author"},
    EmptyMessage: "No posts yet",
}
```

The component renders one row with the empty message spanning all columns.

### Paginated Table

```go
table := renderers.TableData{
    Columns: []string{"Title", "Author"},
    Rows: rows,
    Pagination: &renderers.PaginationData{
        Page:       page,
        TotalPages: totalPages,
        TotalCount: totalCount,
        HasPrev:    page > 1,
        HasNext:    page < totalPages,
        PrevURL:    fmt.Sprintf("?page=%d", page-1),
        NextURL:    fmt.Sprintf("?page=%d", page+1),
    },
}
```

The component handles disabled previous/next states and displays page totals.

---

## 6. Handler Pattern

A typical handler prepares domain data, maps it into a primitive view shape, then renders:

```go
func (h *PostHandler) Index(w http.ResponseWriter, r *http.Request) {
    posts, err := h.Client.Post.Query().All(r.Context())
    if err != nil {
        h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load posts")
        return
    }

    rows := make([]renderers.TableRow, 0, len(posts))
    for _, post := range posts {
        rows = append(rows, renderers.TableRow{
            Cells: []interface{}{post.Subject, post.CreatedAt},
        })
    }

    h.Renderer.Render(w, r, "posts/index.html", &renderers.TemplateData{
        Title: "Posts",
        Data: map[string]interface{}{
            "Table": renderers.TableData{
                Columns:      []string{"Subject", "Created"},
                Rows:         rows,
                EmptyMessage: "No posts yet",
            },
        },
    })
}
```

Template:

```html
{{define "content"}}
<div class="container">
    <h1>Posts</h1>
    {{template "table" .Data.Table}}
</div>
{{end}}
```

Keep mapping logic in handlers or small view-model helpers. Avoid making templates responsible for database-specific or business-specific decisions.

---

## 7. Best Practices

### Keep Components Generic

Good component data:

```go
renderers.TableData{
    Columns: []string{"Name", "Status"},
    Rows: rows,
}
```

Avoid passing raw domain objects when the component does not need them:

```go
// Avoid this for generic table components
Data: map[string]interface{}{
    "Employees": employees,
}
```

### Use Components for Repeated UI

Good candidates:
- Tables
- Empty states
- Status badges
- Pagination controls
- Flash messages
- Repeated summary rows

Less useful candidates:
- One-off page sections
- Highly specific form layouts
- Domain workflows with many conditional rules

### Prefer `RenderPartial` for Fragments

Use `RenderPartial` when the route should always return a fragment. Use `Render` when the route should adapt based on normal browser requests versus HTMX requests.

### Keep `toJSON` in Script Contexts

Good:

```html
<script>
    const filters = {{toJSON .Data.Filters}};
</script>
```

Avoid:

```html
<p>{{toJSON .Data.Filters}}</p>
```

For visible output, format values explicitly with normal template expressions.

---

## Troubleshooting

### Component Not Found

**Problem:** `component template table not found`

**Check:**
- The component file is under `gojang/views/templates/components/`
- The file extension is `.html`
- The component uses `{{define "component_name"}}`
- The name passed to `RenderComponent` matches the defined name

### Partial Renders as Full Page

**Problem:** HTMX endpoint returns the full layout

**Solution:** Use a `.partial.html` template and render it with:

```go
renderer.RenderPartial(w, r, "posts/new", data)
```

### JSON Appears Escaped

**Problem:** JSON appears as `{&#34;name&#34;:&#34;gojang&#34;}`

**Solution:** Use `toJSON` inside a JavaScript context:

```html
<script>
    const payload = {{toJSON .Data.Payload}};
</script>
```

### Date Formatting Fails

**Problem:** `formatDate` does not render as expected

**Check:** The value must be a `time.Time`:

```html
{{formatDate "2006-01-02" .CreatedAt}}
```

---

## Related Documentation

- [HTML Renderer Guide](./html-renderer-guide.md)
- [HTMX Integration Patterns](./htmx-patterns.md)
- [Creating Pages with Data Models](./creating-data-models.md)
