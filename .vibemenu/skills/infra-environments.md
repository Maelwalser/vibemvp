# Environment Management Skill Guide

## Overview

Covers multi-stage pipelines (dev→staging→qa→prod), preview environments, environment-specific secrets, DB migration automation, seeding, and feature flag targeting.

## Multi-Stage Pipeline

```
dev → staging → qa → prod

Promotion gates:
  dev    → staging: all CI checks pass (lint/test/build)
  staging → qa:     integration tests pass + manual approval
  qa      → prod:   smoke tests pass + ops approval + change window
```

```yaml
# GitHub Actions — promotion pipeline
name: Promote to Production

on:
  workflow_dispatch:
    inputs:
      sha:
        description: Git SHA to promote
        required: true

jobs:
  promote:
    environment: production   # requires manual approval
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to production
        run: |
          IMAGE="ghcr.io/org/api:${{ github.event.inputs.sha }}"
          kubectl set image deployment/api api=$IMAGE -n prod
          kubectl rollout status deployment/api -n prod
```

## Preview Environments

### Vercel Preview (per PR)

```yaml
# .github/workflows/preview.yml
- name: Deploy to Vercel Preview
  uses: amondnet/vercel-action@v25
  with:
    vercel-token: ${{ secrets.VERCEL_TOKEN }}
    vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
    vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
    working-directory: ./
  id: vercel
- name: Comment PR with preview URL
  uses: actions/github-script@v7
  with:
    script: |
      github.rest.issues.createComment({
        issue_number: context.issue.number,
        owner: context.repo.owner,
        repo: context.repo.repo,
        body: `Preview deployed: ${{ steps.vercel.outputs.preview-url }}`
      })
```

### Neon DB Branch per PR

```yaml
- name: Create Neon DB branch for PR
  uses: neondatabase/create-branch-action@v5
  with:
    project_id: ${{ secrets.NEON_PROJECT_ID }}
    branch_name: pr-${{ github.event.number }}
    api_key: ${{ secrets.NEON_API_KEY }}
  id: neon

- name: Run migrations on branch DB
  env:
    DATABASE_URL: ${{ steps.neon.outputs.db_url }}
  run: npm run db:migrate

- name: Delete Neon branch on PR close
  if: github.event.action == 'closed'
  uses: neondatabase/delete-branch-action@v3
  with:
    project_id: ${{ secrets.NEON_PROJECT_ID }}
    branch: pr-${{ github.event.number }}
    api_key: ${{ secrets.NEON_API_KEY }}
```

## Environment-Specific Secrets

```
Vault paths per environment:
  secret/dev/api/db
  secret/staging/api/db
  secret/prod/api/db

AWS Secrets Manager naming convention:
  /dev/api/database-url
  /staging/api/database-url
  /prod/api/database-url

Kubernetes Secrets — namespace isolation:
  Namespace: api-dev    → Secret: db-credentials
  Namespace: api-staging → Secret: db-credentials
  Namespace: api-prod   → Secret: db-credentials
```

```yaml
# External Secrets Operator — pulls from Vault per env
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-credentials
  namespace: api-prod
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: db-credentials
    creationPolicy: Owner
  data:
    - secretKey: DATABASE_URL
      remoteRef:
        key: secret/prod/api/db
        property: url
```

## DB Migration Automation on Deploy

```bash
# Pre-deploy hook pattern
# Run migrations BEFORE switching traffic (zero-downtime if backward-compatible)

# Kubernetes init container
initContainers:
  - name: migrate
    image: ghcr.io/org/api:${{ github.sha }}
    command: ["./migrate", "--up"]
    env:
      - name: DATABASE_URL
        valueFrom:
          secretKeyRef:
            name: db-credentials
            key: DATABASE_URL
```

```yaml
# Heroku release phase
# Procfile
release: node dist/migrate.js
web: node dist/server.js
```

```bash
# Rollback script on migration failure
#!/bin/bash
set -e

echo "Running migrations..."
if ! ./migrate --up; then
  echo "Migration failed — rolling back"
  ./migrate --down --steps=1
  exit 1
fi
echo "Migrations complete"
```

## DB Seeding

```typescript
// upsert-pattern seed (idempotent — safe to run multiple times)
async function seed(db: Database) {
  await db.transaction(async (trx) => {
    // Upsert ensures no duplicates on re-run
    await trx
      .insertInto("roles")
      .values([
        { id: "admin", name: "Administrator" },
        { id: "user", name: "Regular User" },
      ])
      .onConflict((oc) => oc.column("id").doUpdateSet({ name: sql`excluded.name` }))
      .execute();

    await trx
      .insertInto("users")
      .values({
        id: "seed-admin-001",
        email: "admin@example.com",
        role_id: "admin",
      })
      .onConflict((oc) => oc.column("id").doNothing())
      .execute();
  });
}
```

```typescript
// Factory pattern for test data
class UserFactory {
  static build(overrides: Partial<User> = {}): User {
    return {
      id: randomUUID(),
      email: `user-${Date.now()}@example.com`,
      role: "user",
      createdAt: new Date(),
      ...overrides,
    };
  }

  static async create(overrides: Partial<User> = {}): Promise<User> {
    const user = this.build(overrides);
    return db.insertInto("users").values(user).returningAll().executeTakeFirstOrThrow();
  }
}

// In tests
const admin = await UserFactory.create({ role: "admin" });
```

## Feature Flag Environment Targeting

```typescript
// LaunchDarkly environment targeting
// dev: all flags on
// staging: flags on for beta users only
// prod: gradual rollout 5% → 100%

const ldClient = LDClient.init(
  process.env.LD_SDK_KEY   // different SDK key per environment
);

// Flag targeting configured in LaunchDarkly UI:
// dev:     serving=true (all users)
// staging: serving=true (if user.email matches /@acme\.com$/)
// prod:    rollout 10% of users by key hash
```

## Key Rules

- DB migrations must be backward-compatible for safe zero-downtime deploys — add columns before removing old ones.
- Preview environments should be ephemeral — create on PR open, destroy on PR close.
- Use separate Vault paths or AWS Secret Manager prefixes per environment — never share secrets.
- Seed data must be idempotent (upsert, not insert) — CI runs seeds on every test run.
- Feature flags: use separate SDK keys per environment for proper targeting and analytics isolation.
- Promotion gates (manual approval) prevent accidental prod deployments during off-hours.
- Always run `--dry-run` migrations in CI before executing against staging/prod.
