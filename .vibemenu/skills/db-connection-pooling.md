# Database Connection Pooling Skill Guide

## Core Concepts

A connection pool maintains a set of open database connections that are reused across requests. Key parameters:

| Parameter | Description | Typical Value |
|-----------|-------------|---------------|
| `min` / `min_idle` | Connections kept open when idle | 2–5 |
| `max` / `max_size` | Hard cap on open connections | CPU cores × 2–4 (Postgres: ≤50) |
| `connection_timeout` | Max time to wait for a connection from pool | 3–10s |
| `idle_timeout` | Idle connection closed after N seconds | 300–600s |
| `max_lifetime` | Connection replaced after N seconds (prevent stale state) | 1800–3600s |
| `keepalive` | SQL or TCP ping to detect dead connections | `SELECT 1` every 60s |

## Pool Size Tuning

```
PostgreSQL optimal pool size ≈ (number of CPU cores) × 2 + effective_spindle_count
```

Common starting point:
- **OLTP workloads:** max = 20–50 per application node
- **Reporting/OLAP queries:** separate pool, max = 5–10 (queries are long, fewer connections needed)
- **Workers/background jobs:** separate pool, max = 5–20

Rule: Total connections across all app instances must not exceed Postgres `max_connections` minus a headroom buffer (10–20) for admin access.

## HikariCP (Java / Kotlin / JVM)

```java
// HikariCP — fastest JVM connection pool
HikariConfig config = new HikariConfig();
config.setJdbcUrl(System.getenv("DATABASE_URL"));
config.setUsername(System.getenv("DB_USER"));
config.setPassword(System.getenv("DB_PASSWORD"));

config.setMaximumPoolSize(20);        // max open connections
config.setMinimumIdle(5);            // min idle connections
config.setConnectionTimeout(10_000); // 10s wait for connection from pool
config.setIdleTimeout(600_000);      // 10m before idle connection closed
config.setMaxLifetime(1_800_000);    // 30m max connection age
config.setKeepaliveTime(60_000);     // ping every 60s to keep alive

// Leak detection (dev/staging)
config.setLeakDetectionThreshold(5_000); // warn after 5s held open

config.setPoolName("main-pool");
config.setAutoCommit(false);         // prefer explicit transactions

HikariDataSource ds = new HikariDataSource(config);
```

Spring Boot YAML:
```yaml
spring:
  datasource:
    url: ${DATABASE_URL}
    hikari:
      maximum-pool-size: 20
      minimum-idle: 5
      connection-timeout: 10000
      idle-timeout: 600000
      max-lifetime: 1800000
      keepalive-time: 60000
      leak-detection-threshold: 5000
      pool-name: main-pool
```

## pgx / pgxpool (Go)

```go
import (
    "context"
    "os"
    "time"
    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
    cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
    if err != nil {
        return nil, fmt.Errorf("pgxpool: parse config: %w", err)
    }

    cfg.MaxConns = 20
    cfg.MinConns = 5
    cfg.MaxConnLifetime = 30 * time.Minute
    cfg.MaxConnIdleTime = 10 * time.Minute
    cfg.HealthCheckPeriod = 1 * time.Minute
    cfg.ConnConfig.ConnectTimeout = 10 * time.Second

    // Connection leak detection via BeforeAcquire/AfterRelease hooks
    cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
        // Return false to reject a connection (e.g., if unhealthy)
        return true
    }

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("pgxpool: create: %w", err)
    }

    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("pgxpool: ping: %w", err)
    }

    return pool, nil
}
```

Usage:
```go
// Acquire returns the connection to the pool after the function returns
conn, err := pool.Acquire(ctx)
if err != nil {
    return fmt.Errorf("acquire: %w", err)
}
defer conn.Release()

rows, err := conn.Query(ctx, "SELECT id, email FROM users LIMIT $1", 10)
// ...

// Or use pool directly for simple queries
var email string
err = pool.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", id).Scan(&email)
```

## node-postgres / pg (Node.js)

```typescript
import { Pool, PoolConfig } from "pg";

const config: PoolConfig = {
  connectionString: process.env.DATABASE_URL,
  max: 20,               // max connections in pool
  min: 2,                // min idle connections
  idleTimeoutMillis: 600_000,   // 10m idle timeout
  connectionTimeoutMillis: 10_000, // 10s wait for connection
  maxUses: 7500,         // recycle connection after N queries (prevents stale state)
  allowExitOnIdle: false,
};

const pool = new Pool(config);

pool.on("error", (err) => {
  console.error("Unexpected pool error", err);
});

// Usage
const client = await pool.connect();
try {
  await client.query("BEGIN");
  const result = await client.query("SELECT * FROM users WHERE id = $1", [id]);
  await client.query("COMMIT");
  return result.rows[0];
} catch (err) {
  await client.query("ROLLBACK");
  throw err;
} finally {
  client.release();   // MUST always release
}
```

## Separate Pools Per Component

Use separate pools with different sizes for different workloads:

```go
// Go example — separate OLTP and reporting pools
type DB struct {
    OLTP      *pgxpool.Pool
    Reporting *pgxpool.Pool
}

func NewDB(ctx context.Context) (*DB, error) {
    oltpCfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
    oltpCfg.MaxConns = 30
    oltpCfg.MaxConnLifetime = 30 * time.Minute

    reportCfg, _ := pgxpool.ParseConfig(os.Getenv("REPORTING_DB_URL"))
    reportCfg.MaxConns = 5
    reportCfg.MaxConnLifetime = 60 * time.Minute
    reportCfg.MaxConnIdleTime = 5 * time.Minute

    oltp, _ := pgxpool.NewWithConfig(ctx, oltpCfg)
    report, _ := pgxpool.NewWithConfig(ctx, reportCfg)

    return &DB{OLTP: oltp, Reporting: report}, nil
}
```

## Connection Leak Detection

A connection leak occurs when code acquires a connection but fails to release it.

```java
// HikariCP: leakDetectionThreshold logs a warning after N ms
config.setLeakDetectionThreshold(5_000); // 5 seconds

// Stack trace in log helps identify the callsite
```

```go
// pgxpool: BeforeAcquire/AfterRelease + timeout context
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

conn, err := pool.Acquire(ctx)
// If context expires before acquiring, returns error — detects pool exhaustion
```

```typescript
// node-postgres: always use try/finally
const client = await pool.connect();
try {
  // ... query
} finally {
  client.release(); // Always called, even on error
}
```

## Failover & Retry

```go
// Retry with exponential backoff on transient connection errors
func withRetry(ctx context.Context, pool *pgxpool.Pool, fn func(*pgxpool.Conn) error) error {
    backoff := 100 * time.Millisecond
    for attempt := 0; attempt < 3; attempt++ {
        conn, err := pool.Acquire(ctx)
        if err != nil {
            time.Sleep(backoff)
            backoff *= 2
            continue
        }
        err = fn(conn)
        conn.Release()
        if err == nil {
            return nil
        }
        // Only retry on connection-level errors, not query errors
        if !isRetryableError(err) {
            return err
        }
        time.Sleep(backoff)
        backoff *= 2
    }
    return fmt.Errorf("max retries exceeded")
}
```

## Metrics to Monitor

```
Pool utilization = active_connections / max_connections
Target: < 80% sustained

Wait queue depth = connections_waiting_for_pool
Target: 0 (any sustained queue = pool exhaustion)

Connection acquisition time
Target: < 5ms p99

Connection errors / timeouts
Target: 0
```

## Anti-Patterns

- Do not set `max` too high — exceeding Postgres `max_connections` causes connection errors.
- Do not set `max` to 1 for shared services — creates a bottleneck.
- Never use a single pool for both OLTP and long-running reporting queries.
- Always release connections in `finally`/`defer`/cleanup — leaks exhaust the pool silently.
- Do not use `idleTimeout = 0` (never expire) — stale connections appear healthy but error on use.
- Do not tune pool size without load testing — theory and practice differ under bursty traffic.
