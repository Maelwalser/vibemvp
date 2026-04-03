# Rate Limiting Skill Guide

## Overview

Rate limiting controls how many requests a client can make in a time window. Three primary algorithms: **token bucket** (bursty traffic allowed), **sliding window** (smooth enforcement), **fixed window** (simple but has boundary burst issue). All production implementations use Redis for atomic cross-instance state.

---

## Algorithm Implementations

### Token Bucket (Redis HASH + Lua)

Allows short bursts up to bucket capacity; refills at a steady rate.

```lua
-- rate_limit_token_bucket.lua
-- KEYS[1] = bucket key (e.g., "rl:user:123")
-- ARGV[1] = capacity (max tokens)
-- ARGV[2] = refill_rate (tokens per second)
-- ARGV[3] = now (unix timestamp float)
-- ARGV[4] = tokens_requested (usually 1)
-- Returns: 1 (allowed) or 0 (denied)

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

-- Refill tokens based on elapsed time
local elapsed = now - last_refill
local new_tokens = math.min(capacity, tokens + elapsed * refill_rate)

if new_tokens < requested then
    -- Update last_refill but not tokens (no partial grant)
    redis.call("HMSET", key, "tokens", new_tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) + 1)
    return 0
end

redis.call("HMSET", key, "tokens", new_tokens - requested, "last_refill", now)
redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) + 1)
return 1
```

Go usage:
```go
func allowTokenBucket(ctx context.Context, rdb *redis.Client, key string, capacity, refillRate int) (bool, error) {
    script := redis.NewScript(tokenBucketLua)
    result, err := script.Run(ctx, rdb,
        []string{key},
        capacity, refillRate,
        float64(time.Now().UnixNano())/1e9,
        1,
    ).Int()
    if err != nil {
        return false, fmt.Errorf("rate limit: %w", err)
    }
    return result == 1, nil
}
```

### Sliding Window (Redis ZSET via EVAL)

Counts requests in a rolling time window; no boundary burst problem.

```lua
-- rate_limit_sliding.lua
-- KEYS[1] = zset key
-- ARGV[1] = window size in seconds
-- ARGV[2] = max requests per window
-- ARGV[3] = now (unix timestamp ms as string)
-- Returns: 1 (allowed) or 0 (denied)

local key = KEYS[1]
local window = tonumber(ARGV[1]) * 1000   -- convert to ms
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local window_start = now - window

-- Remove requests outside the window
redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

-- Count current requests in window
local count = redis.call("ZCARD", key)

if count >= limit then
    return 0
end

-- Add this request
redis.call("ZADD", key, now, now .. "-" .. math.random(1000000))
redis.call("PEXPIRE", key, window)
return 1
```

Go usage:
```go
func allowSlidingWindow(ctx context.Context, rdb *redis.Client, key string, window time.Duration, limit int) (bool, error) {
    script := redis.NewScript(slidingWindowLua)
    nowMs := time.Now().UnixMilli()
    result, err := script.Run(ctx, rdb,
        []string{key},
        int(window.Seconds()), limit, nowMs,
    ).Int()
    if err != nil {
        return false, fmt.Errorf("sliding window rate limit: %w", err)
    }
    return result == 1, nil
}
```

### Fixed Window (Redis INCR + TTL)

Simplest implementation; vulnerable to boundary burst (2x limit at window edge).

```go
func allowFixedWindow(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
    pipe := rdb.Pipeline()
    incr := pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, window)  // no-op if key existed
    if _, err := pipe.Exec(ctx); err != nil {
        return false, fmt.Errorf("fixed window: %w", err)
    }
    return incr.Val() <= int64(limit), nil
}
```

Note: Use `SET key 0 EX <window> NX` + `INCR` pattern for atomic TTL-on-first-set.

---

## Key Strategy

Structure keys to scope rate limits precisely:

```go
// Per-user
key := fmt.Sprintf("rl:user:%s", userID)

// Per-IP
key := fmt.Sprintf("rl:ip:%s", clientIP)

// Per-API-key
key := fmt.Sprintf("rl:apikey:%s", apiKey)

// Per-user per-endpoint (fine-grained)
key := fmt.Sprintf("rl:user:%s:endpoint:%s", userID, endpointSlug)

// Per-IP per-endpoint (unauthenticated endpoints)
key := fmt.Sprintf("rl:ip:%s:endpoint:%s", clientIP, endpointSlug)
```

---

## HTTP 429 Response

Always return `429 Too Many Requests` with `Retry-After` header:

```go
func rateLimitMiddleware(rdb *redis.Client) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := "rl:ip:" + c.IP()
        allowed, err := allowSlidingWindow(c.Context(), rdb, key, time.Minute, 100)
        if err != nil {
            // Fail open on Redis errors to avoid blocking legitimate traffic
            return c.Next()
        }
        if !allowed {
            c.Set("Retry-After", "60")
            c.Set("X-RateLimit-Limit", "100")
            c.Set("X-RateLimit-Window", "60s")
            return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
                "error": "rate limit exceeded",
            })
        }
        return c.Next()
    }
}
```

---

## Framework Libraries

### Node.js — express-rate-limit

```typescript
import rateLimit from 'express-rate-limit'
import RedisStore from 'rate-limit-redis'

const limiter = rateLimit({
  windowMs: 60 * 1000,   // 1 minute
  max: 100,
  standardHeaders: true,  // Return RateLimit-* headers
  legacyHeaders: false,
  store: new RedisStore({
    sendCommand: (...args: string[]) => redis.sendCommand(args),
  }),
  keyGenerator: (req) => req.headers['x-api-key'] as string || req.ip,
  handler: (req, res) => {
    res.status(429).json({ error: 'rate limit exceeded' })
  },
})
app.use('/api/', limiter)
```

### Node.js — @fastify/rate-limit

```typescript
await fastify.register(import('@fastify/rate-limit'), {
  max: 100,
  timeWindow: '1 minute',
  redis: redisClient,
  keyGenerator: (request) => request.headers['x-api-key'] || request.ip,
  errorResponseBuilder: (request, context) => ({
    statusCode: 429,
    error: 'Too Many Requests',
    message: `Rate limit exceeded. Retry in ${context.after}`,
  }),
})
```

### Java — Bucket4j (Spring Boot)

```java
@Component
public class RateLimitFilter extends OncePerRequestFilter {
    private final Cache<String, Bucket> bucketCache;

    public RateLimitFilter() {
        this.bucketCache = Caffeine.newBuilder()
            .expireAfterWrite(1, TimeUnit.HOURS)
            .build();
    }

    @Override
    protected void doFilterInternal(HttpServletRequest req, HttpServletResponse res,
                                    FilterChain chain) throws IOException, ServletException {
        String key = getApiKey(req);
        Bucket bucket = bucketCache.get(key, k -> Bucket.builder()
            .addLimit(Bandwidth.classic(100, Refill.intervally(100, Duration.ofMinutes(1))))
            .build());

        if (bucket.tryConsume(1)) {
            chain.doFilter(req, res);
        } else {
            res.setStatus(429);
            res.addHeader("Retry-After", "60");
            res.getWriter().write("{\"error\":\"rate limit exceeded\"}");
        }
    }
}
```

### Go — golang.org/x/time/rate

```go
import "golang.org/x/time/rate"

// Per-IP in-memory limiter (single instance only)
type IPRateLimiter struct {
    limiters sync.Map
    r        rate.Limit
    b        int
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
    v, _ := l.limiters.LoadOrStore(ip, rate.NewLimiter(l.r, l.b))
    return v.(*rate.Limiter)
}

// r = requests/sec, b = burst capacity
limiter := &IPRateLimiter{r: rate.Limit(10), b: 30}

// In handler:
if !limiter.getLimiter(c.IP()).Allow() {
    return c.Status(429).JSON(fiber.Map{"error": "rate limit exceeded"})
}
```

### Python — fastapi-limiter

```python
from fastapi_limiter import FastAPILimiter
from fastapi_limiter.depends import RateLimiter
import redis.asyncio as aioredis

@app.on_event("startup")
async def startup():
    redis = aioredis.from_url("redis://localhost", encoding="utf-8")
    await FastAPILimiter.init(redis)

@app.post("/login")
@limiter.limit("5/minute")  # or use Depends
async def login(
    request: Request,
    _: None = Depends(RateLimiter(times=5, seconds=60))
):
    ...
```

---

## Key Rules

- Use Redis-based stores for distributed/multi-instance deployments; never use in-memory for scaled services
- Always fail **open** on Redis errors — dropping legitimate traffic is worse than a brief limit bypass
- Prefer **sliding window** for APIs where boundary bursts are unacceptable (auth endpoints)
- Use **token bucket** for bursty workloads (upload, batch jobs) where short bursts are acceptable
- Return `Retry-After` header — clients should use it for backoff
- Apply stricter limits to unauthenticated endpoints and auth/login routes
- Log every 429 with key, endpoint, and timestamp for abuse pattern analysis
- Exempt health check endpoints (`/health`, `/ready`) from rate limits
