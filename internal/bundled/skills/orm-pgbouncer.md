# PgBouncer Skill Guide

## Pooling Modes

| Mode | Description | Best For |
|------|-------------|----------|
| **session** | Connection held for entire client session | Long-lived connections, SET/PREPARE/advisory locks |
| **transaction** | Connection released after each transaction | Most web applications (recommended) |
| **statement** | Connection released after each statement | Simple SELECT-only workloads; no multi-statement transactions |

Transaction mode is the most common choice for web applications — it provides the best connection reuse.

## pgbouncer.ini Configuration

```ini
[databases]
mydb = host=postgres-primary port=5432 dbname=mydb

; Read replica pool (optional)
mydb_ro = host=postgres-replica port=5432 dbname=mydb

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 5432

; Auth
auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt

; Pooling mode
pool_mode = transaction

; Connection limits
max_client_conn = 1000        ; Max inbound connections from apps
default_pool_size = 20        ; Max server connections per database+user pair
min_pool_size = 5             ; Keep at least N connections ready
reserve_pool_size = 5         ; Extra connections for bursts
reserve_pool_timeout = 5.0    ; Seconds to wait before using reserve pool

; Timeouts
connect_timeout = 15          ; Seconds to wait for server connection
query_timeout = 300           ; Seconds before killing long queries (0=disabled)
query_wait_timeout = 120      ; Seconds client waits for an available connection
client_idle_timeout = 600     ; Seconds idle client connection stays open
server_idle_timeout = 600     ; Seconds idle server connection stays alive
server_lifetime = 3600        ; Max seconds a server connection lives

; Logging
log_connections = 0
log_disconnections = 0
log_pooler_errors = 1
stats_period = 60

; Admin
admin_users = pgbouncer_admin
stats_users = pgbouncer_stats

; TLS (optional)
; client_tls_sslmode = require
; client_tls_ca_file = /etc/ssl/certs/ca.crt
; server_tls_sslmode = require
```

## userlist.txt

```
; Format: "username" "md5hash_or_scram_secret"
"app_user" "scram-sha-256$4096:..."
"pgbouncer_admin" "scram-sha-256$4096:..."
```

Generate password hash:

```bash
# PostgreSQL 14+: SCRAM-SHA-256
psql -c "SELECT pg_authid.rolpassword FROM pg_authid WHERE rolname = 'app_user';"
```

## Pool Size Tuning

```
Formula:
  max_client_conn >= (app instances) × (app pool size per instance)
  default_pool_size = (Postgres max_connections × 0.8) ÷ (number of distinct users)

Example:
  Postgres max_connections = 200
  1 app user
  default_pool_size = 160

  10 app instances × 10 threads each = 100 app connections
  max_client_conn = 150 (with headroom)
  default_pool_size = 20 (server-side)
```

```ini
; Typical web app (transaction mode)
max_client_conn = 500
default_pool_size = 20
min_pool_size = 5
```

## Timeout Configuration

```ini
; Queries blocked longer than this will be cancelled (0 = disable)
query_timeout = 300

; Client waits this long for a connection from pool before error
query_wait_timeout = 120

; Idle client connections closed after this many seconds
client_idle_timeout = 600

; Server connections reused for at most this many seconds (prevents stale state)
server_lifetime = 3600

; Connections that haven't been used for this long are closed
server_idle_timeout = 600
```

## Docker Setup

```yaml
# docker-compose.yml
services:
  pgbouncer:
    image: bitnami/pgbouncer:latest
    environment:
      POSTGRESQL_HOST: postgres
      POSTGRESQL_PORT: 5432
      POSTGRESQL_DATABASE: mydb
      POSTGRESQL_USERNAME: app_user
      POSTGRESQL_PASSWORD: ${DB_PASSWORD}
      PGBOUNCER_POOL_MODE: transaction
      PGBOUNCER_MAX_CLIENT_CONN: "500"
      PGBOUNCER_DEFAULT_POOL_SIZE: "20"
      PGBOUNCER_MIN_POOL_SIZE: "5"
      PGBOUNCER_SERVER_IDLE_TIMEOUT: "600"
    ports:
      - "5432:5432"
    depends_on:
      - postgres

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: mydb
      POSTGRES_USER: app_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
```

## Monitoring — SHOW Commands

Connect to PgBouncer admin console:

```bash
psql -h localhost -p 5432 -U pgbouncer_admin pgbouncer
```

```sql
-- Pool statistics
SHOW POOLS;
-- cl_active: client connections in active use
-- cl_waiting: clients waiting for a free server connection
-- sv_active: server connections currently used
-- sv_idle: idle server connections in pool

-- Global stats
SHOW STATS;
-- total_requests, total_received, total_sent, avg_query_duration

-- Per-database stats
SHOW DATABASES;

-- Configuration
SHOW CONFIG;

-- Active client connections
SHOW CLIENTS;

-- Server connections
SHOW SERVERS;
```

Alert thresholds:
- `cl_waiting` > 10 for more than 30s → pool exhaustion
- `avg_query_duration` > 1000ms → query performance issue

## Reload Configuration Without Restart

```sql
-- In pgbouncer admin console
RELOAD;
```

## Centralized vs Distributed Deployment

### Centralized (single PgBouncer)
- Simpler; single point of management.
- Single point of failure — use HA setup (keepalived + VIP) or HAProxy in front.

```
[App instances] → [HAProxy] → [PgBouncer active] → [PostgreSQL]
                            ↘ [PgBouncer standby]
```

### Distributed (PgBouncer per app node)
- No single point of failure.
- App connects to localhost PgBouncer.
- Harder to monitor pool stats centrally.

```
[App + PgBouncer (node 1)] ↘
[App + PgBouncer (node 2)] → [PostgreSQL]
[App + PgBouncer (node 3)] ↗
```

## Incompatibilities with Transaction Mode

Transaction pooling mode does NOT support:
- `SET` statements that persist beyond a transaction
- Named prepared statements (`PREPARE`/`EXECUTE`)
- Advisory locks (pg_advisory_lock)
- `LISTEN`/`NOTIFY`
- `WITH HOLD` cursors

Use session mode for applications that require these features.

## Anti-Patterns

- Do not set `default_pool_size` equal to `max_connections` — leave headroom for direct psql admin access.
- Do not use statement mode with any application that uses explicit transactions.
- Do not use transaction mode with connection-level `SET` statements (e.g., `SET search_path`).
- Do not set `query_timeout` too low for OLAP/reporting queries — use separate pools.
- Never expose PgBouncer admin port (6432) to the internet.
