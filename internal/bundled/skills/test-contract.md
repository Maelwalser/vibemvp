# Testing: Contract Testing Skill Guide

## Overview

Pact consumer/provider contract testing, Schemathesis property-based API testing from OpenAPI specs, and Dredd API Blueprint testing with hooks.

## Pact

### Consumer Side (TypeScript)

```typescript
// user-client.pact.spec.ts
import { PactV3, MatchersV3 } from '@pact-foundation/pact';
import { UserApiClient } from './user-api-client';

const { like, string, integer } = MatchersV3;

const provider = new PactV3({
  consumer: 'frontend-app',
  provider: 'user-service',
  dir: './pacts',  // pact files saved here
});

describe('UserApiClient', () => {
  describe('getUser', () => {
    it('returns a user when found', async () => {
      await provider
        .given('user with id abc123 exists')
        .uponReceiving('a request to get user abc123')
        .withRequest({
          method: 'GET',
          path: '/api/users/abc123',
          headers: { Authorization: like('Bearer some-token') },
        })
        .willRespondWith({
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: {
            id: string('abc123'),
            email: string('alice@example.com'),
            name: string('Alice'),
            createdAt: string('2024-01-15T12:00:00Z'),
          },
        })
        .executeTest(async (mockServer) => {
          const client = new UserApiClient(mockServer.url, 'Bearer some-token');
          const user = await client.getUser('abc123');

          expect(user.id).toBe('abc123');
          expect(user.email).toBe('alice@example.com');
        });
    });

    it('returns 404 when user not found', async () => {
      await provider
        .given('user with id unknown does not exist')
        .uponReceiving('a request for a non-existent user')
        .withRequest({
          method: 'GET',
          path: '/api/users/unknown',
        })
        .willRespondWith({
          status: 404,
          body: { error: string('user not found') },
        })
        .executeTest(async (mockServer) => {
          const client = new UserApiClient(mockServer.url, '');
          await expect(client.getUser('unknown')).rejects.toThrow('user not found');
        });
    });
  });
});
```

### Publish Pact to Broker

```bash
# Install pact-broker CLI
npm install -g @pact-foundation/pact-cli

# Publish pact files
pact-broker publish ./pacts \
  --broker-base-url https://your-broker.pactflow.io \
  --broker-token $PACT_BROKER_TOKEN \
  --consumer-app-version $GIT_SHA \
  --branch $GIT_BRANCH
```

### Provider Verification (TypeScript)

```typescript
// user-service.pact.provider.spec.ts
import { Verifier } from '@pact-foundation/pact';
import app from '../src/app';

describe('Pact Provider Verification', () => {
  let server: http.Server;

  beforeAll(async () => {
    server = app.listen(3001);
  });

  afterAll(() => server.close());

  it('validates consumer contracts', async () => {
    await new Verifier({
      provider: 'user-service',
      providerBaseUrl: 'http://localhost:3001',
      pactBrokerUrl: 'https://your-broker.pactflow.io',
      pactBrokerToken: process.env.PACT_BROKER_TOKEN,
      publishVerificationResult: true,
      providerVersion: process.env.GIT_SHA,
      providerVersionBranch: process.env.GIT_BRANCH,
      // Provider states — set up test data
      stateHandlers: {
        'user with id abc123 exists': async () => {
          await db.query(
            "INSERT INTO users (id, email, name) VALUES ('abc123', 'alice@example.com', 'Alice') ON CONFLICT DO NOTHING"
          );
        },
        'user with id unknown does not exist': async () => {
          await db.query("DELETE FROM users WHERE id = 'unknown'");
        },
      },
    }).verifyProvider();
  });
});
```

### Pact — Python (pact-python)

```python
import pytest
from pact import Consumer, Provider, Like, Term

@pytest.fixture(scope="module")
def pact():
    p = Consumer("frontend").has_pact_with(
        Provider("user-service"),
        pact_dir="./pacts",
    )
    p.start_service()
    yield p
    p.stop_service()

def test_get_user(pact):
    (pact
     .given("user abc123 exists")
     .upon_receiving("a GET request for user abc123")
     .with_request("GET", "/api/users/abc123")
     .will_respond_with(200, body={
         "id": Like("abc123"),
         "email": Like("alice@example.com"),
     }))

    with pact:
        result = UserApiClient(pact.uri).get_user("abc123")

    assert result["id"] == "abc123"
```

## Schemathesis

### CLI Usage

```bash
# Install
pip install schemathesis

# Run all checks against OpenAPI spec
schemathesis run http://localhost:3000/openapi.json \
  --checks all \
  --auth "Bearer $API_TOKEN" \
  --header "X-Request-ID: test-{uuid4}"

# From local spec file
schemathesis run openapi.yaml \
  --base-url http://localhost:3000 \
  --checks all

# Stateful testing: follows API links between operations
schemathesis run http://localhost:3000/openapi.json \
  --stateful=links \
  --checks all

# Specific operations only
schemathesis run http://localhost:3000/openapi.json \
  --endpoint /api/users \
  --method POST,GET

# JUnit output for CI
schemathesis run http://localhost:3000/openapi.json \
  --checks all \
  --junit-xml results.xml
```

### Checks Available

| Check | What it validates |
|-------|-----------------|
| `not_a_server_error` | No 5xx responses |
| `status_code_conformance` | Status code matches spec |
| `content_type_conformance` | Content-Type matches spec |
| `response_schema_conformance` | Response body matches JSON schema |
| `response_headers_conformance` | Response headers match spec |

### Python Integration Test

```python
import schemathesis
from schemathesis import DataGenerationMethod

schema = schemathesis.from_uri(
    "http://localhost:3000/openapi.json",
    data_generation_methods=[DataGenerationMethod.positive, DataGenerationMethod.negative],
)

@schema.parametrize()
def test_api(case):
    case.headers = case.headers or {}
    case.headers["Authorization"] = f"Bearer {TEST_TOKEN}"

    response = case.call()
    case.validate_response(response)
```

## Dredd

### dredd.yml Configuration

```yaml
# dredd.yml
dry-run: false
hookfiles: ./dredd-hooks.js
language: nodejs
sandbox: false
server: npm run start:test
server-wait: 3
endpoint: http://localhost:3000
path: []
blueprint: openapi.yaml
reporter: [junit]
output: [dredd-results.xml]
header: []
sorted: false
```

### Hooks File (JavaScript)

```javascript
// dredd-hooks.js
const hooks = require('hooks');
let authToken = '';
let createdUserId = '';

// Before entire test suite
hooks.beforeAll(async (transactions, done) => {
  const res = await fetch('http://localhost:3000/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: 'test@example.com', password: 'password' }),
  });
  const body = await res.json();
  authToken = body.access_token;
  done();
});

// Before each transaction — inject auth header
hooks.beforeEach((transaction, done) => {
  transaction.request.headers['Authorization'] = `Bearer ${authToken}`;
  done();
});

// Capture created resource ID
hooks.after('Users > Create User > 201', (transaction, done) => {
  const body = JSON.parse(transaction.real.body);
  createdUserId = body.id;
  done();
});

// Inject captured ID into subsequent request
hooks.before('Users > Get User > 200', (transaction, done) => {
  transaction.fullPath = `/api/users/${createdUserId}`;
  done();
});

// Teardown: delete test data
hooks.afterAll(async (transactions, done) => {
  if (createdUserId) {
    await fetch(`http://localhost:3000/api/users/${createdUserId}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${authToken}` },
    });
  }
  done();
});
```

```bash
# Install and run
npm install -g dredd
dredd
```

## Key Rules

- **Pact**: consumer tests produce the pact file; provider tests verify it. Run both in CI.
- **Pact broker**: always publish with a git SHA as `consumerAppVersion` — enables can-i-deploy checks.
- **Provider states**: must set up real database state, not mocks — the provider test hits the real service.
- **Schemathesis**: run `--stateful=links` to test response link chains (create→get→update→delete).
- **Schemathesis**: add to CI as a nightly job against staging — not a blocker on every PR.
- **Dredd**: use hooks to chain dependent API calls; avoid relying on data that already exists in the DB.
- Contract tests complement, not replace, integration tests — they test the interface, not the implementation.
