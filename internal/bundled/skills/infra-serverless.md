# Serverless Skill Guide

## Overview

Serverless functions run on-demand without managing servers. Key constraint: stateless execution, cold starts, per-invocation pricing. Initialize expensive resources (DB connections, SDK clients) outside the handler.

## AWS Lambda

### Handler Pattern

```python
# Python — initialize outside handler for reuse across warm invocations
import boto3
import psycopg2
import os

# Initialized once per container lifecycle (warm start reuse)
db_conn = psycopg2.connect(os.environ["DATABASE_URL"])
s3_client = boto3.client("s3")

def handler(event, context):
    """
    event: dict with trigger-specific data (API GW, SQS, S3, etc.)
    context: Lambda context object (request_id, remaining_time_ms, etc.)
    """
    try:
        # business logic
        return {
            "statusCode": 200,
            "headers": {"Content-Type": "application/json"},
            "body": '{"status": "ok"}'
        }
    except Exception as e:
        print(f"Error: {e}")
        raise
```

```javascript
// Node.js — ES module handler
import { DynamoDBClient } from "@aws-sdk/client-dynamodb";

// Outside handler = reused on warm invocations
const dynamo = new DynamoDBClient({});

export const handler = async (event, context) => {
  console.log(JSON.stringify({ event, requestId: context.awsRequestId }));
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "ok" }),
  };
};
```

### SAM template.yaml

```yaml
AWSTemplateFormatVersion: "2010-09-09"
Transform: AWS::Serverless-2016-10-31

Globals:
  Function:
    Runtime: python3.12
    MemorySize: 512
    Timeout: 30
    Environment:
      Variables:
        DATABASE_URL: !Sub "{{resolve:secretsmanager:prod/api/db:SecretString:url}}"

Resources:
  ApiFunction:
    Type: AWS::Serverless::Function
    Properties:
      FunctionName: api
      Handler: src/handler.handler
      CodeUri: .
      Layers:
        - !Ref SharedLayer
      Events:
        ApiEvent:
          Type: HttpApi
          Properties:
            Path: /{proxy+}
            Method: ANY
      DeadLetterQueue:
        Type: SQS
        TargetArn: !GetAtt DLQ.Arn
      ProvisionedConcurrencyConfig:
        ProvisionedConcurrentExecutions: 5   # cold start mitigation

  SharedLayer:
    Type: AWS::Serverless::LayerVersion
    Properties:
      LayerName: shared-deps
      ContentUri: layer/
      CompatibleRuntimes:
        - python3.12
      RetentionPolicy: Retain

  DLQ:
    Type: AWS::SQS::Queue
    Properties:
      MessageRetentionPeriod: 1209600   # 14 days
```

### Cold Start Mitigation

- Initialize DB/SDK clients outside the handler.
- Use `ProvisionedConcurrencyConfig` for latency-sensitive functions.
- Use Lambda Layers to share dependencies across functions (reduces package size).
- Keep deployment package small — use container images for large runtimes.
- Use `LAMBDA_TASK_ROOT` for file paths inside the Lambda environment.

## GCP Cloud Functions (Functions Framework)

```python
# functions_framework for HTTP trigger
import functions_framework

@functions_framework.http
def handler(request):
    data = request.get_json(silent=True) or {}
    return {"status": "ok", "received": data}, 200
```

```python
# Event-driven (CloudEvent)
@functions_framework.cloudEvent
def process_event(cloud_event):
    print(f"Event: {cloud_event.data}")
```

```yaml
# cloudbuild.yaml deploy
steps:
  - name: gcr.io/google.com/cloudsdktool/cloud-sdk
    args:
      - gcloud functions deploy handler
      - --runtime=python312
      - --trigger-http
      - --allow-unauthenticated
      - --memory=512MB
      - --timeout=60s
      - --set-secrets=DATABASE_URL=projects/PROJECT/secrets/db-url/versions/latest
```

## Cloudflare Workers

```javascript
// export default fetch handler — no cold starts, runs at edge
export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    if (url.pathname === "/health") {
      return new Response(JSON.stringify({ status: "ok" }), {
        headers: { "Content-Type": "application/json" },
      });
    }

    // Access KV, D1, R2 via env bindings
    const value = await env.MY_KV.get("key");
    const db = env.MY_DB;  // D1 SQLite

    return new Response("Hello from edge", { status: 200 });
  },
};
```

```toml
# wrangler.toml
name = "my-worker"
main = "src/index.js"
compatibility_date = "2024-01-01"

[[kv_namespaces]]
binding = "MY_KV"
id = "abc123"

[[d1_databases]]
binding = "MY_DB"
database_name = "my-db"
database_id = "def456"
```

## Key Rules

- Stateless: do not store state in global variables across invocations in a serverless-hostile way — but DO reuse initialized clients.
- Always handle the DLQ for async triggers (SQS, SNS, EventBridge) to avoid silent failures.
- Timeout must be set conservatively — Lambda max is 15 min; most HTTP functions should be under 30s.
- Provisioned concurrency incurs cost even when idle — use it only for user-facing latency-sensitive functions.
- Environment variables are not encrypted in transit within the same account — use Secrets Manager for sensitive values.
