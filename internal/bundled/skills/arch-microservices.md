# Microservices Architecture Skill Guide

## Overview

Microservices are independently deployable services, each owning its data and communicating over the network. Each service is a small autonomous unit. Key concerns: reliable inter-service communication, distributed tracing, service discovery, and graceful degradation.

## Per-Service Project Layout

Each service is its own repository/module:

```
svc-users/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   └── repository/
├── Dockerfile
└── go.mod

svc-orders/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   └── repository/
├── Dockerfile
└── go.mod
```

## Service-to-Service HTTP with Retry and Circuit Breaker

### HTTP Client with Retry

```go
// internal/client/users_client.go
type UsersClient struct {
    baseURL    string
    httpClient *http.Client
    maxRetries int
}

func NewUsersClient(baseURL string) *UsersClient {
    return &UsersClient{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 5 * time.Second,
        },
        maxRetries: 3,
    }
}

func (c *UsersClient) GetUser(ctx context.Context, id string) (UserDTO, error) {
    var last error
    for attempt := 0; attempt < c.maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff with jitter
            delay := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
            delay += time.Duration(rand.Intn(100)) * time.Millisecond
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return UserDTO{}, ctx.Err()
            }
        }
        user, err := c.doGetUser(ctx, id)
        if err == nil {
            return user, nil
        }
        // Only retry on 5xx and network errors; not on 4xx
        if isRetryable(err) {
            last = err
            continue
        }
        return UserDTO{}, err
    }
    return UserDTO{}, fmt.Errorf("get user after %d retries: %w", c.maxRetries, last)
}
```

### Circuit Breaker (state machine)

```go
type CircuitState int
const (
    StateClosed   CircuitState = iota // normal, requests allowed
    StateOpen                         // tripped, requests blocked
    StateHalfOpen                     // probe: one request allowed
)

type CircuitBreaker struct {
    mu           sync.Mutex
    state        CircuitState
    failures     int
    threshold    int
    resetTimeout time.Duration
    lastFailure  time.Time
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    switch cb.state {
    case StateOpen:
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = StateHalfOpen
        } else {
            cb.mu.Unlock()
            return fmt.Errorf("circuit open: upstream unavailable")
        }
    }
    cb.mu.Unlock()

    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        if cb.failures >= cb.threshold {
            cb.state = StateOpen
        }
    } else {
        cb.failures = 0
        cb.state = StateClosed
    }
    return err
}
```

Use battle-tested libraries: `sony/gobreaker` (Go), `resilience4j` (Java/Kotlin), `polly` (.NET), `pybreaker` (Python).

## Distributed Tracing

### Propagate Trace-ID Header

Every service must read and forward the `X-Trace-ID` header:

```go
// Middleware: extract or create trace ID, attach to context
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.NewString()
        }
        ctx := context.WithValue(r.Context(), traceIDKey{}, traceID)
        w.Header().Set("X-Trace-ID", traceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Forward in outbound calls
func (c *UsersClient) doGetUser(ctx context.Context, id string) (UserDTO, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/users/"+id, nil)
    if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
        req.Header.Set("X-Trace-ID", traceID)
    }
    resp, err := c.httpClient.Do(req)
    ...
}
```

For production: use OpenTelemetry SDK with Jaeger or Tempo backend.

```go
// OpenTelemetry setup (Go)
tp := otel.GetTracerProvider()
tracer := tp.Tracer("svc-orders")

ctx, span := tracer.Start(ctx, "PlaceOrder")
defer span.End()

// Inject into outbound HTTP
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
```

## Service Discovery

### Environment-Based (simple)

```bash
USERS_SVC_URL=http://svc-users:8080
ORDERS_SVC_URL=http://svc-orders:8080
```

Kubernetes DNS automatically resolves `svc-users` to the ClusterIP service.

### DNS-Based Discovery Pattern

```go
// Resolve by service name — Kubernetes handles DNS
usersBaseURL := os.Getenv("USERS_SVC_URL")
if usersBaseURL == "" {
    usersBaseURL = "http://svc-users:8080"  // k8s DNS default
}
```

For Consul: services register on startup, clients query `<service-name>.service.consul`.

## Health Check Endpoints

Every service must expose `/health` (liveness) and `/ready` (readiness):

```go
// /health — is the process alive?
app.GET("/health", func(c echo.Context) error {
    return c.JSON(200, map[string]string{"status": "ok", "service": "svc-orders"})
})

// /ready — can the service handle traffic? (checks dependencies)
app.GET("/ready", func(c echo.Context) error {
    if err := db.PingContext(c.Request().Context()); err != nil {
        return c.JSON(503, map[string]string{"status": "not ready", "reason": "db unavailable"})
    }
    return c.JSON(200, map[string]string{"status": "ready"})
})
```

## Inter-Service Authentication

Services authenticate each other with shared secrets or mTLS:

```go
// Simple API key approach
const serviceAPIKeyHeader = "X-Service-Key"

func ServiceAuthMiddleware(expectedKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get(serviceAPIKeyHeader)
            if key != expectedKey {
                http.Error(w, "unauthorized", 401)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Client side: add key to every internal request
req.Header.Set("X-Service-Key", os.Getenv("INTERNAL_SERVICE_KEY"))
```

For production: use mTLS via a service mesh (Istio, Linkerd) — mutual TLS provides both auth and encryption with no code changes.

## Graceful Shutdown

```go
srv := &http.Server{Addr: ":8080", Handler: router}

go func() {
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatalf("listen: %v", err)
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Allow in-flight requests up to 30s to complete
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    log.Printf("shutdown: %v", err)
}

// Close DB pool after server stops accepting requests
db.Close()
```

## Kubernetes Deployment Template

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: svc-orders
spec:
  replicas: 3
  selector:
    matchLabels:
      app: svc-orders
  template:
    spec:
      containers:
      - name: svc-orders
        image: registry/svc-orders:latest
        ports:
        - containerPort: 8080
        env:
        - name: USERS_SVC_URL
          value: "http://svc-users:8080"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: orders-secrets
              key: database-url
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: svc-orders
spec:
  selector:
    app: svc-orders
  ports:
  - port: 8080
    targetPort: 8080
```

## Rules

- Each service owns its own database — no shared databases between services.
- All inter-service calls use HTTP (or gRPC) with retry and circuit breaker.
- Every outbound request propagates the trace ID header.
- Every service exposes `/health` and `/ready` endpoints.
- Services authenticate each other via API keys or mTLS — never via network trust alone.
- Use environment variables for service URLs — never hardcode addresses.
- Graceful shutdown is mandatory: drain in-flight requests before closing DB connections.
- Design for failure: a downstream service being down must not crash the caller.
