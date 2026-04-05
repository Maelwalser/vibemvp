# Domain DTOs Skill Guide

## Overview

DTO categories (Request, Response, EventPayload, Shared), field type mapping from domain to DTO, validation decorators, nested DTO composition, and transformation conventions.

---

## DTO Categories

| Category | Purpose | Direction |
|----------|---------|-----------|
| **Request** | Input from client; includes validation constraints | Inbound |
| **Response** | Output to client; projected subset of domain data | Outbound |
| **EventPayload** | Message published to the event bus / message broker | Internal / async |
| **Shared** | Cross-cutting types reused by multiple DTOs (e.g., pagination, address) | Both |

---

## Field Type Mapping: Domain → DTO

| Domain Type | DTO Type | Notes |
|-------------|----------|-------|
| `UUID` | `string` | Serialize as lowercase UUID string |
| `DateTime` / `TIMESTAMPTZ` | `string` (ISO-8601) | `2026-04-02T14:30:00Z` — never pass raw Date objects |
| `Decimal` / `NUMERIC` | `string` | Preserves precision; client parses as needed |
| `Boolean` | `boolean` | Direct mapping |
| `Enum` | `string` (union type) | Use string literal union in TS; `enum` in Go |
| `Sensitive (email, phone)` | `string` (masked) or omit | Mask unless caller has elevated permission |
| `Binary / BYTEA` | omit or base64 `string` | Never expose raw bytes without encoding |
| `Array` | `T[]` | Typed element array |
| `JSON / JSONB` | typed nested object | Avoid `any` / `object`; define explicit nested DTO |
| `Ref (FK UUID)` | `string` (ID only) | Expand to nested DTO only when required |

---

## TypeScript / NestJS DTO Examples

### Request DTO

```typescript
import { IsEmail, IsString, MinLength, MaxLength, IsOptional, IsIn } from 'class-validator';
import { Transform } from 'class-transformer';

export class CreateUserRequest {
    @IsEmail()
    email: string;

    @IsString()
    @MinLength(8)
    @MaxLength(128)
    password: string;

    @IsString()
    @MaxLength(100)
    fullName: string;

    @IsOptional()
    @IsIn(['admin', 'user', 'viewer'])
    role?: string = 'user';
}
```

### Response DTO

```typescript
export class UserResponse {
    id: string;           // UUID as string
    email: string;
    fullName: string;
    role: string;
    createdAt: string;    // ISO-8601

    // Sensitive fields omitted: password, mfaSecret

    static from(user: UserEntity): UserResponse {
        return {
            id:        user.id,
            email:     user.email,
            fullName:  user.fullName,
            role:      user.role,
            createdAt: user.createdAt.toISOString(),
        };
    }
}
```

### EventPayload DTO

```typescript
export class UserRegisteredPayload {
    eventId:    string;   // UUID — idempotency key
    occurredAt: string;   // ISO-8601
    userId:     string;
    email:      string;
    role:       string;
}
```

### Shared / Pagination DTO

```typescript
export class PaginatedResponse<T> {
    data:       T[];
    total:      number;
    page:       number;
    limit:      number;
    totalPages: number;
}
```

---

## Nested DTO Composition

Use `@ValidateNested` + `@Type(() => NestedDto)` for nested objects in NestJS:

```typescript
import { ValidateNested, IsArray } from 'class-validator';
import { Type } from 'class-transformer';

export class AddressDto {
    @IsString() street: string;
    @IsString() city: string;
    @IsString() country: string;
    @IsOptional() @IsString() postalCode?: string;
}

export class CreateOrderRequest {
    @ValidateNested()
    @Type(() => AddressDto)
    shippingAddress: AddressDto;

    @IsArray()
    @ValidateNested({ each: true })
    @Type(() => OrderLineDto)
    lines: OrderLineDto[];
}
```

---

## Go DTO Examples

```go
// Request DTO with struct tags
type CreateUserRequest struct {
    Email    string `json:"email"    validate:"required,email"`
    Password string `json:"password" validate:"required,min=8,max=128"`
    FullName string `json:"fullName" validate:"required,max=100"`
    Role     string `json:"role"     validate:"omitempty,oneof=admin user viewer"`
}

// Response DTO — domain → DTO mapping
type UserResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    FullName  string `json:"fullName"`
    Role      string `json:"role"`
    CreatedAt string `json:"createdAt"`
}

func UserResponseFromDomain(u domain.User) UserResponse {
    return UserResponse{
        ID:        u.ID.String(),
        Email:     u.Email,
        FullName:  u.FullName,
        Role:      string(u.Role),
        CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
    }
}
```

---

## Python DTO Examples (Pydantic)

```python
from pydantic import BaseModel, EmailStr, Field
from datetime import datetime
from typing import Optional

class CreateUserRequest(BaseModel):
    email:     EmailStr
    password:  str = Field(min_length=8, max_length=128)
    full_name: str = Field(max_length=100)
    role:      str = Field(default="user", pattern="^(admin|user|viewer)$")

class UserResponse(BaseModel):
    id:         str
    email:      str
    full_name:  str
    role:       str
    created_at: str   # ISO-8601

    model_config = {"populate_by_name": True}

    @classmethod
    def from_domain(cls, user) -> "UserResponse":
        return cls(
            id=str(user.id),
            email=user.email,
            full_name=user.full_name,
            role=user.role,
            created_at=user.created_at.isoformat(),
        )
```

---

## Naming and Transformation Conventions

| Convention | Rule |
|------------|------|
| DB columns | `snake_case` — `created_at`, `user_id` |
| JSON / API | `camelCase` — `createdAt`, `userId` |
| Go struct fields | `PascalCase` with json tags |
| Python fields | `snake_case` with alias or model config |
| Request DTO suffix | `Request` or `Input` — e.g. `CreateUserRequest` |
| Response DTO suffix | `Response` — e.g. `UserResponse` |
| Event payload suffix | `Payload` or `Event` — e.g. `UserRegisteredPayload` |
| Shared DTO suffix | `Dto` — e.g. `AddressDto`, `PaginatedResponse<T>` |

---

## Key Rules

- Never expose domain entities directly from API endpoints — always map to a Response DTO.
- Omit sensitive fields (passwords, secrets, encrypted values) from Response DTOs entirely.
- Use `string` for UUIDs and ISO-8601 for dates in all API responses — never raw numeric timestamps.
- Use `string` for monetary/decimal values to preserve precision in transit.
- Validate all Request DTOs at the controller layer before passing to the service.
- EventPayload DTOs must include a unique `eventId` (UUID) for idempotency.
- Map DB snake_case → API camelCase at the DTO serialization boundary, not in business logic.
