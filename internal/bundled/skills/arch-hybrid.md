# Hybrid Architecture Skill Guide

## Overview

A hybrid system combines multiple architectural patterns — some services are monolithic, some are microservices, some are serverless — within a single product. Each service is tagged with its pattern. Shared infrastructure (API gateway, auth, observability) spans all services regardless of their individual patterns.

## Pattern Tagging

Every service in the manifest declares its architectural pattern. This drives code generation decisions:

```json
{
  "services": [
    { "name": "api-core",       "pattern": "modular-monolith", "language": "go" },
    { "name": "media-processor","pattern": "serverless",        "language": "python" },
    { "name": "notification",   "pattern": "event-driven",      "language": "go" },
    { "name": "analytics",      "pattern": "microservice",      "language": "typescript" },
    { "name": "admin-panel",    "pattern": "monolith",          "language": "python" }
  ]
}
```

When generating code, apply the corresponding `arch-*.md` skill per service. Generate the shared infrastructure once.

## Shared Infrastructure (Apply to All Services)

### API Gateway (Single Entry Point)

All external traffic enters through one API gateway regardless of backend pattern:

```yaml
# Kong / Traefik route config
routes:
  - name: core-api
    paths: ["/api/v1"]
    service: api-core:8080        # modular monolith

  - name: analytics-api
    paths: ["/api/analytics"]
    service: analytics:3000       # microservice

  - name: media-upload
    paths: ["/api/media"]
    service: media-processor      # serverless via Lambda URL or API GW

  - name: admin
    paths: ["/admin"]
    service: admin-panel:8000     # monolith
```

The gateway handles:
- TLS termination
- Auth token validation (see below)
- Rate limiting
- Request routing

### Shared Auth (Token Validation at the Gateway)

Validate JWT or session tokens at the gateway — services receive pre-authenticated requests:

```yaml
# Kong plugin config
plugins:
  - name: jwt
    config:
      secret_is_base64: false
      claims_to_verify: [exp]
      key_claim_name: iss
```

Services trust the gateway and read identity from forwarded headers:

```go
// Any backend service — reads pre-validated identity
func getCallerID(r *http.Request) string {
    return r.Header.Get("X-User-ID")   // set by gateway after token validation
}
```

```typescript
// Node.js service
const userID = req.headers["x-user-id"] as string
```

For service-to-service calls (not through the gateway), use internal API keys or mTLS.

### Distributed Tracing (All Services)

Every service, regardless of pattern, propagates the trace ID:

```go
// Middleware for long-running services (monolith, microservice, modular monolith)
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" { traceID = uuid.NewString() }
        ctx := context.WithValue(r.Context(), traceKey{}, traceID)
        w.Header().Set("X-Trace-ID", traceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

```python
# Lambda function — propagate trace ID from event headers
def handler(event, context):
    trace_id = (event.get("headers") or {}).get("x-trace-id", str(uuid.uuid4()))
    # Pass trace_id in all downstream calls and log entries
    logger.info("Processing request", extra={"trace_id": trace_id})
```

Use OpenTelemetry SDK across all services — it works for Lambda, long-running services, and everything in between.

### Structured Logging (All Services)

```go
// All services log JSON with consistent fields
log.Printf(`{"level":"info","trace_id":"%s","service":"api-core","msg":"%s"}`,
    traceID, message)
```

Centralize logs in a single sink (CloudWatch, Datadog, Loki) — configure per deployment regardless of service pattern.

### Health Checks (All Services)

Long-running services expose HTTP health endpoints:

```
GET /health  →  200 OK  {"status": "ok"}
GET /ready   →  200 OK or 503
```

Serverless functions: health is managed by the provider (Lambda, Cloud Functions). Add a lightweight `/ping` handler if needed for synthetic monitoring.

## Routing to Different Backend Patterns

The API gateway routes to different backend types transparently:

```nginx
# nginx upstream config (per backend type)

# Monolith / microservice — forward to running container
upstream api_core {
    server api-core:8080;
    keepalive 32;
}

# Serverless — forward to AWS Lambda Function URL or API Gateway
upstream media_processor {
    server lambda-function-url.lambda-url.us-east-1.on.aws:443;
}

server {
    location /api/v1/ {
        proxy_pass http://api_core;
        proxy_set_header X-Trace-ID $request_id;
        proxy_set_header X-User-ID  $http_x_user_id;
    }

    location /api/media/ {
        proxy_pass https://media_processor;
        proxy_set_header X-Trace-ID $request_id;
    }
}
```

## Per-Service Pattern Decision Guide

| Signal | Recommended Pattern |
|---|---|
| Shared database, team <10 devs, simple domain | Monolith |
| Clear module boundaries, single deploy unit desired | Modular Monolith |
| Independent scaling, separate teams, different tech stacks | Microservices |
| Bursty/async workloads (image processing, notifications) | Serverless |
| Decoupled async workflows, event sourcing | Event-Driven |
| Mix of the above in one product | Hybrid |

## Deployment Configuration

Generate a `docker-compose.yml` for local dev that includes all services regardless of pattern, plus shared infrastructure:

```yaml
services:
  api-gateway:
    image: kong:3.6
    ports: ["8080:8080", "8443:8443"]
    depends_on: [api-core, analytics]

  api-core:          # modular monolith
    build: ./services/api-core
    environment:
      DATABASE_URL: postgres://postgres:password@db:5432/core

  analytics:         # microservice
    build: ./services/analytics
    environment:
      DATABASE_URL: postgres://postgres:password@db:5432/analytics

  admin-panel:       # monolith
    build: ./services/admin-panel
    environment:
      DATABASE_URL: postgres://postgres:password@db:5432/admin

  # Shared infrastructure
  db:
    image: postgres:16-alpine
    volumes: [pgdata:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports: ["16686:16686"]

  kafka:             # for event-driven services
    image: confluentinc/cp-kafka:7.6.0
    environment:
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092

volumes:
  pgdata:
```

## Observability Stack (Shared)

Deploy once, used by all services:

```
Metrics:  Prometheus scrapes all services → Grafana dashboards
Logs:     All services → Loki (or CloudWatch/Datadog)
Traces:   All services → Jaeger / Tempo (via OpenTelemetry)
Alerts:   Grafana alerting on error rate, latency P99, availability
```

Every service must export:
- `http_requests_total` (counter, with status code label)
- `http_request_duration_seconds` (histogram)
- Custom business metrics

## Rules

- Tag every service with its pattern in the manifest — code generation applies the corresponding skill.
- The API gateway is the single external entry point — never expose services directly to the internet.
- Auth validation happens at the gateway; services read identity from forwarded headers.
- All services propagate `X-Trace-ID` — use OpenTelemetry for consistent tracing.
- All services use structured JSON logging with `trace_id`, `service`, and `level` fields.
- Shared infrastructure (gateway, auth, observability, message broker) is generated once and referenced by all services.
- Each service has its own database schema/database; the shared DB instance is a deployment convenience, not a shared schema.
- Generate a `docker-compose.yml` for local development that wires all services and shared infrastructure together.
