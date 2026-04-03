# Contracts: OpenAPI Skill Guide

## Overview

OpenAPI 3.x spec generation, path/operation/schema components, error response definitions, per-language annotations, Swagger UI / ReDoc serving, and client SDK generation.

## OpenAPI 3.x Spec Structure

```yaml
# openapi.yaml
openapi: "3.1.0"
info:
  title: MyApp API
  version: "1.0.0"
  description: REST API for MyApp

servers:
  - url: https://api.example.com/v1
    description: Production
  - url: http://localhost:3000/v1
    description: Development

paths:
  /users:
    get:
      operationId: listUsers
      summary: List all users
      tags: [Users]
      parameters:
        - name: page
          in: query
          schema: { type: integer, default: 1, minimum: 1 }
        - name: limit
          in: query
          schema: { type: integer, default: 20, maximum: 100 }
        - name: email
          in: query
          schema: { type: string, format: email }
      responses:
        "200":
          description: Paginated list of users
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserListResponse'
        "400":
          $ref: '#/components/responses/ValidationError'
        "401":
          $ref: '#/components/responses/Unauthorized'

    post:
      operationId: createUser
      summary: Create a user
      tags: [Users]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        "201":
          description: Created user
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        "400":
          $ref: '#/components/responses/ValidationError'
        "409":
          $ref: '#/components/responses/Conflict'

  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema: { type: string, format: uuid }
    get:
      operationId: getUser
      summary: Get user by ID
      tags: [Users]
      responses:
        "200":
          content:
            application/json:
              schema: { $ref: '#/components/schemas/User' }
        "404":
          $ref: '#/components/responses/NotFound'

components:
  schemas:
    User:
      type: object
      required: [id, email, name, createdAt]
      properties:
        id:
          type: string
          format: uuid
          example: "550e8400-e29b-41d4-a716-446655440000"
        email:
          type: string
          format: email
          example: "alice@example.com"
        name:
          type: string
          example: "Alice"
        role:
          type: string
          enum: [user, admin, moderator]
          default: user
        createdAt:
          type: string
          format: date-time

    CreateUserRequest:
      type: object
      required: [email, name]
      properties:
        email:
          type: string
          format: email
        name:
          type: string
          minLength: 1
          maxLength: 100
        role:
          type: string
          enum: [user, admin, moderator]

    UserListResponse:
      type: object
      required: [data, meta]
      properties:
        data:
          type: array
          items: { $ref: '#/components/schemas/User' }
        meta:
          type: object
          properties:
            total: { type: integer }
            page: { type: integer }
            limit: { type: integer }

    ErrorResponse:
      type: object
      required: [error, message]
      properties:
        error: { type: string }
        message: { type: string }
        details:
          type: array
          items:
            type: object
            properties:
              field: { type: string }
              message: { type: string }

  responses:
    ValidationError:
      description: Validation error
      content:
        application/json:
          schema: { $ref: '#/components/schemas/ErrorResponse' }
          example:
            error: VALIDATION_ERROR
            message: "Invalid request body"
            details:
              - field: email
                message: "must be a valid email"

    Unauthorized:
      description: Authentication required
      content:
        application/json:
          schema: { $ref: '#/components/schemas/ErrorResponse' }

    NotFound:
      description: Resource not found
      content:
        application/json:
          schema: { $ref: '#/components/schemas/ErrorResponse' }

    Conflict:
      description: Resource already exists
      content:
        application/json:
          schema: { $ref: '#/components/schemas/ErrorResponse' }

  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

security:
  - bearerAuth: []
```

## Language-Specific Annotations

### Go (swaggo)

```go
// @title MyApp API
// @version 1.0
// @description REST API for MyApp
// @host api.example.com
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// GetUser godoc
// @Summary Get user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Success 200 {object} User
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{userId} [get]
func (h *UserHandler) GetUser(c *fiber.Ctx) error { ... }
```

```bash
# Generate spec
swag init -g cmd/api/main.go --output docs/
```

### Python (FastAPI — automatic)

```python
from fastapi import FastAPI, Query, Path
from pydantic import BaseModel, EmailStr, UUID4

app = FastAPI(
    title="MyApp API",
    version="1.0.0",
    docs_url="/api-docs",
    redoc_url="/redoc",
)

class CreateUserRequest(BaseModel):
    email: EmailStr
    name: str = Field(min_length=1, max_length=100)

class User(BaseModel):
    id: UUID4
    email: EmailStr
    name: str
    created_at: datetime

@app.post("/v1/users", response_model=User, status_code=201, tags=["Users"],
          summary="Create a user", operation_id="createUser")
async def create_user(body: CreateUserRequest) -> User:
    """Create a new user account."""
    ...
```

### TypeScript (tsoa)

```typescript
import { Route, Get, Post, Body, Path, Query, Tags, Security, Response } from 'tsoa';

@Route('users')
@Tags('Users')
@Security('bearerAuth')
export class UserController extends Controller {
  @Get('{userId}')
  @Response<ErrorResponse>(404, 'Not found')
  @Response<ErrorResponse>(401, 'Unauthorized')
  public async getUser(@Path() userId: string): Promise<User> { ... }

  @Post()
  @Response<ErrorResponse>(400, 'Validation error')
  @Response<ErrorResponse>(409, 'Conflict')
  public async createUser(@Body() body: CreateUserRequest): Promise<User> {
    this.setStatus(201);
    ...
  }
}
```

```bash
# Generate routes and spec
npx tsoa routes
npx tsoa spec
```

## Swagger UI + ReDoc

```typescript
// Express
import swaggerUi from 'swagger-ui-express';
import YAML from 'yamljs';

const spec = YAML.load('./openapi.yaml');

// Swagger UI
app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(spec, {
  swaggerOptions: { persistAuthorization: true },
}));

// ReDoc (read-only, cleaner UI)
app.get('/redoc', (_req, res) => {
  res.send(`
    <!DOCTYPE html>
    <html>
      <head><title>API Docs</title></head>
      <body>
        <redoc spec-url="/openapi.json"></redoc>
        <script src="https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js"></script>
      </body>
    </html>
  `);
});

// Serve raw spec
app.get('/openapi.json', (_req, res) => res.json(spec));
```

## Client SDK Generation

```bash
# Install
npm install -g @openapitools/openapi-generator-cli

# Generate TypeScript fetch client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-fetch \
  -o ./src/generated/api \
  --additional-properties=typescriptThreePlus=true

# Generate Go client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g go \
  -o ./pkg/generated/api

# Generate Python client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g python \
  -o ./generated/api \
  --additional-properties=packageName=myapp_client
```

## Key Rules

- Use `$ref` for all reusable schemas — never inline the same shape twice.
- Define reusable `components/responses` for every standard error (400, 401, 403, 404, 409, 500).
- Always include `operationId` — it becomes the generated client method name.
- Add `example` values to schemas — they render in Swagger UI and improve developer experience.
- Version the spec file alongside the code in the same repository.
- Run `openapi-validator` or `spectral` in CI to catch spec errors before deployment.
- Serve `/api-docs` only in non-production environments or behind auth for sensitive APIs.
