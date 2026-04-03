# SLO / SLI Skill Guide

## Overview

SLI definitions, Prometheus recording rules, error budget calculations, burn rate alerts, Grafana dashboard panels, and error budget policy.

## SLI Definitions

### Availability SLI

```
availability = successful_requests / total_requests

successful_requests = requests where HTTP status is NOT 5xx
```

### Latency SLI (P99)

```
latency_p99 = 99th percentile of request duration
```

### Error Rate SLI

```
error_rate = rate(errors[5m]) / rate(requests[5m])
```

## Error Budget

| SLO Target | Monthly downtime allowed | Error budget (requests) |
|------------|--------------------------|------------------------|
| 99.0% | 7h 18m | 1.0% of requests |
| 99.5% | 3h 39m | 0.5% of requests |
| 99.9% | 43.2 min | 0.1% of requests |
| 99.95% | 21.6 min | 0.05% of requests |
| 99.99% | 4.3 min | 0.01% of requests |

## Prometheus Recording Rules

```yaml
# prometheus/rules/slo.yml
groups:
  - name: slo-recording-rules
    interval: 30s
    rules:

      # Availability SLI: ratio of successful requests
      - record: slo:request_success_rate:ratio_rate5m
        expr: |
          sum by (job, env) (
            rate(http_requests_total{status!~"5.."}[5m])
          )
          /
          sum by (job, env) (
            rate(http_requests_total[5m])
          )

      # Availability SLI: 30-day window
      - record: slo:request_success_rate:ratio_rate30d
        expr: |
          sum by (job, env) (
            rate(http_requests_total{status!~"5.."}[30d])
          )
          /
          sum by (job, env) (
            rate(http_requests_total[30d])
          )

      # Latency SLI: P99 over 5 minutes
      - record: slo:latency_p99:ratio_rate5m
        expr: |
          histogram_quantile(0.99,
            sum by (job, env, le) (
              rate(http_request_duration_seconds_bucket[5m])
            )
          )

      # Error rate SLI
      - record: slo:error_rate:ratio_rate5m
        expr: |
          sum by (job, env) (
            rate(http_requests_total{status=~"5.."}[5m])
          )
          /
          sum by (job, env) (
            rate(http_requests_total[5m])
          )

      # Burn rate: 1h window
      - record: slo:burn_rate:ratio_rate1h
        expr: |
          (1 - slo:request_success_rate:ratio_rate1h)
          /
          (1 - 0.999)   # replace 0.999 with your SLO target

      # Burn rate: 6h window
      - record: slo:burn_rate:ratio_rate6h
        expr: |
          (1 - slo:request_success_rate:ratio_rate6h)
          /
          (1 - 0.999)
```

## Error Budget Burn Rate Alerts

### Fast Burn (exhausts monthly budget in ~2 days)

```yaml
# If burn rate > 30x over 1h window: will exhaust budget in 1/30 of 30d ≈ 1 day
- alert: SLOFastBurn
  expr: slo:burn_rate:ratio_rate1h > 30
  for: 5m
  labels:
    severity: critical
    slo: availability
  annotations:
    summary: "Fast error budget burn for {{ $labels.job }}"
    description: >
      Burn rate is {{ $value | humanize }}x the acceptable rate.
      At this rate the monthly error budget will be exhausted in
      {{ printf "%.1f" (720 / $value) }} hours.
    runbook_url: "https://wiki.example.com/runbooks/slo-fast-burn"
```

### Slow Burn (exhausts monthly budget in ~5 days)

```yaml
# If burn rate > 6x over 6h window: will exhaust budget in 30d/6 = 5 days
- alert: SLOSlowBurn
  expr: slo:burn_rate:ratio_rate6h > 6
  for: 15m
  labels:
    severity: warning
    slo: availability
  annotations:
    summary: "Slow error budget burn for {{ $labels.job }}"
    description: >
      Burn rate is {{ $value | humanize }}x over the last 6 hours.
      Monthly error budget will be exhausted in
      {{ printf "%.1f" (720 / $value) }} hours at this rate.
```

### Latency SLO Alert

```yaml
- alert: LatencySLOBreach
  expr: slo:latency_p99:ratio_rate5m > 0.5  # P99 > 500ms
  for: 5m
  labels:
    severity: warning
    slo: latency
  annotations:
    summary: "P99 latency SLO breach for {{ $labels.job }}"
    description: "P99 is {{ $value | humanizeDuration }}, threshold is 500ms."
```

## Grafana SLO Dashboard

```json
{
  "title": "SLO Dashboard",
  "panels": [
    {
      "title": "Availability (30d)",
      "type": "stat",
      "fieldConfig": {
        "defaults": {
          "unit": "percentunit",
          "thresholds": {
            "steps": [
              { "color": "red",    "value": 0 },
              { "color": "yellow", "value": 0.999 },
              { "color": "green",  "value": 0.9995 }
            ]
          }
        }
      },
      "targets": [
        { "expr": "slo:request_success_rate:ratio_rate30d{job=\"myapp\"}" }
      ]
    },
    {
      "title": "Error Budget Remaining (%)",
      "type": "bargauge",
      "targets": [
        {
          "expr": "((slo:request_success_rate:ratio_rate30d{job=\"myapp\"} - 0.999) / (1 - 0.999)) * 100",
          "legendFormat": "budget remaining"
        }
      ]
    },
    {
      "title": "30-Day Burn Rate",
      "type": "graph",
      "targets": [
        { "expr": "slo:burn_rate:ratio_rate1h{job=\"myapp\"}", "legendFormat": "1h burn rate" },
        { "expr": "slo:burn_rate:ratio_rate6h{job=\"myapp\"}", "legendFormat": "6h burn rate" }
      ]
    },
    {
      "title": "P99 Latency",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{job=\"myapp\"}[5m])))",
          "legendFormat": "p99"
        }
      ]
    }
  ]
}
```

## Error Budget Policy

| Budget Remaining | Action |
|-----------------|--------|
| > 50% | Normal development velocity |
| 25–50% | Review recent changes, increase monitoring |
| 10–25% | Defer non-critical deployments, incident review |
| < 10% | **Freeze deployments** — only critical fixes |
| 0% (exhausted) | Full incident response, post-mortem required |

## Key Rules

- Define SLIs from the user's perspective: did the request succeed and was it fast?
- Use 30-day rolling windows for SLO compliance reporting, not calendar months.
- Pair fast-burn (1h, 30x) and slow-burn (6h, 6x) alerts — they cover different failure modes.
- Never set SLO targets above what your dependencies can provide (cloud providers publish their own SLAs).
- Track error budget in a dashboard visible to the whole team — make the budget real.
- Update SLO targets quarterly based on observed reliability and user expectations.
