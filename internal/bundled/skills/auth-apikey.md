# API Key Auth Skill Guide

## Overview

API keys authenticate machine-to-machine (M2M) requests and developer integrations. Keys are long-lived bearer credentials — treat them like passwords: generate them securely, store only hashes, and support rotation with a grace period.

---

## Implementation Pattern

### Key Generation

Generate 32 bytes (256 bits) of cryptographic random data, then encode as hex or base64url.

```go
// Go — generate API key
import "crypto/rand"
import "encoding/hex"

func GenerateAPIKey() (raw, prefix string, err error) {
    b := make([]byte, 32)
    if _, err = rand.Read(b); err != nil {
        return "", "", fmt.Errorf("generate api key: %w", err)
    }
    raw = hex.EncodeToString(b)  // 64-char hex, 256 bits entropy
    prefix = raw[:8]             // expose only first 8 chars for lookup/display
    return raw, prefix, nil
}
```

```python
# Python
import secrets

def generate_api_key() -> tuple[str, str]:
    raw = secrets.token_hex(32)   # 64-char hex
    prefix = raw[:8]              # for display / lookup hint
    return raw, prefix
```

```typescript
// Node.js
import crypto from 'crypto'

function generateAPIKey(): { raw: string; prefix: string } {
  const raw = crypto.randomBytes(32).toString('hex')
  return { raw, prefix: raw.slice(0, 8) }
}
```

### Hashed Storage

Never store the raw key. Store a SHA-256 hash (with a per-key salt) or use bcrypt.

```go
// Go — SHA-256 + stored salt
import "crypto/sha256"

func HashAPIKey(raw, salt string) string {
    h := sha256.Sum256([]byte(raw + salt))
    return hex.EncodeToString(h[:])
}

// On creation:
raw, prefix, _ := GenerateAPIKey()
salt := hex.EncodeToString(randomBytes(16))
hash := HashAPIKey(raw, salt)
// Store: prefix, salt, hash, scopes, owner_id, created_at, expires_at
// Return raw to user ONCE — never again
```

```typescript
// Node.js — bcrypt
import bcrypt from 'bcrypt'

const BCRYPT_ROUNDS = 12

async function hashAPIKey(raw: string): Promise<string> {
  return bcrypt.hash(raw, BCRYPT_ROUNDS)
}

async function verifyAPIKey(raw: string, storedHash: string): Promise<boolean> {
  return bcrypt.compare(raw, storedHash)
}
```

### Database Schema

```sql
CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    prefix      CHAR(8) NOT NULL,        -- displayed in UI for key identification
    key_hash    TEXT NOT NULL,
    key_salt    TEXT,                    -- null if using bcrypt (salt is embedded)
    scopes      TEXT[] NOT NULL DEFAULT '{}',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    last_used   TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX ON api_keys (prefix) WHERE is_active = TRUE;
CREATE INDEX ON api_keys (owner_id);
```

---

## Token / Session Management

### Request Authentication Middleware

```go
// Go — Fiber middleware
func APIKeyMiddleware(db *pgxpool.Pool) fiber.Handler {
    return func(c *fiber.Ctx) error {
        authHeader := c.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            return fiber.ErrUnauthorized
        }
        raw := strings.TrimPrefix(authHeader, "Bearer ")
        if len(raw) < 8 {
            return fiber.ErrUnauthorized
        }

        prefix := raw[:8]
        row := db.QueryRow(c.Context(),
            `SELECT id, key_hash, key_salt, scopes, owner_id, expires_at
               FROM api_keys
              WHERE prefix = $1 AND is_active = TRUE AND (expires_at IS NULL OR expires_at > NOW())`,
            prefix,
        )

        var keyID, ownerID, keyHash, keySalt string
        var scopes []string
        var expiresAt *time.Time
        if err := row.Scan(&keyID, &keyHash, &keySalt, &scopes, &ownerID, &expiresAt); err != nil {
            return fiber.ErrUnauthorized
        }

        if HashAPIKey(raw, keySalt) != keyHash {
            return fiber.ErrUnauthorized
        }

        // Update last_used asynchronously — don't block the request
        go func() {
            db.Exec(context.Background(),
                `UPDATE api_keys SET last_used = NOW() WHERE id = $1`, keyID)
        }()

        c.Locals("api_key_id", keyID)
        c.Locals("owner_id", ownerID)
        c.Locals("scopes", scopes)
        return c.Next()
    }
}
```

### Scope / Permission Enforcement

```typescript
// Scope check middleware
function requireScope(scope: string) {
  return (req, res, next) => {
    const scopes: string[] = req.apiKey?.scopes ?? []
    if (!scopes.includes(scope) && !scopes.includes('*')) {
      return res.status(403).json({ error: `Missing required scope: ${scope}` })
    }
    next()
  }
}

// Usage
router.delete('/resources/:id', requireScope('resources:delete'), deleteHandler)
```

Predefined scopes:
```
read            read-only access to all resources
write           create and update (no delete)
admin           full access including delete and management
resources:read  scoped to a specific resource type
webhooks:write  manage webhook endpoints
```

### Key Rotation (Issue New + Grace Period + Revoke Old)

```typescript
async function rotateAPIKey(keyID: string, ownerID: string): Promise<{ raw: string }> {
  const { raw, prefix } = generateAPIKey()
  const hash = await hashAPIKey(raw)

  await db.transaction(async (tx) => {
    // 1. Fetch old key scopes
    const old = await tx.query(
      'SELECT scopes, name FROM api_keys WHERE id = $1 AND owner_id = $2',
      [keyID, ownerID]
    )
    if (!old.rows[0]) throw new Error('Key not found')

    // 2. Create new key
    await tx.query(
      `INSERT INTO api_keys (owner_id, name, prefix, key_hash, scopes)
       VALUES ($1, $2, $3, $4, $5)`,
      [ownerID, old.rows[0].name + ' (rotated)', prefix, hash, old.rows[0].scopes]
    )

    // 3. Schedule old key revocation after grace period (24h)
    await tx.query(
      `UPDATE api_keys SET expires_at = NOW() + INTERVAL '24 hours' WHERE id = $1`,
      [keyID]
    )
  })

  return { raw } // return ONCE to the caller
}
```

---

## Security Rules

- Show the raw key exactly once (on creation/rotation). Never store or display it again.
- Always hash before storing: SHA-256+salt or bcrypt (rounds >= 12).
- Rate-limit per API key at the gateway or middleware level (e.g., 1000 req/min).
- Log all key usage with timestamp and endpoint, but never log the raw key value.
- Support immediate revocation (`is_active = FALSE`) and scheduled expiry (`expires_at`).
- Enforce scopes — a key without a required scope must receive HTTP 403.
- On rotation, keep the old key valid for a grace period (24h) to allow zero-downtime migration.

---

## Key Rules

- 32 bytes (256 bits) of `crypto/rand` / `secrets.token_hex` entropy minimum.
- Store prefix (first 8 chars) in plain text for lookup; store hash+salt for verification.
- `Authorization: Bearer <key>` header — never in URL query params (logged by proxies).
- Rate limit per key; log last_used timestamp; alert on sudden usage spikes.
- Scopes define what a key can do — enforce them in middleware.
- Rotation: issue new key, set old key expires_at = NOW() + grace period.
