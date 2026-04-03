# Domain Caching Skill Guide

## Overview

Caching strategies (cache-aside, read-through, write-through, write-behind), invalidation approaches (TTL vs event-driven), TTL presets, cache key patterns, and CDN cache integration.

---

## Cache-Aside (Lazy Loading)

Application checks cache first; on a miss it queries the DB and populates the cache. Most common pattern.

```python
def get_user(user_id: str) -> dict:
    key = f"user:{user_id}:v1"
    cached = redis.get(key)
    if cached:
        return json.loads(cached)          # cache HIT

    user = db.query("SELECT * FROM users WHERE id = %s", [user_id])
    if user:
        redis.setex(key, 300, json.dumps(user))  # populate, TTL = 5m
    return user
```

```go
func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    key := fmt.Sprintf("user:%s:v1", id)
    if data, err := r.cache.Get(ctx, key).Bytes(); err == nil {
        var u User
        return &u, json.Unmarshal(data, &u)
    }
    u, err := r.db.QueryUser(ctx, id)
    if err != nil { return nil, err }
    if b, _ := json.Marshal(u); b != nil {
        r.cache.Set(ctx, key, b, 5*time.Minute)
    }
    return u, nil
}
```

**Tradeoff:** First request after expiry is a cache miss (thundering herd risk — use probabilistic early expiry or mutex lock).

---

## Read-Through

Cache layer itself queries the DB on miss. Application always reads from cache.

```go
// Typically provided by the cache library (e.g., groupcache, Momento)
cache := groupcache.NewGroup("users", 64<<20, groupcache.GetterFunc(
    func(ctx context.Context, key string, dest groupcache.Sink) error {
        user, err := db.QueryUser(ctx, key)
        if err != nil { return err }
        data, _ := json.Marshal(user)
        return dest.SetBytes(data, time.Now().Add(5*time.Minute))
    },
))
```

**Tradeoff:** Cache controls DB access; harder to control cache population logic per caller.

---

## Write-Through

Write to cache and DB synchronously in the same operation. Cache is always consistent.

```python
def update_user(user_id: str, data: dict) -> dict:
    db.execute("UPDATE users SET ... WHERE id = %s", [user_id, ...])
    key = f"user:{user_id}:v1"
    redis.setex(key, 300, json.dumps(data))   # update cache immediately
    return data
```

**Tradeoff:** Every write hits both DB and cache; slightly higher write latency.

---

## Write-Behind (Write-Back)

Write to cache immediately; persist to DB asynchronously. Optimizes write throughput.

```python
def update_user(user_id: str, data: dict):
    key = f"user:{user_id}:v1"
    redis.setex(key, 300, json.dumps(data))   # fast cache write
    queue.publish("user.write_behind", {"id": user_id, "data": data})

# Background worker
def process_write_behind(msg):
    db.execute("UPDATE users SET ... WHERE id = %s", [msg["id"], ...])
```

**Tradeoff:** Risk of data loss if the cache node fails before the async write completes. Use only for non-critical or re-computable data.

---

## Invalidation Strategies

### TTL-Based

Set an expiry on every cache write. No explicit invalidation needed.

```python
redis.setex(key, ttl_seconds, value)
```

Best for: read-heavy data with tolerable staleness, reference data.

### Event-Driven Invalidation

Publish a cache invalidation message on every update or delete.

```python
# On write
def update_product(product_id: str, data: dict):
    db.update(product_id, data)
    event_bus.publish("cache.invalidate", {"key": f"product:{product_id}:v1"})

# Cache invalidation consumer
def on_invalidate(event):
    redis.delete(event["key"])
```

Best for: highly consistent requirements, low read-to-write ratio.

### Hybrid

Use TTL as safety net + event-driven invalidation for immediate consistency on writes.

---

## TTL Presets

| Preset | TTL | Use Case |
|--------|-----|----------|
| Real-time | 30s | Live scores, dashboards, prices |
| Semi-real-time | 1m | Social feeds, notifications |
| Short | 5m | User sessions, search results |
| Medium | 15m | Product catalog, recommendations |
| Long | 1h | Reference data, categories |
| Static | 24h | Configuration, i18n strings |

---

## Cache Key Pattern

```
{entity}:{id}:{version}

Examples:
  user:550e8400-e29b-41d4-a716-446655440000:v1
  product:42:v3
  product:42:related:v1       # computed / derived
  search:users:q=john:page=2:v1
```

Rules:
- Include a version suffix (`v1`, `v2`) to allow zero-downtime cache schema changes via key rotation.
- Namespace by entity type to avoid collisions across services.
- Avoid embedding secrets or PII in keys.

---

## Redis / Valkey Configuration

```python
import redis

cache = redis.Redis(
    host=os.environ["REDIS_HOST"],
    port=int(os.environ.get("REDIS_PORT", 6379)),
    db=0,
    decode_responses=True,
    socket_timeout=0.5,          # fail fast on cache miss
    socket_connect_timeout=0.5,
    retry_on_timeout=False,      # don't retry — fall through to DB
)
```

```go
rdb := redis.NewClient(&redis.Options{
    Addr:         os.Getenv("REDIS_ADDR"),
    DialTimeout:  500 * time.Millisecond,
    ReadTimeout:  500 * time.Millisecond,
    WriteTimeout: 500 * time.Millisecond,
})
```

---

## CDN Cache

Use `Cache-Control` for browser/CDN behavior and `Surrogate-Key` (or `Cache-Tag`) for targeted purging.

```http
HTTP/1.1 200 OK
Cache-Control: public, max-age=3600, stale-while-revalidate=60
Surrogate-Key: product product:42 category:electronics
ETag: "abc123"
```

### Purge by Tag (Cloudflare, Fastly, Varnish)

```bash
# Cloudflare Cache Tag purge
curl -X POST "https://api.cloudflare.com/client/v4/zones/{zone_id}/purge_cache" \
  -H "Authorization: Bearer $CF_TOKEN" \
  -d '{"tags":["product:42"]}'
```

```python
# On product update, purge related CDN cache
def update_product(product_id: str, data: dict):
    db.update(product_id, data)
    redis.delete(f"product:{product_id}:v1")       # Redis invalidation
    cdn.purge_tags([f"product:{product_id}"])       # CDN purge
```

---

## Key Rules

- Always set a TTL on every cache entry — unbounded entries cause memory exhaustion.
- Use cache-aside as the default; only switch to write-through when strong consistency is required.
- Never cache sensitive PII without encrypting the cached value.
- Cache key must include a version component to support zero-downtime schema changes.
- Set short socket timeouts on the cache client (500ms) and fall through to the DB on timeout — never let a slow cache degrade the application.
- Pair CDN `Surrogate-Key` tags with event-driven purging for publicly cacheable API responses.
