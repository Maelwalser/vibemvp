# Cloudflare Workers Skill Guide

## Overview

Cloudflare Workers run JavaScript/TypeScript at the edge (200+ PoPs globally) using the V8 isolate model — no cold starts, sub-millisecond startup. Workers have access to KV (key-value), D1 (SQLite), R2 (object storage), Queues, and Durable Objects.

## Request Handler Pattern

```typescript
// src/index.ts — main entry point
export interface Env {
  // Bindings declared in wrangler.toml
  KV_STORE: KVNamespace;
  DB: D1Database;
  BUCKET: R2Bucket;
  QUEUE: Queue;
  API_KEY: string;             // secret from wrangler secret put
  ENVIRONMENT: string;         // var from wrangler.toml [vars]
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);

    try {
      return await router(request, url, env, ctx);
    } catch (err) {
      console.error("Unhandled error:", err);
      return new Response(JSON.stringify({ error: "Internal server error" }), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      });
    }
  },
};

async function router(request: Request, url: URL, env: Env, ctx: ExecutionContext): Promise<Response> {
  const { pathname } = url;
  const method = request.method;

  // Simple router
  if (method === "GET" && pathname === "/health") {
    return json({ status: "ok", region: request.cf?.colo });
  }

  if (method === "GET" && pathname.startsWith("/api/users/")) {
    const id = pathname.split("/")[3];
    return handleGetUser(id, env);
  }

  if (method === "POST" && pathname === "/api/users") {
    return handleCreateUser(request, env);
  }

  return new Response("Not Found", { status: 404 });
}

// Helpers
function json(data: unknown, status = 200, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
  });
}

function cors(response: Response, origin = "*"): Response {
  const headers = new Headers(response.headers);
  headers.set("Access-Control-Allow-Origin", origin);
  headers.set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
  headers.set("Access-Control-Allow-Headers", "Content-Type, Authorization");
  return new Response(response.body, { status: response.status, headers });
}
```

## wrangler.toml Configuration

```toml
name = "my-worker"
main = "src/index.ts"
compatibility_date = "2024-03-01"
compatibility_flags = ["nodejs_compat"]

[vars]
ENVIRONMENT = "production"
APP_URL = "https://app.example.com"

# KV namespace — create with: wrangler kv namespace create "CACHE"
[[kv_namespaces]]
binding = "KV_STORE"
id = "abc123...production"
preview_id = "def456...preview"

# D1 database — create with: wrangler d1 create my-db
[[d1_databases]]
binding = "DB"
database_name = "my-db"
database_id = "uuid-from-create-command"

# R2 bucket — create with: wrangler r2 bucket create my-bucket
[[r2_buckets]]
binding = "BUCKET"
bucket_name = "my-bucket"

# Queue — create with: wrangler queues create my-queue
[[queues.producers]]
binding = "QUEUE"
queue = "my-queue"

# Queue consumer
[[queues.consumers]]
queue = "my-queue"
max_batch_size = 10
max_batch_timeout = 5
max_retries = 3

# Routes (production)
routes = [
  { pattern = "api.example.com/*", zone_name = "example.com" }
]

# Development (wrangler dev)
[dev]
port = 8787
local_protocol = "http"
```

## KV Store Operations

```typescript
async function handleGetUser(id: string, env: Env): Promise<Response> {
  // Try cache first
  const cached = await env.KV_STORE.get(`user:${id}`, { type: "json" });
  if (cached) {
    return json(cached, 200, { "X-Cache": "HIT" });
  }

  // Fallback to D1
  const user = await env.DB.prepare("SELECT * FROM users WHERE id = ?")
    .bind(id)
    .first();

  if (!user) {
    return json({ error: "not_found" }, 404);
  }

  // Cache for 5 minutes
  await env.KV_STORE.put(`user:${id}`, JSON.stringify(user), {
    expirationTtl: 300,  // seconds
  });

  return json(user, 200, { "X-Cache": "MISS" });
}

// KV with metadata
await env.KV_STORE.put("session:abc", JSON.stringify({ userId: "123" }), {
  expirationTtl: 3600,
  metadata: { createdAt: new Date().toISOString() },
});

// List keys with prefix
const list = await env.KV_STORE.list({ prefix: "session:", limit: 100 });
for (const key of list.keys) {
  console.log(key.name, key.expiration, key.metadata);
}

// Delete
await env.KV_STORE.delete(`user:${id}`);
```

## D1 SQL Database

```typescript
async function handleCreateUser(request: Request, env: Env): Promise<Response> {
  const body = await request.json() as { email: string; name: string };

  if (!body.email || !body.name) {
    return json({ error: "email and name required" }, 400);
  }

  const id = crypto.randomUUID();

  // Single insert
  const result = await env.DB.prepare(
    "INSERT INTO users (id, email, name, created_at) VALUES (?, ?, ?, ?)"
  )
    .bind(id, body.email, body.name, new Date().toISOString())
    .run();

  if (!result.success) {
    return json({ error: "insert failed" }, 500);
  }

  return json({ id, email: body.email, name: body.name }, 201);
}

// Batch statements (atomic)
async function batchInsert(env: Env, items: Array<{ id: string; name: string }>) {
  const statements = items.map(item =>
    env.DB.prepare("INSERT INTO items (id, name) VALUES (?, ?)")
      .bind(item.id, item.name)
  );
  await env.DB.batch(statements);
}

// Query with typed result
interface UserRow {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

const { results } = await env.DB.prepare("SELECT * FROM users WHERE email = ?")
  .bind("alice@example.com")
  .all<UserRow>();
```

## D1 Migrations

```bash
# Create migration file
wrangler d1 migrations create my-db create-users-table

# migrations/0001_create_users_table.sql
# CREATE TABLE users (
#   id TEXT PRIMARY KEY,
#   email TEXT UNIQUE NOT NULL,
#   name TEXT NOT NULL,
#   created_at TEXT NOT NULL
# );

# Apply migrations
wrangler d1 migrations apply my-db              # production
wrangler d1 migrations apply my-db --local      # local dev
```

## Queues

```typescript
// Producer — enqueue a message
await env.QUEUE.send({ type: "email", to: "user@example.com", template: "welcome" });

// Consumer — process messages
export default {
  async queue(batch: MessageBatch<{ type: string; to: string; template: string }>, env: Env) {
    for (const message of batch.messages) {
      try {
        await sendEmail(message.body, env);
        message.ack();     // acknowledge success
      } catch (err) {
        message.retry();   // retry this message
      }
    }
  },
};
```

## Environment Bindings and Secrets

```bash
# Set secrets (not stored in wrangler.toml)
wrangler secret put API_KEY
wrangler secret put DATABASE_URL

# List secrets
wrangler secret list

# Deploy
wrangler deploy

# Tail logs in production
wrangler tail

# Local dev (uses local KV/D1 simulators)
wrangler dev
```

## Rules

- Workers have a 10ms CPU time limit on the free plan (50ms on paid) — offload heavy work to Queues or DO
- Never store sensitive data in KV metadata — it is returned in list responses
- Use `ctx.waitUntil(promise)` to run background work after returning the response (logging, analytics)
- D1 is transactional within a `batch()` call — use batch for multi-statement atomic writes
- KV is eventually consistent across regions — not suitable as a primary database or for strong consistency needs
- Always validate input before SQL bind parameters — D1 bind protects against injection but not business logic errors
- Use `compatibility_flags = ["nodejs_compat"]` to access Node.js built-ins (Buffer, crypto, etc.)
