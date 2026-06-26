---
name: gojang-render-htmx-ui
description: Build or modify Gojang server-rendered UI with Go templates and HTMX. Use for partial templates, modals, fragment responses, renderer helpers, reusable components, table rendering, CSRF-aware HTMX forms, and dynamic interactions without heavy JavaScript.
---

# Gojang Render HTMX UI

Use this skill to build public server-rendered interfaces and HTMX interactions.

## Workflow

1. Read `docs/htmx-patterns.md` for interaction patterns and `docs/rendering-primitives-guide.md` for renderer/component APIs.
2. Inspect existing templates in `app/posts/templates/` and shared templates in `app/views/templates/`.
3. Use full page templates for normal navigation and `.partial.html` templates for modal, list, card, and refresh fragments.
4. Render with:
   - `Renderer.Render` for full pages or request-aware rendering
   - `Renderer.RenderPartial` when the endpoint should always return a fragment
   - `Renderer.RenderComponent` for direct reusable component responses
5. Include CSRF tokens in forms with `{{.CSRFToken}}`; `base.html` also wires HTMX CSRF headers.
6. Run `go test ./...`; for renderer changes, include `go test ./app/gojang/views/renderers`.

## HTMX Patterns

- Open modals with `hx-get`, `hx-target="#modal"` or the existing modal target, and `hx-swap="innerHTML"`.
- Submit creates with `hx-post`; submit edits with `hx-put`; submit deletes with `hx-delete`.
- Use `HX-Trigger: closeModal` after successful modal actions.
- Use `HX-Retarget` and `HX-Reswap` when a handler needs to update a list or card different from the original form target.
- Return validation errors by re-rendering the form partial with `TemplateData.Errors`.
- Prevent direct browser access to modal-only endpoints when the existing feature does so, by redirecting non-HTMX requests back to the index.

## Template Conventions

- Full pages define `title` and `content`.
- Partials use `.partial.html` suffix and render without `base.html`.
- Shared components live in `app/views/templates/components/` and define named templates.
- Use existing renderer functions: `add`, `sub`, `lower`, `join`, `contains`, `hasPrefix`, `until`, `iterate`, `formatDate`, `formatNumber`, `toJSON`, `t`, and `tArray`.
- Keep business logic in handlers or view-model helpers, not templates.
- Auth templates live under `app/views/templates/auth/`; keep forgot-password, reset-password, verification, and expired-link pages generic enough to avoid account enumeration.
- Use renderer session flash support for auth success/error messages instead of adding one-off query parameters.

## Useful References

- HTMX guide: `docs/htmx-patterns.md`
- Rendering guide: `docs/rendering-primitives-guide.md`
- Renderer code: `app/gojang/views/renderers/renderer.go`
- Example CRUD UI: `app/posts/templates/`
