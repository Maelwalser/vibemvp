# Observability: Metrics Skill Guide

## Overview

Prometheus instrumentation, Grafana dashboards, Datadog custom metrics, CloudWatch, and New Relic. Choose the right instrument type and expose a `/metrics` endpoint.

## Instrument Types

| Type | When to Use | Example |
|------|-------------|---------|
| Counter | Monotonically increasing count | requests total, errors total |
| Gauge | Value that goes up and down | active connections, queue depth |
| Histogram | Distribution of values, compute percentiles | request duration, payload size |
| Summary | Client-side quantile calculation (avoid in distributed) | latency (single instance) |

Prefer **Histogram** over Summary for latency — enables server-side aggregation.

## Go: prometheus/client_golang

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total HTTP requests by status and method.",
    }, []string{"method", "path", "status"})

    httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request duration in seconds.",
        Buckets: prometheus.DefBuckets, // .005 .01 .025 .05 .1 .25 .5 1 2.5 5 10
    }, []string{"method", "path"})

    activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "http_active_connections",
        Help: "Number of active HTTP connections.",
    })
)

// Middleware
func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        timer := prometheus.NewTimer(httpDuration.WithLabelValues(r.Method, r.URL.Path))
        defer timer.ObserveDuration()

        activeConnections.Inc()
        defer activeConnections.Dec()

        next.ServeHTTP(w, r)
    })
}

// Expose endpoint
http.Handle("/metrics", promhttp.Handler())
```

## TypeScript: prom-client

```typescript
import { Counter, Gauge, Histogram, register } from 'prom-client';
import { collectDefaultMetrics } from 'prom-client';

collectDefaultMetrics({ prefix: 'myapp_' });

export const httpRequestsTotal = new Counter({
  name: 'http_requests_total',
  help: 'Total HTTP requests',
  labelNames: ['method', 'path', 'status'],
});

export const httpDuration = new Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'path'],
  buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
});

// Express /metrics route
app.get('/metrics', async (_req, res) => {
  res.set('Content-Type', register.contentType);
  res.send(await register.metrics());
});
```

## Prometheus Recording Rules

```yaml
# prometheus/rules/app.yml
groups:
  - name: myapp
    interval: 1m
    rules:
      - record: job:request_success_rate:5m
        expr: |
          sum by (job) (rate(http_requests_total{status!~"5.."}[5m]))
          /
          sum by (job) (rate(http_requests_total[5m]))

      - record: job:request_latency_p99:5m
        expr: |
          histogram_quantile(0.99,
            sum by (job, le) (rate(http_request_duration_seconds_bucket[5m]))
          )

      - record: job:error_rate:5m
        expr: |
          sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
          /
          sum by (job) (rate(http_requests_total[5m]))
```

## Prometheus Alert Rules

```yaml
groups:
  - name: myapp-alerts
    rules:
      - alert: HighErrorRate
        expr: job:error_rate:5m > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Error rate above 1% for {{ $labels.job }}"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: HighLatency
        expr: job:request_latency_p99:5m > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P99 latency above 500ms for {{ $labels.job }}"
```

## Grafana Dashboard JSON (Key Panels)

```json
{
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum by (job) (rate(http_requests_total[5m]))",
          "legendFormat": "{{ job }}"
        }
      ]
    },
    {
      "title": "P99 Latency",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket[5m])))",
          "legendFormat": "p99"
        }
      ]
    },
    {
      "title": "Error Rate",
      "type": "stat",
      "targets": [
        {
          "expr": "job:error_rate:5m",
          "legendFormat": "error rate"
        }
      ]
    }
  ]
}
```

## Datadog: DogStatsD Custom Metrics

```python
from datadog import initialize, statsd

initialize(statsd_host='localhost', statsd_port=8125)

# Counter
statsd.increment('http.requests', tags=['method:POST', 'status:200', 'env:prod'])

# Gauge
statsd.gauge('queue.depth', 42, tags=['queue:orders', 'env:prod'])

# Histogram (auto computes p50/p75/p95/p99)
statsd.histogram('http.request.duration', 0.123, tags=['path:/api/users'])

# Datadog APM trace→metric (automatic)
# All spans generate metrics: trace.service.hits / trace.service.errors / trace.service.duration
```

## CloudWatch: PutMetricData + Composite Alarm

```python
import boto3

cloudwatch = boto3.client('cloudwatch', region_name='us-east-1')

cloudwatch.put_metric_data(
    Namespace='MyApp',
    MetricData=[{
        'MetricName': 'OrdersProcessed',
        'Value': 42,
        'Unit': 'Count',
        'Dimensions': [
            {'Name': 'Environment', 'Value': 'production'},
            {'Name': 'Service', 'Value': 'order-service'},
        ],
    }]
)
```

```hcl
resource "aws_cloudwatch_metric_alarm" "composite" {
  alarm_name        = "myapp-composite-alarm"
  alarm_description = "High error rate AND high latency"
  alarm_rule = join(" AND ", [
    "ALARM(myapp-error-rate)",
    "ALARM(myapp-high-latency)",
  ])
  alarm_actions = [aws_sns_topic.alerts.arn]
}

resource "aws_cloudwatch_metric_alarm" "anomaly" {
  alarm_name          = "myapp-anomaly-detection"
  comparison_operator = "GreaterThanUpperThreshold"
  evaluation_periods  = 2
  threshold_metric_id = "ad1"

  metric_query {
    id          = "m1"
    metric {
      metric_name = "RequestCount"
      namespace   = "MyApp"
      period      = 60
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "ad1"
    expression  = "ANOMALY_DETECTION_BAND(m1, 2)"
    return_data = true
  }
}
```

## New Relic Custom Metrics

```python
import newrelic.agent

# Custom metric
newrelic.agent.record_custom_metric('Custom/OrdersProcessed', 42)

# Custom attributes on current transaction
newrelic.agent.add_custom_attribute('user_id', user_id)
newrelic.agent.add_custom_attribute('order_id', order_id)
```

## Key Rules

- Use Histogram for latency metrics — never Counter.
- Always add `job` and `env` labels to every metric for filtering.
- Keep cardinality low — never use user IDs or request IDs as label values.
- Define recording rules for commonly queried expressions to reduce query load.
- Expose `/metrics` on a dedicated port or internal path, not the public API port.
- Use `promauto` in Go to register metrics at init time, avoiding runtime errors.
