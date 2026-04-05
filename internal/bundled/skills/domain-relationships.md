# Domain Relationships Skill Guide

## Overview

Relational modeling patterns for One-to-One, One-to-Many, and Many-to-Many relationships, FK naming conventions, cascade behaviors, and join table design for M2M with extra attributes.

---

## One-to-One

Use when each row in table A corresponds to exactly one row in table B.

### Option A — Shared Primary Key

The child table's PK is also a FK to the parent. Enforces strict 1:1.

```sql
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_profiles (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio        TEXT,
    avatar_url TEXT,
    website    TEXT
);
```

### Option B — Unique Foreign Key

The child table has its own PK plus a unique FK. Allows the child to exist independently.

```sql
CREATE TABLE user_settings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    theme      TEXT NOT NULL DEFAULT 'light',
    locale     TEXT NOT NULL DEFAULT 'en'
);
```

---

## One-to-Many

Place the foreign key on the "many" side. This is the most common relationship.

```sql
CREATE TABLE organizations (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    email           TEXT NOT NULL UNIQUE
);

-- Index the FK column for efficient joins and lookups
CREATE INDEX users_organization_id_idx ON users (organization_id);
```

---

## Many-to-Many

Use a join table. Two main options:

### Option A — Composite PK (simple pivot)

```sql
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
```

### Option B — Surrogate PK + Unique Constraint (preferred when extra attributes needed)

```sql
CREATE TABLE project_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    role        TEXT        NOT NULL DEFAULT 'member',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invited_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (project_id, user_id)
);

CREATE INDEX project_members_project_id_idx ON project_members (project_id);
CREATE INDEX project_members_user_id_idx    ON project_members (user_id);
```

---

## Cascade Behaviors

| Behavior | SQL | Effect |
|----------|-----|--------|
| `CASCADE` | `ON DELETE CASCADE` | Delete child rows when parent is deleted |
| `SET NULL` | `ON DELETE SET NULL` | Nullify FK on child rows when parent is deleted |
| `SET DEFAULT` | `ON DELETE SET DEFAULT` | Set FK to its default value when parent is deleted |
| `RESTRICT` | `ON DELETE RESTRICT` | Prevent parent delete if any child row exists |
| `NO ACTION` | `ON DELETE NO ACTION` | Same as RESTRICT but deferred (checked at end of transaction) |

### Choosing the Right Cascade

```
User deleted → their sessions should be deleted     → CASCADE
User deleted → their posts should remain (orphaned) → SET NULL (author_id nullable)
Category deleted → prevent if products reference it → RESTRICT
Order deleted → line items must also be deleted     → CASCADE
Team member removed → their tasks stay, unassigned  → SET NULL
```

---

## FK Naming Convention

```sql
-- Pattern: {referencing_table}_{referenced_table}_fk
-- Column:  {referenced_table_singular}_id

ALTER TABLE orders
    ADD CONSTRAINT orders_users_fk
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_orders_fk
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE;

ALTER TABLE project_members
    ADD CONSTRAINT project_members_projects_fk
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    ADD CONSTRAINT project_members_users_fk
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

---

## M2M Join Table with Additional Attributes

When the relationship itself carries data, use Option B with a surrogate PK.

```sql
-- Team memberships with extra metadata
CREATE TABLE team_members (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id        UUID        NOT NULL REFERENCES teams(id)    ON DELETE CASCADE,
    user_id        UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    -- Relationship attributes
    role           TEXT        NOT NULL DEFAULT 'member'
                               CHECK (role IN ('owner','admin','member','viewer')),
    permissions    TEXT[]      NOT NULL DEFAULT '{}',
    joined_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ,
    invited_by_id  UUID        REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (team_id, user_id)
);

-- Supporting indexes
CREATE INDEX team_members_team_id_idx ON team_members (team_id);
CREATE INDEX team_members_user_id_idx ON team_members (user_id);
CREATE INDEX team_members_expires_at_idx ON team_members (expires_at)
    WHERE expires_at IS NOT NULL;
```

---

## Relationship-Aware DTO Generation

When serializing relationships in API responses, choose the right projection:

```typescript
// Embed nested objects for 1:1 and 1:N (avoid N+1 — always eager-load)
type UserResponse = {
    id: string;
    email: string;
    profile: UserProfileResponse | null;   // 1:1 embedded
    roles: RoleResponse[];                  // M:M embedded (limit to IDs if list is large)
};

// For M:N, expose IDs by default; provide a separate endpoint for full objects
type ProjectMemberResponse = {
    userId: string;
    role: string;
    joinedAt: string;  // ISO-8601
};
```

---

## Key Rules

- Always index FK columns — unindexed FKs cause full table scans on JOIN and cascading operations.
- Prefer `ON DELETE CASCADE` for composition relationships (child cannot exist without parent).
- Prefer `ON DELETE RESTRICT` for reference relationships (child can exist independently; prevent orphan loss).
- Use `ON DELETE SET NULL` only when the FK column is nullable and a null parent is a valid state.
- Join tables for M2M with extra attributes must use a surrogate PK plus a unique constraint on `(a_id, b_id)`.
- Never use `ON DELETE NO ACTION` unless you need deferred constraint checking within a transaction.
