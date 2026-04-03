# PostgreSQL Advanced Skill Guide

## Overview

Advanced PostgreSQL patterns for production use: JSONB document storage, window functions for analytics, CTEs for complex queries, fuzzy search with pg_trgm, partial and composite indexes, query analysis, advisory locks, and pub/sub with LISTEN/NOTIFY.

---

## JSONB Operators & Indexes

### Operators

```sql
-- -> returns JSON (preserves type)
SELECT data -> 'user' -> 'name' FROM events;

-- ->> returns text
SELECT data ->> 'email' FROM users WHERE data ->> 'status' = 'active';

-- ? checks key existence
SELECT * FROM products WHERE attributes ? 'color';

-- @> containment (left contains right)
SELECT * FROM orders WHERE metadata @> '{"status": "paid"}';

-- #> path navigation (array-safe)
SELECT data #> '{address,city}' FROM profiles;

-- #>> text extraction via path
SELECT data #>> '{tags,0}' FROM posts;
```

### JSONB Indexes

```sql
-- GIN index for containment (@>) and key existence (?)
CREATE INDEX idx_products_attrs ON products USING gin (attributes);

-- GIN with jsonb_path_ops (smaller, only supports @>)
CREATE INDEX idx_events_meta ON events USING gin (metadata jsonb_path_ops);

-- B-tree on extracted field (for =, <, > on specific key)
CREATE INDEX idx_users_email ON users ((data ->> 'email'));

-- Partial GIN for subset of rows
CREATE INDEX idx_active_tags ON posts USING gin (tags)
  WHERE published = true;
```

---

## Window Functions

```sql
-- ROW_NUMBER: unique sequential rank per partition
SELECT
  user_id,
  order_id,
  amount,
  ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at DESC) AS rn
FROM orders;

-- RANK: same rank for ties, gaps after ties
-- DENSE_RANK: same rank for ties, no gaps
SELECT
  product_id,
  sales,
  RANK()       OVER (ORDER BY sales DESC) AS rank,
  DENSE_RANK() OVER (ORDER BY sales DESC) AS dense_rank
FROM daily_sales;

-- LAG / LEAD: access previous / next row
SELECT
  date,
  revenue,
  LAG(revenue, 1, 0)  OVER (ORDER BY date) AS prev_day,
  LEAD(revenue, 1, 0) OVER (ORDER BY date) AS next_day,
  revenue - LAG(revenue, 1, 0) OVER (ORDER BY date) AS day_over_day
FROM daily_revenue;

-- SUM / AVG with frame spec (running totals)
SELECT
  date,
  amount,
  SUM(amount) OVER (ORDER BY date ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running_total
FROM transactions;

-- NTILE: divide into N buckets
SELECT user_id, NTILE(4) OVER (ORDER BY lifetime_value DESC) AS quartile
FROM users;
```

---

## CTEs (Common Table Expressions)

### Standard CTE

```sql
WITH active_users AS (
  SELECT id, email, created_at
  FROM users
  WHERE deleted_at IS NULL
    AND last_login > NOW() - INTERVAL '30 days'
),
user_orders AS (
  SELECT user_id, COUNT(*) AS order_count, SUM(total) AS total_spent
  FROM orders
  WHERE status = 'completed'
  GROUP BY user_id
)
SELECT
  u.email,
  COALESCE(o.order_count, 0) AS orders,
  COALESCE(o.total_spent, 0) AS spent
FROM active_users u
LEFT JOIN user_orders o ON o.user_id = u.id
ORDER BY spent DESC;
```

### Recursive CTE (org chart / tree traversal)

```sql
WITH RECURSIVE org_tree AS (
  -- Anchor: start from root node
  SELECT id, name, parent_id, 0 AS depth, ARRAY[id] AS path
  FROM employees
  WHERE parent_id IS NULL

  UNION ALL

  -- Recursive: join children to parents
  SELECT e.id, e.name, e.parent_id, t.depth + 1, t.path || e.id
  FROM employees e
  JOIN org_tree t ON t.id = e.parent_id
  WHERE NOT e.id = ANY(t.path)  -- cycle protection
)
SELECT id, name, depth, path FROM org_tree ORDER BY path;
```

---

## pg_trgm — Fuzzy Search

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Similarity search (0.0-1.0)
SELECT name, similarity(name, 'postgres') AS sim
FROM products
WHERE similarity(name, 'postgres') > 0.3
ORDER BY sim DESC;

-- GIN index for fast trigram search
CREATE INDEX idx_products_name_trgm ON products USING gin (name gin_trgm_ops);

-- ILIKE with index support (much faster with trgm index)
SELECT * FROM products WHERE name ILIKE '%postgre%';

-- word_similarity for partial matches
SELECT name FROM articles
WHERE word_similarity('quick fox', content) > 0.4
ORDER BY word_similarity('quick fox', content) DESC;
```

---

## Indexes: Partial & Composite

```sql
-- Partial index: only indexes rows matching WHERE
CREATE INDEX idx_orders_pending ON orders (created_at)
  WHERE status = 'pending';

-- Composite index: equality columns first, then range
CREATE INDEX idx_orders_user_date ON orders (user_id, created_at DESC);
-- Covers: WHERE user_id = $1 AND created_at > $2

-- Covering index (INCLUDE avoids table heap fetch)
CREATE INDEX idx_users_email_covering ON users (email) INCLUDE (id, name, status);

-- Expression index
CREATE INDEX idx_users_lower_email ON users (lower(email));
-- Used by: WHERE lower(email) = 'user@example.com'

-- BRIN for time-series (very small, for append-only data)
CREATE INDEX idx_events_ts_brin ON events USING brin (created_at);
```

---

## EXPLAIN ANALYZE

```sql
-- Read execution plan
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM orders WHERE user_id = 42 AND created_at > NOW() - INTERVAL '7 days';

-- Key metrics to examine:
-- "Seq Scan" on large tables → missing index
-- "rows=X loops=Y" actual vs estimated rows → statistics outdated → ANALYZE table
-- "Buffers: hit=X read=Y" → high read = disk I/O → caching or index issue
-- cost=start..total → higher total = more work

-- Force fresh statistics
ANALYZE orders;

-- Update planner statistics aggressiveness (default 100)
ALTER TABLE orders ALTER COLUMN user_id SET STATISTICS 500;
```

---

## pg_stat_statements

```sql
-- Enable extension
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find slowest queries
SELECT
  query,
  calls,
  mean_exec_time,
  total_exec_time,
  rows / calls AS avg_rows
FROM pg_stat_statements
WHERE calls > 100
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Find most called queries
SELECT query, calls, mean_exec_time
FROM pg_stat_statements
ORDER BY calls DESC
LIMIT 10;

-- Reset stats
SELECT pg_stat_statements_reset();
```

---

## Advisory Locks

Distributed application locks without a lock table:

```sql
-- Session-level lock (released when session ends)
SELECT pg_advisory_lock(12345);
-- ... do work ...
SELECT pg_advisory_unlock(12345);

-- Transaction-level lock (released at commit/rollback)
SELECT pg_advisory_xact_lock(hashtext('job:email-digest'));

-- Non-blocking try (returns false if already locked)
SELECT pg_try_advisory_lock(42) AS acquired;

-- Lock with two int4 keys (namespace + id)
SELECT pg_advisory_lock(1, user_id) FROM users WHERE id = $1;
```

Go usage:
```go
func withAdvisoryLock(ctx context.Context, db *pgxpool.Pool, lockID int64, fn func() error) error {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquire conn: %w", err)
    }
    defer conn.Release()

    if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
        return fmt.Errorf("advisory lock: %w", err)
    }
    defer conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockID)

    return fn()
}
```

---

## LISTEN / NOTIFY (Pub/Sub)

```sql
-- Publisher: send notification with optional payload (<8000 bytes)
NOTIFY channel_name, '{"event":"user_created","id":42}';

-- Inside a trigger:
CREATE OR REPLACE FUNCTION notify_order_update() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('order_updates', row_to_json(NEW)::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER order_update_notify
  AFTER INSERT OR UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION notify_order_update();
```

Go listener:
```go
conn, _ := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
if _, err := conn.Exec(ctx, "LISTEN order_updates"); err != nil {
    log.Fatal(err)
}
for {
    notification, err := conn.WaitForNotification(ctx)
    if err != nil {
        log.Printf("notification error: %v", err)
        break
    }
    log.Printf("channel=%s payload=%s", notification.Channel, notification.Payload)
}
```

---

## Key Rules

- Always use `timestamptz` (not `timestamp`) to avoid timezone bugs
- Prefer `text` over `varchar(n)` — PostgreSQL stores them identically; length constraints belong in application logic
- Run `EXPLAIN (ANALYZE, BUFFERS)` on any query taking >100ms before optimizing indexes
- Composite index column order: equality filters first, then range/sort columns
- JSONB GIN indexes can be large — use `jsonb_path_ops` for containment-only queries to reduce size
- Advisory locks are per-connection; always release in a `defer` or finally block
- LISTEN/NOTIFY payloads are limited to ~8000 bytes; send IDs, not full records
- `pg_stat_statements` is essential for production query monitoring — enable at cluster startup
