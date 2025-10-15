# Database Migration Command

This command handles database migrations for the Gojang framework using golang-migrate.

## Usage

```bash
# Apply all pending migrations
go run ./gojang/cmd/migrate/main.go up

# Rollback the last migration
go run ./gojang/cmd/migrate/main.go down
```

Or using Task:

```bash
# Apply all pending migrations
task migrate

# Rollback the last migration
task migrate-down
```

## How It Works

The migration command:
1. Reads the database URL from your `.env` file
2. Connects to the database (supports both SQLite and PostgreSQL)
3. Executes migrations from the `gojang/models/migrations/` directory
4. Keeps track of which migrations have been applied

## Migration Files

Migration files are located in `gojang/models/migrations/` and follow this naming pattern:
- `NNNNNN_description.up.sql` - Migration to apply
- `NNNNNN_description.down.sql` - Migration to rollback

Example:
```
000001_create_users.up.sql
000001_create_users.down.sql
```

## Creating New Migrations

Use the `task migrate-create` command:

```bash
task migrate-create name=add_products_table
```

This creates empty migration files that you can fill in with your SQL.

## Dependencies

- `github.com/golang-migrate/migrate/v4` - Migration library
- `github.com/golang-migrate/migrate/v4/database/sqlite3` - SQLite driver
- `github.com/golang-migrate/migrate/v4/database/postgres` - PostgreSQL driver
- `github.com/golang-migrate/migrate/v4/source/file` - File source driver

## See Also

- [Taskfile Commands Guide](../../../docs/taskfile-guide.md)
- [Creating Data Models](../../../docs/creating-data-models.md)
