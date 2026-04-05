# OAuth2 / OIDC Skill Guide

## Overview

OAuth2 is an authorization framework; OIDC (OpenID Connect) extends it with identity. Use Authorization Code + PKCE for user-facing flows. Use Client Credentials for M2M (service-to-service). Use OIDC ID tokens to establish user identity.

---

## Implementation Pattern

### Authorization Code + PKCE Flow

PKCE (Proof Key for Code Exchange) prevents authorization code interception. Required for public clients (SPAs, mobile) and recommended for all flows.

```typescript
// Step 1: Generate PKCE pair before redirect
import crypto from 'crypto'

function generatePKCE(): { verifier: string; challenge: string } {
  const verifier = crypto.randomBytes(32).toString('base64url')
  const challenge = crypto
    .createHash('sha256')
    .update(verifier)
    .digest('base64url')
  return { verifier, challenge }
}

// Step 2: Build authorization URL
function buildAuthURL(params: {
  authorizationEndpoint: string
  clientID: string
  redirectURI: string
  scopes: string[]
  codeChallenge: string
  state: string
}): string {
  const url = new URL(params.authorizationEndpoint)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('client_id', params.clientID)
  url.searchParams.set('redirect_uri', params.redirectURI)
  url.searchParams.set('scope', params.scopes.join(' '))
  url.searchParams.set('code_challenge', params.codeChallenge)
  url.searchParams.set('code_challenge_method', 'S256')
  url.searchParams.set('state', params.state)
  return url.toString()
}

// Step 3: Exchange code for tokens (server-side)
async function exchangeCode(params: {
  tokenEndpoint: string
  code: string
  codeVerifier: string
  clientID: string
  clientSecret: string
  redirectURI: string
}): Promise<{ access_token: string; refresh_token?: string; id_token?: string }> {
  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    code: params.code,
    code_verifier: params.codeVerifier,
    client_id: params.clientID,
    client_secret: params.clientSecret,
    redirect_uri: params.redirectURI,
  })
  const res = await fetch(params.tokenEndpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  })
  if (!res.ok) throw new Error(`Token exchange failed: ${res.status}`)
  return res.json()
}
```

### State Parameter (CSRF for OAuth)

```typescript
// Generate and store state before redirect
const state = crypto.randomBytes(16).toString('hex')
req.session.oauthState = state

// Verify on callback
if (req.query.state !== req.session.oauthState) {
  return res.status(400).json({ error: 'State mismatch — possible CSRF' })
}
delete req.session.oauthState
```

### Go — Authorization Code + PKCE

```go
import "golang.org/x/oauth2"

var oauthConfig = &oauth2.Config{
    ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
    ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
    RedirectURL:  os.Getenv("OAUTH_REDIRECT_URI"),
    Scopes:       []string{"openid", "email", "profile"},
    Endpoint: oauth2.Endpoint{
        AuthURL:  "https://provider.example.com/oauth/authorize",
        TokenURL: "https://provider.example.com/oauth/token",
    },
}

func StartLogin(w http.ResponseWriter, r *http.Request) {
    verifier := oauth2.GenerateVerifier()
    state := generateState()
    setSessionValues(w, r, verifier, state)
    url := oauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
    http.Redirect(w, r, url, http.StatusFound)
}

func HandleCallback(w http.ResponseWriter, r *http.Request) {
    verifier, state := getSessionValues(r)
    if r.URL.Query().Get("state") != state {
        http.Error(w, "state mismatch", http.StatusBadRequest)
        return
    }
    code := r.URL.Query().Get("code")
    token, err := oauthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(verifier))
    if err != nil {
        http.Error(w, "token exchange failed", http.StatusInternalServerError)
        return
    }
    idToken := token.Extra("id_token").(string)
    _ = validateIDToken(idToken)
}
```

---

## OIDC ID Token Validation

Validate every ID token before trusting its claims.

```typescript
import { createRemoteJWKSet, jwtVerify } from 'jose'

const JWKS = createRemoteJWKSet(
  new URL('https://provider.example.com/.well-known/jwks.json')
)

async function validateIDToken(idToken: string, clientID: string) {
  const { payload } = await jwtVerify(idToken, JWKS, {
    issuer: 'https://provider.example.com',
    audience: clientID,
    clockTolerance: 30, // seconds
  })

  // Required claims
  if (!payload.sub) throw new Error('Missing sub claim')
  if (!payload.iat || !payload.exp) throw new Error('Missing iat/exp claims')
  if (payload.exp < Math.floor(Date.now() / 1000)) throw new Error('ID token expired')

  return payload
}
```

### Userinfo Endpoint

```typescript
async function fetchUserInfo(accessToken: string): Promise<Record<string, unknown>> {
  const res = await fetch('https://provider.example.com/userinfo', {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
  if (!res.ok) throw new Error(`Userinfo failed: ${res.status}`)
  return res.json()
}
```

---

## Refresh Token Flow

```typescript
async function refreshTokens(refreshToken: string): Promise<{
  access_token: string
  refresh_token?: string
}> {
  const body = new URLSearchParams({
    grant_type: 'refresh_token',
    refresh_token: refreshToken,
    client_id: process.env.OAUTH_CLIENT_ID!,
    client_secret: process.env.OAUTH_CLIENT_SECRET!,
  })
  const res = await fetch(process.env.TOKEN_ENDPOINT!, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  })
  if (!res.ok) throw new Error(`Refresh failed: ${res.status}`)
  return res.json()
}
```

---

## Client Credentials (M2M)

No user involvement — service authenticates directly with the provider.

```go
// Go — client credentials
config := &clientcredentials.Config{
    ClientID:     os.Getenv("M2M_CLIENT_ID"),
    ClientSecret: os.Getenv("M2M_CLIENT_SECRET"),
    TokenURL:     "https://provider.example.com/oauth/token",
    Scopes:       []string{"api:read", "api:write"},
}

// TokenSource handles caching and auto-refresh
tokenSource := config.TokenSource(context.Background())

// Use as HTTP transport
httpClient := &http.Client{
    Transport: oauth2.NewClient(context.Background(), tokenSource).Transport,
}
```

### Token Introspection

Validate opaque tokens issued by an authorization server.

```typescript
async function introspectToken(token: string): Promise<{ active: boolean; sub?: string; scope?: string }> {
  const body = new URLSearchParams({ token, token_type_hint: 'access_token' })
  const credentials = Buffer.from(
    `${process.env.CLIENT_ID}:${process.env.CLIENT_SECRET}`
  ).toString('base64')

  const res = await fetch(process.env.INTROSPECTION_ENDPOINT!, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      Authorization: `Basic ${credentials}`,
    },
    body,
  })
  return res.json()
}
```

---

## Security Rules

- Always use PKCE (`S256` method) — plain method is insecure.
- Validate `state` parameter on every callback to prevent CSRF.
- Validate ID token `iss`, `aud`, `exp`, and `iat` before trusting claims.
- Fetch the JWKS from the provider's well-known endpoint; cache with TTL.
- Never expose `client_secret` in browser/mobile code — exchange codes server-side only.
- Store `code_verifier` in the session (not in a cookie the client can read).
- Use short-lived access tokens; refresh in the background before expiry.

---

## Key Rules

- Authorization Code + PKCE for all user-facing flows.
- State parameter required to prevent OAuth CSRF.
- ID token validation: verify signature via JWKS, check iss/aud/exp/iat.
- Userinfo endpoint provides additional claims beyond what's in the ID token.
- Client Credentials for M2M — use an auto-refreshing token source.
- Token introspection for validating opaque tokens from a third-party AS.
