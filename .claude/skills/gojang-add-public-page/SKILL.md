---
name: gojang-add-public-page
description: Add or modify public, user-facing pages in a Gojang app. Use when creating static pages, marketing/content pages, dashboard-style public pages, page handlers, public routes, navigation links, or templates under app/views/templates or app/pages.
---

# Gojang Add Public Page

Use this skill to add public site pages without creating a new Ent model.

## Workflow

1. Read `docs/creating-static-pages.md` when the request is more than a trivial route/template addition.
2. Inspect existing public page patterns in:
   - `app/pages/pages.handler.go`
   - `app/pages/pages.route.go`
   - `app/views/templates/base.html`
   - related templates in `app/views/templates/`
3. Create top-level public templates in `app/views/templates/` unless the page belongs to a feature package.
4. Add a `PageHandler` method that calls `h.Renderer.Render(w, r, "<template>.html", &renderers.TemplateData{...})`.
5. Register the route in `PageRoutes`. Use `middleware.RequireAuth(sm, client)` for private user pages.
6. Add navigation in `base.html` only when the page should be globally discoverable.
7. Run `go test ./...` after code or template changes.

## Gojang Conventions

- Define both `{{define "title"}}...{{end}}` and `{{define "content"}}...{{end}}` in full page templates.
- Pass dynamic values through `TemplateData.Data` and read them as `.Data.Key` in templates.
- Use `middleware.GetUser(r.Context())` in handlers only when the handler needs explicit user data beyond what the renderer injects as `.User`.
- Keep public site code separate from admin code. Do not use admin templates, admin renderers, or `/admin` path checks in public handlers.
- Prefer existing CSS classes and public layout structure over introducing one-off styling.

## Useful References

- Static page guide: `docs/creating-static-pages.md`
- Architecture separation: `docs/architecture-separation.md`
- Renderer API: `app/gojang/views/renderers/renderer.go`
