# Valkey Skill Guide

## Overview

Valkey is an open-source, Redis 7.2 API-compatible key-value store, forked from Redis after the license change to SSPL. It is a drop-in replacement for Redis with zero application code changes required, adding multi-threaded I/O, RDMA support, and an active open-source community under the Linux Foundation.

## Connection — Zero Code Changes

```javascript
// Existing Redis client code works unchanged — just update the host
import Redis from 'ioredis';

const valkey = new Redis({
  host: process.env.VALKEY_HOST,   // was REDIS_HOST
  port: 6379,                       // same default port
  password: process.env.VALKEY_PASSWORD,
  tls: process.env.VALKEY_TLS === 'true' ? {} : undefined,
});

// Cluster mode — same API
const cluster = new Redis.Cluster([
  { host: 'valkey-1', port: 6379 },
  { host: 'valkey-2', port: 6379 },
]);
```

```go
// Go — redis/v9 client works with Valkey
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
    Addr: os.Getenv("VALKEY_ADDR"),  // e.g., "valkey:6379"
})
```

## Multi-Threaded I/O Configuration

Valkey introduces true multi-threaded I/O (not just command processing). Configure in `valkey.conf`:

```bash
# Enable multi-threaded I/O (default: disabled for backward compat)
io-threads 4          # set to number of CPU cores (leave 1 for main thread)
io-threads-do-reads yes  # enable multi-threaded reads too

# Recommendation: io-threads = CPU cores - 1 (max 8 for typical workloads)
# For 8-core machine: io-threads 7

# Verify thread count at runtime
valkey-cli INFO server | grep io_threads
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: valkey
spec:
  serviceName: valkey
  replicas: 1
  selector:
    matchLabels:
      app: valkey
  template:
    metadata:
      labels:
        app: valkey
    spec:
      containers:
        - name: valkey
          image: valkey/valkey:8.0-alpine
          command: ["valkey-server"]
          args:
            - "--io-threads"
            - "4"
            - "--io-threads-do-reads"
            - "yes"
            - "--maxmemory"
            - "1gb"
            - "--maxmemory-policy"
            - "allkeys-lru"
            - "--requirepass"
            - "$(VALKEY_PASSWORD)"
          env:
            - name: VALKEY_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: valkey-secret
                  key: password
          ports:
            - containerPort: 6379
          volumeMounts:
            - name: data
              mountPath: /data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
```

## RDMA Support (Low Latency)

Valkey supports RDMA (Remote Direct Memory Access) for sub-microsecond latency in HPC/AI workloads:

```bash
# Enable RDMA in valkey.conf (requires RDMA-capable NICs and kernel modules)
bind-source-addr ""
enable-debug-command yes

# Build with RDMA support (from source)
make BUILD_WITH_MODULES=1 RDMA=yes

# Connection string for RDMA (separate port)
# rdma://valkey-host:6379
```

## Migration from Redis

```bash
# 1. Verify API compatibility — no code changes needed for Redis 7.2 API
# Valkey maintains full compatibility with Redis 7.2 commands

# 2. Docker image swap
# Before: image: redis:7.2-alpine
# After:  image: valkey/valkey:8.0-alpine

# 3. RDB file migration — RDB format is compatible
# Copy dump.rdb from Redis data directory to Valkey data directory

# 4. AOF migration
# Valkey reads Redis AOF files directly

# 5. Verify after migration
valkey-cli PING  # PONG
valkey-cli INFO server | grep valkey_version
valkey-cli DBSIZE

# 6. Run test suite against Valkey before switching production
```

## Divergences from Redis 8+

| Feature | Valkey | Redis 8+ |
|---------|--------|----------|
| License | BSD-3-Clause (OSS) | SSPL + RSAL (source-available) |
| Multi-threaded I/O | Built-in | Built-in |
| Vector similarity search | Via module | Native (Redis Stack) |
| JSON support | Via module | Native (Redis Stack) |
| Time Series | Via module | Native (Redis Stack) |
| RDMA | Supported | Not supported |
| Active-active geo-replication | Planned | Redis Enterprise only |
| Governance | Linux Foundation | Redis Ltd |

## Connection String Compatibility

```bash
# All existing connection string formats work unchanged
redis://valkey-host:6379
redis://:password@valkey-host:6379
redis://user:password@valkey-host:6379/0
rediss://valkey-host:6380  # TLS
redis://valkey-host:6379?maxReconnects=5

# Sentinel
redis://sentinel-host:26379/mymaster

# Cluster
redis://valkey-node-1:6379,valkey-node-2:6379,valkey-node-3:6379
```

## Key Rules

- Zero application code changes when migrating from Redis 7.2 — only change the host/image
- Enable `io-threads` only when CPU is the bottleneck, not network; benchmark before enabling
- RDMA requires specific hardware (Mellanox/NVIDIA InfiniBand NICs) — not for general use
- Valkey modules (JSON, Search, TimeSeries) are available at https://github.com/valkey-io
- Redis Sentinel and Cluster protocols are fully supported
- Valkey maintains backward compatibility with Redis 7.2 — test before upgrading past that baseline
- Use `valkey-cli` (same interface as `redis-cli`) for administration and debugging
