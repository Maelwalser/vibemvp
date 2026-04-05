# Container Runtime Skill Guide

## Overview

Best-practice Dockerfiles for each language runtime. Priorities: minimal image size, security hardening (non-root user, read-only filesystem, no capabilities), fast builds via layer caching.

## Node.js Alpine Multi-Stage

```dockerfile
# Stage 1: Build
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

# Stage 2: Runtime
FROM node:22-alpine AS runtime
RUN apk add --no-cache tini

# Non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/node_modules ./node_modules
COPY --from=builder --chown=appuser:appgroup /app/dist ./dist
COPY --from=builder --chown=appuser:appgroup /app/package.json .

USER appuser
EXPOSE 8080

# tini as PID 1 for proper signal handling
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["node", "dist/server.js"]
```

## Go Scratch (minimal static binary)

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /api ./cmd/api

# Stage 2: Scratch runtime (zero OS, no shell)
FROM scratch
# Copy CA certs for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy timezone data if needed
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy binary
COPY --from=builder /api /api

EXPOSE 8080
ENTRYPOINT ["/api"]
```

## Python Slim

```dockerfile
FROM python:3.12-slim AS builder
WORKDIR /app

# Install uv for fast dependency resolution
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev --no-install-project

COPY . .
RUN uv sync --frozen --no-dev

FROM python:3.12-slim AS runtime
RUN addgroup --system appgroup && adduser --system --ingroup appgroup appuser
WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/.venv ./.venv
COPY --from=builder --chown=appuser:appgroup /app/src ./src

ENV PATH="/app/.venv/bin:$PATH"
USER appuser
EXPOSE 8080
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
```

```dockerfile
# Without uv (pip only)
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN adduser --disabled-password --no-create-home appuser
USER appuser
CMD ["python", "-m", "uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
```

## Distroless

```dockerfile
# Go with distroless (no shell, no package manager, nonroot user)
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
```

```dockerfile
# Node.js with distroless
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

FROM gcr.io/distroless/nodejs22-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist
EXPOSE 8080
CMD ["dist/server.js"]
```

## Security Hardening — Kubernetes Pod Security Context

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

containers:
  - name: api
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
          - ALL
    volumeMounts:
      - name: tmp
        mountPath: /tmp      # writable tmpfs for temp files

volumes:
  - name: tmp
    emptyDir:
      medium: Memory
      sizeLimit: 64Mi
```

## Security Hardening — Docker Compose

```yaml
services:
  api:
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp:size=64m,mode=1777
    cap_drop:
      - ALL
    user: "1000:1000"
```

## Key Rules

- Always use multi-stage builds — never ship build tools or dev dependencies in the final image.
- Use `tini` as PID 1 in Node.js Alpine images for proper signal handling and zombie process reaping.
- Go binaries: set `CGO_ENABLED=0` for a fully static binary that works in `scratch` or `distroless`.
- Copy CA certificates into scratch images or HTTPS connections will fail.
- `readOnlyRootFilesystem: true` prevents runtime writes — mount `tmpfs` for `/tmp` if the app needs it.
- Drop ALL Linux capabilities and never run as root in production containers.
- `--no-cache-dir` in pip prevents the pip cache from bloating the image layer.
- Use `uv` instead of pip for 10-100x faster Python dependency installation in CI.
