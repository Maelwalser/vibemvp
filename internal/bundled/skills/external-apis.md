# External APIs Skill Guide

## Overview

Patterns for integrating third-party providers: authentication mechanisms, circuit breakers, retry with backoff, fallback strategies, inbound webhook verification, and SDK vs raw HTTP tradeoffs.

---

## Provider Integration Pattern

Every external provider integration should define these components up front:

```
Provider Config:
  base_url:    https://api.provider.com/v1
  auth:        API Key | OAuth2 | Bearer | Basic | mTLS
  rate_limit:  1000 req/min (per key)
  timeout:     10s
  retryable:   429, 502, 503, 504
```

---

## Authentication Mechanisms

### API Key — Header

```http
GET /users HTTP/1.1
Host: api.provider.com
X-API-Key: sk_live_abc123
```

```go
req.Header.Set("X-API-Key", os.Getenv("PROVIDER_API_KEY"))
```

### OAuth2 — Client Credentials (M2M)

```python
import httpx

def get_access_token() -> str:
    resp = httpx.post(
        "https://auth.provider.com/oauth2/token",
        data={
            "grant_type":    "client_credentials",
            "client_id":     os.environ["PROVIDER_CLIENT_ID"],
            "client_secret": os.environ["PROVIDER_CLIENT_SECRET"],
            "scope":         "read:users write:users",
        },
        timeout=10,
    )
    resp.raise_for_status()
    return resp.json()["access_token"]
```

Cache the token until `expires_in - 60` seconds to avoid per-request token fetches.

### Bearer Token

```http
Authorization: Bearer eyJhbGciOiJSUzI1NiJ9...
```

### Basic Auth

```python
import base64
credentials = base64.b64encode(b"user:password").decode()
headers = {"Authorization": f"Basic {credentials}"}
```

### mTLS — Mutual TLS

```go
cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
transport := &http.Transport{TLSClientConfig: tlsConfig}
client := &http.Client{Transport: transport}
```

### PKCE (for user-facing OAuth2 flows)

Generate `code_verifier` (random 43-128 chars), derive `code_challenge = BASE64URL(SHA256(verifier))`. Include in authorization redirect. Exchange code + verifier at token endpoint.

---

## Circuit Breaker

Prevent cascading failures when an external service is degraded. Transitions: Closed → Open → Half-Open → Closed.

```go
// Using github.com/sony/gobreaker
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "provider-api",
    MaxRequests: 5,                    // max requests in Half-Open
    Interval:    60 * time.Second,     // Closed state stat reset interval
    Timeout:     30 * time.Second,     // time in Open before trying Half-Open
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures >= 5
    },
    OnStateChange: func(name string, from, to gobreaker.State) {
        log.Printf("circuit %s: %s → %s", name, from, to)
    },
})

result, err := cb.Execute(func() (interface{}, error) {
    return callProviderAPI()
})
if errors.Is(err, gobreaker.ErrOpenState) {
    return fallbackValue()
}
```

```python
# Using pybreaker
import pybreaker

breaker = pybreaker.CircuitBreaker(fail_max=5, reset_timeout=30)

@breaker
def call_provider():
    ...

try:
    result = call_provider()
except pybreaker.CircuitBreakerError:
    result = fallback_value()
```

---

## Retry with Exponential Backoff + Jitter

Retry on transient errors (429, 502, 503, 504) with jitter to avoid synchronized thundering herds.

```python
import time
import random
import httpx

def call_with_retry(url: str, headers: dict, max_retries: int = 3) -> dict:
    base_delay = 1.0
    for attempt in range(max_retries + 1):
        try:
            resp = httpx.get(url, headers=headers, timeout=10)
            if resp.status_code == 429:
                retry_after = int(resp.headers.get("Retry-After", base_delay))
                time.sleep(retry_after)
                continue
            resp.raise_for_status()
            return resp.json()
        except (httpx.TimeoutException, httpx.HTTPStatusError) as e:
            if attempt == max_retries:
                raise
            delay = base_delay * (2 ** attempt) + random.uniform(0, 1)
            time.sleep(delay)
```

```go
func callWithRetry(ctx context.Context, url string) ([]byte, error) {
    var lastErr error
    for attempt := 0; attempt <= 3; attempt++ {
        if attempt > 0 {
            jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
            delay := time.Duration(math.Pow(2, float64(attempt)))*time.Second + jitter
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
        resp, err := http.Get(url)
        if err != nil { lastErr = err; continue }
        if resp.StatusCode == 429 || resp.StatusCode >= 500 {
            lastErr = fmt.Errorf("status %d", resp.StatusCode)
            continue
        }
        return io.ReadAll(resp.Body)
    }
    return nil, lastErr
}
```

---

## Fallback Value Strategy

When the circuit is open or retries are exhausted, return a safe default:

```go
func GetRecommendations(userID string) ([]Product, error) {
    result, err := cb.Execute(func() (interface{}, error) {
        return fetchFromRecommendationEngine(userID)
    })
    if err != nil {
        // Fallback: return top-selling products from local cache
        return localCache.GetTopSellers(), nil
    }
    return result.([]Product), nil
}
```

Fallback strategies:
- Return cached last-known-good value
- Return default/empty response
- Serve from a degraded local dataset
- Return a 503 with `Retry-After` to the caller

---

## Timeout + Fail-Fast

Always set explicit timeouts. Never rely on OS defaults.

```go
client := &http.Client{Timeout: 10 * time.Second}
```

```python
resp = httpx.get(url, timeout=httpx.Timeout(connect=2.0, read=8.0, write=5.0, pool=1.0))
```

Rule: connection timeout < read timeout < overall request timeout. Fail fast — a slow external service should not block your request handler indefinitely.

---

## Inbound Webhook

### Route Definition

```
POST /webhooks/{provider}
```

One route per provider; use path or query param to route to the correct handler.

### HMAC-SHA256 Signature Verification

```python
import hmac, hashlib

def verify_webhook_signature(payload: bytes, signature_header: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256,
    ).hexdigest()
    # Constant-time comparison prevents timing attacks
    return hmac.compare_digest(f"sha256={expected}", signature_header)

@app.post("/webhooks/stripe")
async def stripe_webhook(request: Request):
    payload   = await request.body()
    signature = request.headers.get("Stripe-Signature", "")
    if not verify_webhook_signature(payload, signature, os.environ["STRIPE_WEBHOOK_SECRET"]):
        raise HTTPException(status_code=401, detail="Invalid signature")
    event = json.loads(payload)
    # process event
```

### Idempotency Key

```python
@app.post("/webhooks/stripe")
async def stripe_webhook(request: Request):
    ...
    event_id = event["id"]   # provider-assigned idempotency key
    if redis.exists(f"webhook:processed:{event_id}"):
        return {"status": "already_processed"}
    redis.setex(f"webhook:processed:{event_id}", 86400, "1")
    # process event
```

---

## SDK Wrapper vs Raw HTTP Client

| Factor | SDK | Raw HTTP |
|--------|-----|----------|
| Dev speed | Fast — types and auth built-in | Slower |
| Dependency risk | Locked to SDK version | Control your own |
| Customization | Limited | Full control |
| Maintenance | Provider updates SDK | You maintain |
| Circuit breaker | Must wrap SDK | Direct integration |

**Rule:** Use the official SDK for complex providers (Stripe, Twilio, AWS). Use raw HTTP for simple REST APIs where the SDK adds more friction than value.

```go
// SDK wrapper pattern — hide SDK details behind a domain interface
type PaymentProvider interface {
    Charge(ctx context.Context, amount Money, token string) (ChargeID, error)
}

type StripeProvider struct { client *stripe.Client }

func (s *StripeProvider) Charge(ctx context.Context, amount Money, token string) (ChargeID, error) {
    params := &stripe.ChargeParams{
        Amount:   stripe.Int64(amount.Cents()),
        Currency: stripe.String(string(amount.Currency)),
        Source:   &stripe.SourceParams{Token: stripe.String(token)},
    }
    ch, err := s.client.Charges.New(params)
    if err != nil { return "", mapStripeError(err) }
    return ChargeID(ch.ID), nil
}
```

---

## Error Mapping to Internal Domain Errors

Never let provider error types leak into business logic.

```go
func mapStripeError(err error) error {
    var stripeErr *stripe.Error
    if errors.As(err, &stripeErr) {
        switch stripeErr.Code {
        case stripe.ErrorCodeCardDeclined:
            return ErrPaymentDeclined
        case stripe.ErrorCodeInsufficientFunds:
            return ErrInsufficientFunds
        case stripe.ErrorCodeRateLimitExceeded:
            return ErrRateLimited
        default:
            return fmt.Errorf("payment provider error: %w", ErrProviderUnavailable)
        }
    }
    return fmt.Errorf("unexpected payment error: %w", err)
}
```

---

## Key Rules

- Never hardcode API keys or secrets — always read from environment variables.
- Always apply a circuit breaker to external calls; configure `OnStateChange` to emit a metric or alert.
- Retry only on idempotent operations (GET) or with an idempotency key on POST/PATCH.
- Verify webhook signatures with constant-time comparison (`hmac.compare_digest`) to prevent timing attacks.
- Deduplicate webhooks via the provider's event ID stored in Redis or the DB.
- Map all provider errors to internal domain error types before they reach business logic.
- Set explicit timeouts on every HTTP client — never use zero/default.
