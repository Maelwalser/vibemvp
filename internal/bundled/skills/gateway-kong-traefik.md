# Kong & Traefik Gateway Skill Guide

## Overview

API gateways centralize cross-cutting concerns: routing, rate limiting, auth, TLS termination, and observability. Kong is plugin-rich and database-backed; Traefik is cloud-native and auto-discovers services from Docker/Kubernetes labels.

---

## Kong

### Declarative Configuration (deck / kong.yml)

```yaml
# kong.yml — managed with: deck sync
_format_version: "3.0"
_transform: true

services:
  - name: user-service
    url: http://user-service:8080
    connect_timeout: 5000
    read_timeout: 30000
    write_timeout: 30000
    routes:
      - name: user-routes
        paths:
          - /api/v1/users
        methods:
          - GET
          - POST
          - PUT
          - DELETE
        strip_path: false
        preserve_host: false

  - name: order-service
    url: http://order-service:8080
    routes:
      - name: order-routes
        paths:
          - /api/v1/orders
        methods:
          - GET
          - POST

# Global plugins (apply to all services)
plugins:
  - name: correlation-id
    config:
      header_name: X-Correlation-ID
      generator: uuid#counter
      echo_downstream: true

  - name: request-size-limiting
    config:
      allowed_payload_size: 8   # MB

# Consumers (for API key / JWT auth)
consumers:
  - username: mobile-app
    keyauth_credentials:
      - key: sk_live_mobile_abc123
  - username: partner-api
    keyauth_credentials:
      - key: sk_live_partner_xyz789
```

### Rate Limiting Plugin

```yaml
plugins:
  # Per-service rate limiting
  - name: rate-limiting
    service: user-service
    config:
      minute: 100
      hour: 5000
      policy: redis          # local | redis | cluster
      redis_host: redis
      redis_port: 6379
      hide_client_headers: false
      error_code: 429
      error_message: "Rate limit exceeded"

  # Per-consumer (API key) rate limiting
  - name: rate-limiting
    consumer: mobile-app
    config:
      minute: 500
      policy: redis
      redis_host: redis
      redis_port: 6379
```

### JWT Plugin

```yaml
plugins:
  - name: jwt
    service: order-service
    config:
      secret_is_base64: false
      claims_to_verify:
        - exp
        - nbf
      key_claim_name: iss          # which claim holds the consumer key
      run_on_preflight: false       # skip OPTIONS for CORS

# Register consumer JWT credential
consumers:
  - username: web-app
    jwt_secrets:
      - key: web-app-issuer         # must match iss claim in token
        secret: jwt-signing-secret
        algorithm: HS256
```

### Proxy Cache Plugin

```yaml
plugins:
  - name: proxy-cache
    service: user-service
    config:
      response_code:
        - 200
        - 301
      request_method:
        - GET
        - HEAD
      content_type:
        - application/json
      cache_ttl: 300           # seconds
      strategy: memory         # memory | redis
      cache_control: true      # respect Cache-Control headers
```

### Docker Compose + Kong

```yaml
# docker-compose.yml
services:
  kong-db:
    image: postgres:15
    environment:
      POSTGRES_DB: kong
      POSTGRES_USER: kong
      POSTGRES_PASSWORD: kong

  kong-migration:
    image: kong:3.5
    command: kong migrations bootstrap
    environment:
      KONG_DATABASE: postgres
      KONG_PG_HOST: kong-db
    depends_on: [kong-db]

  kong:
    image: kong:3.5
    environment:
      KONG_DATABASE: postgres
      KONG_PG_HOST: kong-db
      KONG_PROXY_ACCESS_LOG: /dev/stdout
      KONG_ADMIN_ACCESS_LOG: /dev/stdout
      KONG_PROXY_ERROR_LOG: /dev/stderr
      KONG_ADMIN_LISTEN: 0.0.0.0:8001
    ports:
      - "8000:8000"   # proxy
      - "8443:8443"   # proxy TLS
      - "8001:8001"   # admin API
    depends_on: [kong-migration]
```

---

## Traefik

### Static Configuration (traefik.yml)

```yaml
# traefik.yml
global:
  checkNewVersion: false
  sendAnonymousUsage: false

log:
  level: INFO

api:
  dashboard: true
  insecure: false   # only expose dashboard via router with auth

entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false    # require explicit opt-in via labels
    network: traefik-net
  file:
    directory: /etc/traefik/dynamic
    watch: true

certificatesResolvers:
  letsencrypt:
    acme:
      email: ops@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
```

### Dynamic Config (File Provider)

```yaml
# /etc/traefik/dynamic/middlewares.yml
http:
  middlewares:
    rate-limit:
      rateLimit:
        average: 100
        burst: 50

    auth-headers:
      headers:
        customRequestHeaders:
          X-Internal: "true"

    compress:
      compress: {}

    retry:
      retry:
        attempts: 3
        initialInterval: "100ms"

    strip-api-prefix:
      stripPrefix:
        prefixes:
          - /api

  # Manual service + router if not using Docker labels
  services:
    legacy-api:
      loadBalancer:
        servers:
          - url: "http://legacy:9000"

  routers:
    legacy-router:
      entryPoints: [websecure]
      rule: "Host(`api.example.com`) && PathPrefix(`/legacy`)"
      middlewares: [rate-limit, compress]
      service: legacy-api
      tls:
        certResolver: letsencrypt
```

### Docker Labels

```yaml
# docker-compose.yml
services:
  api:
    image: my-api:latest
    networks:
      - traefik-net
    labels:
      - "traefik.enable=true"

      # Router
      - "traefik.http.routers.api.rule=Host(`api.example.com`)"
      - "traefik.http.routers.api.entrypoints=websecure"
      - "traefik.http.routers.api.tls.certresolver=letsencrypt"

      # Middlewares
      - "traefik.http.routers.api.middlewares=rate-limit@file,compress@file"

      # Service (port)
      - "traefik.http.services.api.loadbalancer.server.port=8080"

      # Health check
      - "traefik.http.services.api.loadbalancer.healthcheck.path=/health"
      - "traefik.http.services.api.loadbalancer.healthcheck.interval=10s"

  traefik:
    image: traefik:v3.0
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik.yml:/traefik.yml:ro
      - ./dynamic:/etc/traefik/dynamic:ro
      - letsencrypt:/letsencrypt

networks:
  traefik-net:
    external: true
```

### Middleware Chains (Traefik)

```yaml
# Chain multiple middlewares in order
http:
  middlewares:
    secure-chain:
      chain:
        middlewares:
          - rate-limit
          - auth-headers
          - compress
```

### Kubernetes IngressRoute CRD

```yaml
# Traefik CRD for Kubernetes
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: api-route
  namespace: default
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`api.example.com`) && PathPrefix(`/api/v1`)
      kind: Rule
      services:
        - name: user-service
          port: 8080
      middlewares:
        - name: rate-limit-middleware
  tls:
    certResolver: letsencrypt

---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: rate-limit-middleware
spec:
  rateLimit:
    average: 100
    burst: 50
```

## Rules

- Always use declarative config (deck for Kong, file/labels for Traefik) — avoid manual Admin API calls in production
- Set health check paths on every upstream service so the gateway stops routing to unhealthy instances
- Use Redis as the rate limiting backend when running multiple gateway instances (not in-memory)
- In Traefik, always set `exposedByDefault: false` to require explicit opt-in per container
- Kong: put global plugins (correlation ID, request size) at the service or global level, not per-route, to avoid duplication
- Traefik: put the Traefik container on a dedicated network and only attach app containers to `traefik-net`
- Never expose the Kong Admin API (port 8001) or Traefik Dashboard publicly without authentication
