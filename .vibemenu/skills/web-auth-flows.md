# Web Authentication Flows Skill Guide

## Overview

This guide covers the complete set of frontend and full-stack authentication patterns: redirect OAuth/OIDC with PKCE, modal login, magic links, passwordless WebAuthn, social-only aggregation, token storage strategies, and silent token refresh.

---

## Redirect OAuth/OIDC with PKCE

The standard flow for user-facing web applications. The user is redirected to the provider and back.

```typescript
// auth.ts — complete PKCE flow
import crypto from 'crypto'

// Step 1: initiate login
export function initiateLogin(): void {
  const verifier = crypto.randomBytes(32).toString('base64url')
  const challenge = crypto.createHash('sha256').update(verifier).digest('base64url')
  const state = crypto.randomBytes(16).toString('hex')

  // Store in sessionStorage (cleared on tab close) — not localStorage
  sessionStorage.setItem('pkce_verifier', verifier)
  sessionStorage.setItem('oauth_state', state)

  const params = new URLSearchParams({
    response_type: 'code',
    client_id: import.meta.env.VITE_OAUTH_CLIENT_ID,
    redirect_uri: `${window.location.origin}/auth/callback`,
    scope: 'openid email profile',
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state,
  })

  window.location.href = `https://provider.example.com/authorize?${params}`
}

// Step 2: handle callback (on the /auth/callback page)
export async function handleCallback(): Promise<void> {
  const params = new URLSearchParams(window.location.search)
  const code = params.get('code')
  const returnedState = params.get('state')
  const storedState = sessionStorage.getItem('oauth_state')
  const verifier = sessionStorage.getItem('pkce_verifier')

  sessionStorage.removeItem('oauth_state')
  sessionStorage.removeItem('pkce_verifier')

  if (!code || returnedState !== storedState) {
    throw new Error('Invalid state — possible CSRF attack')
  }

  // Server-side token exchange (never expose client_secret in browser)
  const res = await fetch('/api/auth/exchange', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code, codeVerifier: verifier }),
    credentials: 'include', // needed for cookie response
  })

  if (!res.ok) throw new Error('Token exchange failed')
  // Server sets HttpOnly cookies; no tokens returned to JS
  window.location.replace('/dashboard')
}
```

---

## Modal Login (Intercept 401, Show Modal, Retry)

```typescript
// api-client.ts — global 401 interceptor
let loginResolve: ((token: string) => void) | null = null

export async function apiRequest(url: string, options: RequestInit = {}): Promise<Response> {
  const res = await fetch(url, { ...options, credentials: 'include' })

  if (res.status === 401) {
    // Show login modal and wait for user to authenticate
    const newToken = await showLoginModal()
    // Retry original request — server will use the new session cookie
    return fetch(url, { ...options, credentials: 'include' })
  }

  return res
}

// Show modal and return a promise that resolves on successful login
function showLoginModal(): Promise<string> {
  return new Promise((resolve) => {
    loginResolve = resolve
    document.dispatchEvent(new CustomEvent('show-login-modal'))
  })
}

// Modal component calls this on successful login
export function onLoginSuccess(): void {
  loginResolve?.('')
  loginResolve = null
}
```

```tsx
// LoginModal.tsx (React)
import { useEffect, useState } from 'react'
import { onLoginSuccess } from './api-client'

export function LoginModal() {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const handler = () => setVisible(true)
    document.addEventListener('show-login-modal', handler)
    return () => document.removeEventListener('show-login-modal', handler)
  }, [])

  async function handleSubmit(email: string, password: string) {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
      credentials: 'include',
    })
    if (res.ok) {
      setVisible(false)
      onLoginSuccess()
    }
  }

  if (!visible) return null
  return <LoginForm onSubmit={handleSubmit} />
}
```

---

## Magic Link

Passwordless login via a signed, short-TTL URL sent to the user's email.

```typescript
// Server — generate magic link
import crypto from 'crypto'

async function sendMagicLink(email: string): Promise<void> {
  const token = crypto.randomBytes(32).toString('hex')
  const expiresAt = new Date(Date.now() + 15 * 60 * 1000) // 15 minutes

  const hash = crypto.createHash('sha256').update(token).digest('hex')
  await db.query(
    `INSERT INTO magic_links (email, token_hash, expires_at, used)
     VALUES ($1, $2, $3, FALSE)
     ON CONFLICT (email) DO UPDATE SET token_hash = $2, expires_at = $3, used = FALSE`,
    [email, hash, expiresAt]
  )

  const link = `${process.env.APP_URL}/auth/magic?token=${token}&email=${encodeURIComponent(email)}`
  await sendEmail({
    to: email,
    subject: 'Your sign-in link',
    text: `Click to sign in (expires in 15 minutes): ${link}`,
  })
}

// Server — verify magic link token
async function verifyMagicLink(token: string, email: string): Promise<string> {
  const hash = crypto.createHash('sha256').update(token).digest('hex')
  const { rows } = await db.query(
    `SELECT id FROM magic_links
      WHERE email = $1 AND token_hash = $2 AND used = FALSE AND expires_at > NOW()`,
    [email, hash]
  )
  if (!rows[0]) throw new Error('Invalid or expired magic link')

  // Mark as used (single-use)
  await db.query(`UPDATE magic_links SET used = TRUE WHERE id = $1`, [rows[0].id])

  // Create or find user and issue session
  const userID = await findOrCreateUser(email)
  return createSession(userID)
}
```

---

## Passwordless via WebAuthn

See `auth-mfa.md` for full ceremony implementation. Frontend trigger:

```typescript
import { startAuthentication } from '@simplewebauthn/browser'

async function passkeyLogin(): Promise<void> {
  // Fetch options from server
  const optionsRes = await fetch('/api/auth/webauthn/options', { credentials: 'include' })
  const options = await optionsRes.json()

  // Browser prompts user for biometric/PIN
  const assertion = await startAuthentication(options)

  // Verify with server
  const verifyRes = await fetch('/api/auth/webauthn/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(assertion),
    credentials: 'include',
  })

  if (!verifyRes.ok) throw new Error('Passkey authentication failed')
  window.location.replace('/dashboard')
}
```

---

## Social-Only Login Aggregation

```typescript
// Aggregate multiple OAuth providers under one user account
type Provider = 'google' | 'github' | 'microsoft'

// On callback — link provider to existing account or create new user
async function handleSocialCallback(
  provider: Provider,
  providerUserID: string,
  email: string
): Promise<string> {
  // 1. Check if this provider account is already linked
  const { rows: linked } = await db.query(
    `SELECT user_id FROM social_accounts WHERE provider = $1 AND provider_user_id = $2`,
    [provider, providerUserID]
  )
  if (linked[0]) return createSession(linked[0].user_id)

  // 2. Check if a user with this email exists
  const { rows: existing } = await db.query(
    `SELECT id FROM users WHERE email = $1`, [email]
  )
  const userID = existing[0]?.id ?? await createUser(email)

  // 3. Link the provider account
  await db.query(
    `INSERT INTO social_accounts (user_id, provider, provider_user_id)
     VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
    [userID, provider, providerUserID]
  )

  return createSession(userID)
}
```

---

## Frontend Token Storage

### HttpOnly Cookie (Recommended)

```typescript
// Server sets cookie — JS never touches the token
res.cookie('access_token', token, {
  httpOnly: true,    // inaccessible to JS
  secure: true,      // HTTPS only
  sameSite: 'strict',
  maxAge: 15 * 60 * 1000,
})

// Client just sends credentials: 'include' and the browser attaches the cookie
fetch('/api/protected', { credentials: 'include' })
```

### In-Memory Storage (SPA with no backend)

```typescript
// Store in module-level variable — survives re-renders, lost on page refresh
let accessToken: string | null = null

export function setToken(token: string): void {
  accessToken = token
}

export function getToken(): string | null {
  return accessToken
}

// On page load, use silent refresh (iframe) or redirect to refresh
```

Never use `localStorage` or `sessionStorage` for tokens — they are accessible to any JS on the page (XSS risk).

---

## Silent Token Refresh

Renew access tokens before they expire without user interaction.

### Cookie-Based (Recommended)

```typescript
// Middleware on every request: check if access token is within 2 minutes of expiry
// and automatically refresh using the HttpOnly refresh_token cookie
export async function refreshIfNeeded(req, res, next) {
  const accessPayload = verifyAccessToken(req.cookies.access_token)
  const expiresIn = accessPayload.exp - Math.floor(Date.now() / 1000)

  if (expiresIn < 120) { // refresh if expiring within 2 minutes
    try {
      const newTokens = await rotateTokens(req.cookies.refresh_token)
      res.cookie('access_token', newTokens.accessToken, { httpOnly: true, secure: true, sameSite: 'strict', maxAge: 15 * 60 * 1000 })
      res.cookie('refresh_token', newTokens.refreshToken, { httpOnly: true, secure: true, sameSite: 'strict', path: '/api/auth/refresh', maxAge: 7 * 24 * 60 * 60 * 1000 })
    } catch {
      // Refresh failed — clear cookies and redirect to login
      res.clearCookie('access_token')
      res.clearCookie('refresh_token')
      return res.status(401).json({ error: 'Session expired' })
    }
  }
  next()
}
```

### In-Memory with Proactive Refresh Timer

```typescript
let refreshTimer: ReturnType<typeof setTimeout> | null = null

function scheduleRefresh(expiresAt: number): void {
  if (refreshTimer) clearTimeout(refreshTimer)
  const msUntilRefresh = (expiresAt * 1000) - Date.now() - 60_000 // refresh 1 min early

  refreshTimer = setTimeout(async () => {
    try {
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        credentials: 'include',
      })
      if (res.ok) {
        const { expiresAt: newExp } = await res.json()
        scheduleRefresh(newExp)
      } else {
        // Session ended — redirect to login
        window.location.href = '/login'
      }
    } catch {
      console.error('Silent refresh failed')
    }
  }, Math.max(msUntilRefresh, 0))
}
```

---

## Security Rules

- PKCE is mandatory for all browser-based OAuth flows — never use implicit flow.
- State parameter is required on every authorization request to prevent CSRF.
- Magic links are single-use and expire in 15 minutes maximum.
- Never store tokens in `localStorage` or `sessionStorage` — use HttpOnly cookies.
- For SPA in-memory storage: implement proactive refresh before token expiry.
- The server must exchange OAuth codes — never expose `client_secret` in browser code.
- Set `credentials: 'include'` on all fetch calls to send HttpOnly cookies cross-origin.

---

## Key Rules

- Redirect OAuth: PKCE (S256) + state parameter + server-side code exchange.
- Modal login: intercept 401, queue original request, retry after re-auth.
- Magic links: SHA-256 hash stored; single-use; 15-minute TTL.
- Social aggregation: link providers to a unified user account by email.
- Token storage: HttpOnly cookie always preferred; in-memory as fallback for SPAs.
- Silent refresh: proactive (timer-based) or reactive (on 401) renewal of access tokens.
