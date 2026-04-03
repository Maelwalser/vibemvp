# Contracts: API Versioning Skill Guide

## Overview

URL path versioning, Accept-Version header, query param versioning, Sunset/Deprecation headers, and version lifecycle management.

## Versioning Strategies

| Strategy | Example | Pros | Cons |
|----------|---------|------|------|
| URL path | `/v1/users` | Visible, cacheable, easy to test | URL changes on version |
| Header | `Version: 2024-01-01` | Clean URLs | Invisible, harder to test |
| Query param | `?api_version=2` | Easy to test | Hard to cache, pollutes URLs |

**Recommendation**: Use URL path versioning as default. Use date-based header versioning (Stripe-style) for APIs with frequent incremental changes.

## URL Path Versioning

```typescript
// Express
import express from 'express';
const app = express();

// v1 router
const v1Router = express.Router();
v1Router.get('/users', v1Controllers.listUsers);
v1Router.post('/users', v1Controllers.createUser);

// v2 router (new features, breaking changes)
const v2Router = express.Router();
v2Router.get('/users', v2Controllers.listUsers);   // new pagination format
v2Router.post('/users', v2Controllers.createUser); // new fields

app.use('/v1', v1Router);
app.use('/v2', v2Router);

// Redirect unversioned to latest (optional)
app.use('/api/users', (_req, res) => res.redirect(301, '/v2/users'));
```

```go
// Go (Fiber)
v1 := app.Group("/v1")
v1.Get("/users", handlers.V1ListUsers)
v1.Post("/users", handlers.V1CreateUser)

v2 := app.Group("/v2")
v2.Get("/users", handlers.V2ListUsers)  // returns cursor-based pagination
v2.Post("/users", handlers.V2CreateUser)
```

```python
# FastAPI
from fastapi import FastAPI, APIRouter

v1_router = APIRouter(prefix="/v1", tags=["v1"])
v2_router = APIRouter(prefix="/v2", tags=["v2"])

@v1_router.get("/users")
async def list_users_v1(): ...

@v2_router.get("/users")
async def list_users_v2(): ...   # new response shape

app = FastAPI()
app.include_router(v1_router)
app.include_router(v2_router)
```

## Accept-Version Header (Date-Based)

```typescript
// Stripe-style: Stripe-Version: 2024-01-15
// Custom header: Version: 2024-01-15

function resolveApiVersion(req: express.Request): string {
  const requested = req.header('Version') || req.header('Accept-Version');
  const supported = ['2024-01-15', '2023-07-01', '2023-01-01'];

  if (!requested) return supported[0]; // default to latest

  // Find exact match or nearest earlier version
  const match = supported.find(v => v <= requested);
  return match || supported[supported.length - 1];
}

app.use((req, res, next) => {
  req.apiVersion = resolveApiVersion(req);
  res.setHeader('API-Version', req.apiVersion);
  next();
});

app.get('/users', (req, res) => {
  if (req.apiVersion >= '2024-01-15') {
    return v2Controller.listUsers(req, res);
  }
  return v1Controller.listUsers(req, res);
});
```

## Query Parameter Versioning (least preferred)

```typescript
// Only use when URL path and header versioning aren't viable
app.get('/users', (req, res) => {
  const version = req.query.api_version || '2';
  if (version === '1') return v1Controller.listUsers(req, res);
  return v2Controller.listUsers(req, res);
});
```

## Deprecation Headers

RFC 8594 `Sunset` header tells clients when the version will be removed.

```typescript
// Middleware: add deprecation headers to v1 responses
app.use('/v1', (req, res, next) => {
  // Sunset header: RFC 8594 — date after which endpoint is removed
  res.setHeader('Sunset', 'Sat, 31 Dec 2024 23:59:59 GMT');
  res.setHeader('Deprecation', 'true');
  res.setHeader(
    'Link',
    '</v2/users>; rel="successor-version", <https://docs.example.com/migration/v1-to-v2>; rel="deprecation"'
  );
  next();
});
```

```go
// Go middleware
func deprecationMiddleware(sunset string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        c.Set("Sunset", sunset)
        c.Set("Deprecation", "true")
        c.Set("Link", `</v2>; rel="successor-version"`)
        return c.Next()
    }
}

v1 := app.Group("/v1", deprecationMiddleware("Sat, 31 Dec 2024 23:59:59 GMT"))
```

## Version Lifecycle

```
v1 released → v2 released → v1 deprecation announced → Sunset header added to v1
→ 6-month window → v1 removed
```

```markdown
### Version Lifecycle Policy

1. **Release** — new version published, documented in changelog.
2. **Deprecation announcement** — blog post + email to API consumers.
3. **Sunset header** — added to all deprecated version responses.
4. **Grace period** — minimum 6 months from deprecation announcement.
5. **Removal** — version endpoint returns 410 Gone with migration instructions.
6. **Support policy** — N-1 version always supported (current + 1 previous major).
```

### 410 Gone Response (after removal)

```typescript
app.all('/v1/*', (_req, res) => {
  res.status(410).json({
    error: 'API_VERSION_REMOVED',
    message: 'API v1 was removed on 2025-01-01. Please migrate to v2.',
    migration_guide: 'https://docs.example.com/migration/v1-to-v2',
    current_version: 'v2',
  });
});
```

## OpenAPI Version Metadata

```yaml
# openapi.yaml — per-version spec files
info:
  title: MyApp API
  version: "2.0.0"
  description: |
    ## Changelog
    ### v2.0.0 (2024-01-15)
    - **BREAKING**: `/users` response now uses cursor-based pagination.
    - Added `role` field to User object.

    ### v1.0.0 (2023-01-01) — DEPRECATED (Sunset: 2024-12-31)
    - Initial release.

x-api-version: "2.0.0"
x-sunset: "2025-12-31"
```

## Changelog Entry Format

```markdown
## [2.0.0] - 2024-01-15

### Breaking Changes
- `GET /users` response now returns cursor-based pagination.
  Before: `{ data: [], total: 100, page: 1 }`
  After:  `{ data: [], pageInfo: { hasNextPage, endCursor } }`
- Removed `username` field from User object (use `name` instead).

### Added
- `role` field on User object (values: `user`, `admin`, `moderator`).
- `GET /users?role=admin` filter parameter.

### Changed
- `POST /users` now returns 201 instead of 200.

### Deprecated
- v1 API (Sunset: 2024-12-31). See migration guide.
```

## Key Rules

- Use URL path versioning (`/v1/`) as the default — it is the most widely understood pattern.
- Always support at least N-1 versions — never remove a version without a minimum 6-month sunset window.
- Add `Sunset` and `Deprecation` headers from the moment you release the successor version.
- Define a single OpenAPI spec per version — never combine versions in one spec.
- Include a changelog entry for every breaking change, even minor ones.
- Return `410 Gone` (not `404`) from removed version endpoints — the difference matters to clients.
- Monitor version usage metrics — check which version each API key is calling before removal.
