# Session-Based Auth Skill Guide

## Overview

Server-side sessions store authentication state on the server and issue the client an opaque session ID cookie. Unlike JWTs, the server controls session lifetime directly — invalidation is immediate and requires no token blacklisting.

Use sessions when you need centralized control, instant revocation, and don't require a stateless architecture.

---

## Implementation Pattern

### Session ID Generation

Session IDs must be cryptographically random and unpredictable.

```go
// Go — crypto/rand session ID
import "crypto/rand"
import "encoding/hex"

func generateSessionID() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("session ID generation: %w", err)
    }
    return hex.EncodeToString(b), nil
}
```

```python
# Python
import secrets
def generate_session_id() -> str:
    return secrets.token_hex(32)  # 64-char hex string, 256 bits of entropy
```

```typescript
// Node.js
import crypto from 'crypto'
function generateSessionID(): string {
  return crypto.randomBytes(32).toString('hex')
}
```

### Redis Session Store

```go
// Go — store and retrieve session in Redis
type Session struct {
    UserID    string    `json:"user_id"`
    Email     string    `json:"email"`
    Roles     []string  `json:"roles"`
    CreatedAt time.Time `json:"created_at"`
    LastSeen  time.Time `json:"last_seen"`
}

func SaveSession(ctx context.Context, rdb *redis.Client, sid string, s Session, ttl time.Duration) error {
    data, err := json.Marshal(s)
    if err != nil {
        return fmt.Errorf("marshal session: %w", err)
    }
    return rdb.Set(ctx, "session:"+sid, data, ttl).Err()
}

func GetSession(ctx context.Context, rdb *redis.Client, sid string) (*Session, error) {
    data, err := rdb.Get(ctx, "session:"+sid).Bytes()
    if err != nil {
        return nil, fmt.Errorf("session not found: %w", err)
    }
    var s Session
    if err := json.Unmarshal(data, &s); err != nil {
        return nil, fmt.Errorf("unmarshal session: %w", err)
    }
    return &s, nil
}

func DeleteSession(ctx context.Context, rdb *redis.Client, sid string) error {
    return rdb.Del(ctx, "session:"+sid).Err()
}
```

### Express-Session (Node.js)

```typescript
import session from 'express-session'
import RedisStore from 'connect-redis'
import { createClient } from 'redis'

const redisClient = createClient({ url: process.env.REDIS_URL })
await redisClient.connect()

app.use(session({
  store: new RedisStore({ client: redisClient }),
  secret: process.env.SESSION_SECRET!,
  name: '__sid',             // obscure default name
  resave: false,
  saveUninitialized: false,
  cookie: {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 24 * 60 * 60 * 1000, // 1 day
  },
}))
```

### Django SessionMiddleware

```python
# settings.py
SESSION_ENGINE = 'django.contrib.sessions.backends.cache'
SESSION_CACHE_ALIAS = 'default'    # points to Redis cache config
SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SECURE = True       # HTTPS only
SESSION_COOKIE_SAMESITE = 'Strict'
SESSION_COOKIE_AGE = 86400         # seconds
SESSION_COOKIE_NAME = '__sid'

CACHES = {
    'default': {
        'BACKEND': 'django.core.cache.backends.redis.RedisCache',
        'LOCATION': os.environ['REDIS_URL'],
    }
}
```

### Rails Session Store

```ruby
# config/initializers/session_store.rb
Rails.application.config.session_store :redis_store,
  servers: [ENV['REDIS_URL']],
  expire_after: 24.hours,
  key: '__sid',
  secure: Rails.env.production?,
  httponly: true,
  same_site: :strict
```

---

## Token / Session Management

### Session Expiry and Renewal

Extend the session TTL on each authenticated request (sliding expiry):

```go
func sessionMiddleware(rdb *redis.Client) fiber.Handler {
    return func(c *fiber.Ctx) error {
        sid := extractSessionCookie(c)
        session, err := GetSession(c.Context(), rdb, sid)
        if err != nil {
            return fiber.ErrUnauthorized
        }
        // Slide TTL
        rdb.Expire(c.Context(), "session:"+sid, 24*time.Hour)
        session.LastSeen = time.Now()
        _ = SaveSession(c.Context(), rdb, sid, *session, 24*time.Hour)
        c.Locals("user", session)
        return c.Next()
    }
}
```

### Instant Revocation (Logout)

```typescript
app.post('/auth/logout', async (req, res) => {
  req.session.destroy((err) => {
    if (err) {
      console.error('Session destroy error:', err)
      return res.status(500).json({ error: 'Logout failed' })
    }
    res.clearCookie('__sid')
    res.json({ ok: true })
  })
})
```

---

## CSRF Protection

Sessions require CSRF protection because the browser automatically sends cookies.

### Double-Submit Cookie Pattern

```typescript
import crypto from 'crypto'

// On login: issue a CSRF token in a non-HttpOnly cookie
res.cookie('csrf_token', crypto.randomBytes(32).toString('hex'), {
  httpOnly: false,  // readable by JS so client can read and echo it
  secure: true,
  sameSite: 'strict',
})

// Middleware: verify header matches cookie
function csrfProtect(req, res, next) {
  if (['GET', 'HEAD', 'OPTIONS'].includes(req.method)) return next()
  const header = req.headers['x-csrf-token']
  const cookie = req.cookies['csrf_token']
  if (!header || !cookie || header !== cookie) {
    return res.status(403).json({ error: 'Invalid CSRF token' })
  }
  next()
}
```

### Synchronizer Token Pattern (Django)

Django uses this pattern by default via `{% csrf_token %}` in templates and `CsrfViewMiddleware`. For APIs, include the `X-CSRFToken` header.

```python
# Exempt views that don't need CSRF (e.g., webhook receivers)
from django.views.decorators.csrf import csrf_exempt

@csrf_exempt
def webhook(request):
    ...
```

---

## Security Rules

- Session IDs must be at least 128 bits (32 bytes) of cryptographically random data.
- Rotate the session ID immediately after login (`req.session.regenerate()` in express-session) to prevent session fixation.
- Always set `HttpOnly`, `Secure`, and `SameSite=Strict` on the session cookie.
- Apply CSRF protection to all state-changing requests (POST/PUT/PATCH/DELETE).
- Set an absolute session timeout (even with sliding expiry) to cap maximum session age.
- Store sessions in Redis with TTL — never in process memory (does not survive restarts or scale-out).
- On privilege escalation (e.g., password change, email change), invalidate the existing session and issue a new one.

---

## Key Rules

- Minimum 256-bit session ID entropy — use `crypto/rand`, `secrets.token_hex`, or `crypto.randomBytes`.
- Regenerate session ID on login to prevent session fixation.
- HttpOnly + Secure + SameSite=Strict cookie attributes always.
- CSRF protection required — double-submit cookie or synchronizer token.
- Immediate revocation by deleting the session key from the store.
- Sliding TTL extended on each authenticated request; absolute timeout for inactivity limits.
