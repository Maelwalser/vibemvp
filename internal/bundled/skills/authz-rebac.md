# ReBAC (Relationship-Based Access Control) Skill Guide

## Overview

Relationship-Based Access Control (ReBAC) makes authorization decisions based on the graph of relationships between users and resources. Rather than assigning roles to users globally, you store tuples like `user:alice#owner@document:report` and answer questions like "can alice edit document:report?" by traversing the relationship graph.

This model (inspired by Google Zanzibar) scales to fine-grained, per-object permissions. SpiceDB is the leading open-source Zanzibar implementation.

---

## Core Concepts

### Tuple Structure

```
subject_type : subject_id # relation @ object_type : object_id

Examples:
user:alice   #owner    @document:q3-report
user:bob     #viewer   @document:q3-report
group:eng    #member   @team:platform
user:carol   #member   @group:eng
```

Tuples are the atomic unit of permission. The check API traverses the graph to determine effective access.

### Schema Definition (SpiceDB)

```zed
// schema.zed
definition user {}

definition group {
    relation member: user
}

definition team {
    relation member: user | group#member
}

definition document {
    relation owner:  user
    relation editor: user | team#member
    relation viewer: user | team#member | group#member

    permission view   = viewer + editor + owner
    permission edit   = editor + owner
    permission delete = owner
}

definition folder {
    relation owner:  user
    relation viewer: user

    permission view   = viewer + owner
}

// Hierarchical: document inherits folder visibility
definition document_in_folder {
    relation folder:  folder
    relation owner:   user
    relation editor:  user

    permission view   = editor + owner + folder->view
    permission edit   = editor + owner
}
```

---

## SpiceDB Integration

### SpiceDB Client Setup (Go)

```go
import (
    v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
    "github.com/authzed/authzed-go/v1"
    "github.com/authzed/grpcutil"
    "google.golang.org/grpc"
)

func NewSpiceDBClient() (*authzed.Client, error) {
    return authzed.NewClient(
        os.Getenv("SPICEDB_ENDPOINT"),
        grpcutil.WithBearerToken(os.Getenv("SPICEDB_TOKEN")),
        grpc.WithTransportCredentials(insecure.NewCredentials()), // use TLS in prod
    )
}
```

### Write Relationship Tuple

```go
func WriteRelationship(ctx context.Context, client *authzed.Client,
    subjectType, subjectID, relation, objectType, objectID string) error {

    _, err := client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
        Updates: []*v1.RelationshipUpdate{
            {
                Operation: v1.RelationshipUpdate_OPERATION_CREATE,
                Relationship: &v1.Relationship{
                    Resource: &v1.ObjectReference{
                        ObjectType: objectType,
                        ObjectId:   objectID,
                    },
                    Relation: relation,
                    Subject: &v1.SubjectReference{
                        Object: &v1.ObjectReference{
                            ObjectType: subjectType,
                            ObjectId:   subjectID,
                        },
                    },
                },
            },
        },
    })
    return err
}

// Example: alice owns document:q3-report
err := WriteRelationship(ctx, client, "user", "alice", "owner", "document", "q3-report")
```

### Check Permission

```go
func CanDo(ctx context.Context, client *authzed.Client,
    subjectType, subjectID, permission, objectType, objectID string) (bool, error) {

    resp, err := client.CheckPermission(ctx, &v1.CheckPermissionRequest{
        Resource: &v1.ObjectReference{
            ObjectType: objectType,
            ObjectId:   objectID,
        },
        Permission: permission,
        Subject: &v1.SubjectReference{
            Object: &v1.ObjectReference{
                ObjectType: subjectType,
                ObjectId:   subjectID,
            },
        },
    })
    if err != nil {
        return false, fmt.Errorf("check permission: %w", err)
    }
    return resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, nil
}

// Usage
allowed, err := CanDo(ctx, client, "user", "alice", "edit", "document", "q3-report")
if !allowed {
    return fiber.ErrForbidden
}
```

### TypeScript — SpiceDB Client

```typescript
import { v1 } from '@authzed/authzed-node'

const client = v1.NewClient(
  process.env.SPICEDB_TOKEN!,
  process.env.SPICEDB_ENDPOINT!,
  v1.ClientSecurity.INSECURE_LOCALHOST_ALLOWED // use SECURE in prod
)

async function checkPermission(
  userID: string,
  permission: string,
  objectType: string,
  objectID: string
): Promise<boolean> {
  const { permissionship } = await client.checkPermission({
    resource: { objectType, objectId: objectID },
    permission,
    subject: {
      object: { objectType: 'user', objectId: userID },
    },
  })
  return permissionship === v1.CheckPermissionResponse_Permissionship.HAS_PERMISSION
}
```

---

## Hierarchical Ownership Patterns

### Team → Document Inheritance

```
Relationship tuples:
user:carol   #member  @team:platform
team:platform #editor @document:arch-doc

Check: can carol edit document:arch-doc?
→ carol → member → team:platform → editor → document:arch-doc  ✓
```

### Group Membership Chain

```go
// Add carol to group:eng → automatically grants all group:eng permissions
WriteRelationship(ctx, client, "user", "carol", "member", "group", "eng")

// Grant group:eng viewer access to folder:shared
WriteRelationship(ctx, client, "group", "eng#member", "viewer", "folder", "shared")

// carol can now view all documents in folder:shared through:
// carol → group:eng#member → folder:shared#viewer → document_in_folder:*#view
```

### Delete Relationship (Revoke Access)

```go
func DeleteRelationship(ctx context.Context, client *authzed.Client,
    subjectType, subjectID, relation, objectType, objectID string) error {

    _, err := client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
        Updates: []*v1.RelationshipUpdate{
            {
                Operation: v1.RelationshipUpdate_OPERATION_DELETE,
                Relationship: &v1.Relationship{
                    Resource: &v1.ObjectReference{
                        ObjectType: objectType, ObjectId: objectID,
                    },
                    Relation: relation,
                    Subject: &v1.SubjectReference{
                        Object: &v1.ObjectReference{
                            ObjectType: subjectType, ObjectId: subjectID,
                        },
                    },
                },
            },
        },
    })
    return err
}
```

---

## Tuple Storage (Without SpiceDB)

For simpler use cases, store tuples in PostgreSQL and implement a check API yourself.

```sql
CREATE TABLE relationship_tuples (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_type TEXT NOT NULL,
    subject_id   TEXT NOT NULL,
    relation     TEXT NOT NULL,
    object_type  TEXT NOT NULL,
    object_id    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subject_type, subject_id, relation, object_type, object_id)
);

CREATE INDEX ON relationship_tuples (object_type, object_id, relation);
CREATE INDEX ON relationship_tuples (subject_type, subject_id);
```

```sql
-- Direct permission check (non-recursive)
SELECT EXISTS (
    SELECT 1 FROM relationship_tuples
    WHERE object_type = 'document'
      AND object_id   = 'q3-report'
      AND relation    IN ('owner', 'editor')
      AND subject_type = 'user'
      AND subject_id   = 'alice'
) AS has_access;
```

---

## Security Rules

- Use ZedTokens (SpiceDB consistency tokens) to avoid new-enemy-problem: always pass a ZedToken from a write operation to subsequent reads on the same object.
- Deny by default — if the check returns `PERMISSIONSHIP_NO_PERMISSION`, access is denied.
- Scope delete operations: when a resource is deleted, delete all its tuples to prevent orphaned permissions.
- Audit tuple writes in an append-only log for compliance.
- Never trust the subject identity from the client — always derive it from the verified auth token.

---

## Key Rules

- Tuples: `subject_type:subject_id#relation@object_type:object_id`.
- Check API: returns HAS_PERMISSION or NO_PERMISSION — fail closed on NO_PERMISSION.
- Hierarchical groups: `group:eng#member` as subject in a tuple propagates all member permissions.
- SpiceDB is the production-grade choice; PostgreSQL tuples table works for simpler graphs.
- Delete all related tuples when deleting a resource — no orphaned grants.
- ZedToken consistency prevents stale reads after writes.
