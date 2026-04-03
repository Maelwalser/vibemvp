# Data Compliance Skill Guide

## Overview

Compliance patterns for GDPR, HIPAA, PCI-DSS, SOC2, data residency, CCPA, and classification levels. Each regulation maps to a concrete implementation pattern.

---

## GDPR — Right to Deletion

Hard-delete PII columns; retain an anonymized audit record for accountability.

```sql
-- Step 1: Anonymize all PII columns in-place
UPDATE users
SET
    email           = 'deleted-' || id || '@gdpr.invalid',
    full_name       = 'DELETED',
    phone           = NULL,
    date_of_birth   = NULL,
    address         = NULL,
    deleted_at      = NOW(),
    gdpr_erased_at  = NOW()
WHERE id = $1;

-- Step 2: Delete from child tables that contain raw PII
DELETE FROM user_sessions     WHERE user_id = $1;
DELETE FROM user_addresses    WHERE user_id = $1;
DELETE FROM payment_methods   WHERE user_id = $1;

-- Step 3: Append anonymized audit entry (retained for legal basis)
INSERT INTO gdpr_audit_log (user_id_hash, action, performed_at, legal_basis)
VALUES (encode(sha256($1::text::bytea), 'hex'), 'RIGHT_TO_ERASURE', NOW(), 'GDPR Art.17');
```

---

## HIPAA — Immutable Audit Trail

Every access to PHI (Protected Health Information) must produce an append-only log entry.

```sql
CREATE TABLE phi_audit_log (
    id           BIGSERIAL PRIMARY KEY,
    user_id      UUID        NOT NULL,
    patient_id   UUID        NOT NULL,
    action       TEXT        NOT NULL,  -- READ | WRITE | DELETE | EXPORT
    resource     TEXT        NOT NULL,  -- e.g. 'medical_records', 'prescriptions'
    purpose      TEXT        NOT NULL,  -- e.g. 'treatment', 'payment', 'operations'
    ip_address   INET,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent any modification of audit rows
CREATE RULE no_update_phi_audit AS ON UPDATE TO phi_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_phi_audit AS ON DELETE TO phi_audit_log DO INSTEAD NOTHING;

-- Revoke direct table modification from all app roles
REVOKE UPDATE, DELETE ON phi_audit_log FROM app_role;
```

Application: call before every PHI read or write:

```python
def log_phi_access(user_id, patient_id, action, resource, purpose, ip=None):
    db.execute(
        "INSERT INTO phi_audit_log (user_id, patient_id, action, resource, purpose, ip_address) "
        "VALUES (%s, %s, %s, %s, %s, %s)",
        [user_id, patient_id, action, resource, purpose, ip]
    )
```

---

## PCI-DSS — Tokenization

Never store raw card numbers. Replace with an opaque token; the actual PAN lives in a vault.

```
Flow:
  Client submits card → Tokenization service → Store token in DB
                                             → Store PAN in vault (e.g. HashiCorp Vault, AWS Payment Cryptography)
  Charge flow: retrieve PAN from vault using token → pass to payment processor
```

```sql
CREATE TABLE payment_tokens (
    token        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vault_ref    TEXT    NOT NULL UNIQUE,  -- opaque reference to vault record
    card_last4   CHAR(4) NOT NULL,
    card_brand   TEXT    NOT NULL,
    expiry_month SMALLINT NOT NULL,
    expiry_year  SMALLINT NOT NULL,
    user_id      UUID    NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- NO card number column ever
);
```

```go
// Tokenize card (call before any persistence)
func TokenizeCard(pan string) (token string, vaultRef string, err error) {
    vaultRef, err = vault.Store(pan) // encrypted in vault
    if err != nil { return }
    token = uuid.New().String()
    return
}
```

---

## SOC2 — Schema Change Audit Log

Track all DDL operations to satisfy Change Management controls.

```sql
-- DDL audit trigger (PostgreSQL event trigger)
CREATE TABLE ddl_audit_log (
    id           BIGSERIAL PRIMARY KEY,
    event_type   TEXT        NOT NULL,  -- ALTER TABLE, CREATE INDEX, DROP TABLE …
    object_type  TEXT,
    object_name  TEXT,
    schema_name  TEXT,
    executed_by  TEXT        NOT NULL DEFAULT current_user,
    query        TEXT,
    executed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION log_ddl_event() RETURNS event_trigger AS $$
BEGIN
    INSERT INTO ddl_audit_log (event_type, object_type, object_name, schema_name, query)
    SELECT
        tg_event,
        object_type,
        object_identity,
        schema_name,
        current_query()
    FROM pg_event_trigger_ddl_commands();
END;
$$ LANGUAGE plpgsql;

CREATE EVENT TRIGGER ddl_audit_trigger
    ON ddl_command_end
    EXECUTE FUNCTION log_ddl_event();
```

---

## Data Residency — Geographically Partitioned Storage

Tag each row with its region and enforce placement via application-level routing.

```sql
CREATE TABLE user_data (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region     TEXT NOT NULL CHECK (region IN ('EU', 'US', 'APAC', 'CA')),
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY LIST (region);

CREATE TABLE user_data_eu   PARTITION OF user_data FOR VALUES IN ('EU');
CREATE TABLE user_data_us   PARTITION OF user_data FOR VALUES IN ('US');
CREATE TABLE user_data_apac PARTITION OF user_data FOR VALUES IN ('APAC');
CREATE TABLE user_data_ca   PARTITION OF user_data FOR VALUES IN ('CA');
```

Route at application layer: map user's jurisdiction → correct database host in that region.

---

## CCPA — Opt-Out Handling

```sql
ALTER TABLE users ADD COLUMN ccpa_do_not_sell BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN ccpa_opted_out_at TIMESTAMPTZ;

-- All data-sale or sharing queries must filter
SELECT * FROM users WHERE ccpa_do_not_sell = FALSE;
```

```python
def record_ccpa_optout(user_id: str):
    db.execute(
        "UPDATE users SET ccpa_do_not_sell = TRUE, ccpa_opted_out_at = NOW() WHERE id = %s",
        [user_id]
    )
    audit_log("CCPA_OPTOUT", user_id)
```

---

## Data Classification Levels

Apply consistently across all tables, APIs, and storage buckets.

| Level | Examples | Controls |
|-------|----------|----------|
| **Public** | Marketing copy, public docs | No special handling |
| **Internal** | Employee names, internal metrics | Access log, internal-only ACL |
| **Confidential** | Customer PII, contracts | Encrypted at rest, access-controlled, audit trail |
| **Restricted** | PAN, SSN, PHI, credentials | Tokenized or encrypted, strict ACL, immutable audit, break-glass access only |

```sql
-- Column-level classification comment
COMMENT ON COLUMN users.email IS 'classification:Confidential; pii:true; encrypt:AES-256';
COMMENT ON COLUMN users.full_name IS 'classification:Confidential; pii:true';
COMMENT ON COLUMN payment_tokens.card_last4 IS 'classification:Restricted; pii:true; tokenized:true';
```

---

## Key Rules

- GDPR deletion: anonymize in-place, then hard-delete child PII tables; keep anonymized audit row.
- HIPAA audit log: append-only via DDL rules and role revocations — no application path for UPDATE/DELETE.
- PCI-DSS: never store raw PAN in the application database; vault reference only.
- SOC2: DDL event trigger provides tamper-evident change history for auditors.
- CCPA opt-out: filter `ccpa_do_not_sell = FALSE` in every data-sharing query path.
- Label every sensitive column with a classification comment for automated scanner discovery.
