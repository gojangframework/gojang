---
name: gojang-operate-test
description: Run, test, migrate, seed, build, or prepare deployment for a Gojang app. Use for Taskfile commands, Ent generation, database migrations, seed data, local development server workflow, structured logging checks, deployment readiness, and test strategy.
---

# Gojang Operate Test

Use this skill for operational workflows: local runs, tests, migrations, seeds, builds, and deployment checks.

## Core Commands

Prefer Taskfile commands when Task is available:

```bash
task dev
task build
task test
task schema-gen
task migrate
task migrate-down
task seed
```

Use plain Go fallbacks when needed:

```bash
go run ./app/cmd/web
go build -o bin/web ./app/cmd/web
go test ./...
go generate ./app/gojang/models
go run ./app/cmd/migrate/main.go up
go run ./app/cmd/seed/main.go
```

## Workflow

1. Read `Taskfile.yml` before assuming command names.
2. Read `docs/taskfile-guide.md` for migration workflows.
3. Read `docs/testing-best-practices.md` before adding or reshaping tests.
4. Read `docs/deployment-guide.md` or `docs/distributed-deployment.md` for production deployment work.
5. Read `docs/logging-guide.md` before changing logging behavior.
6. Run the narrowest useful test first, then `go test ./...` before finishing code changes.

## Migration And Generation Rules

- Run `go generate ./app/gojang/models` after Ent schema changes.
- Use `task migrate-create name=<snake_case_name>` for new SQL migration files when explicit migrations are needed.
- Auto-migration runs on app startup through `db.AutoMigrate`.
- Do not delete or rewrite existing migrations unless the user explicitly asks and the change is safe for the current project state.

## Environment Checks

- Local config comes from `.env`; example defaults live in `.env.example`.
- Server address is built from `DEVHOST` and `PORT`.
- SQLite and PostgreSQL are supported through `DATABASE_URL`.
- SMTP is optional; the email service is disabled when `SMTP_HOST` is empty.

## Useful References

- Task guide: `docs/taskfile-guide.md`
- Testing guide: `docs/testing-best-practices.md`
- Deployment guide: `docs/deployment-guide.md`
- Distributed deployment: `docs/distributed-deployment.md`
- Logging guide: `docs/logging-guide.md`
