# Observability: Health Checks Skill Guide

## Overview

Liveness (`/healthz`), readiness (`/readyz`), startup probes, version endpoint, and per-language middleware patterns.

## Endpoint Contracts

| Endpoint | Checks | Returns |
|----------|--------|---------|
| `GET /healthz` | Process alive only — no dependencies | Always `200 OK` |
| `GET /readyz` | DB + cache + critical external APIs | `200` ready / `503` not ready |
| `GET /version` | Static build metadata | Always `200 OK` |

Never block `/healthz` on dependencies — it is purely "is the process alive?".

## Go

```go
package health

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
)

type Check struct {
    Status string `json:"status"`
    Error  string `json:"error,omitempty"`
}

type ReadyResponse struct {
    Status string            `json:"status"`
    Checks map[string]Check  `json:"checks"`
}

// Liveness — always 200 if process is running
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readiness — checks dependencies
func ReadinessHandler(db *sql.DB, cache *redis.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
        defer cancel()

        checks := map[string]Check{}
        allOK := true

        // DB check
        if err := db.PingContext(ctx); err != nil {
            checks["database"] = Check{Status: "fail", Error: err.Error()}
            allOK = false
        } else {
            checks["database"] = Check{Status: "ok"}
        }

        // Cache check
        if err := cache.Ping(ctx).Err(); err != nil {
            checks["cache"] = Check{Status: "fail", Error: err.Error()}
            allOK = false
        } else {
            checks["cache"] = Check{Status: "ok"}
        }

        resp := ReadyResponse{Checks: checks}
        w.Header().Set("Content-Type", "application/json")
        if allOK {
            resp.Status = "ready"
            w.WriteHeader(http.StatusOK)
        } else {
            resp.Status = "not_ready"
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        json.NewEncoder(w).Encode(resp)
    }
}
```

## Node.js / Express

```typescript
import express from 'express';
import { pool } from './db';
import { redis } from './cache';

const router = express.Router();

router.get('/healthz', (_req, res) => {
  res.json({ status: 'ok' });
});

router.get('/readyz', async (_req, res) => {
  const checks: Record<string, { status: string; error?: string }> = {};
  let allOK = true;

  try {
    await pool.query('SELECT 1');
    checks.database = { status: 'ok' };
  } catch (err) {
    checks.database = { status: 'fail', error: String(err) };
    allOK = false;
  }

  try {
    await redis.ping();
    checks.cache = { status: 'ok' };
  } catch (err) {
    checks.cache = { status: 'fail', error: String(err) };
    allOK = false;
  }

  res.status(allOK ? 200 : 503).json({
    status: allOK ? 'ready' : 'not_ready',
    checks,
  });
});

export { router as healthRouter };
```

## Python / FastAPI

```python
from fastapi import APIRouter, Response
from sqlalchemy import text
import redis as redis_lib

router = APIRouter()

@router.get("/healthz", status_code=200)
async def liveness():
    return {"status": "ok"}

@router.get("/readyz")
async def readiness(response: Response, db=Depends(get_db), cache=Depends(get_cache)):
    checks = {}
    all_ok = True

    try:
        await db.execute(text("SELECT 1"))
        checks["database"] = {"status": "ok"}
    except Exception as e:
        checks["database"] = {"status": "fail", "error": str(e)}
        all_ok = False

    try:
        await cache.ping()
        checks["cache"] = {"status": "ok"}
    except Exception as e:
        checks["cache"] = {"status": "fail", "error": str(e)}
        all_ok = False

    response.status_code = 200 if all_ok else 503
    return {"status": "ready" if all_ok else "not_ready", "checks": checks}
```

## Version Endpoint

```go
// Go
var startTime = time.Now()

type VersionInfo struct {
    Version   string `json:"version"`
    Commit    string `json:"commit"`
    BuildTime string `json:"build_time"`
    Uptime    string `json:"uptime"`
}

// Populated at link time: -ldflags="-X main.version=1.4.2 -X main.commit=abc123"
var (
    version   = "dev"
    commit    = "none"
    buildTime = "unknown"
)

func VersionHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(VersionInfo{
        Version:   version,
        Commit:    commit,
        BuildTime: buildTime,
        Uptime:    time.Since(startTime).String(),
    })
}
```

## Kubernetes Probe Configuration

```yaml
spec:
  containers:
    - name: myapp
      image: myapp:1.4.2

      # Slow-start apps: startupProbe runs first, then liveness takes over
      startupProbe:
        httpGet:
          path: /healthz
          port: 8080
        failureThreshold: 30   # 30 * 2s = 60s max startup time
        periodSeconds: 2

      livenessProbe:
        httpGet:
          path: /healthz
          port: 8080
        initialDelaySeconds: 0
        periodSeconds: 10
        failureThreshold: 3    # restart after 3 consecutive failures
        timeoutSeconds: 2

      readinessProbe:
        httpGet:
          path: /readyz
          port: 8080
        initialDelaySeconds: 5
        periodSeconds: 5
        failureThreshold: 3
        successThreshold: 1
        timeoutSeconds: 3
```

## Sample /readyz Response Bodies

```json
// 200 OK — healthy
{
  "status": "ready",
  "checks": {
    "database": { "status": "ok" },
    "cache":    { "status": "ok" }
  }
}

// 503 Service Unavailable — degraded
{
  "status": "not_ready",
  "checks": {
    "database": { "status": "ok" },
    "cache":    { "status": "fail", "error": "dial tcp: connection refused" }
  }
}
```

## Key Rules

- `/healthz` must never fail due to a dependency — it reflects only process health.
- Wrap all `/readyz` dependency checks in a timeout (3 seconds maximum per check).
- Return `503` from `/readyz` when any critical dependency fails — Kubernetes will stop routing traffic.
- Non-critical dependencies (analytics, notification service) should not fail readiness.
- Expose health endpoints on the same port as the application for simplicity; exclude from auth middleware.
- Set `startupProbe` for services with slow JVM/Python startup — prevents premature liveness kills.
