# CockroachDB Skill Guide

## Overview

CockroachDB is a distributed SQL database with full PostgreSQL wire protocol compatibility. Designed for horizontal scaling, multi-region deployments, and automatic failover. Key concepts: distributed transactions, multi-region survivability zones, CDC via CHANGEFEED, and hot-spot avoidance with distributed primary keys.

---

## Connection String

```
postgresql://user:password@host:26257/dbname?sslmode=verify-full&sslrootcert=/certs/ca.crt
```

For a multi-node cluster (list multiple hosts for automatic failover):
```
postgresql://user:password@node1:26257,node2:26257,node3:26257/dbname?sslmode=verify-full
```

CockroachDB Cloud (Serverless/Dedicated):
```
postgresql://user:password@cluster-name.region.cockroachlabs.cloud:26257/defaultdb?sslmode=verify-full
```

### Go (pgx)

```go
pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    return fmt.Errorf("db connect: %w", err)
}
```

CockroachDB is compatible with `pgx`, `database/sql`, and any PostgreSQL driver.

---

## Multi-Region Cluster

### Cluster Setup

```sql
-- Show regions available in the cluster
SHOW REGIONS FROM CLUSTER;

-- Set the primary region for a database
ALTER DATABASE mydb PRIMARY REGION "us-east1";

-- Add additional regions
ALTER DATABASE mydb ADD REGION "eu-west1";
ALTER DATABASE mydb ADD REGION "ap-southeast1";
```

### Survivability Goals

```sql
-- Survive zone failure (default: loses availability if a zone goes down)
ALTER DATABASE mydb SURVIVE ZONE FAILURE;

-- Survive region failure (requires 3+ regions; stronger guarantee)
ALTER DATABASE mydb SURVIVE REGION FAILURE;
```

### Regional Table Placement

```sql
-- REGIONAL BY TABLE: pin all rows to the primary region (fast reads/writes there)
ALTER TABLE orders SET LOCALITY REGIONAL BY TABLE IN PRIMARY REGION;

-- REGIONAL BY ROW: each row is pinned to a region (fast local reads everywhere)
ALTER TABLE user_profiles SET LOCALITY REGIONAL BY ROW;
-- Automatically adds a 'crdb_region' column; update it per row:
UPDATE user_profiles SET crdb_region = 'eu-west1' WHERE id = $1;

-- GLOBAL: replicate to all regions (low-latency reads everywhere, higher write latency)
ALTER TABLE config SET LOCALITY GLOBAL;
```

---

## Distributed Transaction Patterns

### Basic Transaction

```go
// CockroachDB recommends retry loops for serialization errors
func withRetry(ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
    const maxRetries = 5
    for attempt := 0; attempt < maxRetries; attempt++ {
        tx, err := db.Begin(ctx)
        if err != nil {
            return fmt.Errorf("begin tx: %w", err)
        }
        if err := fn(tx); err != nil {
            tx.Rollback(ctx)
            if isRetryable(err) {
                continue
            }
            return err
        }
        if err := tx.Commit(ctx); err != nil {
            if isRetryable(err) {
                continue
            }
            return fmt.Errorf("commit: %w", err)
        }
        return nil
    }
    return fmt.Errorf("transaction failed after %d retries", maxRetries)
}

func isRetryable(err error) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "40001"  // serialization_failure
    }
    return false
}
```

### SAVEPOINT for Application-Level Retry

CockroachDB supports PostgreSQL-compatible savepoints for retry:

```sql
BEGIN;
SAVEPOINT cockroach_restart;

-- ... statements ...

RELEASE SAVEPOINT cockroach_restart;  -- Commit
COMMIT;

-- On 40001 error:
ROLLBACK TO SAVEPOINT cockroach_restart;  -- Retry without full rollback
```

---

## Avoid Hot Spots with UUID Primary Keys

Sequential integer PKs (SERIAL/SEQUENCE) cause **hot spots** — all inserts go to the same range (the tail of the key space).

```sql
-- Bad: monotonically increasing PK creates hot spot
CREATE TABLE events (id SERIAL PRIMARY KEY, ...);

-- Good: UUID distributes inserts across ranges
CREATE TABLE events (
  id   UUID     DEFAULT gen_random_uuid() PRIMARY KEY,
  data JSONB    NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Also good: hash-sharded index for time-series
CREATE TABLE metrics (
  id         SERIAL,
  sensor_id  INT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL,
  value      FLOAT,
  PRIMARY KEY (id) USING HASH WITH (bucket_count=8)
);
```

---

## IMPORT INTO for Bulk Loading

```sql
-- Import from CSV stored in cloud storage
IMPORT INTO users (id, email, name, created_at)
CSV DATA (
  'gs://my-bucket/users-export.csv',
  'gs://my-bucket/users-export-2.csv'
)
WITH
  skip = '1',              -- skip header row
  nullif = '',             -- treat empty string as NULL
  allow_quoted_null = 'true';

-- Import from Postgres dump (PGDUMP format)
IMPORT TABLE legacy_orders FROM PGDUMP 's3://bucket/orders.sql'
WITH ignore_unsupported_statements;

-- Monitor import job
SHOW JOBS;
```

IMPORT INTO is faster than `INSERT` for large datasets because it bypasses the transaction log for the duration of the import.

---

## CHANGEFEED (CDC)

Change Data Capture streams row-level changes to external sinks.

```sql
-- Emit changes to Kafka
CREATE CHANGEFEED FOR TABLE orders
  INTO 'kafka://kafka-broker:9092'
  WITH
    updated,
    resolved='10s',            -- emit resolved timestamps every 10s
    format='json',
    key_in_value,
    topic_prefix='crdb-';

-- Emit to Google Cloud Pub/Sub
CREATE CHANGEFEED FOR TABLE users, orders
  INTO 'gcpubsub://my-project?topic=crdb-changes'
  WITH updated, resolved='5s', format='json';

-- Emit to S3 (for data lake ingestion)
CREATE CHANGEFEED FOR TABLE events
  INTO 's3://bucket/crdb-cdc?AWS_ACCESS_KEY_ID=...&AWS_SECRET_ACCESS_KEY=...'
  WITH updated, format='csv';

-- Monitor changefeeds
SHOW CHANGEFEED JOBS;

-- Pause / Resume
PAUSE JOB <job_id>;
RESUME JOB <job_id>;
```

---

## Schema Migration with cockroach sql

```bash
# Connect to cluster
cockroach sql --url "postgresql://user@host:26257/dbname?sslmode=verify-full" \
  --certs-dir=/certs

# Execute migration file
cockroach sql --url "$DATABASE_URL" < migrations/001_create_users.sql

# Check cluster/schema status
cockroach sql --url "$DATABASE_URL" --execute "SHOW TABLES;"
cockroach sql --url "$DATABASE_URL" --execute "SHOW CREATE TABLE users;"
```

### Recommended Migration Tools

Use **golang-migrate** or **Flyway** — both work with the PostgreSQL dialect:

```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

### Online Schema Changes

CockroachDB supports online schema changes (no full table lock):
```sql
-- Adding columns, indexes — non-blocking
ALTER TABLE orders ADD COLUMN discount DECIMAL(10,2) DEFAULT 0;
CREATE INDEX CONCURRENTLY idx_orders_status ON orders (status);

-- Backfilling data before adding NOT NULL constraint
ALTER TABLE orders ADD COLUMN region TEXT;
UPDATE orders SET region = 'us-east1' WHERE region IS NULL;
ALTER TABLE orders ALTER COLUMN region SET NOT NULL;
```

---

## Key Rules

- Use `UUID` or hash-sharded sequences as primary keys — sequential integers create write hot spots that degrade distributed performance
- Always implement a **retry loop** for `40001 serialization_failure` errors — these are normal in a distributed system and expected under contention
- `REGIONAL BY ROW` locality requires explicitly setting `crdb_region` per row — plan this into your data model early
- CHANGEFEED requires an **enterprise license** for Kafka/cloud sinks; the `experimental-sql` sink is available in all editions for development
- IMPORT INTO is atomic — if it fails, no partial data is committed
- Use `cockroach sql` for migrations in CI; pair with `golang-migrate` or `Flyway` for versioned schema management
- `SURVIVE REGION FAILURE` requires at least 3 regions; `SURVIVE ZONE FAILURE` requires at least 3 zones
- Test failover during development by simulating node loss: `cockroach quit --insecure --host=node2`
