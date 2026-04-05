# Testing: API Testing Skill Guide

## Overview

Bruno (`.bru` file format), Hurl (HTTP DSL), and Postman/Newman for CI — collection structure, assertions, variable capture, and authentication.

## Bruno

### Collection Structure

```
api-tests/
├── bruno.json
├── environments/
│   ├── local.bru
│   └── staging.bru
├── users/
│   ├── create-user.bru
│   ├── get-user.bru
│   └── delete-user.bru
└── orders/
    ├── create-order.bru
    └── list-orders.bru
```

### bruno.json

```json
{
  "version": "1",
  "name": "MyApp API Tests",
  "type": "collection"
}
```

### Environment File (local.bru)

```
vars {
  baseUrl: http://localhost:3000
  adminToken: dev-token-123
}
```

### Request File (create-user.bru)

```
meta {
  name: Create User
  type: http
  seq: 1
}

post {
  url: {{baseUrl}}/api/users
  body: json
  auth: bearer
}

auth:bearer {
  token: {{adminToken}}
}

headers {
  Content-Type: application/json
  X-Request-ID: {{$randomString}}
}

body:json {
  {
    "email": "alice@example.com",
    "name": "Alice"
  }
}

assert {
  res.status: eq 201
  res.body.id: isDefined
  res.body.email: eq alice@example.com
  res.responseTime: lt 500
}

script:post-response {
  bru.setVar("userId", res.body.id);
}
```

### Get User (uses captured variable)

```
meta {
  name: Get User
  type: http
  seq: 2
}

get {
  url: {{baseUrl}}/api/users/{{userId}}
  auth: bearer
}

auth:bearer {
  token: {{adminToken}}
}

assert {
  res.status: eq 200
  res.body.id: eq {{userId}}
  res.body.email: eq alice@example.com
}
```

### Run Bruno CLI

```bash
# Install
npm install -g @usebruno/cli

# Run entire collection
bru run --env local

# Run specific folder
bru run users/ --env staging

# Run with output
bru run --env local --reporter junit --output results.xml
```

## Hurl

### Basic Request File (users.hurl)

```hurl
# Create user
POST http://localhost:3000/api/users
Content-Type: application/json
Authorization: Bearer {{admin_token}}
{
  "email": "alice@example.com",
  "name": "Alice"
}

HTTP 201
[Asserts]
status == 201
header "Content-Type" contains "application/json"
jsonpath "$.id" exists
jsonpath "$.email" == "alice@example.com"
duration < 500

[Captures]
user_id: jsonpath "$.id"

---

# Get user using captured ID
GET http://localhost:3000/api/users/{{user_id}}
Authorization: Bearer {{admin_token}}

HTTP 200
[Asserts]
jsonpath "$.id" == {{user_id}}
jsonpath "$.email" == "alice@example.com"

---

# Delete user
DELETE http://localhost:3000/api/users/{{user_id}}
Authorization: Bearer {{admin_token}}

HTTP 204
```

### Authentication Flow

```hurl
# Login and capture token
POST http://localhost:3000/api/auth/login
Content-Type: application/json
{
  "email": "admin@example.com",
  "password": "{{admin_password}}"
}

HTTP 200
[Captures]
token: jsonpath "$.access_token"

---

# Use token in subsequent request
GET http://localhost:3000/api/protected
Authorization: Bearer {{token}}

HTTP 200
[Asserts]
jsonpath "$.data" exists
```

### Hurl Variables and Options

```bash
# Run with variables
hurl users.hurl \
  --variable admin_token=my-token \
  --variable admin_password=secret \
  --variable base_url=http://localhost:3000

# Run against different environments
hurl users.hurl --variables-file staging.env

# Output formats
hurl users.hurl --report-junit results.xml
hurl users.hurl --report-html reports/

# Verbose output
hurl users.hurl --very-verbose
```

```ini
# staging.env (variables file)
base_url=https://staging.example.com
admin_token=staging-token-xyz
admin_password=stagingpass
```

## Postman / Newman

### Collection JSON Structure (v2.1)

```json
{
  "info": {
    "name": "MyApp API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    { "key": "baseUrl", "value": "http://localhost:3000" }
  ],
  "item": [
    {
      "name": "Create User",
      "request": {
        "method": "POST",
        "url": "{{baseUrl}}/api/users",
        "header": [
          { "key": "Content-Type", "value": "application/json" },
          { "key": "Authorization", "value": "Bearer {{adminToken}}" }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\"email\": \"alice@example.com\", \"name\": \"Alice\"}",
          "options": { "raw": { "language": "json" } }
        }
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test('Status is 201', () => pm.response.to.have.status(201));",
              "pm.test('Has user ID', () => {",
              "  const body = pm.response.json();",
              "  pm.expect(body.id).to.be.a('string');",
              "  pm.collectionVariables.set('userId', body.id);",
              "});"
            ]
          }
        }
      ]
    }
  ]
}
```

### Newman CLI (CI Integration)

```bash
# Install
npm install -g newman newman-reporter-htmlextra

# Run collection
newman run collection.json \
  --environment environments/staging.json \
  --reporters cli,junit,htmlextra \
  --reporter-junit-export results.xml \
  --reporter-htmlextra-export reports/index.html \
  --bail  # stop on first failure in CI
```

### GitHub Actions Step

```yaml
- name: Run API tests
  run: |
    newman run tests/api/collection.json \
      --environment tests/api/environments/staging.json \
      --reporters cli,junit \
      --reporter-junit-export test-results/api-results.xml

- name: Upload test results
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: api-test-results
    path: test-results/
```

## Key Rules

- Use variable capture (`bru.setVar` / `[Captures]` / `pm.collectionVariables.set`) to chain requests — never hardcode IDs.
- Always assert both the status code AND key response fields — a 200 with an empty body is a bug.
- Store environment files separately from collection files — never commit credentials.
- Run API tests against a running service, not mocks — they must exercise the real HTTP layer.
- Use `--bail` in Newman CI runs to stop on first failure and report quickly.
- Add response time assertions (`res.responseTime: lt 500` / `duration < 500`) as performance regression guards.
- For authenticated flows, capture the token in the first request and pass it to all subsequent requests.
