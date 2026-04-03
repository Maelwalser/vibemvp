# Serverless Architecture Skill Guide

## Overview

Serverless (FaaS) executes code in response to events with no persistent server management. Each invocation is stateless. Providers: AWS Lambda, Google Cloud Functions, Azure Functions, Cloudflare Workers. Key concerns: cold start latency, execution time limits, stateless design, and per-invocation cost.

## Execution Constraints by Provider

| Provider | Max Timeout | Memory | Cold Start Mitigation |
|---|---|---|---|
| AWS Lambda | 15 minutes | 128 MB – 10 GB | Provisioned Concurrency |
| Google Cloud Functions | 60 minutes | 128 MB – 32 GB | Min instances |
| Azure Functions | 10 min (Consumption) | 1.5 GB | Premium Plan, always-on |
| Cloudflare Workers | 30 seconds | 128 MB | V8 Isolates (no cold start) |

## Function Structure

Initialize expensive resources **outside** the handler. The runtime reuses the execution context across warm invocations:

```python
# Python (AWS Lambda)

import os
import boto3
import psycopg2

# CORRECT: initialize at module level — runs once per container, not per request
DB_CONN = psycopg2.connect(os.environ["DATABASE_URL"])
S3_CLIENT = boto3.client("s3")
SECRET = os.environ["API_SECRET"]  # loaded once

def handler(event, context):
    # DB_CONN is reused on warm invocations
    with DB_CONN.cursor() as cur:
        cur.execute("SELECT count(*) FROM orders")
        count = cur.fetchone()[0]
    return {"statusCode": 200, "body": str(count)}
```

```go
// Go (AWS Lambda)
package main

import (
    "context"
    "database/sql"
    "os"
    "github.com/aws/aws-lambda-go/lambda"
    _ "github.com/lib/pq"
)

// Initialize outside handler — reused across warm invocations
var db *sql.DB

func init() {
    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        panic("db init: " + err.Error())
    }
    db.SetMaxOpenConns(5)   // Lambda: keep pool small; each instance handles 1 req at a time
    db.SetMaxIdleConns(2)
}

func handler(ctx context.Context, event APIGatewayEvent) (APIGatewayResponse, error) {
    var count int
    if err := db.QueryRowContext(ctx, "SELECT count(*) FROM orders").Scan(&count); err != nil {
        return APIGatewayResponse{StatusCode: 500}, err
    }
    return APIGatewayResponse{StatusCode: 200, Body: strconv.Itoa(count)}, nil
}

func main() {
    lambda.Start(handler)
}
```

```typescript
// TypeScript (AWS Lambda — Node.js runtime)
import { DynamoDBClient } from "@aws-sdk/client-dynamodb"
import { SecretsManagerClient, GetSecretValueCommand } from "@aws-sdk/client-secrets-manager"

// Initialize outside handler
const dynamo = new DynamoDBClient({ region: process.env.AWS_REGION })
let cachedSecret: string | null = null  // cache secrets in memory

async function getSecret(): Promise<string> {
    if (cachedSecret) return cachedSecret
    const sm = new SecretsManagerClient({})
    const resp = await sm.send(new GetSecretValueCommand({ SecretId: process.env.SECRET_ARN }))
    cachedSecret = resp.SecretString!
    return cachedSecret
}

export const handler = async (event: AWSLambda.APIGatewayProxyEvent) => {
    const secret = await getSecret()  // cached after first call
    return { statusCode: 200, body: JSON.stringify({ ok: true }) }
}
```

## Stateless Design

```
CORRECT: each invocation reads and writes to external state (DB, cache, S3)
WRONG:   storing state in module-level variables that change between requests

// WRONG — global mutable state
let requestCount = 0
export const handler = async () => {
    requestCount++  // NOT reliable — each Lambda container has its own counter
    return { count: requestCount }
}

// CORRECT — read/write from external store
export const handler = async () => {
    const count = await redis.incr("request_count")
    return { count }
}
```

## Cold Start Mitigation

### Provisioned Concurrency (AWS Lambda)

```yaml
# serverless.yml / SAM template
Resources:
  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      FunctionName: my-api
      MemorySize: 512
      Timeout: 30

  ProvisionedConcurrency:
    Type: AWS::Lambda::ProvisionedConcurrencyConfig
    Properties:
      FunctionName: !Ref MyFunction
      Qualifier: prod
      ProvisionedConcurrentExecutions: 5   # keeps 5 instances warm
```

### Minimize Bundle Size

Cold start time is proportional to bundle size. Minimize dependencies:

```
# Node.js: use tree-shaking, avoid large libraries
# Install only what Lambda needs:
npm install --omit=dev

# Use esbuild to bundle:
esbuild src/handler.ts --bundle --platform=node --minify --outfile=dist/handler.js

# Target bundle size: < 5 MB unzipped, < 50 MB with dependencies
```

```python
# Python: use Lambda Layers for large dependencies
# Layer: numpy, pandas, scipy (up to 250 MB unzipped total including function)
# Function package: only your code
```

### Connection Pool Sizing for Lambda

Lambda processes one request per container. Adjust pool settings accordingly:

```go
db.SetMaxOpenConns(3)   // Lambda: 1 concurrent req per container; small pool
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(10 * time.Minute)  // Recycle connections to avoid stale ones
```

Use RDS Proxy (AWS) or PgBouncer in front of Postgres to limit total connections across all Lambda instances.

## Timeout Configuration

```python
# Always set context-aware timeouts shorter than the Lambda timeout
import boto3

def handler(event, context):
    # context.get_remaining_time_in_millis() gives ms until Lambda kills the function
    remaining_ms = context.get_remaining_time_in_millis()
    timeout_sec = min(remaining_ms / 1000 - 1, 10)  # leave 1s buffer

    # Apply timeout to downstream calls
    s3 = boto3.client("s3", config=Config(connect_timeout=timeout_sec, read_timeout=timeout_sec))
```

```typescript
export const handler = async (event: any, context: Context) => {
    const deadline = Date.now() + context.getRemainingTimeInMillis() - 1000  // 1s buffer

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), deadline - Date.now())

    try {
        const resp = await fetch("https://api.example.com/data", { signal: controller.signal })
        return { statusCode: 200, body: await resp.text() }
    } finally {
        clearTimeout(timeoutId)
    }
}
```

## Environment Variables and Secrets

```bash
# Lambda environment variables (set in deployment config)
DATABASE_URL=...          # for non-secret config
SECRET_ARN=arn:aws:...    # reference, not the value

# Never embed secrets directly — use Secrets Manager or Parameter Store
# Cache the secret in a module-level variable (see example above)
```

## API Gateway Integration

```yaml
# AWS SAM
Resources:
  ApiFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: dist/handler.handler
      Runtime: nodejs22.x
      MemorySize: 512
      Timeout: 29          # API Gateway hard limit is 29s
      Environment:
        Variables:
          TABLE_NAME: !Ref OrdersTable
      Events:
        Api:
          Type: Api
          Properties:
            Path: /orders
            Method: POST
```

## Cloudflare Workers (No Cold Start)

```typescript
// workers use V8 Isolates — no cold start, but no Node.js APIs
export default {
    async fetch(request: Request, env: Env): Promise<Response> {
        const url = new URL(request.url)

        if (url.pathname === "/health") {
            return new Response(JSON.stringify({ ok: true }), {
                headers: { "Content-Type": "application/json" },
            })
        }

        // Use env bindings instead of environment variables
        const data = await env.KV_STORE.get("config")
        return new Response(data, { status: 200 })
    },
}

interface Env {
    KV_STORE: KVNamespace
    DB: D1Database
}
```

## Rules

- Initialize DB connections, HTTP clients, and secrets **outside** the handler function.
- Never store mutable request-scoped state in module-level variables.
- Set DB connection pool max to 2–5; use a connection pooler (RDS Proxy, PgBouncer) for Postgres.
- Lambda timeout: always leave at least 1 second buffer for cleanup; set downstream timeouts shorter.
- Bundle size matters for cold starts — tree-shake, use Layers for large dependencies.
- Cache secrets in memory after first fetch; refresh on expiry or error.
- Use Provisioned Concurrency or min-instances for latency-sensitive endpoints.
- Return 202 Accepted for operations exceeding 29 seconds (API Gateway limit) and process async via SQS or Step Functions.
- No global mutable state — Lambda containers may be reused or replaced at any time.
