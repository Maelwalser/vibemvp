# golang-migrate Skill Guide

## Installation

```bash
# CLI tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Library
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

## Migration File Pairs

```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_add_user_role.up.sql
├── 000002_add_user_role.down.sql
└── 000003_add_posts.up.sql
└── 000003_add_posts.down.sql
```

- Files must be numbered sequentially (zero-padded).
- Each version requires both `.up.sql` and `.down.sql`.
- Never edit or delete applied migration files.

### Create Migration Files

```bash
migrate create -ext sql -dir migrations -seq create_users
# Creates: 000001_create_users.up.sql and 000001_create_users.down.sql
```

## Migration File Examples

```sql
-- 000001_create_users.up.sql
CREATE TABLE users (
    id         UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    name       VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
```

```sql
-- 000001_create_users.down.sql
DROP TABLE IF EXISTS users;
```

```sql
-- 000002_add_user_role.up.sql
ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'USER';
```

```sql
-- 000002_add_user_role.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS role;
```

## CLI Usage

```bash
export DATABASE_URL="postgres://user:pass@localhost/mydb?sslmode=disable"

# Apply all pending migrations
migrate -path migrations -database "$DATABASE_URL" up

# Apply N migrations
migrate -path migrations -database "$DATABASE_URL" up 2

# Rollback last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Rollback all
migrate -path migrations -database "$DATABASE_URL" down

# Goto specific version
migrate -path migrations -database "$DATABASE_URL" goto 3

# Show current version
migrate -path migrations -database "$DATABASE_URL" version

# Force version (fix dirty state after failed migration)
migrate -path migrations -database "$DATABASE_URL" force 2
```

## Embedded Library Usage

```go
package main

import (
    "database/sql"
    "fmt"
    "log"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/lib/pq"
)

func runMigrations(db *sql.DB) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("migrate: creating driver: %w", err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        "file://migrations",
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("migrate: creating instance: %w", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrate: running up: %w", err)
    }

    return nil
}
```

## Embed FS (Bundle Migrations in Binary)

```go
package main

import (
    "embed"
    "fmt"
    "log"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
    _ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(connStr string) error {
    db, err := sql.Open("pgx", connStr)
    if err != nil {
        return fmt.Errorf("open db: %w", err)
    }
    defer db.Close()

    src, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return fmt.Errorf("iofs source: %w", err)
    }

    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("driver: %w", err)
    }

    m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
    if err != nil {
        return fmt.Errorf("migrate instance: %w", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrate up: %w", err)
    }

    log.Println("migrations: applied successfully")
    return nil
}
```

## Advisory Locks (Multi-Instance Safety)

golang-migrate uses PostgreSQL advisory locks automatically when using the postgres driver — only one instance can run migrations at a time. No additional configuration needed.

```go
// Config options for advisory lock behavior
driver, err := postgres.WithInstance(db, &postgres.Config{
    MigrationsTable: "schema_migrations",  // default: schema_migrations
    DatabaseName:    "mydb",
    // Advisory lock is always used for postgres driver
})
```

## migrate.Steps vs migrate.Up vs migrate.Down

```go
// Apply all pending
m.Up()

// Apply exactly N migrations forward
m.Steps(3)

// Rollback exactly N migrations
m.Steps(-2)

// Rollback all
m.Down()

// Go to specific version
m.Migrate(5)
```

## Graceful Shutdown

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

done := make(chan error, 1)
go func() { done <- m.Up() }()

select {
case err := <-done:
    if err != nil && err != migrate.ErrNoChange {
        log.Fatalf("migration failed: %v", err)
    }
case <-ctx.Done():
    m.GracefulStop <- true   // Signal to stop after current migration finishes
    log.Fatal("migration timed out")
}
```

## Dirty State Recovery

If a migration fails mid-way, the database is marked "dirty":

```bash
# Check current state
migrate -path migrations -database "$DATABASE_URL" version
# Output: 3 (dirty)

# Manually fix the migration side effects in the DB, then:
migrate -path migrations -database "$DATABASE_URL" force 2
# Forces version to 2 (the last successfully applied migration)

# Re-apply
migrate -path migrations -database "$DATABASE_URL" up 1
```

## Anti-Patterns

- Never delete or rename applied migration files.
- Never modify an applied migration's SQL content.
- Do not use `down 0` (full rollback) in production without a tested recovery plan.
- Always handle `migrate.ErrNoChange` — it is not an error, just means nothing to apply.
- Keep `.up.sql` and `.down.sql` as strict inverses of each other.
