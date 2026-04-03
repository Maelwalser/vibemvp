# Domain Modeling Skill Guide

## Overview

Domain-driven design primitives: bounded context definition, entity vs value object vs aggregate distinctions, attribute types and constraints, sensitive field handling, and domain event patterns.

---

## Bounded Context Definition

A bounded context is an explicit boundary within which a domain model applies. Each context has its own ubiquitous language and owns its data.

```
┌─────────────────────────────┐     ┌──────────────────────────────┐
│  Identity Context           │     │  Billing Context             │
│  - User (Aggregate Root)    │     │  - Account (Aggregate Root)  │
│  - Role (Entity)            │ ──> │  - Invoice (Entity)          │
│  - Permission (Value Object)│     │  - LineItem (Value Object)   │
└─────────────────────────────┘     └──────────────────────────────┘
         context map: Customer/Supplier
```

### Context Map Relationships

- **Shared Kernel** — two contexts share a subset of the model and coordinate changes.
- **Customer/Supplier** — upstream context (supplier) publishes; downstream (customer) consumes.
- **Anti-Corruption Layer (ACL)** — downstream translates the upstream model through an adapter to protect its own language.
- **Open Host Service** — upstream publishes a stable published language (e.g. API contract).

---

## Entity vs Value Object vs Aggregate

| Concept | Identity | Mutability | Example |
|---------|----------|------------|---------|
| **Entity** | Has a unique ID | Mutable over time | `User`, `Order`, `Product` |
| **Value Object** | No identity — defined by attributes | Immutable | `Money`, `Address`, `Email` |
| **Aggregate** | Cluster of entities/VOs with one root | Root is the only public entry point | `Order` (root) + `OrderLine[]` |

```go
// Value Object — immutable, no ID
type Money struct {
    Amount   decimal.Decimal
    Currency string
}

func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
    if currency == "" { return Money{}, errors.New("currency required") }
    return Money{Amount: amount, Currency: currency}, nil
}

// Entity — has stable identity
type User struct {
    ID        uuid.UUID
    Email     Email       // value object
    CreatedAt time.Time
}

// Aggregate Root — controls access to internals
type Order struct {
    ID    uuid.UUID
    Lines []OrderLine   // only mutated through Order methods
}

func (o *Order) AddLine(product ProductID, qty int) error { ... }
```

---

## Domain Attribute Types

| Type | DB Column | Notes |
|------|-----------|-------|
| `String` | `TEXT` / `VARCHAR(n)` | Length constraint via `max_length` |
| `Int` | `INTEGER` / `BIGINT` | Range via `min` / `max` |
| `Float` | `NUMERIC(p,s)` | Use NUMERIC for money, never FLOAT |
| `Boolean` | `BOOLEAN` | Default explicit |
| `DateTime` | `TIMESTAMPTZ` | Always timezone-aware |
| `UUID` | `UUID` | Default `gen_random_uuid()` |
| `Enum` | `TEXT` with CHECK / native ENUM | Use text + check for portability |
| `JSON` | `JSONB` | Prefer JSONB for indexability |
| `Binary` | `BYTEA` | Encrypted PII, raw blobs |
| `Array` | `TEXT[]` / `INTEGER[]` | Avoid for normalized data |
| `Ref` | `UUID REFERENCES ...` | FK to another aggregate root |

---

## Constraint Types

Apply at the database level AND at the application validation layer.

| Constraint | DB Enforcement | Application Enforcement |
|------------|---------------|------------------------|
| `required` | `NOT NULL` | `@IsNotEmpty()` / required field |
| `unique` | `UNIQUE INDEX` | Pre-check or catch unique violation |
| `not_null` | `NOT NULL` | Non-nullable type |
| `min` | `CHECK (val >= n)` | `@Min(n)` |
| `max` | `CHECK (val <= n)` | `@Max(n)` |
| `min_length` | `CHECK (length(s) >= n)` | `@MinLength(n)` |
| `max_length` | `VARCHAR(n)` or CHECK | `@MaxLength(n)` |
| `email` | None (pattern too complex) | `@IsEmail()` |
| `url` | None | `@IsUrl()` |
| `regex` | `CHECK (s ~ 'pattern')` | `@Matches(/pattern/)` |
| `positive` | `CHECK (val > 0)` | `@IsPositive()` |
| `future` | `CHECK (dt > NOW())` | Custom validator |
| `past` | `CHECK (dt < NOW())` | Custom validator |
| `enum` | `CHECK (val IN (...))` | `@IsIn([...])` |

---

## Sensitive Field Handling

### Identify Sensitive Fields

Mark during modeling with a `sensitive: true` flag. Sensitive field types include:
- PII: `email`, `phone`, `date_of_birth`, `ssn`, `address`
- Financial: `card_number`, `bank_account`
- Health: `diagnosis`, `medication`
- Credentials: `password_hash`, `api_key`

### Encrypt at Rest

```sql
-- Store ciphertext in BYTEA; decrypt only in authorized contexts
ALTER TABLE users ADD COLUMN email_enc BYTEA NOT NULL;
ALTER TABLE users DROP COLUMN email;  -- remove plaintext after migration
```

### Mask in Logs and API Responses

```go
type User struct {
    ID    uuid.UUID `json:"id"`
    Email string    `json:"email" log:"-"`       // omit from log output
    Phone string    `json:"phone" log:"masked"`  // trigger masking
}

// Mask before serializing to JSON for non-privileged callers
func (u User) Sanitized() User {
    return User{
        ID:    u.ID,
        Email: maskEmail(u.Email),
        Phone: maskPhone(u.Phone),
    }
}
```

---

## Domain Event Pattern

Domain events capture what happened inside an aggregate, enabling decoupled side effects.

```go
// Event definition
type UserRegistered struct {
    OccurredAt time.Time
    UserID     uuid.UUID
    Email      string
}

// Aggregate raises events rather than calling services directly
func (u *User) Register(email string) error {
    u.Email = email
    u.raise(UserRegistered{
        OccurredAt: time.Now(),
        UserID:     u.ID,
        Email:      email,
    })
    return nil
}

// Repository persists aggregate + dispatches events after commit
func (r *UserRepo) Save(ctx context.Context, u *User) error {
    if err := r.db.save(u); err != nil { return err }
    for _, ev := range u.Events() {
        r.bus.Publish(ctx, ev)
    }
    return nil
}
```

---

## Key Rules

- Every aggregate has exactly one root; external code only calls methods on the root.
- Value objects are immutable — replace, never mutate.
- Sensitive fields are encrypted at rest; plaintext never appears in logs or non-privileged API responses.
- Apply constraints at both the DB (CHECK, NOT NULL, FK) and application (validation decorators/structs) layers.
- Use `TIMESTAMPTZ` for all date/time columns — never `TIMESTAMP WITHOUT TIME ZONE`.
- Domain events are raised inside aggregates and dispatched after the DB transaction commits.
