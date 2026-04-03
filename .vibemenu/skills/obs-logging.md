# Observability: Structured Logging Skill Guide

## Overview

Structured JSON logging with mandatory fields, ingestion pipelines for Loki/ELK/CloudWatch/Datadog, and log level guidance.

## Mandatory Log Fields

Every log line must contain:

```json
{
  "level": "info",
  "timestamp": "2024-01-15T12:00:00.000Z",
  "message": "user created account",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "service": "user-service",
  "env": "production",
  "version": "1.4.2"
}
```

## Log Level Guidance

| Level | When to Use |
|-------|-------------|
| DEBUG | Dev/staging only — verbose state, SQL queries, raw payloads |
| INFO | Business events — user created, order placed, job completed |
| WARN | Degraded but not broken — retry succeeded, cache miss, slow query |
| ERROR | Failures needing alert — DB down, payment failed, auth rejected |

Never log PII (email, password, card number) at any level.

## Go: slog Structured Logging

```go
import (
    "log/slog"
    "os"
)

func NewLogger(env, version, service string) *slog.Logger {
    opts := &slog.HandlerOptions{Level: slog.LevelInfo}
    if env == "development" {
        opts.Level = slog.LevelDebug
    }
    handler := slog.NewJSONHandler(os.Stdout, opts)
    return slog.New(handler).With(
        "service", service,
        "env", env,
        "version", version,
    )
}

// Usage
logger.InfoContext(ctx, "user created account",
    "user_id", userID,
    "trace_id", traceID,
)
logger.ErrorContext(ctx, "payment failed",
    "order_id", orderID,
    "error", err.Error(),
)
```

## TypeScript: pino

```typescript
import pino from 'pino';

const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  base: {
    service: 'user-service',
    env: process.env.NODE_ENV,
    version: process.env.APP_VERSION,
  },
  timestamp: pino.stdTimeFunctions.isoTime,
  formatters: {
    level: (label) => ({ level: label }),
  },
});

logger.info({ trace_id: traceId, user_id: userId }, 'user created account');
logger.error({ trace_id: traceId, err }, 'payment failed');
```

## Python: structlog

```python
import structlog

structlog.configure(
    processors=[
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ]
)

log = structlog.get_logger().bind(
    service="user-service",
    env=os.getenv("ENV"),
    version=os.getenv("APP_VERSION"),
)

log.info("user created account", user_id=user_id, trace_id=trace_id)
log.error("payment failed", order_id=order_id, error=str(e))
```

## Loki + Grafana (promtail / Alloy)

```yaml
# promtail config
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: app-logs
    static_configs:
      - targets: [localhost]
        labels:
          job: myapp
          env: production
          __path__: /var/log/**/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            trace_id: trace_id
            service: service
      - labels:
          level:
          service:
      - timestamp:
          source: timestamp
          format: RFC3339
```

```logql
# LogQL: filter errors in user-service
{service="user-service"} | json | level="error"

# LogQL: count errors per minute
sum by (service) (rate({env="production"} | json | level="error" [1m]))

# LogQL: search by trace_id
{env="production"} | json | trace_id="4bf92f3577b34da6a3ce929d0e0e4736"
```

## ELK: Logstash Pipeline

```ruby
# logstash.conf
input {
  beats {
    port => 5044
  }
}

filter {
  json {
    source => "message"
    target => "parsed"
  }

  mutate {
    rename => {
      "[parsed][trace_id]" => "trace_id"
      "[parsed][level]"    => "level"
      "[parsed][service]"  => "service"
    }
    add_field => { "[@metadata][index]" => "logs-%{[parsed][env]}-%{+YYYY.MM.dd}" }
  }

  date {
    match => ["[parsed][timestamp]", "ISO8601"]
    target => "@timestamp"
  }
}

output {
  elasticsearch {
    hosts => ["http://elasticsearch:9200"]
    index => "%{[@metadata][index]}"
  }
}
```

## CloudWatch

```hcl
resource "aws_cloudwatch_log_group" "app" {
  name              = "/myapp/production"
  retention_in_days = 30
}
```

```bash
# CloudWatch Insights query — top errors last 1h
fields @timestamp, level, message, service, trace_id
| filter level = "error"
| stats count(*) as error_count by service
| sort error_count desc
| limit 20

# Tail specific trace
fields @timestamp, message
| filter trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"
| sort @timestamp asc
```

## Datadog Log Collection

```yaml
# datadog-agent log config
logs:
  - type: file
    path: /var/log/myapp/*.log
    service: user-service
    source: go
    tags:
      - env:production
      - version:1.4.2
```

```bash
# Environment variables for Datadog agent
DD_API_KEY=<key>
DD_SITE=datadoghq.com
DD_LOGS_ENABLED=true
DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL=true
```

## Key Rules

- Always include `trace_id` and `span_id` — propagated from incoming W3C `traceparent` header.
- Use structured fields (key-value), never format strings with dynamic values into the `message`.
- Set log retention to 30 days minimum; use lifecycle policies to archive to cold storage after 90 days.
- In production, run at `INFO` level. Enable `DEBUG` only via dynamic config or feature flag.
- Never log secrets, tokens, passwords, or full request/response bodies by default.
