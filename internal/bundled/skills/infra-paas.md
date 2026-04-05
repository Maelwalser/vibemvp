# PaaS Deployment Skill Guide

## Overview

Platform-as-a-Service tools abstract server management. Covered: Render (render.yaml), Railway (railway.json), Fly.io (fly.toml), Heroku (Procfile + buildpacks).

## Render (render.yaml)

```yaml
services:
  - type: web
    name: api
    env: docker
    dockerfilePath: ./Dockerfile
    dockerContext: .
    region: oregon
    plan: starter
    healthCheckPath: /healthz
    envVars:
      - key: APP_ENV
        value: production
      - key: DATABASE_URL
        fromDatabase:
          name: my-postgres
          property: connectionString
      - key: REDIS_URL
        fromService:
          type: redis
          name: my-redis
          property: connectionString
      - key: JWT_SECRET
        generateValue: true   # auto-generate and store

  - type: worker
    name: worker
    env: docker
    dockerfilePath: ./Dockerfile.worker
    envVars:
      - key: DATABASE_URL
        fromDatabase:
          name: my-postgres
          property: connectionString

  - type: cron
    name: cleanup-job
    env: docker
    schedule: "0 2 * * *"   # 2am UTC daily
    dockerfilePath: ./Dockerfile.cron
    envVars:
      - key: DATABASE_URL
        fromDatabase:
          name: my-postgres
          property: connectionString

databases:
  - name: my-postgres
    plan: starter
    databaseName: appdb

  - name: my-redis
    plan: starter
```

## Railway (railway.json)

```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "DOCKERFILE",
    "dockerfilePath": "Dockerfile"
  },
  "deploy": {
    "startCommand": "node dist/server.js",
    "healthcheckPath": "/healthz",
    "healthcheckTimeout": 300,
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 3
  }
}
```

Railway uses environment variables from the Railway dashboard or `railway variables set KEY=value`. Link services within the same project via auto-injected variables (`${{Postgres.DATABASE_URL}}`).

## Fly.io (fly.toml)

```toml
app = "my-api"
primary_region = "iad"

[build]
  dockerfile = "Dockerfile"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 1
  processes = ["app"]

  [http_service.concurrency]
    type = "requests"
    hard_limit = 250
    soft_limit = 200

[[http_service.checks]]
  grace_period = "10s"
  interval = "15s"
  method = "GET"
  path = "/healthz"
  protocol = "http"
  timeout = "2s"

[machines]
  [machines.app]
    cpus = 1
    memory_mb = 512

[[vm]]
  size = "shared-cpu-1x"

[env]
  APP_ENV = "production"
  LOG_LEVEL = "info"
```

```bash
# Deploy
fly deploy

# Set secrets
fly secrets set DATABASE_URL="postgres://..." JWT_SECRET="..."

# Scale
fly scale count 3 --region iad
fly scale vm shared-cpu-2x

# Autoscaling config
fly autoscale set min=1 max=10
```

## Heroku

```
# Procfile — defines process types
web: node dist/server.js
worker: node dist/worker.js
release: node dist/migrate.js   # runs before web starts on each deploy
```

```bash
# Set environment variables
heroku config:set APP_ENV=production JWT_SECRET=changeme --app my-app

# Add managed addons
heroku addons:create heroku-postgresql:mini --app my-app
heroku addons:create heroku-redis:mini --app my-app

# Buildpacks (specify runtime)
heroku buildpacks:set heroku/nodejs --app my-app
# For multi-buildpack:
heroku buildpacks:add --index 1 heroku/nodejs
heroku buildpacks:add --index 2 heroku/python
```

```yaml
# heroku.yml (Docker-based Heroku)
build:
  docker:
    web: Dockerfile
    worker: Dockerfile.worker
run:
  web: node dist/server.js
  worker: node dist/worker.js
```

## Key Rules

- Render: use `fromDatabase`/`fromService` env var references so URLs update automatically on re-provision.
- Railway: link service variables with `${{ServiceName.VAR_NAME}}` syntax in railway dashboard or config.
- Fly.io: set `auto_stop_machines = true` and `min_machines_running = 1` to balance cost and cold starts.
- Heroku `release:` dyno runs migrations before web dynos receive traffic — ideal for zero-downtime schema changes.
- All PaaS platforms: never commit secrets — use the platform's secret/config management UI or CLI.
- Health check paths must return 200 quickly (< 2s) — use lightweight `/healthz` that doesn't hit DB.
