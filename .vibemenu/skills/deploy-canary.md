# Canary Deployment Skill Guide

## Overview

Canary deployments gradually shift traffic from stable to new version, monitoring error rate and latency. Automatic rollback fires if thresholds are breached. Pattern: 5%→10%→25%→50%→100%.

## Kubernetes — Flagger Canary CRD

```yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: api
  namespace: default
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api

  progressDeadlineSeconds: 60

  service:
    port: 80
    targetPort: 8080
    gateways:
      - public-gateway.istio-system.svc.cluster.local
    hosts:
      - api.example.com

  analysis:
    interval: 1m
    threshold: 5          # max failed checks before rollback
    maxWeight: 50         # max traffic sent to canary
    stepWeight: 10        # increment per step

    metrics:
      - name: request-success-rate
        threshold: 99     # minimum % success rate
        interval: 1m
      - name: request-duration
        threshold: 500    # maximum P99 latency ms
        interval: 1m

    webhooks:
      - name: smoke-test
        type: pre-rollout
        url: http://flagger-loadtester.test/
        timeout: 15s
        metadata:
          type: bash
          cmd: "curl -sd 'test' http://api-canary/healthz | grep ok"
```

## Istio VirtualService with Weights

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: api
spec:
  hosts:
    - api.example.com
  http:
    - route:
        - destination:
            host: api-stable
            port:
              number: 80
          weight: 90     # stable
        - destination:
            host: api-canary
            port:
              number: 80
          weight: 10     # canary

---
# Update weights via kubectl patch or CI script
# kubectl patch vs api --type='json' \
#   -p='[{"op":"replace","path":"/spec/http/0/route/0/weight","value":50},
#         {"op":"replace","path":"/spec/http/0/route/1/weight","value":50}]'
```

## AWS CodeDeploy Canary Config

```json
{
  "deploymentConfigName": "canary-10-percent-5-minutes",
  "computePlatform": "ECS",
  "trafficRoutingConfig": {
    "type": "TimeBasedCanary",
    "timeBasedCanary": {
      "canaryPercentage": 10,
      "canaryInterval": 5
    }
  }
}
```

```yaml
# appspec.yml with hooks for monitoring
Hooks:
  AfterAllowTestTraffic:
    - functionName: CheckCanaryHealth
      functionVersion: "$LATEST"
```

## Automated Promotion/Rollback Script

```bash
#!/bin/bash
# canary-promote.sh
METRICS_URL="http://prometheus:9090/api/v1/query"
CANARY_NS="default"
CANARY_SVC="api-canary"

check_error_rate() {
  local QUERY='sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100'
  local RESULT=$(curl -s "$METRICS_URL?query=$QUERY" | jq '.data.result[0].value[1]' -r)
  echo "$RESULT"
}

check_p99_latency() {
  local QUERY='histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) * 1000'
  local RESULT=$(curl -s "$METRICS_URL?query=$QUERY" | jq '.data.result[0].value[1]' -r)
  echo "$RESULT"
}

WEIGHTS=(5 10 25 50 100)
for WEIGHT in "${WEIGHTS[@]}"; do
  echo "Setting canary weight to ${WEIGHT}%..."
  # Update VirtualService weight
  kubectl patch virtualservice api --type='json' \
    -p="[{\"op\":\"replace\",\"path\":\"/spec/http/0/route/1/weight\",\"value\":$WEIGHT}]"

  echo "Waiting 5 minutes for metrics..."
  sleep 300

  ERROR_RATE=$(check_error_rate)
  P99=$(check_p99_latency)

  echo "Error rate: ${ERROR_RATE}%, P99 latency: ${P99}ms"

  if (( $(echo "$ERROR_RATE > 0.1" | bc -l) )); then
    echo "ERROR RATE EXCEEDED THRESHOLD — rolling back"
    kubectl patch virtualservice api --type='json' \
      -p='[{"op":"replace","path":"/spec/http/0/route/0/weight","value":100},
           {"op":"replace","path":"/spec/http/0/route/1/weight","value":0}]'
    exit 1
  fi

  if (( $(echo "$P99 > 1000" | bc -l) )); then
    echo "P99 LATENCY EXCEEDED THRESHOLD — rolling back"
    exit 1
  fi
done

echo "Canary promotion complete"
```

## Feature Flags for Code-Level Gradual Rollout

```typescript
// Use feature flags to control new code path independently of deployment
import { LDClient } from "@launchdarkly/node-server-sdk";

const client = LDClient.init(process.env.LD_SDK_KEY);

async function handleRequest(req) {
  const user = { key: req.userId, email: req.email };

  const useNewAlgorithm = await client.variation(
    "new-recommendation-algorithm",
    user,
    false   // default = old path
  );

  if (useNewAlgorithm) {
    return newRecommendationAlgorithm(req);
  }
  return legacyRecommendationAlgorithm(req);
}
```

## Prometheus Metrics for Canary Analysis

```yaml
# Alert on canary degradation
groups:
  - name: canary
    rules:
      - alert: CanaryHighErrorRate
        expr: |
          sum(rate(http_requests_total{service="api-canary",status=~"5.."}[5m]))
          /
          sum(rate(http_requests_total{service="api-canary"}[5m]))
          > 0.001   # 0.1% threshold
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: Canary error rate above threshold — auto-rollback

      - alert: CanaryHighLatency
        expr: |
          histogram_quantile(0.99,
            sum(rate(http_request_duration_seconds_bucket{service="api-canary"}[5m])) by (le)
          ) > 1.0   # 1000ms P99 threshold
        for: 2m
```

## Key Rules

- Start at 5% — even 1% traffic on a production canary is statistically meaningful for error detection.
- Automated rollback is essential — do not rely on human intervention during off-hours.
- Monitor both error rate (<0.1%) AND latency P99 — latency degradation often precedes errors.
- Feature flags enable code-level canaries independent of deployment canaries.
- Database migrations must be backward-compatible — canary pods run new code against shared DB.
- Keep stable and canary deployments in the same Kubernetes namespace for shared ConfigMaps/Secrets.
- After 100% promotion, delete the canary deployment and clean up the VirtualService weights.
