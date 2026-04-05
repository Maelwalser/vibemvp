# Docker Compose Skill Guide

## Overview

Docker Compose defines multi-container applications. Use a base `compose.yml` with `compose.override.yml` for dev and `compose.prod.yml` for production overrides.

## Base compose.yml

```yaml
services:
  api:
    image: ghcr.io/org/api:${TAG:-latest}
    environment:
      APP_ENV: ${APP_ENV}
      DATABASE_URL: ${DATABASE_URL}
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --save 60 1 --loglevel warning
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
    restart: unless-stopped

volumes:
  pg_data:
  redis_data:
```

## compose.override.yml (dev — auto-loaded)

```yaml
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
      target: dev
    volumes:
      - .:/app
      - /app/node_modules
    environment:
      APP_ENV: development
      LOG_LEVEL: debug
    ports:
      - "8080:8080"
      - "9229:9229"   # debugger

  db:
    ports:
      - "5432:5432"   # expose for local dev tools

  redis:
    ports:
      - "6379:6379"
```

## compose.prod.yml

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M
        reservations:
          cpus: "0.25"
          memory: 128M
      replicas: 2
      restart_policy:
        condition: on-failure
        max_attempts: 3
    secrets:
      - db_password
      - jwt_secret

secrets:
  db_password:
    external: true
  jwt_secret:
    external: true
```

## Running with override files

```bash
# Dev (auto-loads compose.override.yml)
docker compose up

# Production
docker compose -f compose.yml -f compose.prod.yml up -d

# Explicit override
docker compose -f compose.yml -f compose.override.yml up
```

## Docker Secrets

```yaml
# Create secret from file
# echo "mysecretpassword" | docker secret create db_password -

services:
  api:
    secrets:
      - db_password
    environment:
      # Read secret from /run/secrets/db_password in container
      DB_PASSWORD_FILE: /run/secrets/db_password

secrets:
  db_password:
    file: ./secrets/db_password.txt   # dev
    # external: true                  # prod (Docker Swarm)
```

## Converting to Kubernetes with Kompose

```bash
# Install kompose
curl -L https://github.com/kubernetes/kompose/releases/latest/download/kompose-linux-amd64 -o kompose

# Convert compose.yml to K8s manifests
kompose convert -f compose.yml -o k8s/

# Convert and apply directly
kompose up
```

## Key Rules

- Use `condition: service_healthy` in `depends_on` to prevent app startup before dependencies are ready.
- Always define `healthcheck` for stateful services (DB, cache, message brokers).
- Use `restart: unless-stopped` for production services; `no` for one-shot jobs.
- Set resource limits in prod to prevent a single container from consuming all host resources.
- Use named volumes (not bind mounts) for persistent data in production.
- Never commit `.env` files with secrets — use `.env.example` as template and populate via CI secrets.
- The `json-file` logging driver with `max-size` prevents disk fill from unbounded log growth.
