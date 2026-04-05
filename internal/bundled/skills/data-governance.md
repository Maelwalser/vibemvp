# Data Governance Skill Guide

## Overview

Data governance covers soft-delete patterns, hard-delete cascade planning, archive strategies, and PII encryption/masking for production databases.

---

## Soft-Delete Pattern

Add a nullable timestamp column — never remove rows directly.

```sql
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;
```

### Partial Unique Index

Enforce uniqueness only among live rows:

```sql
-- Email must be unique among non-deleted users
CREATE UNIQUE INDEX users_email_active_uidx
    ON users (email)
    WHERE deleted_at IS NULL;
```

### View-Based Filtering

Always query through the active view to avoid accidentally including deleted rows:

```sql
CREATE VIEW active_users AS
    SELECT * FROM users WHERE deleted_at IS NULL;

CREATE VIEW active_orders AS
    SELECT * FROM orders WHERE deleted_at IS NULL;
```

### Soft Delete & Restore

```sql
-- Soft delete
UPDATE users SET deleted_at = NOW() WHERE id = $1;

-- Restore
UPDATE users SET deleted_at = NULL WHERE id = $1;
```

---

## Hard-Delete Cascade Planning

Before issuing a hard DELETE, enumerate all FK references and decide on cascade behavior:

```sql
-- Discover all FK references to a table
SELECT
    tc.table_name  AS referencing_table,
    kcu.column_name AS referencing_column,
    rc.delete_rule
FROM information_schema.referential_constraints rc
JOIN information_schema.table_constraints tc
    ON rc.constraint_name = tc.constraint_name
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
WHERE rc.unique_constraint_name IN (
    SELECT constraint_name FROM information_schema.table_constraints
    WHERE table_name = 'users'
);
```

Cascade options per FK:
- `CASCADE` — delete child rows automatically
- `SET NULL` — nullify FK on child rows
- `RESTRICT` — prevent parent delete if children exist
- `NO ACTION` — same as RESTRICT but deferred check

---

## Archive Table Strategy

Move old soft-deleted rows to a dedicated archive table to keep live tables lean.

```sql
CREATE TABLE users_archive (LIKE users INCLUDING ALL);

-- Migration job (run via pg_cron or scheduled worker)
WITH moved AS (
    DELETE FROM users
    WHERE deleted_at IS NOT NULL
      AND deleted_at < NOW() - INTERVAL '90 days'
    RETURNING *
)
INSERT INTO users_archive SELECT * FROM moved;
```

### pg_cron Scheduling

```sql
-- Requires pg_cron extension
SELECT cron.schedule(
    'archive-deleted-users',
    '0 2 * * *',   -- daily at 02:00
    $$
    WITH moved AS (
        DELETE FROM users
        WHERE deleted_at IS NOT NULL
          AND deleted_at < NOW() - INTERVAL '90 days'
        RETURNING *
    )
    INSERT INTO users_archive SELECT * FROM moved;
    $$
);
```

---

## PII Encryption (Field-Level)

### pgcrypto — Database-Level

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Encrypt on insert
INSERT INTO users (email_encrypted)
VALUES (pgp_sym_encrypt('user@example.com', current_setting('app.pii_key')));

-- Decrypt for authorized queries
SELECT pgp_sym_decrypt(email_encrypted::bytea, current_setting('app.pii_key'))
    AS email
FROM users
WHERE id = $1;
```

Set the key at session start (never hardcode):
```sql
SET app.pii_key = 'your-256-bit-key-from-env';
```

### Application-Level Encryption (preferred for key rotation)

```python
from cryptography.fernet import Fernet

KEY = Fernet(os.environ["PII_ENCRYPTION_KEY"])

def encrypt_pii(value: str) -> bytes:
    return KEY.encrypt(value.encode())

def decrypt_pii(ciphertext: bytes) -> str:
    return KEY.decrypt(ciphertext).decode()
```

```go
import "golang.org/x/crypto/nacl/secretbox"

func EncryptPII(plaintext string, key [32]byte) []byte { ... }
func DecryptPII(ciphertext []byte, key [32]byte) (string, error) { ... }
```

### View-Based Decryption for Authorized Roles

```sql
CREATE VIEW users_pii_view AS
    SELECT
        id,
        pgp_sym_decrypt(email_encrypted::bytea, current_setting('app.pii_key')) AS email,
        pgp_sym_decrypt(phone_encrypted::bytea, current_setting('app.pii_key')) AS phone
    FROM users;

-- Only grant to privileged roles
GRANT SELECT ON users_pii_view TO pii_reader_role;
```

---

## PII Masking Functions

Use masking for display/logging where raw PII must not appear.

```sql
-- Mask email: user@example.com → u***@example.com
CREATE OR REPLACE FUNCTION mask_email(email TEXT) RETURNS TEXT AS $$
    SELECT SUBSTRING(email, 1, 1)
        || '***@'
        || SPLIT_PART(email, '@', 2);
$$ LANGUAGE SQL IMMUTABLE;

-- Mask phone: +14155552671 → +1*******671
CREATE OR REPLACE FUNCTION mask_phone(phone TEXT) RETURNS TEXT AS $$
    SELECT SUBSTRING(phone, 1, 2)
        || REPEAT('*', LENGTH(phone) - 5)
        || RIGHT(phone, 3);
$$ LANGUAGE SQL IMMUTABLE;
```

---

## Key Rules

- Never expose raw PII in logs, error messages, or API responses — always mask.
- Store encryption keys in environment variables or a secrets manager (Vault, AWS Secrets Manager), never in source code or the database.
- Use partial unique indexes (`WHERE deleted_at IS NULL`) on every unique column in soft-deleted tables.
- Query live data exclusively through active views; never `SELECT * FROM users` in application code.
- Schedule archive jobs off-peak and run inside a transaction to avoid partial migrations.
- Rotate PII encryption keys by re-encrypting with the new key in a background job before retiring the old key.
