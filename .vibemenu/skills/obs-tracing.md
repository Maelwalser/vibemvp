# Observability: Distributed Tracing Skill Guide

## Overview

OpenTelemetry SDK setup, auto and manual instrumentation, OTLP export to Jaeger and Grafana Tempo, sampling strategies, and W3C traceparent propagation.

## W3C Traceparent Header Format

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
             ^  ^                                ^               ^
             |  traceId (32 hex chars)          spanId (16)    flags
             version=00                                         01=sampled
```

Always propagate `traceparent` (and optionally `tracestate`) across service boundaries.

## Go: OpenTelemetry SDK

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "go.opentelemetry.io/otel/trace"
)

func InitTracer(ctx context.Context, serviceName, version, env string) (func(), error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("otel-collector:4317"),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("create exporter: %w", err)
    }

    res := resource.NewWithAttributes(semconv.SchemaURL,
        semconv.ServiceName(serviceName),
        semconv.ServiceVersion(version),
        semconv.DeploymentEnvironment(env),
    )

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        // Production: sample 10% of traces
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return func() { tp.Shutdown(ctx) }, nil
}
```

### Manual Span

```go
tracer := otel.Tracer("user-service")

func CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    ctx, span := tracer.Start(ctx, "CreateUser",
        trace.WithAttributes(
            attribute.String("user.email", req.Email),
            attribute.String("db.operation", "INSERT"),
        ),
    )
    defer span.End()

    user, err := db.Insert(ctx, req)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("insert user: %w", err)
    }

    span.SetAttributes(attribute.String("user.id", user.ID))
    return user, nil
}
```

### Auto-Instrumentation (HTTP + DB)

```go
import (
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql"
)

// HTTP server
mux := http.NewServeMux()
handler := otelhttp.NewHandler(mux, "myapp")

// HTTP client
client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

// Database
db, err := otelsql.Open("pgx", dsn)
```

## TypeScript: OpenTelemetry SDK

```typescript
import { NodeSDK } from '@opentelemetry/sdk-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { Resource } from '@opentelemetry/resources';
import { SEMRESATTRS_SERVICE_NAME, SEMRESATTRS_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { TraceIdRatioBasedSampler } from '@opentelemetry/sdk-trace-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';

const sdk = new NodeSDK({
  resource: new Resource({
    [SEMRESATTRS_SERVICE_NAME]: 'user-service',
    [SEMRESATTRS_SERVICE_VERSION]: process.env.APP_VERSION,
  }),
  traceExporter: new OTLPTraceExporter({
    url: 'grpc://otel-collector:4317',
  }),
  sampler: new TraceIdRatioBasedSampler(0.1),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
process.on('SIGTERM', () => sdk.shutdown());
```

```typescript
// Manual span
import { trace, SpanStatusCode } from '@opentelemetry/api';

const tracer = trace.getTracer('user-service');

async function createUser(req: CreateUserRequest): Promise<User> {
  return tracer.startActiveSpan('createUser', async (span) => {
    try {
      span.setAttributes({ 'user.email': req.email });
      const user = await db.insert(req);
      span.setAttributes({ 'user.id': user.id });
      return user;
    } catch (err) {
      span.recordException(err as Error);
      span.setStatus({ code: SpanStatusCode.ERROR });
      throw err;
    } finally {
      span.end();
    }
  });
}
```

## Python: OpenTelemetry SDK

```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.trace import StatusCode

resource = Resource(attributes={SERVICE_NAME: "user-service"})
provider = TracerProvider(resource=resource)
provider.add_span_processor(
    BatchSpanProcessor(OTLPSpanExporter(endpoint="http://otel-collector:4317"))
)
trace.set_tracer_provider(provider)

tracer = trace.get_tracer("user-service")

# Manual span
with tracer.start_as_current_span("create_user") as span:
    span.set_attribute("user.email", req.email)
    try:
        user = db.insert(req)
        span.set_attribute("user.id", user.id)
    except Exception as e:
        span.record_exception(e)
        span.set_status(StatusCode.ERROR, str(e))
        raise
```

## Export to Jaeger

```yaml
# OpenTelemetry Collector config
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [jaeger]
```

## Export to Grafana Tempo (S3 Backend)

```yaml
# tempo.yaml
server:
  http_listen_port: 3200

distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317

storage:
  trace:
    backend: s3
    s3:
      bucket: my-tempo-traces
      endpoint: s3.us-east-1.amazonaws.com
      region: us-east-1

compactor:
  compaction:
    block_retention: 336h  # 14 days
```

## Sampling Strategies

```go
// Development: always sample
sdktrace.WithSampler(sdktrace.AlwaysSample())

// Production: sample 10% by trace ID (deterministic)
sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1))

// Tail-based sampling via OTel Collector
// (sample 100% of error traces, 10% of success traces)
```

```yaml
# OTel Collector tail sampling config
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: errors-policy
        type: status_code
        status_code: { status_codes: [ERROR] }
      - name: sample-policy
        type: probabilistic
        probabilistic: { sampling_percentage: 10 }
```

## Baggage Propagation

```go
import "go.opentelemetry.io/otel/baggage"

// Set baggage (propagates to downstream services)
b, _ := baggage.Parse("user_id=abc123,tenant=acme")
ctx = baggage.ContextWithBaggage(ctx, b)

// Read baggage in downstream service
b := baggage.FromContext(ctx)
userID := b.Member("user_id").Value()
```

## Key Rules

- Initialize the TracerProvider before any other application code.
- Always call `span.End()` via `defer` to avoid leaking spans.
- Record errors with `span.RecordError(err)` AND set status to ERROR.
- Use `TraceIdRatioBased(0.1)` in production — never `AlwaysSample()`.
- Propagate context through every function call; never use `context.Background()` mid-request.
- Keep span names short and static (no dynamic IDs in span names — use attributes instead).
- Auto-instrumentation covers HTTP, gRPC, DB, Redis, and messaging — enable before manual spans.
