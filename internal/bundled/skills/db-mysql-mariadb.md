# MySQL / MariaDB Skill Guide

## Overview

MySQL and MariaDB share the InnoDB storage engine and most SQL syntax. Key differentiators: InnoDB B-tree index strategy, composite index column ordering, transaction isolation levels, JSON column support (MySQL 5.7.8+, MariaDB 10.2+), binary logging for replication, and query optimization via EXPLAIN FORMAT=JSON.

---

## InnoDB B-Tree Index Strategy

InnoDB uses a **clustered index** (primary key defines row storage order). Secondary indexes store the primary key value as a pointer — so wide primary keys bloat all secondary indexes.

```sql
-- Good: narrow surrogate PK
CREATE TABLE orders (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  status     VARCHAR(20)     NOT NULL,
  total      DECIMAL(10,2)   NOT NULL,
  created_at DATETIME        NOT NULL
) ENGINE=InnoDB;

-- Bad: wide natural PK (UUID stored as CHAR(36) wastes space in every secondary index)
-- Use BINARY(16) + UUID_TO_BIN() if UUIDs are required

-- Covering index: query resolved entirely from index without table access
CREATE INDEX idx_orders_user_status ON orders (user_id, status, total);
-- SELECT total FROM orders WHERE user_id = 42 AND status = 'paid' → index-only scan
```

### Index Cardinality

```sql
-- Check cardinality (estimated distinct values)
SHOW INDEX FROM orders;

-- Analyze to update statistics
ANALYZE TABLE orders;
```

---

## Composite Index Column Order

Rule: **equality columns first, range or ORDER BY columns last**.

```sql
-- Query: WHERE status = 'active' AND created_at > '2024-01-01' ORDER BY created_at
CREATE INDEX idx_status_created ON orders (status, created_at);
-- ✓ status equality, then created_at range + sort — single index satisfies all

-- Wrong order:
CREATE INDEX idx_created_status ON orders (created_at, status);
-- ✗ range on created_at prevents using status from the index

-- Covering index including SELECT columns
CREATE INDEX idx_user_status_cover ON orders (user_id, status) INCLUDE (total, created_at);
-- MySQL 8.0+ supports INCLUDE; MariaDB uses partial workaround
```

For MariaDB (which lacks INCLUDE), add the columns to the index key itself if they are low-cardinality:
```sql
CREATE INDEX idx_user_status_cover ON orders (user_id, status, total, created_at);
```

---

## Transaction Isolation Levels

```sql
-- View current level
SELECT @@GLOBAL.transaction_isolation;
SELECT @@SESSION.transaction_isolation;

-- Set for session
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;

-- Set globally (requires restart or SET GLOBAL)
SET GLOBAL transaction_isolation = 'READ-COMMITTED';
```

| Level | Dirty Read | Non-repeatable Read | Phantom Read | Use Case |
|-------|-----------|---------------------|--------------|----------|
| `READ UNCOMMITTED` | Yes | Yes | Yes | Never use in production |
| `READ COMMITTED` | No | Yes | Yes | High-concurrency OLTP (PostgreSQL default) |
| `REPEATABLE READ` | No | No | Yes* | MySQL/MariaDB default; consistent snapshot |
| `SERIALIZABLE` | No | No | No | Financial transactions; high lock contention |

*InnoDB avoids phantom reads at REPEATABLE READ via gap locks.

```sql
-- Explicit transaction with isolation
START TRANSACTION;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;

SELECT balance FROM accounts WHERE id = 1 FOR UPDATE;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
UPDATE accounts SET balance = balance + 100 WHERE id = 2;

COMMIT;
```

---

## JSON Column

```sql
-- Schema
CREATE TABLE events (
  id      BIGINT AUTO_INCREMENT PRIMARY KEY,
  payload JSON NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert
INSERT INTO events (payload) VALUES ('{"type": "login", "user_id": 42, "ip": "1.2.3.4"}');

-- JSON_EXTRACT (returns JSON value)
SELECT JSON_EXTRACT(payload, '$.user_id') AS user_id FROM events;

-- -> shorthand (MySQL 5.7.9+)
SELECT payload -> '$.type' AS event_type FROM events;

-- ->> shorthand (unquoted text)
SELECT payload ->> '$.ip' AS ip_address FROM events;

-- WHERE clause filtering
SELECT * FROM events WHERE payload ->> '$.type' = 'login';
SELECT * FROM events WHERE JSON_EXTRACT(payload, '$.user_id') = 42;

-- JSON_CONTAINS for array membership
SELECT * FROM events WHERE JSON_CONTAINS(payload -> '$.tags', '"urgent"');

-- Functional index on JSON field (MySQL 8.0+)
ALTER TABLE events ADD COLUMN user_id BIGINT AS (payload ->> '$.user_id') VIRTUAL;
CREATE INDEX idx_events_user_id ON events (user_id);

-- MariaDB: use generated column + index
ALTER TABLE events ADD COLUMN event_type VARCHAR(50)
  AS (JSON_UNQUOTE(JSON_EXTRACT(payload, '$.type'))) PERSISTENT;
CREATE INDEX idx_event_type ON events (event_type);
```

---

## EXPLAIN FORMAT=JSON

```sql
EXPLAIN FORMAT=JSON
SELECT o.id, u.email, o.total
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE o.status = 'pending' AND o.created_at > '2024-01-01';
```

Key fields in JSON output:
```json
{
  "query_block": {
    "select_id": 1,
    "cost_info": { "query_cost": "245.80" },
    "nested_loop": [{
      "table": {
        "table_name": "o",
        "access_type": "ref",        // ref = index lookup; ALL = full scan (bad)
        "key": "idx_status_created", // which index used
        "used_columns": ["id", "user_id", "total", "status", "created_at"],
        "rows_examined_per_scan": 150,
        "filtered": 33.33,           // % of rows passing WHERE after index
        "using_index": false         // true = covering index (no table lookup)
      }
    }]
  }
}
```

Anti-patterns to spot:
- `"access_type": "ALL"` on large tables → missing index
- `"filtered": < 10` → low selectivity; consider composite index
- `"using_temporary": true` + `"using_filesort": true` → expensive GROUP BY / ORDER BY

---

## Binary Logging for Replication

```sql
-- Check binary logging status
SHOW VARIABLES LIKE 'log_bin';
SHOW BINARY LOGS;
SHOW MASTER STATUS;

-- my.cnf / my.ini config for replication
```

```ini
[mysqld]
server-id          = 1
log_bin            = /var/log/mysql/mysql-bin.log
binlog_format      = ROW          # ROW (safe), STATEMENT (compact), MIXED
binlog_row_image   = MINIMAL      # MINIMAL (efficient), FULL (complete before/after)
expire_logs_days   = 7
max_binlog_size    = 500M
sync_binlog        = 1            # Flush to disk on every commit (ACID safe)
```

### Replica Setup

```sql
-- On replica
CHANGE MASTER TO
  MASTER_HOST='primary-db.internal',
  MASTER_USER='repl',
  MASTER_PASSWORD='secret',
  MASTER_LOG_FILE='mysql-bin.000001',
  MASTER_LOG_POS=154;

START SLAVE;
SHOW SLAVE STATUS\G
-- Check: Seconds_Behind_Master, Last_Error
```

### GTID-Based Replication (MySQL 5.6+, MariaDB 10.0+)

```ini
gtid_mode          = ON
enforce_gtid_consistency = ON
```

```sql
CHANGE MASTER TO
  MASTER_HOST='primary-db.internal',
  MASTER_AUTO_POSITION=1;
```

---

## Slow Query Log

```ini
[mysqld]
slow_query_log         = 1
slow_query_log_file    = /var/log/mysql/slow.log
long_query_time        = 1.0      # seconds (0.1 for sub-second)
log_queries_not_using_indexes = 1
min_examined_row_limit = 1000     # Only log if >1000 rows examined
```

```sql
-- Enable at runtime
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 0.5;

-- Analyze with mysqldumpslow
-- mysqldumpslow -s t -t 20 /var/log/mysql/slow.log
```

---

## Key Rules

- Use `BIGINT UNSIGNED AUTO_INCREMENT` as primary key unless using UUID (then prefer `BINARY(16)` + `UUID_TO_BIN()`)
- Default isolation level `REPEATABLE READ` is safe for most apps; switch to `READ COMMITTED` for high-concurrency workloads to reduce gap lock contention
- `ROW` binlog format is required for safe replication with non-deterministic functions (NOW(), UUID())
- Always run `ANALYZE TABLE` after bulk inserts to refresh optimizer statistics
- Generated/virtual columns + indexes are the correct approach for indexing JSON fields in MySQL 8.0+
- Never use `SELECT *` in application code — specifying columns enables covering index usage
- `EXPLAIN FORMAT=JSON` gives more detail than plain `EXPLAIN`; use it for complex joins
- Set `sync_binlog=1` and `innodb_flush_log_at_trx_commit=1` together for ACID durability
