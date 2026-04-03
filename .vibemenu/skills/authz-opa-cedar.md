# OPA / Cedar Policy Authorization Skill Guide

## Overview

External policy engines decouple authorization logic from application code. Policies are versioned, testable, and auditable independently of the services they protect.

- **OPA (Open Policy Agent)**: General-purpose policy engine with Rego language. Deploy as a sidecar or standalone service.
- **Cedar**: Amazon's policy language for AWS Verified Permissions. Principal/action/resource model with strong typing.

---

## OPA (Open Policy Agent)

### Rego Policy Authoring

```rego
# policies/authz.rego
package authz

import future.keywords.if
import future.keywords.in

# Default deny
default allow := false

# Allow if user has the required role
allow if {
    required_role := data.routes[input.method][input.path]
    required_role in input.user.roles
}

# Allow admins to do anything
allow if {
    "admin" in input.user.roles
}

# Allow resource owner
allow if {
    input.action == "update"
    input.resource.owner_id == input.user.id
}

# Deny inactive users regardless of roles
deny if {
    input.user.active == false
}

# Final decision: allow unless explicitly denied
final_allow if {
    allow
    not deny
}
```

### Input Document Structure

```json
{
  "user": {
    "id": "user-123",
    "roles": ["editor"],
    "department": "engineering",
    "active": true
  },
  "action": "update",
  "resource": {
    "type": "document",
    "id": "doc-456",
    "owner_id": "user-123",
    "department": "engineering"
  },
  "method": "PUT",
  "path": "/api/documents/doc-456"
}
```

### OPA Sidecar Deployment

```yaml
# docker-compose.yml
services:
  opa:
    image: openpolicyagent/opa:latest
    command:
      - run
      - --server
      - --addr=0.0.0.0:8181
      - --bundle=/policies
      - --log-format=json
    volumes:
      - ./policies:/policies
    ports:
      - "8181:8181"

  app:
    build: .
    environment:
      OPA_URL: http://opa:8181
```

```yaml
# Kubernetes sidecar
spec:
  containers:
    - name: app
      image: your-app:latest
    - name: opa
      image: openpolicyagent/opa:latest
      args:
        - run
        - --server
        - --addr=0.0.0.0:8181
        - --bundle=/policies
      volumeMounts:
        - name: policies
          mountPath: /policies
      resources:
        limits: { memory: 128Mi, cpu: 250m }
```

### Policy Bundle Loading

```bash
# Build a policy bundle
opa build -b policies/ -o bundle.tar.gz

# OPA loads bundles from disk or HTTP
opa run --server \
  --bundle https://your-bundle-server/bundle.tar.gz \
  --set=services.bundle-server.url=https://your-bundle-server
```

### Go — OPA HTTP Query

```go
type OPAInput struct {
    User     map[string]any `json:"user"`
    Action   string         `json:"action"`
    Resource map[string]any `json:"resource"`
}

type OPAResult struct {
    Result struct {
        Allow bool `json:"final_allow"`
    } `json:"result"`
}

func CheckWithOPA(ctx context.Context, input OPAInput) (bool, error) {
    body, err := json.Marshal(map[string]any{"input": input})
    if err != nil {
        return false, fmt.Errorf("marshal opa input: %w", err)
    }

    opaURL := os.Getenv("OPA_URL") + "/v1/data/authz/final_allow"
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, opaURL, bytes.NewReader(body))
    if err != nil {
        return false, fmt.Errorf("create opa request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return false, fmt.Errorf("opa request: %w", err)
    }
    defer resp.Body.Close()

    var result OPAResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return false, fmt.Errorf("decode opa response: %w", err)
    }
    return result.Result.Allow, nil
}
```

### TypeScript — OPA Query

```typescript
async function checkWithOPA(input: {
  user: { id: string; roles: string[] }
  action: string
  resource: Record<string, unknown>
}): Promise<boolean> {
  const res = await fetch(`${process.env.OPA_URL}/v1/data/authz/final_allow`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input }),
  })
  if (!res.ok) throw new Error(`OPA check failed: ${res.status}`)
  const { result } = await res.json()
  return result === true
}
```

### Partial Evaluation (Pre-compute filters)

```rego
# Generate a list of document IDs the user can read
allowed_documents[id] {
    data.documents[id].owner_id == input.user.id
}

allowed_documents[id] {
    data.documents[id].department == input.user.department
    "viewer" in input.user.roles
}
```

```go
// Query: which documents can this user read?
// POST /v1/data/authz/allowed_documents
// Returns: { "result": ["doc-1", "doc-5", "doc-9"] }
```

### Policy Testing

```rego
# policies/authz_test.rego
package authz_test

import data.authz

test_admin_can_do_anything if {
    authz.allow with input as {
        "user": {"roles": ["admin"], "active": true},
        "action": "delete",
        "resource": {"owner_id": "other-user"},
    }
}

test_owner_can_update_own_resource if {
    authz.allow with input as {
        "user": {"id": "user-1", "roles": ["user"], "active": true},
        "action": "update",
        "resource": {"owner_id": "user-1"},
    }
}

test_inactive_user_denied if {
    not authz.final_allow with input as {
        "user": {"roles": ["admin"], "active": false},
        "action": "read",
        "resource": {},
    }
}
```

```bash
opa test policies/ -v
```

---

## Cedar (AWS Verified Permissions)

### Policy Language

```cedar
// ALLOW principals with admin role
permit (
  principal in Role::"admin",
  action in [Action::"read", Action::"write", Action::"delete"],
  resource
);

// ALLOW resource owners to edit
permit (
  principal,
  action == Action::"edit",
  resource
) when {
  resource.owner == principal
};

// ALLOW team members to view team documents
permit (
  principal,
  action == Action::"view",
  resource is Document
) when {
  principal in resource.team
};

// FORBID inactive users
forbid (
  principal,
  action,
  resource
) when {
  principal.status == "inactive"
};
```

### AWS Verified Permissions Setup

```typescript
import {
  VerifiedPermissionsClient,
  IsAuthorizedCommand,
} from '@aws-sdk/client-verifiedpermissions'

const avp = new VerifiedPermissionsClient({ region: process.env.AWS_REGION })

async function isAuthorized(params: {
  policyStoreId: string
  principalType: string
  principalID: string
  action: string
  resourceType: string
  resourceID: string
  context?: Record<string, unknown>
}): Promise<boolean> {
  const cmd = new IsAuthorizedCommand({
    policyStoreId: params.policyStoreId,
    principal: {
      entityType: params.principalType,
      entityId: params.principalID,
    },
    action: {
      actionType: 'Action',
      actionId: params.action,
    },
    resource: {
      entityType: params.resourceType,
      entityId: params.resourceID,
    },
    context: params.context
      ? {
          contextMap: Object.fromEntries(
            Object.entries(params.context).map(([k, v]) => [
              k,
              { boolean: v as boolean },
            ])
          ),
        }
      : undefined,
  })

  const result = await avp.send(cmd)
  return result.decision === 'ALLOW'
}

// Usage
const allowed = await isAuthorized({
  policyStoreId: process.env.AVP_POLICY_STORE_ID!,
  principalType: 'User',
  principalID: userID,
  action: 'edit',
  resourceType: 'Document',
  resourceID: documentID,
})
if (!allowed) return res.status(403).json({ error: 'Forbidden' })
```

---

## Security Rules

- Policy evaluation must be the authoritative decision point — never short-circuit with local checks that bypass the engine.
- Fail closed: if the policy engine returns an error or is unreachable, deny the request.
- Version control all policies alongside the codebase; require tests for every new policy rule.
- Log all authorization decisions with the input document for audit trails.
- Use partial evaluation to push database filters into SQL — avoid fetching all records and filtering in memory.

---

## Key Rules

- OPA: Rego `default allow := false` pattern; deny by default.
- Input: structured JSON with user, action, resource, and environment attributes.
- OPA sidecar: query `/v1/data/package/rule` via HTTP POST.
- Policy bundles: version and distribute via OPA bundle server.
- Test policies with `opa test` before deployment.
- Cedar: ALLOW/FORBID statements with typed principal/action/resource model; use FORBID to override ALLOW.
- AWS Verified Permissions: `IsAuthorizedCommand` with entity references.
