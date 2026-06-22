---
name: gojang-add-data-model
description: Add or modify data-backed features in a Gojang app. Use when creating Ent schemas, generated model code, CRUD handlers, forms, feature route packages, templates, migrations, or admin registration for a new resource.
---

# Gojang Add Data Model

Use this skill to add an application resource with database-backed behavior.

## Workflow

1. Read `docs/quick-start-data-model.md` for simple resources or `docs/creating-data-models.md` for relationships, validation, or production-ready flows.
2. Add or update the Ent schema in `app/schema/<resource>.go`.
3. Run `go generate ./app/gojang/models` after schema changes.
4. Add form structs and validation tags in `app/views/forms/forms.go` when the public UI accepts user input.
5. Create a feature package such as `app/products/` with:
   - `<resource>.handler.go`
   - `<resource>.route.go`
   - `templates/index.html`
   - `.partial.html` templates for modal/forms/list fragments as needed
6. Register the feature route in `app/cmd/web/main.go`.
7. Register or customize admin behavior in `app/gojang/admin/models.go` when the resource should appear in the admin workspace.
8. Run `go test ./...`; if generated code changes, also inspect generated files for expected field and edge names.

## Patterns

- Follow `app/posts/` as the primary example for user-owned CRUD with HTMX modals and authorization checks.
- Keep public feature handlers under the app feature package. Do not add model-specific admin handlers unless the generic admin registry cannot express the behavior.
- Use Ent predicate packages for queries, for example `post.IDEQ(id)`.
- Use generated field constants such as `post.FieldCreatedAt` where available.
- Protect create/update/delete routes with `middleware.RequireAuth(sm, client)`.
- Check ownership with `middleware.OwnsResource(r, ownerID)` for user-owned records.

## Admin Integration

- Prefer `registry.RegisterModel(admin.ModelRegistration{...})` overrides in `app/gojang/admin/models.go`.
- Use `BeforeCreate`, `BeforeUpdate`, or `BeforeSave` hooks for admin-only data transforms.
- Use `QueryModifier` for eager loading relationships that list fields display.

## Useful References

- Quick start: `docs/quick-start-data-model.md`
- Full guide: `docs/creating-data-models.md`
- Existing feature: `app/posts/`
- Admin model registration: `app/gojang/admin/models.go`
