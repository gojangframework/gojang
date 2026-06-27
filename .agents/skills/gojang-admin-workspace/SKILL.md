---
name: gojang-admin-workspace
description: Work on Gojang's registry-driven admin panel. Use when customizing admin model overrides, fields, hooks, list columns, readonly or hidden fields, admin CRUD behavior, admin workspace templates, or staff-only admin routes under /admin.
---

# Gojang Admin Workspace

Use this skill for the staff-only admin panel and generic CRUD workspace.

## Workflow

1. Read `app/gojang/admin/README.md` and `docs/architecture-separation.md` before changing admin behavior.
2. Inspect current registry overrides in `app/gojang/admin/models.go`.
3. Remember that plain generated Ent models are discovered from `*models.Client`; add `RegisterModel` only for admin-specific overrides.
4. Prefer registry configuration over model-specific admin handlers:
   - `ListFields`
   - `HiddenFields`
   - `ReadonlyFields`
   - `OptionalFields`
   - `CustomFields`
   - `BeforeSave`, `BeforeCreate`, `BeforeUpdate`
   - `QueryModifier`
5. Change admin templates only for generic workspace behavior shared by resources.
6. Run focused admin tests first, then all tests:
   - `go test ./app/gojang/admin`
   - `go test ./...`

## Architecture Rules

- Keep admin and public site concerns separate.
- Admin routes are mounted under `/admin` and protected by auth, staff, and audit middleware.
- Canonical resource workspaces use `/admin/t/{resource}`.
- Public handlers must not branch on `/admin` paths or render admin templates.
- Admin renderer and templates live under `app/gojang/admin/`; public renderer and templates live under `app/gojang/views` and `app/views`.

## Registry Tips

- Do not add `RegisterModel` only to make a generated Ent model appear; `NewRegistry(client)` discovers Ent clients automatically.
- Add virtual admin-only inputs with `CustomFields` and clean them from the data map in hooks.
- Hash or validate sensitive values in `BeforeSave`, as the User registration does for password fields.
- Assign current-user-owned fields in `BeforeCreate` with `middleware.GetUser(ctx)`.
- Use `QueryModifier` to eager load relationships displayed in grid/list fields.
- Hide sensitive generated fields such as password hashes, email verification tokens, and password reset tokens.

## Useful References

- Admin package guide: `app/gojang/admin/README.md`
- Registry implementation: `app/gojang/admin/registry.go`
- Admin handlers: `app/gojang/admin/handler.go`
- Admin templates: `app/gojang/admin/views/`
