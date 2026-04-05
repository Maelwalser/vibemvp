# Observability: Alerting Skill Guide

## Overview

Grafana Alerting with PromQL, PagerDuty Events API v2, OpsGenie, and CloudWatch Alarms with composite metrics.

## Grafana Alerting

### Alert Rule (UI / Terraform)

```yaml
# grafana_alert_rule Terraform resource
resource "grafana_rule_group" "myapp" {
  name             = "myapp-alerts"
  folder_uid       = grafana_folder.alerts.uid
  interval_seconds = 60

  rule {
    name      = "HighErrorRate"
    condition = "C"

    # Query A: raw metric
    data {
      ref_id = "A"
      relative_time_range { from = 300; to = 0 }
      datasource_uid = "prometheus"
      model = jsonencode({
        expr = "sum(rate(http_requests_total{status=~\"5..\",job=\"myapp\"}[5m]))"
      })
    }

    # Query B: total requests
    data {
      ref_id = "B"
      relative_time_range { from = 300; to = 0 }
      datasource_uid = "prometheus"
      model = jsonencode({
        expr = "sum(rate(http_requests_total{job=\"myapp\"}[5m]))"
      })
    }

    # Reducer C: error rate
    data {
      ref_id         = "C"
      datasource_uid = "-100"  # expression
      model = jsonencode({
        type       = "math"
        expression = "$A / $B"
      })
    }

    condition = "C"

    annotations = {
      summary     = "Error rate above 1% for myapp"
      description = "Current error rate: {{ $values.C.Value | humanizePercentage }}"
      runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    }

    labels = {
      severity = "critical"
      team     = "backend"
    }

    no_data_state  = "NoData"
    exec_err_state = "Error"

    for = "5m"   # pending period before firing
  }
}
```

### Notification Policy

```yaml
# Route alerts by severity label
resource "grafana_notification_policy" "root" {
  group_by      = ["alertname", "job"]
  contact_point = grafana_contact_point.default.name

  policy {
    matcher {
      label = "severity"
      match = "="
      value = "critical"
    }
    contact_point   = grafana_contact_point.pagerduty.name
    group_wait      = "30s"
    group_interval  = "5m"
    repeat_interval = "1h"
  }

  policy {
    matcher {
      label = "severity"
      match = "="
      value = "warning"
    }
    contact_point   = grafana_contact_point.slack.name
    repeat_interval = "4h"
  }
}
```

### Common PromQL Alert Expressions

```promql
# Error rate > 1%
avg(rate(http_requests_total{status=~"5.."}[5m]))
/ avg(rate(http_requests_total[5m])) > 0.01

# P99 latency > 500ms
histogram_quantile(0.99,
  sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
) > 0.5

# Pod memory usage > 80%
container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.8

# Service down (no scrape for 2m)
up{job="myapp"} == 0
```

## PagerDuty Events API v2

```bash
# Trigger alert
curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "routing_key": "'"$PD_ROUTING_KEY"'",
    "event_action": "trigger",
    "dedup_key": "myapp-high-error-rate",
    "payload": {
      "summary": "High error rate in myapp (production)",
      "severity": "critical",
      "source": "prometheus",
      "timestamp": "2024-01-15T12:00:00Z",
      "component": "user-service",
      "group": "backend",
      "class": "error_rate",
      "custom_details": {
        "error_rate": "3.2%",
        "threshold": "1%",
        "runbook": "https://wiki.example.com/runbooks/high-error-rate"
      }
    },
    "links": [
      {"href": "https://grafana.example.com/d/xxx", "text": "Dashboard"}
    ]
  }'

# Resolve alert (use same dedup_key)
curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "routing_key": "'"$PD_ROUTING_KEY"'",
    "event_action": "resolve",
    "dedup_key": "myapp-high-error-rate"
  }'
```

### Severity Mapping

| Prometheus severity | PagerDuty severity |
|--------------------|--------------------|
| critical | critical |
| high | error |
| warning | warning |
| info | info |

## OpsGenie

```bash
# Create alert
curl -X POST https://api.opsgenie.com/v2/alerts \
  -H "Authorization: GenieKey $OPSGENIE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "High error rate in myapp",
    "description": "Error rate is 3.2%, threshold is 1%. See: https://grafana.example.com",
    "priority": "P1",
    "tags": ["production", "backend", "slo-breach"],
    "teams": [{"name": "backend-team"}],
    "details": {
      "error_rate": "3.2%",
      "service": "myapp",
      "runbook": "https://wiki.example.com/runbooks/high-error-rate"
    },
    "alias": "myapp-high-error-rate"
  }'

# Close alert
curl -X DELETE "https://api.opsgenie.com/v2/alerts/myapp-high-error-rate?identifierType=alias" \
  -H "Authorization: GenieKey $OPSGENIE_API_KEY"
```

## CloudWatch Alarms

### Standard Metric Alarm

```hcl
resource "aws_cloudwatch_metric_alarm" "error_rate" {
  alarm_name          = "myapp-high-error-rate"
  alarm_description   = "Error rate above 1%"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  threshold           = 1
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "errors"
    metric {
      metric_name = "5xxCount"
      namespace   = "AWS/ApplicationELB"
      stat        = "Sum"
      period      = 60
    }
  }

  metric_query {
    id          = "total"
    metric {
      metric_name = "RequestCount"
      namespace   = "AWS/ApplicationELB"
      stat        = "Sum"
      period      = 60
    }
  }

  metric_query {
    id          = "error_rate"
    expression  = "errors / total * 100"
    return_data = true
    label       = "Error Rate %"
  }

  alarm_actions = [aws_sns_topic.alerts.arn]
  ok_actions    = [aws_sns_topic.alerts.arn]
}
```

### Composite Alarm

```hcl
resource "aws_cloudwatch_composite_alarm" "slo_breach" {
  alarm_name  = "myapp-slo-breach"
  alarm_rule  = "ALARM(${aws_cloudwatch_metric_alarm.error_rate.alarm_name}) AND ALARM(${aws_cloudwatch_metric_alarm.latency_p99.alarm_name})"
  alarm_actions = [aws_sns_topic.critical_alerts.arn]
}
```

## Key Rules

- Set a `for` / `pending` period (minimum 5 minutes) before an alert fires — prevents flapping.
- Always define `ok_actions` to auto-resolve PagerDuty/OpsGenie incidents when the alarm clears.
- Use `dedup_key` / `alias` consistently — without it, each alert triggers a new incident.
- Route `critical` alerts to on-call (PagerDuty), `warning` alerts to Slack only.
- Include a `runbook_url` annotation on every alert — on-call engineers need immediate guidance.
- Never page on a single data point — require at least 3 consecutive evaluation periods.
- Anomaly detection alarms (`ANOMALY_DETECTION_BAND`) supplement but do not replace threshold alarms.
