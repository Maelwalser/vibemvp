# Atlas Skill Guide (ariga.io)

## Installation

```bash
# macOS / Linux
curl -sSf https://atlasgo.sh | sh

# Go install
go install ariga.io/atlas/cmd/atlas@latest

# Atlas Go SDK
go get ariga.io/atlas
```

## HCL Schema Definition

```hcl
# schema.hcl
schema "public" {}

table "users" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }

  column "email" {
    type = varchar(255)
    null = false
  }

  column "name" {
    type = varchar(255)
    null = true
  }

  column "role" {
    type    = varchar(50)
    null    = false
    default = "USER"
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_users_email" {
    columns = [column.email]
    unique  = true
  }
}

table "posts" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }

  column "title" {
    type = varchar(500)
    null = false
  }

  column "author_id" {
    type = uuid
    null = false
  }

  column "published" {
    type    = boolean
    null    = false
    default = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_posts_author" {
    columns     = [column.author_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  index "idx_posts_author" {
    columns = [column.author_id]
  }
}
```

## Schema Inspection (Reverse-Engineering)

```bash
# Inspect existing database and output HCL schema
atlas schema inspect \
  --url "postgres://user:pass@localhost/mydb?sslmode=disable" \
  --format '{{ hcl . }}' \
  > schema.hcl

# Inspect and output SQL
atlas schema inspect \
  --url "postgres://user:pass@localhost/mydb?sslmode=disable" \
  --format '{{ sql . }}'
```

## Schema Diff (Drift Detection)

```bash
# Compare two databases
atlas schema diff \
  --from "postgres://user:pass@localhost/mydb_dev" \
  --to "postgres://user:pass@localhost/mydb_prod"

# Compare HCL file to live database
atlas schema diff \
  --from "file://schema.hcl" \
  --to "postgres://user:pass@localhost/mydb"

# Detect drift between desired state and current state
atlas schema diff \
  --from "postgres://user:pass@localhost/mydb" \
  --to "file://schema.hcl"
```

## Declarative Apply

```bash
# Apply schema.hcl to database (calculates diff and applies)
atlas schema apply \
  --url "postgres://user:pass@localhost/mydb?sslmode=disable" \
  --to "file://schema.hcl" \
  --dev-url "docker://postgres/15/dev"

# Dry run (print SQL without applying)
atlas schema apply \
  --url "postgres://user:pass@localhost/mydb?sslmode=disable" \
  --to "file://schema.hcl" \
  --dry-run
```

## Versioned Migration Workflow

```bash
# atlas.hcl project file
# atlas.hcl
env "dev" {
  src = "file://schema.hcl"
  url = "postgres://user:pass@localhost/mydb?sslmode=disable"
  dev = "docker://postgres/15/dev"
  migration {
    dir = "file://migrations"
  }
}

env "prod" {
  url = env("DATABASE_URL")
  migration {
    dir = "file://migrations"
  }
}
```

```bash
# Generate migration from schema diff
atlas migrate diff add_user_role \
  --env dev

# Apply pending migrations
atlas migrate apply --env prod

# Validate migration directory integrity
atlas migrate validate --env dev

# Show migration status
atlas migrate status --env prod

# Lint migration files (safety checks)
atlas migrate lint --env dev --latest 1
```

Generated migration files:

```
migrations/
├── 20240115130000_add_user_role.sql
├── 20240115130000_add_user_role.sum   # checksum file
└── atlas.sum                          # directory checksum
```

## CI Integration — GitHub Actions

```yaml
# .github/workflows/atlas.yml
name: Atlas CI

on:
  pull_request:
    paths:
      - "migrations/**"
      - "schema.hcl"

jobs:
  lint:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0   # needed for --latest comparison

      - uses: arigaio/atlas-action@master
        with:
          working-directory: "."
          dev-url: "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
          dir: "file://migrations"
```

## Multi-Tenant Migration Pattern

```hcl
# Per-tenant schema using HCL template
# tenant_schema.hcl.tmpl
schema "tenant_{{ .TenantID }}" {}

table "orders" {
  schema = schema["tenant_{{ .TenantID }}"]
  # ...
}
```

```bash
# Apply per-tenant (iterate over tenants in CI)
for tenant in tenant1 tenant2 tenant3; do
  atlas schema apply \
    --url "postgres://user:pass@localhost/mydb?sslmode=disable&search_path=${tenant}" \
    --to "file://schema.hcl" \
    --dev-url "docker://postgres/15/dev"
done
```

## atlas.hcl Full Example

```hcl
# atlas.hcl
data "hcl_template" "schema" {
  path = "schema.hcl"
}

env "local" {
  src = data.hcl_template.schema.url
  url = "postgres://postgres:postgres@localhost/mydb?sslmode=disable"
  dev = "docker://postgres/15/dev"
  migration {
    dir    = "file://migrations"
    format = atlas
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
```

## Anti-Patterns

- Never manually edit files in the `migrations/` directory — the `.sum` file will detect tampering.
- Always use `--dev-url` for a throw-away database during diff generation (not your real DB).
- Do not use declarative `schema apply` in production CI — use `migrate apply` for versioned, auditable deployments.
- Declarative apply is safe for development; versioned migrations provide rollback history for production.
