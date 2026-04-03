# SQLite Skill Guide

## Overview

SQLite is an embedded, serverless database — the entire database is a single file. Ideal for: local-first apps, CLI tools, edge/embedded systems, test databases, and single-user services. Limitations: one writer at a time (WAL mode helps), no multi-process clustering, file-locking constraints.

---

## WAL Mode (Write-Ahead Logging)

WAL mode dramatically improves write concurrency: readers never block writers, writers never block readers. Required for any application with concurrent access.

```sql
PRAGMA journal_mode=WAL;
-- Returns: wal

-- Verify
PRAGMA journal_mode;
```

Set at connection open time (persists in the database file):
```go
db, err := sql.Open("sqlite3", "file:app.db?_journal_mode=WAL&_busy_timeout=5000")
```

WAL creates two additional files: `app.db-wal` and `app.db-shm`. Include them in backups.

### WAL Checkpoint

WAL accumulates changes; checkpoint writes them back to the main file:
```sql
PRAGMA wal_checkpoint(TRUNCATE);  -- Checkpoint and truncate WAL file
PRAGMA wal_checkpoint(PASSIVE);   -- Checkpoint without blocking readers
```

---

## Busy Timeout for Concurrent Writes

SQLite allows only **one writer at a time**. Without a busy timeout, concurrent writes fail immediately with `SQLITE_BUSY`.

```sql
PRAGMA busy_timeout=5000;  -- Wait up to 5000ms before returning SQLITE_BUSY
```

Via connection string (mattn/go-sqlite3):
```go
db, err := sql.Open("sqlite3", "file:app.db?_busy_timeout=5000&_journal_mode=WAL")
```

Via modernc.org/sqlite (CGo-free):
```go
db, err := sql.Open("sqlite", "file:app.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
```

---

## Foreign Key Enforcement

Foreign key constraints are **disabled by default** in SQLite. Enable per connection:

```sql
PRAGMA foreign_keys=ON;
```

```go
// Enable via connection hook (mattn/go-sqlite3)
sql.Register("sqlite3_fk", &sqlite3.SQLiteDriver{
    ConnectHook: func(conn *sqlite3.SQLiteConn) error {
        _, err := conn.Exec("PRAGMA foreign_keys=ON", nil)
        return err
    },
})
db, _ := sql.Open("sqlite3_fk", "app.db")
```

---

## FTS5 Full-Text Search

```sql
-- Create FTS5 virtual table
CREATE VIRTUAL TABLE articles_fts USING fts5(
  title,
  body,
  content=articles,    -- links to real table for content retrieval
  content_rowid=id     -- maps FTS rowid to articles.id
);

-- Populate from existing data
INSERT INTO articles_fts(rowid, title, body)
  SELECT id, title, body FROM articles;

-- Keep in sync with triggers
CREATE TRIGGER articles_ai AFTER INSERT ON articles BEGIN
  INSERT INTO articles_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
END;

CREATE TRIGGER articles_ad AFTER DELETE ON articles BEGIN
  INSERT INTO articles_fts(articles_fts, rowid, title, body)
    VALUES ('delete', old.id, old.title, old.body);
END;

CREATE TRIGGER articles_au AFTER UPDATE ON articles BEGIN
  INSERT INTO articles_fts(articles_fts, rowid, title, body)
    VALUES ('delete', old.id, old.title, old.body);
  INSERT INTO articles_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
END;

-- Search
SELECT a.id, a.title, rank
FROM articles_fts
JOIN articles a ON a.id = articles_fts.rowid
WHERE articles_fts MATCH 'quick brown fox'
ORDER BY rank;

-- Phrase search
WHERE articles_fts MATCH '"quick brown"'

-- Prefix search
WHERE articles_fts MATCH 'quick*'

-- Column filter
WHERE articles_fts MATCH 'title:SQLite body:performance'

-- Highlight snippet
SELECT snippet(articles_fts, 1, '<b>', '</b>', '...', 20) AS excerpt
FROM articles_fts WHERE articles_fts MATCH 'sqlite';
```

---

## JSON1 Extension

JSON1 is bundled with SQLite 3.38.0+ (enabled by default in most distributions).

```sql
-- Store JSON in TEXT column
CREATE TABLE events (id INTEGER PRIMARY KEY, data TEXT);
INSERT INTO events (data) VALUES ('{"type":"login","user_id":42,"tags":["web","mobile"]}');

-- Extract field
SELECT json_extract(data, '$.type') AS event_type FROM events;
SELECT json_extract(data, '$.user_id') AS uid FROM events;

-- Filter on JSON field
SELECT * FROM events WHERE json_extract(data, '$.type') = 'login';

-- Array access
SELECT json_extract(data, '$.tags[0]') AS first_tag FROM events;

-- Aggregate JSON into array
SELECT json_group_array(json_extract(data, '$.type')) AS types FROM events;

-- Build JSON object
SELECT json_object('id', id, 'type', json_extract(data, '$.type')) FROM events;

-- json_each to expand array
SELECT e.id, t.value AS tag
FROM events e, json_each(json_extract(e.data, '$.tags')) t;

-- Generated column + index on JSON field (SQLite 3.31.0+)
ALTER TABLE events ADD COLUMN event_type TEXT
  GENERATED ALWAYS AS (json_extract(data, '$.type')) STORED;
CREATE INDEX idx_events_type ON events (event_type);
```

---

## Embedded Use Cases & Constraints

### Connection Per Goroutine/Thread

SQLite connections are **not safe for concurrent use from multiple goroutines**. Use a connection pool with max 1 writer:

```go
// Write pool: 1 connection (serializes writes)
writeDB, err := sql.Open("sqlite3", "file:app.db?_journal_mode=WAL&_busy_timeout=5000")
if err != nil {
    log.Fatal(err)
}
writeDB.SetMaxOpenConns(1)

// Read pool: multiple connections (WAL allows concurrent reads)
readDB, err := sql.Open("sqlite3", "file:app.db?_journal_mode=WAL&mode=ro")
if err != nil {
    log.Fatal(err)
}
readDB.SetMaxOpenConns(4)
```

### In-Memory Database

```go
// Shared in-memory DB (all connections see same data)
db, _ := sql.Open("sqlite3", "file::memory:?cache=shared&_journal_mode=WAL")

// Per-connection in-memory (isolated, for testing)
db, _ := sql.Open("sqlite3", ":memory:")
```

### File Locking Limitations

- On **NFS/network shares**: SQLite file locking is unreliable — do not use SQLite over NFS
- On **Docker/container volumes**: works correctly with local volume mounts, not network-backed volumes
- Multiple **processes** can read concurrently (WAL mode), but write contention between processes requires `busy_timeout`
- **macOS**: `F_FULLFSYNC` is used for durability — slower than Linux but more crash-safe

### Optimal PRAGMA Configuration

```sql
-- Run at every connection open
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA foreign_keys=ON;
PRAGMA synchronous=NORMAL;   -- Safe with WAL; faster than FULL
PRAGMA cache_size=-64000;    -- 64MB page cache (negative = kibibytes)
PRAGMA temp_store=MEMORY;    -- Temp tables in RAM
PRAGMA mmap_size=268435456;  -- 256MB memory-mapped I/O
```

---

## Schema Patterns

```sql
-- Strict typing (SQLite 3.37.0+) — prevents type coercion surprises
CREATE TABLE users (
  id    INTEGER PRIMARY KEY,
  email TEXT    NOT NULL UNIQUE,
  name  TEXT    NOT NULL
) STRICT;

-- Autoincrement vs INTEGER PRIMARY KEY
-- INTEGER PRIMARY KEY = alias for rowid (fast, reuses deleted IDs)
-- AUTOINCREMENT = never reuses IDs (requires extra table scan, slower)
-- Use INTEGER PRIMARY KEY unless you need never-reuse guarantee

-- Upsert
INSERT INTO settings (user_id, key, value) VALUES (1, 'theme', 'dark')
  ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value;
```

---

## Key Rules

- Always set `PRAGMA journal_mode=WAL` and `PRAGMA busy_timeout` at connection open — default DELETE journal mode serializes reads with writes
- Enable `PRAGMA foreign_keys=ON` per connection — it is off by default and does not persist
- Use max 1 connection in the write pool; use multiple connections for the read pool (WAL allows concurrent reads)
- Never use SQLite over network file systems (NFS, SMB, CIFS)
- FTS5 triggers must be maintained manually — content tables do not auto-sync
- `GENERATED ALWAYS AS ... STORED` columns + indexes are the correct way to index JSON fields
- For testing, prefer named in-memory databases (`file::memory:?cache=shared`) to avoid test isolation issues with separate in-memory instances
- SQLite is not a replacement for PostgreSQL/MySQL in multi-server deployments; use it for embedded, local-first, or single-process services
