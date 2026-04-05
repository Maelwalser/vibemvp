# Secrets Management Skill Guide

## Overview

Covers: HashiCorp Vault (KV v2, dynamic credentials, AppRole), AWS Secrets Manager, GCP Secret Manager, and GitHub Secrets.

## HashiCorp Vault

### KV v2 Secrets Engine

```bash
# Enable KV v2
vault secrets enable -path=secret kv-v2

# Write secrets
vault kv put secret/myapp/db \
  url="postgres://user:pass@host:5432/db" \
  password="mysecret"

# Read secrets
vault kv get secret/myapp/db
vault kv get -field=url secret/myapp/db

# Update one field (patch — does not overwrite other fields)
vault kv patch secret/myapp/db password="newpass"

# List secrets
vault kv list secret/myapp
```

### Dynamic Database Credentials (TTL-based)

```bash
# Enable database secrets engine
vault secrets enable database

# Configure PostgreSQL connection
vault write database/config/postgres \
  plugin_name=postgresql-database-plugin \
  allowed_roles="api-role" \
  connection_url="postgresql://{{username}}:{{password}}@postgres:5432/appdb" \
  username="vault" \
  password="vaultpass"

# Create role with TTL
vault write database/roles/api-role \
  db_name=postgres \
  creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
  default_ttl="1h" \
  max_ttl="24h"

# Generate credentials (app calls this at startup)
vault read database/creds/api-role
# Output: username=v-api-xxx, password=xxx, lease_duration=1h
```

### AppRole Authentication

```bash
# Enable AppRole
vault auth enable approle

# Create role with policy
vault write auth/approle/role/api-role \
  secret_id_ttl=10m \
  token_ttl=1h \
  token_max_ttl=24h \
  policies="api-policy"

# Get role_id (embed in app config)
vault read auth/approle/role/api-role/role-id

# Generate secret_id (generate per-deployment, keep short TTL)
vault write -f auth/approle/role/api-role/secret-id

# Login with AppRole (app does this at startup)
vault write auth/approle/login \
  role_id=<ROLE_ID> \
  secret_id=<SECRET_ID>
```

### Vault Agent Sidecar (auto-renewal)

```yaml
# Kubernetes — Vault Agent as init container + sidecar
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/agent-inject-secret-db: "secret/data/myapp/db"
  vault.hashicorp.com/agent-inject-template-db: |
    {{- with secret "secret/data/myapp/db" -}}
    DATABASE_URL={{ .Data.data.url }}
    {{- end }}
  vault.hashicorp.com/role: "api-role"
  vault.hashicorp.com/agent-pre-populate-only: "false"  # keep renewing
```

### Transit Engine (application-level encryption)

```bash
# Enable transit
vault secrets enable transit

# Create encryption key
vault write -f transit/keys/myapp

# Encrypt data
vault write transit/encrypt/myapp plaintext=$(echo "sensitive data" | base64)
# Returns: ciphertext=vault:v1:xxx

# Decrypt
vault write transit/decrypt/myapp ciphertext="vault:v1:xxx"
# Returns: plaintext (base64)
```

## AWS Secrets Manager

```typescript
// Node.js
import {
  SecretsManagerClient,
  GetSecretValueCommand,
} from "@aws-sdk/client-secrets-manager";

const client = new SecretsManagerClient({ region: "us-east-1" });

async function getSecret(secretName: string): Promise<Record<string, string>> {
  const response = await client.send(
    new GetSecretValueCommand({ SecretId: secretName })
  );
  return JSON.parse(response.SecretString!);
}

// Cache at module level — avoid calling on every request
let dbConfig: Record<string, string>;

async function getDbConfig() {
  if (!dbConfig) {
    dbConfig = await getSecret("prod/api/db");
  }
  return dbConfig;
}
```

```bash
# Create secret
aws secretsmanager create-secret \
  --name prod/api/db \
  --secret-string '{"url":"postgres://...","password":"secret"}'

# Rotate secret (requires rotation Lambda)
aws secretsmanager rotate-secret \
  --secret-id prod/api/db \
  --rotation-lambda-arn arn:aws:lambda:...:function:SecretsRotation \
  --rotation-rules AutomaticallyAfterDays=30
```

```yaml
# CloudFormation dynamic reference
Resources:
  DBInstance:
    Type: AWS::RDS::DBInstance
    Properties:
      MasterUserPassword: "{{resolve:secretsmanager:prod/api/db:SecretString:password}}"
```

## GCP Secret Manager

```python
from google.cloud import secretmanager

client = secretmanager.SecretManagerServiceClient()

def access_secret(project_id: str, secret_id: str, version: str = "latest") -> str:
    name = f"projects/{project_id}/secrets/{secret_id}/versions/{version}"
    response = client.access_secret_version(request={"name": name})
    return response.payload.data.decode("UTF-8")

# Usage
db_url = access_secret("my-project", "db-url")
```

```bash
# Create secret
echo -n "postgres://user:pass@host/db" | \
  gcloud secrets create db-url --data-file=-

# Add new version
echo -n "postgres://user:pass@host/db-v2" | \
  gcloud secrets versions add db-url --data-file=-

# Disable old version
gcloud secrets versions disable 1 --secret=db-url

# IAM binding — grant service account access
gcloud secrets add-iam-policy-binding db-url \
  --member="serviceAccount:api-sa@PROJECT.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

## GitHub Secrets

```yaml
# GitHub Actions — access secrets
jobs:
  deploy:
    environment: production   # environment-scoped secrets
    steps:
      - name: Deploy
        env:
          API_KEY: ${{ secrets.API_KEY }}
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
        run: |
          echo "Deploying with secrets from environment"
```

```bash
# Set secrets via GitHub CLI
gh secret set DATABASE_URL --body "postgres://..."
gh secret set API_KEY --body "sk-..." --env production

# List secrets (names only, values are never shown)
gh secret list
gh secret list --env production
```

## Key Rules

- Never log or print secret values — mask them in CI/CD output.
- Rotate Vault dynamic credentials frequently — 1h TTL is appropriate for DB credentials.
- AppRole secret_id should have a short TTL (10m) — generate fresh per deployment.
- AWS Secrets Manager: cache the secret at process startup — calling per-request adds latency and cost.
- GCP: always bind IAM at the secret level, not the project level — principle of least privilege.
- GitHub environment-scoped secrets require PR approval for protected environments — use this for production.
- Use Vault transit engine for field-level encryption of PII stored in the database.
