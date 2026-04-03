# RBAC / ABAC Authorization Skill Guide

## Overview

Role-Based Access Control (RBAC) grants permissions based on a user's assigned role. Attribute-Based Access Control (ABAC) evaluates dynamic attributes (user department, resource owner, request context) to make access decisions. Use RBAC for simple, hierarchical permission models; layer ABAC on top for fine-grained, context-aware rules.

---

## RBAC Implementation

### Database Schema

```sql
-- Core RBAC tables
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,   -- 'admin', 'user', 'moderator', etc.
    description TEXT
);

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    TEXT NOT NULL,          -- 'posts', 'users', 'billing'
    action      TEXT NOT NULL,          -- 'read', 'create', 'update', 'delete', 'manage'
    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id       UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Efficient permission lookup view
CREATE VIEW user_permissions AS
SELECT ur.user_id, p.resource, p.action
FROM user_roles ur
JOIN role_permissions rp ON ur.role_id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id;
```

### Predefined Roles

```sql
INSERT INTO roles (name, description) VALUES
  ('owner',       'Full access including ownership transfer'),
  ('admin',       'Full access within tenant'),
  ('superadmin',  'Cross-tenant administrative access'),
  ('manager',     'Manage resources and team members'),
  ('moderator',   'Content moderation and user management'),
  ('editor',      'Create and edit content'),
  ('auditor',     'Read-only access with audit log visibility'),
  ('user',        'Standard authenticated user access'),
  ('viewer',      'Read-only access to shared content');
```

### Go — RBAC Middleware

```go
type Permission struct {
    Resource string
    Action   string
}

func RequirePermission(db *pgxpool.Pool, resource, action string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        userID := c.Locals("user_id").(string)

        var count int
        err := db.QueryRow(c.Context(),
            `SELECT COUNT(*) FROM user_permissions
              WHERE user_id = $1 AND resource = $2 AND action = $3`,
            userID, resource, action,
        ).Scan(&count)
        if err != nil {
            return fiber.ErrInternalServerError
        }
        if count == 0 {
            return fiber.ErrForbidden
        }
        return c.Next()
    }
}

// Usage
app.Delete("/posts/:id", RequirePermission(db, "posts", "delete"), deletePostHandler)
app.Get("/admin/users", RequirePermission(db, "users", "manage"), listUsersHandler)
```

### TypeScript — RBAC Middleware

```typescript
function requirePermission(resource: string, action: string) {
  return async (req, res, next) => {
    const userID = req.user?.id
    if (!userID) return res.status(401).json({ error: 'Unauthenticated' })

    const { rows } = await db.query(
      `SELECT 1 FROM user_permissions
        WHERE user_id = $1 AND resource = $2 AND action = $3`,
      [userID, resource, action]
    )
    if (rows.length === 0) {
      return res.status(403).json({ error: 'Insufficient permissions' })
    }
    next()
  }
}

// Routes
router.delete('/posts/:id', requirePermission('posts', 'delete'), deletePost)
router.post('/users', requirePermission('users', 'create'), createUser)
```

### Python — RBAC Decorator (FastAPI)

```python
from functools import wraps
from fastapi import HTTPException, Depends

def require_permission(resource: str, action: str):
    async def dependency(user: User = Depends(get_current_user), db: AsyncSession = Depends(get_db)):
        result = await db.execute(
            select(func.count()).select_from(UserPermission).where(
                UserPermission.user_id == user.id,
                UserPermission.resource == resource,
                UserPermission.action == action,
            )
        )
        if result.scalar() == 0:
            raise HTTPException(status_code=403, detail="Insufficient permissions")
    return Depends(dependency)

# Usage
@router.delete("/posts/{post_id}", dependencies=[require_permission("posts", "delete")])
async def delete_post(post_id: str):
    ...
```

### Permission Cache (Redis)

```go
// Cache user permissions to avoid a DB hit per request
func GetUserPermissions(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, userID string) ([]Permission, error) {
    key := "perms:" + userID
    cached, err := rdb.Get(ctx, key).Bytes()
    if err == nil {
        var perms []Permission
        _ = json.Unmarshal(cached, &perms)
        return perms, nil
    }

    rows, err := db.Query(ctx,
        `SELECT resource, action FROM user_permissions WHERE user_id = $1`, userID)
    if err != nil {
        return nil, fmt.Errorf("fetch permissions: %w", err)
    }
    defer rows.Close()

    var perms []Permission
    for rows.Next() {
        var p Permission
        _ = rows.Scan(&p.Resource, &p.Action)
        perms = append(perms, p)
    }

    data, _ := json.Marshal(perms)
    rdb.Set(ctx, key, data, 5*time.Minute) // 5-minute TTL
    return perms, nil
}
```

---

## ABAC Implementation

ABAC evaluates policies against attributes of the subject (user), resource, and environment.

### Policy Rule Structure

```go
type ABACPolicy struct {
    Effect    string            // "allow" or "deny"
    Resource  string            // "posts", "documents"
    Action    string            // "update", "delete"
    Condition func(user User, resource map[string]any, env map[string]any) bool
}

// Example policies
var policies = []ABACPolicy{
    {
        Effect:   "allow",
        Resource: "posts",
        Action:   "update",
        Condition: func(user User, resource map[string]any, env map[string]any) bool {
            // User can update their own posts
            return resource["owner_id"] == user.ID
        },
    },
    {
        Effect:   "allow",
        Resource: "documents",
        Action:   "read",
        Condition: func(user User, resource map[string]any, env map[string]any) bool {
            // User must belong to the same department as the document
            return resource["department"] == user.Department
        },
    },
    {
        Effect:   "allow",
        Resource: "reports",
        Action:   "export",
        Condition: func(user User, resource map[string]any, env map[string]any) bool {
            // Only during business hours and only for auditors
            hour := time.Now().Hour()
            return user.HasRole("auditor") && hour >= 9 && hour < 18
        },
    },
}

func Evaluate(user User, resource, action string, resourceAttrs map[string]any) bool {
    env := map[string]any{
        "time": time.Now(),
        "ip":   user.LastIP,
    }
    for _, policy := range policies {
        if policy.Resource == resource && policy.Action == action {
            if policy.Condition(user, resourceAttrs, env) {
                return policy.Effect == "allow"
            }
        }
    }
    return false // deny by default
}
```

### TypeScript ABAC

```typescript
interface PolicyContext {
  user: { id: string; roles: string[]; department?: string }
  resource: Record<string, unknown>
  env: { now: Date; ip?: string }
}

type PolicyFn = (ctx: PolicyContext) => boolean

const policies: Record<string, PolicyFn> = {
  'posts:update': ({ user, resource }) => resource['ownerID'] === user.id,
  'posts:delete': ({ user }) => user.roles.includes('admin') || user.roles.includes('moderator'),
  'documents:read': ({ user, resource }) => resource['department'] === user.department,
}

function canDo(user: PolicyContext['user'], resource: string, action: string, resourceAttrs: Record<string, unknown>): boolean {
  const key = `${resource}:${action}`
  const policy = policies[key]
  if (!policy) return false  // deny unknown combinations
  return policy({ user, resource: resourceAttrs, env: { now: new Date() } })
}
```

---

## Security Rules

- Default to DENY — only grant access when an explicit allow rule matches.
- Invalidate the permissions cache immediately on role change or permission update.
- Log all authorization decisions for sensitive resources (especially denies).
- Separate RBAC from authentication — permissions are evaluated after identity is confirmed.
- For ABAC: fail closed on missing attributes (treat `null` owner as deny, not allow).
- Never derive permissions solely from user-supplied input — always re-fetch from the authority source.

---

## Key Rules

- RBAC tables: `roles`, `permissions`, `role_permissions`, `user_roles`.
- Predefined roles: owner / admin / superadmin / manager / moderator / editor / auditor / user / viewer.
- Middleware checks: user_permissions view, then `fiber.ErrForbidden` (403) on miss.
- Cache permissions in Redis with 5-minute TTL; invalidate on role/permission change.
- ABAC: evaluate condition functions against subject + resource + environment attributes.
- Deny by default — explicit allow rules required for access.
