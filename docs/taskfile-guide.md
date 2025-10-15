# Taskfile Commands Guide

This guide provides documentation for the Taskfile commands available in the Gojang framework. Task is a cross-platform task runner and build tool that replaces Make.

## Prerequisites

Install Task (if not already installed):

**macOS/Linux:**
```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Or using Homebrew:
```bash
brew install go-task
```

**Windows:**
```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Or using Chocolatey:
```bash
choco install go-task
```

## Available Commands

Run `task --list` to see all available tasks.

---

## Database Migration Commands

### task migrate

Applies all pending database migrations.

**Usage:**
```bash
task migrate
```

**Output:**
- `✅ All migrations applied successfully` - All pending migrations were applied
- `✅ No pending migrations` - Database is already up to date

**Example:**
```bash
$ task migrate
task: [migrate] go run ./gojang/cmd/migrate/main.go up
✅ All migrations applied successfully
```

---

### task migrate-down

Rolls back the last applied migration.

**Usage:**
```bash
task migrate-down
```

**Output:**
- `✅ Last migration rolled back successfully` - Migration was rolled back
- `⚠️  No migrations to rollback` - No migrations to rollback
- `⚠️  No migrations to rollback (database is empty)` - Database has no migration history

**Example:**
```bash
$ task migrate-down
task: [migrate-down] go run ./gojang/cmd/migrate/main.go down
✅ Last migration rolled back successfully
```

**Warning:** Use with caution in production environments. This will drop tables and lose data.

---

### task migrate-create

Creates a new migration file pair (up and down).

**Usage:**
```bash
task migrate-create name=<migration_name>
```

**Parameters:**
- `name` - Name for the migration (e.g., `add_products_table`, `add_user_status_field`)

**Output:**
Creates two files in `gojang/models/migrations/`:
- `NNNNNN_<name>.up.sql` - Migration to apply changes
- `NNNNNN_<name>.down.sql` - Migration to rollback changes

**Example:**
```bash
$ task migrate-create name=add_products_table
task: [migrate-create] migrate create -ext sql -dir gojang/models/migrations -seq add_products_table
/home/runner/work/gojang-dev/gojang-dev/gojang/models/migrations/000002_add_products_table.up.sql
/home/runner/work/gojang-dev/gojang-dev/gojang/models/migrations/000002_add_products_table.down.sql
```

**Next Steps:**
1. Edit the `.up.sql` file to add your schema changes
2. Edit the `.down.sql` file to add the rollback logic
3. Run `task migrate` to apply the new migration

**Example Migration Files:**

`000002_add_products_table.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    price REAL NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
```

`000002_add_products_table.down.sql`:
```sql
DROP TABLE IF EXISTS products;
```

---

## Other Development Commands

### task dev
Run server with live reload (requires Air).

```bash
task dev
```

### task build
Build the web binary.

```bash
task build
```

### task test
Run tests.

```bash
task test
```

### task seed
Seed database with initial admin login.

```bash
task seed
```

### task schema-gen
Generate Ent code after schema changes.

```bash
task schema-gen
```

### task addpage
Create a new static page interactively.

```bash
task addpage
```

### task addmodel
Create a new data model interactively or via command-line flags.

**Interactive Mode:**
```bash
task addmodel
```

The command will prompt you for:
- Model name (e.g., 'Product', 'Category', 'Order')
- Model icon (e.g., '📦', '🏷️', '📋')
- Fields with types and validation

**Non-Interactive Mode (for automation):**
```bash
# Create a model with command-line flags
go run ./gojang/cmd/addmodel \
  --model Product \
  --icon "📦" \
  --fields "name:string:required,description:text,price:float:required,stock:int"
```

**Dry-Run Mode (preview changes):**
```bash
# See what files would be created without making changes
go run ./gojang/cmd/addmodel \
  --model Product \
  --fields "name:string:required,price:float" \
  --dry-run
```

**Available Flags:**
- `--model` - Model name (PascalCase)
- `--icon` - Model icon (default: 📄)
- `--fields` - Comma-separated fields in format `name:type[:required]`
- `--dry-run` - Preview changes without writing files
- `--timestamps` - Add created_at field (default: true)
- `--examples` - Show detailed usage examples

**Field Format:**
```
name:type[:required]
```

**Supported Types:** `string`, `text`, `int`, `float`, `bool`, `time`

**Example:**
```bash
# Quick model creation
go run ./gojang/cmd/addmodel \
  --model Article \
  --icon "📰" \
  --fields "title:string:required,content:text:required,published:bool"
```

**What It Creates:**
- Ent schema file
- Generated Ent code
- Form validation struct
- CRUD handlers
- Routes
- HTML templates (index, new, edit)
- Admin panel registration

See `gojang/cmd/addmodel/README.md` for comprehensive documentation.

### task clean
Clean build artifacts and generated files.

```bash
task clean
```

### task run
Build and run the server.

```bash
task run
```

---

## Migration Workflow

### 1. Creating a New Migration

When you need to add or modify database schema:

```bash
# Create migration files
task migrate-create name=add_email_verification

# Edit the generated files
# - gojang/models/migrations/NNNNNN_add_email_verification.up.sql
# - gojang/models/migrations/NNNNNN_add_email_verification.down.sql

# Apply the migration
task migrate
```

### 2. Rolling Back a Migration

If something goes wrong or you need to undo changes:

```bash
# Rollback the last migration
task migrate-down

# Fix the migration files if needed
# Then reapply
task migrate
```

### 3. Development to Production

**Development:**
- Use `task migrate` to apply migrations
- Test thoroughly
- Commit migration files to version control

**Production:**
- Pull latest code with migration files
- Run `task migrate` to apply new migrations
- Backup database before running migrations!

---

## Tips and Best Practices

### Migration Best Practices

1. **Always create both up and down migrations** - This allows you to rollback if needed

2. **Use IF EXISTS / IF NOT EXISTS** - Makes migrations idempotent and safer to rerun

3. **Test migrations locally first** - Always test on a development database before production

4. **Backup before migrating** - Especially in production environments
   ```bash
   # SQLite backup
   sqlite3 app.db ".backup backup.db"
   
   # PostgreSQL backup
   pg_dump -U gojang gojang > backup.sql
   ```

5. **Never modify existing migrations** - Once a migration is applied, create a new one to make changes

6. **Use sequential naming** - The `migrate-create` command automatically creates sequential numbers

7. **Keep migrations small and focused** - One logical change per migration

### Troubleshooting

**Problem:** `migrate` command not found when running `task migrate-create`

**Solution:** Install golang-migrate CLI:
```bash
go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Problem:** Migration fails with "dirty database"

**Solution:** This happens when a migration partially fails. You can force the version:
```bash
# Check current version
migrate -path gojang/models/migrations -database "sqlite://app.db" version

# Force to a specific version (use with caution!)
migrate -path gojang/models/migrations -database "sqlite://app.db" force VERSION
```

**Problem:** Need to see migration status

**Solution:** You can use the migrate CLI directly:
```bash
migrate -path gojang/models/migrations -database "sqlite://app.db" version
```

---

## Related Documentation

- [Creating Data Models](./creating-data-models.md)
- [Deployment Guide](./deployment-guide.md)
- [Quick Start Guide](./quick-start-data-model.md)

---

For more information about Task, visit [taskfile.dev](https://taskfile.dev/).
