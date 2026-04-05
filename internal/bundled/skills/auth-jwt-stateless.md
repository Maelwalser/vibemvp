# JWT Stateless Auth Skill Guide

## Overview

JSON Web Tokens provide stateless authentication: the server issues a signed token, the client stores and presents it on every request. No session lookup is required — validity is verified by checking the signature and claims.

Use RS256 (asymmetric) when multiple services need to verify tokens independently without sharing a secret. Use HS256 (symmetric) for single-service scenarios where the same process signs and verifies.

---

## Implementation Pattern

### Token Structure

```
Header.Payload.Signature

Header:  { "alg": "RS256", "typ": "JWT" }
Payload: { "sub": "user-id", "email": "...", "roles": ["user"], "iat": 1700000000, "exp": 1700003600 }
```

### Go — Issue Tokens (RS256)

```go
import "github.com/golang-jwt/jwt/v5"

type Claims struct {
    UserID string   `json:"sub"`
    Email  string   `json:"email"`
    Roles  []string `json:"roles"`
    jwt.RegisteredClaims
}

func IssueAccessToken(userID, email string, roles []string, privateKey *rsa.PrivateKey) (string, error) {
    now := time.Now()
    claims := Claims{
        UserID: userID,
        Email:  email,
        Roles:  roles,
        RegisteredClaims: jwt.RegisteredClaims{
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
            Issuer:    "your-service",
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    return token.SignedString(privateKey)
}

func IssueRefreshToken(userID string, privateKey *rsa.PrivateKey) (string, error) {
    now := time.Now()
    claims := jwt.RegisteredClaims{
        Subject:   userID,
        IssuedAt:  jwt.NewNumericDate(now),
        ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
        Issuer:    "your-service",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    return token.SignedString(privateKey)
}
```

### TypeScript — Issue Tokens (HS256)

```typescript
import jwt from 'jsonwebtoken'

const ACCESS_SECRET = process.env.JWT_ACCESS_SECRET!
const REFRESH_SECRET = process.env.JWT_REFRESH_SECRET!

export function issueAccessToken(userID: string, roles: string[]): string {
  return jwt.sign({ sub: userID, roles }, ACCESS_SECRET, {
    algorithm: 'HS256',
    expiresIn: '15m',
    issuer: 'your-service',
  })
}

export function issueRefreshToken(userID: string): string {
  return jwt.sign({ sub: userID }, REFRESH_SECRET, {
    algorithm: 'HS256',
    expiresIn: '7d',
    issuer: 'your-service',
  })
}
```

### Python — Issue Tokens

```python
import jwt
from datetime import datetime, timedelta, timezone

ACCESS_SECRET = os.environ["JWT_ACCESS_SECRET"]
REFRESH_SECRET = os.environ["JWT_REFRESH_SECRET"]

def issue_access_token(user_id: str, roles: list[str]) -> str:
    now = datetime.now(timezone.utc)
    payload = {
        "sub": user_id,
        "roles": roles,
        "iat": now,
        "exp": now + timedelta(minutes=15),
        "iss": "your-service",
    }
    return jwt.encode(payload, ACCESS_SECRET, algorithm="HS256")

def issue_refresh_token(user_id: str) -> str:
    now = datetime.now(timezone.utc)
    payload = {
        "sub": user_id,
        "iat": now,
        "exp": now + timedelta(days=7),
        "iss": "your-service",
    }
    return jwt.encode(payload, REFRESH_SECRET, algorithm="HS256")
```

---

## Token / Session Management

### Access + Refresh Token Pair

- **Access token**: short-lived (15 minutes). Sent on every API request.
- **Refresh token**: long-lived (7–30 days). Used only to obtain a new access token.

### Refresh Token Rotation (Rotating / Sliding Window)

On every refresh:
1. Validate the incoming refresh token.
2. Check it has not been used before (see blacklisting below).
3. Issue a new access token AND a new refresh token.
4. Invalidate the old refresh token (store its `jti` in Redis).

```go
// Rotating refresh in Go
func RotateRefreshToken(oldToken string) (accessToken, refreshToken string, err error) {
    claims, err := verifyRefreshToken(oldToken)
    if err != nil {
        return "", "", fmt.Errorf("invalid refresh token: %w", err)
    }

    jti := claims.ID // unique token ID
    used, err := redisClient.Exists(ctx, "rt:blacklist:"+jti).Result()
    if err != nil || used > 0 {
        return "", "", errors.New("refresh token already used")
    }

    // Blacklist old token for its remaining TTL
    ttl := time.Until(claims.ExpiresAt.Time)
    if err := redisClient.Set(ctx, "rt:blacklist:"+jti, "1", ttl).Err(); err != nil {
        return "", "", fmt.Errorf("blacklist: %w", err)
    }

    newAccess, _ := IssueAccessToken(claims.Subject, claims.Email, claims.Roles, privateKey)
    newRefresh, _ := IssueRefreshToken(claims.Subject, privateKey)
    return newAccess, newRefresh, nil
}
```

### Token Blacklisting with Redis (Logout)

On logout, store the access token's `jti` (JWT ID) in Redis with TTL equal to its remaining lifetime.

```typescript
export async function logout(token: string): Promise<void> {
  const decoded = jwt.verify(token, ACCESS_SECRET) as jwt.JwtPayload
  const remaining = (decoded.exp! - Math.floor(Date.now() / 1000))
  if (remaining > 0) {
    await redis.set(`at:blacklist:${decoded.jti}`, '1', 'EX', remaining)
  }
}
```

Middleware must check the blacklist on every request:

```typescript
export async function jwtMiddleware(req, res, next) {
  const token = extractBearerToken(req) // or read from cookie
  const decoded = jwt.verify(token, ACCESS_SECRET) as jwt.JwtPayload
  const blacklisted = await redis.exists(`at:blacklist:${decoded.jti}`)
  if (blacklisted) return res.status(401).json({ error: 'Token revoked' })
  req.user = decoded
  next()
}
```

### HttpOnly Cookie Storage

Never store tokens in `localStorage`. Set tokens as HttpOnly cookies.

```typescript
res.cookie('access_token', accessToken, {
  httpOnly: true,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'strict',
  maxAge: 15 * 60 * 1000, // 15 minutes in ms
})
res.cookie('refresh_token', refreshToken, {
  httpOnly: true,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'strict',
  path: '/auth/refresh', // scope refresh cookie to refresh endpoint only
  maxAge: 7 * 24 * 60 * 60 * 1000,
})
```

---

## Security Rules

- Always include `jti` (unique ID) in every token to enable targeted revocation.
- Validate `iss` (issuer) and `aud` (audience) claims in middleware.
- Use RS256 for multi-service architectures — never share the private key.
- Rotate the signing key periodically; serve the public key at a JWKS endpoint.
- Store secrets/private keys in environment variables or a secrets manager — never in source code.
- Set `path` on the refresh token cookie to limit scope to the refresh endpoint.
- Log failed verification attempts; alert on anomalous patterns.

---

## Key Rules

- Access token TTL: 15 minutes max.
- Refresh token TTL: 7–30 days; rotate on every use.
- Blacklist revoked tokens in Redis with TTL = token's remaining lifetime.
- HttpOnly + Secure + SameSite=Strict cookies only — no localStorage.
- Include `jti` in every token for selective revocation.
- Reject tokens that fail signature, expiry, issuer, or blacklist checks — never silently ignore.
