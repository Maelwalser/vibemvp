# Blue/Green Deployment Skill Guide

## Overview

Blue/green deployment maintains two identical environments: blue (current active) and green (new version). Traffic switches atomically, enabling instant rollback by switching back. Zero-downtime guaranteed.

## Concept

```
[Load Balancer]
     |
  [Router] ──── 100% ──▶ [Blue  — v1.0 — ACTIVE]
                          [Green — v2.0 — IDLE]

After cut-over:
  [Router] ──── 100% ──▶ [Green — v2.0 — ACTIVE]
                          [Blue  — v1.0 — STANDBY (keep for rollback)]
```

## Kubernetes — Service Selector Swap

```yaml
# blue deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-blue
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
      slot: blue
  template:
    metadata:
      labels:
        app: api
        slot: blue
        version: "1.0.0"
    spec:
      containers:
        - name: api
          image: ghcr.io/org/api:1.0.0

---
# green deployment (new version)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-green
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
      slot: green
  template:
    metadata:
      labels:
        app: api
        slot: green
        version: "2.0.0"
    spec:
      containers:
        - name: api
          image: ghcr.io/org/api:2.0.0

---
# Service points to active slot
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  selector:
    app: api
    slot: blue   # ← change to 'green' to cut over
  ports:
    - port: 80
      targetPort: 8080
```

```bash
# Step 1: Deploy green
kubectl apply -f deployment-green.yaml

# Step 2: Wait for green to be ready
kubectl rollout status deployment/api-green

# Step 3: Verify green health
kubectl run smoke-test --rm -it --image=curlimages/curl -- \
  curl -f http://api-green-svc/healthz

# Step 4: Cut over (atomic selector patch)
kubectl patch service api -p '{"spec":{"selector":{"slot":"green"}}}'

# Step 5: Rollback if needed (instant)
kubectl patch service api -p '{"spec":{"selector":{"slot":"blue"}}}'

# Step 6: Scale down blue after confidence period
kubectl scale deployment/api-blue --replicas=0
```

## AWS ALB — Listener Rule Update

```bash
# Create target groups for blue and green
aws elbv2 create-target-group \
  --name api-blue \
  --protocol HTTP --port 8080 \
  --vpc-id vpc-xxx \
  --health-check-path /healthz

aws elbv2 create-target-group \
  --name api-green \
  --protocol HTTP --port 8080 \
  --vpc-id vpc-xxx \
  --health-check-path /healthz

# Register instances to green target group
aws elbv2 register-targets \
  --target-group-arn arn:aws:...:targetgroup/api-green/xxx \
  --targets Id=i-xxx Id=i-yyy

# Wait for green targets to be healthy
aws elbv2 wait target-in-service \
  --target-group-arn arn:aws:...:targetgroup/api-green/xxx

# Switch ALB listener to green (atomic)
aws elbv2 modify-listener \
  --listener-arn arn:aws:...:listener/app/my-alb/xxx/yyy \
  --default-actions Type=forward,TargetGroupArn=arn:aws:...:targetgroup/api-green/xxx
```

## AWS ECS Blue/Green (CodeDeploy)

```json
// appspec.yml for ECS blue/green
{
  "version": 0.0,
  "Resources": [
    {
      "TargetService": {
        "Type": "AWS::ECS::Service",
        "Properties": {
          "TaskDefinition": "<TASK_DEFINITION>",
          "LoadBalancerInfo": {
            "ContainerName": "api",
            "ContainerPort": 8080
          }
        }
      }
    }
  ],
  "Hooks": [
    { "BeforeInstall": "LambdaFunctionToValidateBeforeInstall" },
    { "AfterAllowTestTraffic": "LambdaFunctionToSmokeTest" },
    { "AfterAllowTraffic": "LambdaFunctionToValidateAfterTraffic" }
  ]
}
```

## Database Schema Compatibility Requirement

Blue/green deployments require backward-compatible schema changes:

```sql
-- Safe: additive change (both v1 and v2 work with new column)
ALTER TABLE users ADD COLUMN display_name VARCHAR(255);

-- Unsafe: renaming (breaks v1 which still uses old_name)
-- ALTER TABLE users RENAME COLUMN name TO display_name;

-- Multi-step rename (safe):
-- Step 1 (deploy with v1 still running): add new column
ALTER TABLE users ADD COLUMN display_name VARCHAR(255);
-- Step 2: backfill
UPDATE users SET display_name = name;
-- Step 3 (after v2 is fully deployed): drop old column
ALTER TABLE users DROP COLUMN name;
```

## Health Check Before Cut-Over

```bash
#!/bin/bash
# verify-green.sh — run before switching traffic
HEALTH_URL="http://api-green.internal/healthz"
MAX_RETRIES=10
RETRY_DELAY=5

for i in $(seq 1 $MAX_RETRIES); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$HEALTH_URL")
  if [ "$STATUS" = "200" ]; then
    echo "Green is healthy (attempt $i)"
    exit 0
  fi
  echo "Attempt $i: status $STATUS, retrying in ${RETRY_DELAY}s..."
  sleep $RETRY_DELAY
done

echo "Green failed health check — aborting cut-over"
exit 1
```

## Key Rules

- Database migrations must be backward-compatible — old and new code must work simultaneously during cutover.
- Keep blue running for at least 15-30 minutes after cut-over before scaling it down.
- Run smoke tests against green on the test listener port before switching production traffic.
- Blue/green is not suitable for stateful services with local disk state — use read replicas or shared storage.
- Zero-downtime is only guaranteed if the load balancer uses connection draining (ELB deregistration delay).
- Always automate the rollback command — manual steps under pressure lead to mistakes.
