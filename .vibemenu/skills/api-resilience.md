# API Resilience Skill Guide

## Overview

Resilience patterns protect services from cascading failures. The key patterns are: circuit breaker (fail fast), retry with backoff (transient faults), timeout (bound latency), and bulkhead (isolate failures). Apply them in layers — not mutually exclusive.

## Circuit Breaker

States: **Closed** (normal, requests flow) → **Open** (failures exceeded threshold, requests fail immediately) → **Half-Open** (probe request sent, if success → Closed, else → Open).

```
Closed ──(failures > threshold)──▶ Open
  ▲                                  │
  │                                  │ (after reset timeout)
  │                                  ▼
  └──(probe succeeds)────────── Half-Open
```

### Go (sony/gobreaker)

```go
import "github.com/sony/gobreaker"

cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "order-service",
    MaxRequests: 3,              // requests allowed in half-open state
    Interval:    60 * time.Second, // window for counting failures (closed state)
    Timeout:     30 * time.Second, // how long to stay open before trying half-open
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
        return counts.Requests >= 5 && failureRatio >= 0.5
    },
    OnStateChange: func(name string, from, to gobreaker.State) {
        log.Printf("Circuit breaker %s: %s → %s", name, from, to)
    },
})

// Wrap calls
result, err := cb.Execute(func() (interface{}, error) {
    return orderService.GetOrder(ctx, id)
})
if err != nil {
    if errors.Is(err, gobreaker.ErrOpenState) {
        // Circuit is open — return cached/fallback value
        return fallbackOrder(id), nil
    }
    return nil, err
}
```

### TypeScript (opossum)

```typescript
import CircuitBreaker from "opossum";

const breaker = new CircuitBreaker(fetchUserFromService, {
  timeout: 5000,               // ms before timing out
  errorThresholdPercentage: 50, // open if 50%+ failures in window
  resetTimeout: 30000,          // ms to wait before half-open probe
  volumeThreshold: 5,           // minimum calls before evaluating threshold
  rollingCountTimeout: 10000,   // rolling window size
  rollingCountBuckets: 10,      // buckets in the window
});

breaker.fallback(() => ({ id: "fallback", name: "Unknown" }));

breaker.on("open", () => console.log("Circuit opened"));
breaker.on("halfOpen", () => console.log("Circuit half-open, probing"));
breaker.on("close", () => console.log("Circuit closed — healthy"));

// Use the breaker
const user = await breaker.fire(userId);
```

### Java (Resilience4j)

```java
import io.github.resilience4j.circuitbreaker.*;

CircuitBreakerConfig config = CircuitBreakerConfig.custom()
    .failureRateThreshold(50)            // open at 50% failure rate
    .slowCallRateThreshold(80)           // also open on 80% slow calls
    .slowCallDurationThreshold(Duration.ofSeconds(2))
    .waitDurationInOpenState(Duration.ofSeconds(30))
    .permittedNumberOfCallsInHalfOpenState(3)
    .slidingWindowType(SlidingWindowType.COUNT_BASED)
    .slidingWindowSize(10)
    .build();

CircuitBreaker cb = CircuitBreaker.of("order-service", config);

// Decorate the call
Supplier<Order> decoratedSupplier = CircuitBreaker.decorateSupplier(cb, () -> orderService.get(id));

// Execute with fallback
Order order = Try.ofSupplier(decoratedSupplier)
    .recover(CallNotPermittedException.class, e -> fallbackOrder(id))
    .get();
```

## Retry with Exponential Backoff and Jitter

Only retry idempotent operations and known transient errors (network timeout, 503, 429).

### Backoff Formula

```
delay = min(base * 2^attempt + random_jitter, max_delay)

Example with base=100ms, max=30s:
  attempt 0: 100ms + jitter
  attempt 1: 200ms + jitter
  attempt 2: 400ms + jitter
  attempt 3: 800ms + jitter
  attempt 4: 1600ms + jitter  (capped at max_delay)
```

### Go

```go
import (
    "math"
    "math/rand"
    "time"
)

type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    // IsRetryable returns true for errors that should trigger a retry
    IsRetryable func(error) bool
}

func Retry[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error) {
    var zero T
    var lastErr error

    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        result, err := fn()
        if err == nil {
            return result, nil
        }

        lastErr = err

        if !cfg.IsRetryable(err) {
            return zero, err  // non-retryable — fail immediately
        }

        if attempt == cfg.MaxAttempts-1 {
            break  // last attempt, don't sleep
        }

        delay := exponentialBackoff(attempt, cfg.BaseDelay, cfg.MaxDelay)
        select {
        case <-ctx.Done():
            return zero, ctx.Err()
        case <-time.After(delay):
        }
    }

    return zero, fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

func exponentialBackoff(attempt int, base, max time.Duration) time.Duration {
    exp := math.Pow(2, float64(attempt))
    delay := time.Duration(float64(base) * exp)
    // Add jitter: ±25% of delay
    jitter := time.Duration(rand.Int63n(int64(delay) / 2))
    delay = delay + jitter - time.Duration(int64(delay)/4)
    if delay > max {
        delay = max
    }
    return delay
}

// Usage
result, err := Retry(ctx, RetryConfig{
    MaxAttempts: 4,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    30 * time.Second,
    IsRetryable: func(err error) bool {
        var httpErr *HTTPError
        if errors.As(err, &httpErr) {
            return httpErr.StatusCode == 429 || httpErr.StatusCode >= 500
        }
        return errors.Is(err, context.DeadlineExceeded)
    },
}, func() (Order, error) {
    return orderService.GetOrder(ctx, id)
})
```

### TypeScript

```typescript
async function withRetry<T>(
  fn: () => Promise<T>,
  opts: { maxAttempts: number; baseDelayMs: number; maxDelayMs: number; isRetryable: (err: unknown) => boolean },
): Promise<T> {
  let lastErr: unknown;
  for (let attempt = 0; attempt < opts.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (err) {
      lastErr = err;
      if (!opts.isRetryable(err) || attempt === opts.maxAttempts - 1) throw err;

      const exp = Math.pow(2, attempt) * opts.baseDelayMs;
      const jitter = Math.random() * exp * 0.5;
      const delay = Math.min(exp + jitter, opts.maxDelayMs);
      await sleep(delay);
    }
  }
  throw lastErr;
}
```

### Python (tenacity)

```python
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type
import httpx

@retry(
    stop=stop_after_attempt(4),
    wait=wait_exponential(multiplier=0.1, min=0.1, max=30),
    retry=retry_if_exception_type((httpx.TransportError, httpx.HTTPStatusError)),
    reraise=True,
)
async def fetch_user(client: httpx.AsyncClient, user_id: str) -> dict:
    resp = await client.get(f"/users/{user_id}")
    resp.raise_for_status()
    return resp.json()
```

## Timeout Configuration

```go
// Per-request timeout (Go)
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
result, err := service.Call(ctx, request)
if errors.Is(err, context.DeadlineExceeded) {
    return nil, ErrServiceTimeout
}
```

```typescript
// Node.js with AbortController
async function callWithTimeout<T>(fn: (signal: AbortSignal) => Promise<T>, timeoutMs: number): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fn(controller.signal);
  } finally {
    clearTimeout(timer);
  }
}
```

## Bulkhead Pattern

Isolate resource pools per dependency to prevent one slow downstream from exhausting all resources.

```typescript
// Limit concurrent calls per dependency
class Bulkhead {
  private active = 0;
  constructor(private maxConcurrent: number) {}

  async execute<T>(fn: () => Promise<T>): Promise<T> {
    if (this.active >= this.maxConcurrent) {
      throw new Error("Bulkhead full — request rejected");
    }
    this.active++;
    try {
      return await fn();
    } finally {
      this.active--;
    }
  }
}

const userServiceBulkhead = new Bulkhead(50);   // max 50 concurrent calls to user-service
const orderServiceBulkhead = new Bulkhead(20);  // max 20 concurrent calls to order-service
```

```java
// Resilience4j Bulkhead
BulkheadConfig config = BulkheadConfig.custom()
    .maxConcurrentCalls(20)
    .maxWaitDuration(Duration.ofMillis(100))
    .build();

Bulkhead bulkhead = Bulkhead.of("order-service", config);
Supplier<Order> decorated = Bulkhead.decorateSupplier(bulkhead, () -> orderService.get(id));
```

## Combined Pattern (Circuit Breaker + Retry + Timeout)

Apply in this order (outermost to innermost): Bulkhead → Circuit Breaker → Retry → Timeout → Call.

```java
// Resilience4j — decorator chain
Supplier<Order> decorated = Decorators.ofSupplier(() -> orderService.get(id))
    .withBulkhead(bulkhead)
    .withCircuitBreaker(circuitBreaker)
    .withRetry(retry)
    .withTimeLimiter(timeLimiter, scheduledExecutorService)
    .withFallback(List.of(CallNotPermittedException.class, TimeoutException.class),
        e -> fallbackOrder(id))
    .decorate();
```

## Retryable vs Non-Retryable Errors

| Retryable | Non-Retryable |
|-----------|--------------|
| 429 Too Many Requests | 400 Bad Request |
| 503 Service Unavailable | 401 Unauthorized |
| 504 Gateway Timeout | 403 Forbidden |
| Network timeout / connection reset | 404 Not Found |
| 500 Internal Server Error (sometimes) | 409 Conflict |

## Rules

- Always combine circuit breaker with retry — retrying into an open circuit wastes resources
- Add jitter to avoid thundering herd (all clients retrying at the same time after an outage)
- Set timeouts at every network boundary — never rely on upstream to enforce its own timeout
- Log circuit breaker state changes for observability
- Use bulkheads to give critical paths (payment, auth) dedicated resource pools separate from lower-priority paths
- Expose circuit breaker state in health checks so orchestrators can route traffic accordingly
