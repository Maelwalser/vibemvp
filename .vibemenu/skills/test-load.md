# Testing: Load Testing Skill Guide

## Overview

k6, Locust, and Artillery — VU scripts, scenarios, thresholds, ramp-up profiles, and CI integration.

## k6

### Basic Script

```javascript
// k6 run load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const createUserTrend = new Trend('create_user_duration');

export const options = {
  // Staged ramp-up scenario
  stages: [
    { duration: '30s', target: 10 },   // ramp up to 10 VUs
    { duration: '1m',  target: 50 },   // ramp to 50 VUs
    { duration: '2m',  target: 50 },   // stay at 50 VUs
    { duration: '30s', target: 0 },    // ramp down
  ],

  // Thresholds — test fails if violated
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],  // P95 < 500ms, P99 < 1s
    http_req_failed:   ['rate<0.01'],                 // error rate < 1%
    errors:            ['rate<0.01'],
    create_user_duration: ['p(99)<1000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';

export default function () {
  // Create user
  const payload = JSON.stringify({
    email: `user_${Date.now()}_${Math.random()}@example.com`,
    name: 'Load Test User',
  });

  const createRes = http.post(`${BASE_URL}/api/users`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'create_user' },
  });

  const createOK = check(createRes, {
    'create status 201': (r) => r.status === 201,
    'has user id': (r) => r.json('id') !== undefined,
  });
  errorRate.add(!createOK);
  createUserTrend.add(createRes.timings.duration);

  const userId = createRes.json('id');

  // Get user
  if (userId) {
    const getRes = http.get(`${BASE_URL}/api/users/${userId}`, {
      tags: { name: 'get_user' },
    });

    check(getRes, {
      'get status 200': (r) => r.status === 200,
    });
  }

  sleep(1);  // think time between iterations
}
```

### Named Scenarios

```javascript
export const options = {
  scenarios: {
    // Constant VU load
    constant_load: {
      executor: 'constant-vus',
      vus: 20,
      duration: '5m',
    },

    // Constant arrival rate (requests/second)
    rps_load: {
      executor: 'constant-arrival-rate',
      rate: 100,                // 100 requests/second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },

    // Ramping arrival rate (spike test)
    spike: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      stages: [
        { duration: '30s', target: 10 },
        { duration: '10s', target: 500 },  // spike
        { duration: '30s', target: 10 },   // recover
      ],
      preAllocatedVUs: 100,
      maxVUs: 1000,
    },
  },
};
```

### Setup / Teardown

```javascript
export function setup() {
  // Authenticate once before load test
  const res = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
    email: 'loadtest@example.com',
    password: 'password',
  }), { headers: { 'Content-Type': 'application/json' } });

  return { token: res.json('access_token') };
}

export default function (data) {
  http.get(`${BASE_URL}/api/protected`, {
    headers: { Authorization: `Bearer ${data.token}` },
  });
  sleep(1);
}
```

```bash
# Run locally
k6 run load-test.js

# Run with env vars
BASE_URL=https://staging.example.com k6 run load-test.js

# Output results to JSON
k6 run --out json=results.json load-test.js

# Output to InfluxDB + Grafana
k6 run --out influxdb=http://localhost:8086/k6 load-test.js
```

## Locust

### Basic Script

```python
# locustfile.py
from locust import HttpUser, task, between, events
import json
import random

class APIUser(HttpUser):
    host = "http://localhost:3000"
    wait_time = between(1, 3)  # think time 1–3 seconds

    def on_start(self):
        """Called once per user on start — perform login."""
        res = self.client.post("/api/auth/login", json={
            "email": "loadtest@example.com",
            "password": "password",
        })
        self.token = res.json()["access_token"]
        self.client.headers.update({"Authorization": f"Bearer {self.token}"})

    @task(3)  # weight: called 3x more than task(1)
    def get_users(self):
        with self.client.get("/api/users", name="/api/users", catch_response=True) as res:
            if res.status_code == 200:
                res.success()
            else:
                res.failure(f"Got {res.status_code}")

    @task(1)
    def create_user(self):
        payload = {
            "email": f"user_{random.randint(1,1000000)}@example.com",
            "name": "Load Test User",
        }
        with self.client.post("/api/users", json=payload, name="/api/users [POST]",
                               catch_response=True) as res:
            if res.status_code == 201:
                res.success()
            else:
                res.failure(f"Got {res.status_code}: {res.text}")
```

```bash
# Headless run (CI)
locust -f locustfile.py \
  --headless \
  --users 100 \
  --spawn-rate 10 \
  --run-time 5m \
  --host http://localhost:3000 \
  --html report.html \
  --csv results

# Web UI (interactive)
locust -f locustfile.py
# Open http://localhost:8089
```

### Failure Thresholds

```python
# Custom assertion on exit
import sys
from locust import events

@events.quitting.add_listener
def on_quitting(environment, **kwargs):
    if environment.stats.total.fail_ratio > 0.01:
        print("ERROR: fail ratio exceeds 1%")
        environment.process_exit_code = 1
    if environment.stats.total.avg_response_time > 500:
        print("ERROR: avg response time exceeds 500ms")
        environment.process_exit_code = 1
```

## Artillery

### YAML Configuration

```yaml
# load-test.yml
config:
  target: "http://localhost:3000"
  http:
    timeout: 10
  phases:
    - duration: 60
      arrivalRate: 10
      name: "Warm up"
    - duration: 120
      arrivalRate: 10
      rampTo: 50
      name: "Ramp up"
    - duration: 120
      arrivalRate: 50
      name: "Sustained load"
    - duration: 30
      arrivalRate: 10
      name: "Ramp down"
  ensure:
    p99: 1000   # P99 < 1000ms
    p95: 500    # P95 < 500ms
    maxErrorRate: 1  # error rate < 1%

scenarios:
  - name: "User CRUD flow"
    weight: 80
    flow:
      - post:
          url: "/api/auth/login"
          json:
            email: "loadtest@example.com"
            password: "password"
          capture:
            - json: "$.access_token"
              as: "token"
          expect:
            - statusCode: 200

      - get:
          url: "/api/users"
          headers:
            Authorization: "Bearer {{ token }}"
          expect:
            - statusCode: 200

      - post:
          url: "/api/users"
          headers:
            Authorization: "Bearer {{ token }}"
          json:
            email: "{{ $randomString }}@example.com"
            name: "Artillery User"
          expect:
            - statusCode: 201
          capture:
            - json: "$.id"
              as: "userId"

  - name: "Read-only flow"
    weight: 20
    flow:
      - get:
          url: "/api/users"
          expect:
            - statusCode: 200
```

```bash
# Install
npm install -g artillery

# Run test
artillery run load-test.yml

# Run and generate HTML report
artillery run load-test.yml --output results.json
artillery report results.json

# Quick test (CLI)
artillery quick --count 20 --num 10 http://localhost:3000/api/healthz
```

## Key Rules

- Always define `thresholds` / `ensure` — a load test without pass/fail criteria is just noise.
- Use named requests (`tags: { name: 'create_user' }` in k6, `name:` in Artillery) for readable reports.
- Start with a ramp-up phase — hitting peak load instantly hides warm-up effects.
- Parameterize user data (random emails, IDs) to avoid caching and uniqueness constraint errors.
- Run load tests against an isolated environment with production-like data volumes.
- For authenticated flows, capture tokens in `setup()` / `on_start` — do not authenticate on every request.
- Fail CI if P95 or error rate thresholds are violated; treat load tests as gates, not metrics only.
