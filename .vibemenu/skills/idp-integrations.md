# Identity Provider Integrations Skill Guide

## Overview

Identity providers (IdPs) handle the complexity of secure authentication, credential storage, and user management. Integrate with them via their SDKs rather than building from scratch. This guide covers Auth0, Clerk, Supabase Auth, Keycloak, and AWS Cognito.

---

## Auth0

### SDK Setup (React + Node.js)

```typescript
// React SPA — @auth0/auth0-react
import { Auth0Provider } from '@auth0/auth0-react'

function App() {
  return (
    <Auth0Provider
      domain={process.env.REACT_APP_AUTH0_DOMAIN!}
      clientId={process.env.REACT_APP_AUTH0_CLIENT_ID!}
      authorizationParams={{
        redirect_uri: window.location.origin,
        audience: process.env.REACT_APP_AUTH0_AUDIENCE,
        scope: 'openid profile email',
      }}
    >
      <Router />
    </Auth0Provider>
  )
}

// Login and token retrieval
import { useAuth0 } from '@auth0/auth0-react'

function LoginButton() {
  const { loginWithRedirect, isAuthenticated, getAccessTokenSilently } = useAuth0()

  async function callAPI() {
    const token = await getAccessTokenSilently()
    await fetch('/api/protected', {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  return isAuthenticated
    ? <button onClick={callAPI}>Call API</button>
    : <button onClick={() => loginWithRedirect()}>Login</button>
}
```

```typescript
// Node.js API — validate Auth0 JWT
import { auth } from 'express-oauth2-jwt-bearer'

const checkJwt = auth({
  audience: process.env.AUTH0_AUDIENCE,
  issuerBaseURL: `https://${process.env.AUTH0_DOMAIN}`,
  tokenSigningAlg: 'RS256',
})

app.get('/api/protected', checkJwt, (req, res) => {
  res.json({ user: req.auth.payload })
})
```

### User Sync to Local DB

```typescript
// Auth0 Action (Post Login trigger) — sync user to your DB
exports.onExecutePostLogin = async (event, api) => {
  const namespace = 'https://yourapp.com'
  // Set custom claims
  api.idToken.setCustomClaim(`${namespace}/roles`, event.authorization?.roles ?? [])
  api.accessToken.setCustomClaim(`${namespace}/roles`, event.authorization?.roles ?? [])
  // Sync to DB via Management API or direct HTTP call to your webhook
}
```

---

## Clerk

### Setup (Next.js App Router)

```typescript
// app/layout.tsx
import { ClerkProvider } from '@clerk/nextjs'

export default function RootLayout({ children }) {
  return (
    <ClerkProvider>
      <html><body>{children}</body></html>
    </ClerkProvider>
  )
}

// middleware.ts — protect routes
import { clerkMiddleware, createRouteMatcher } from '@clerk/nextjs/server'

const isProtected = createRouteMatcher(['/dashboard(.*)'])

export default clerkMiddleware((auth, req) => {
  if (isProtected(req)) auth().protect()
})
```

```typescript
// Server component
import { auth, currentUser } from '@clerk/nextjs/server'

export default async function Dashboard() {
  const { userId } = auth()
  const user = await currentUser()
  return <div>Hello {user?.firstName}</div>
}

// API route
import { auth } from '@clerk/nextjs/server'

export async function GET() {
  const { userId } = auth()
  if (!userId) return Response.json({ error: 'Unauthorized' }, { status: 401 })
  return Response.json({ userId })
}
```

### User Hook (Client)

```typescript
import { useUser } from '@clerk/nextjs'

function Profile() {
  const { isLoaded, isSignedIn, user } = useUser()
  if (!isLoaded || !isSignedIn) return null
  return <div>{user.emailAddresses[0].emailAddress}</div>
}
```

---

## Supabase Auth

### Setup

```typescript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  process.env.SUPABASE_URL!,
  process.env.SUPABASE_ANON_KEY!
)

// Sign up
const { data, error } = await supabase.auth.signUp({
  email: 'user@example.com',
  password: 'secure-password',
})

// Sign in
const { data, error } = await supabase.auth.signInWithPassword({
  email: 'user@example.com',
  password: 'secure-password',
})

// OAuth sign in (GitHub, Google, etc.)
await supabase.auth.signInWithOAuth({ provider: 'github' })

// Get session
const { data: { session } } = await supabase.auth.getSession()

// Sign out
await supabase.auth.signOut()
```

### Server-Side Session Management (Next.js)

```typescript
import { createServerClient } from '@supabase/ssr'
import { cookies } from 'next/headers'

export function createSupabaseServerClient() {
  return createServerClient(
    process.env.SUPABASE_URL!,
    process.env.SUPABASE_ANON_KEY!,
    { cookies: { getAll: () => cookies().getAll() } }
  )
}

// In a Server Action or Route Handler
const supabase = createSupabaseServerClient()
const { data: { user } } = await supabase.auth.getUser()
```

### Row-Level Security with Auth

```sql
-- Users can only access their own rows
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;

CREATE POLICY "users_own_data" ON profiles
  USING (auth.uid() = user_id);

CREATE POLICY "users_insert_own" ON profiles
  FOR INSERT WITH CHECK (auth.uid() = user_id);
```

---

## Keycloak

### Realm and Client Setup

```bash
# Create realm via Admin CLI
kcadm.sh create realms -s realm=myrealm -s enabled=true

# Create client
kcadm.sh create clients -r myrealm \
  -s clientId=myapp \
  -s publicClient=false \
  -s 'redirectUris=["https://app.example.com/callback"]' \
  -s 'webOrigins=["https://app.example.com"]'
```

```typescript
// Node.js — Keycloak adapter
import Keycloak from 'keycloak-connect'
import session from 'express-session'

const memoryStore = new session.MemoryStore()
app.use(session({ secret: process.env.SESSION_SECRET!, store: memoryStore }))

const keycloak = new Keycloak({ store: memoryStore }, {
  realm: 'myrealm',
  'auth-server-url': 'https://keycloak.example.com',
  'resource': 'myapp',
  'credentials': { secret: process.env.KEYCLOAK_CLIENT_SECRET },
})

app.use(keycloak.middleware())
app.get('/protected', keycloak.protect('realm:user'), (req, res) => {
  res.json({ user: req.kauth.grant.access_token.content })
})
```

### Role Mapping

```typescript
// Extract roles from Keycloak access token
function extractRoles(token: any): string[] {
  const realmRoles = token.realm_access?.roles ?? []
  const clientRoles = token.resource_access?.myapp?.roles ?? []
  return [...realmRoles, ...clientRoles]
}
```

---

## AWS Cognito

### UserPool + Hosted UI

```typescript
// Amplify SDK (frontend)
import { Amplify } from 'aws-amplify'
import { signIn, signOut, getCurrentUser, fetchAuthSession } from 'aws-amplify/auth'

Amplify.configure({
  Auth: {
    Cognito: {
      userPoolId: process.env.COGNITO_USER_POOL_ID!,
      userPoolClientId: process.env.COGNITO_CLIENT_ID!,
      loginWith: {
        oauth: {
          domain: process.env.COGNITO_DOMAIN!,
          scopes: ['openid', 'email', 'profile'],
          redirectSignIn: ['https://app.example.com/callback'],
          redirectSignOut: ['https://app.example.com/'],
          responseType: 'code',
        }
      }
    }
  }
})

// Sign in with hosted UI
await signIn({ provider: { custom: 'HostedUI' } })

// Get current user
const user = await getCurrentUser()

// Get tokens
const { tokens } = await fetchAuthSession()
const accessToken = tokens?.accessToken.toString()
```

```go
// Go — validate Cognito JWT
import "github.com/lestrrat-go/jwx/v2/jwk"
import "github.com/lestrrat-go/jwx/v2/jwt"

func validateCognitoToken(tokenStr string) (jwt.Token, error) {
    keySetURL := fmt.Sprintf(
        "https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json",
        os.Getenv("AWS_REGION"),
        os.Getenv("COGNITO_USER_POOL_ID"),
    )
    keySet, err := jwk.Fetch(context.Background(), keySetURL)
    if err != nil {
        return nil, fmt.Errorf("fetch JWKS: %w", err)
    }
    return jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet), jwt.WithValidate(true))
}
```

---

## Security Rules

- Never store client secrets in frontend code — exchange tokens server-side only.
- Always validate ID tokens and access tokens on the server before trusting claims.
- Use environment variables for all IdP credentials (domain, clientId, secret, etc.).
- Sync only necessary user attributes to your local database on first login.
- Implement webhooks / post-login actions to propagate role changes promptly.
- For Supabase: enable Row Level Security on every table that contains user data.

---

## Key Rules

- Auth0: `getAccessTokenSilently()` client-side; `express-oauth2-jwt-bearer` server-side.
- Clerk: `auth()` in server components; `useUser()` in client components.
- Supabase Auth: `@supabase/ssr` for server-side session; RLS enforces data isolation.
- Keycloak: adapter middleware + `keycloak.protect('role')` per route.
- Cognito: Amplify SDK for frontend; JWKS-based JWT validation on backend.
- All IdPs: validate tokens server-side on every request — never trust client-provided identity without verification.
