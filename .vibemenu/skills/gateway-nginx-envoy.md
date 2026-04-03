# Nginx & Envoy Gateway Skill Guide

## Overview

Nginx is the industry-standard reverse proxy and load balancer. Envoy is a modern service mesh data plane with dynamic xDS configuration. Use Nginx for traditional deployments; Envoy for service mesh and dynamic microservice routing.

---

## Nginx

### Upstream with Least Connections

```nginx
# /etc/nginx/nginx.conf or included conf file

upstream api_servers {
    least_conn;                      # route to server with fewest active connections
    keepalive 32;                    # persistent connections to upstream

    server api1.internal:8080 weight=3;
    server api2.internal:8080 weight=2;
    server api3.internal:8080;
    server api4.internal:8080 backup; # only used if others fail

    # Health checks (nginx plus only — use third-party module for OSS)
    # check interval=3000 rise=2 fall=3 timeout=1000 type=http;
}

upstream static_servers {
    ip_hash;   # sticky sessions by client IP
    server static1.internal:8080;
    server static2.internal:8080;
}
```

### Location Blocks with proxy_pass

```nginx
server {
    listen 80;
    server_name api.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    # SSL
    ssl_certificate     /etc/ssl/certs/example.com.crt;
    ssl_certificate_key /etc/ssl/private/example.com.key;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 1d;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options DENY always;

    # Gzip
    gzip on;
    gzip_types application/json text/plain text/css application/javascript;
    gzip_min_length 1024;
    gzip_comp_level 6;

    # API proxy
    location /api/ {
        proxy_pass         http://api_servers;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   Connection        "";   # keep-alive to upstream

        proxy_connect_timeout 5s;
        proxy_read_timeout    30s;
        proxy_send_timeout    30s;

        proxy_buffering    on;
        proxy_buffer_size  4k;
        proxy_buffers      8 4k;

        # Pass correlation ID
        proxy_set_header   X-Request-ID $request_id;
    }

    # Static files
    location /static/ {
        root /var/www;
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    # Health check (no rate limiting)
    location /health {
        proxy_pass http://api_servers;
        access_log off;
    }
}
```

### Rate Limiting

```nginx
http {
    # Define rate limit zones
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/m;
    limit_req_zone $http_x_api_key     zone=key_limit:10m rate=1000r/m;

    # Zone for burst protection
    limit_req_zone $binary_remote_addr zone=strict:10m rate=10r/s;

    server {
        location /api/ {
            limit_req zone=api_limit burst=20 nodelay;
            limit_req_status 429;

            proxy_pass http://api_servers;
        }

        location /api/auth/ {
            limit_req zone=strict burst=5 nodelay;
            limit_req_status 429;

            proxy_pass http://api_servers;
        }
    }
}
```

### WebSocket Support

```nginx
location /ws/ {
    proxy_pass          http://ws_servers;
    proxy_http_version  1.1;
    proxy_set_header    Upgrade    $http_upgrade;
    proxy_set_header    Connection "upgrade";
    proxy_set_header    Host       $host;
    proxy_read_timeout  3600s;    # long-lived connections
}
```

---

## Envoy

### Static Bootstrap Configuration

```yaml
# envoy.yaml
static_resources:
  listeners:
    - name: main_listener
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 8080
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                codec_type: AUTO
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: api_service
                      domains: ["*"]
                      routes:
                        - match:
                            prefix: "/api/users"
                          route:
                            cluster: user_service
                            timeout: 30s
                            retry_policy:
                              retry_on: "5xx,reset,connect-failure"
                              num_retries: 3
                              per_try_timeout: 10s
                        - match:
                            prefix: "/api/orders"
                          route:
                            cluster: order_service
                            timeout: 30s
                http_filters:
                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router

  clusters:
    - name: user_service
      connect_timeout: 5s
      type: STRICT_DNS
      lb_policy: LEAST_REQUEST
      load_assignment:
        cluster_name: user_service
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: user-service
                      port_value: 8080
      health_checks:
        - timeout: 5s
          interval: 10s
          unhealthy_threshold: 3
          healthy_threshold: 1
          http_health_check:
            path: /health

    - name: order_service
      connect_timeout: 5s
      type: STRICT_DNS
      lb_policy: ROUND_ROBIN
      load_assignment:
        cluster_name: order_service
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: order-service
                      port_value: 8080

admin:
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 9901
```

### Circuit Breaker (Outlier Detection)

```yaml
clusters:
  - name: backend_service
    # Circuit breaker thresholds
    circuit_breakers:
      thresholds:
        - priority: DEFAULT
          max_connections: 1000
          max_pending_requests: 200
          max_requests: 1000
          max_retries: 10
          track_remaining: true

    # Outlier detection — eject unhealthy hosts
    outlier_detection:
      consecutive_5xx: 5                        # eject after 5 consecutive 5xx
      consecutive_gateway_failure: 5
      interval: 10s                             # check interval
      base_ejection_time: 30s                   # minimum ejection duration
      max_ejection_percent: 50                  # never eject more than 50% of hosts
      success_rate_minimum_hosts: 3             # need at least 3 hosts to analyze
      success_rate_stale_after_warmup_window: 300s
```

### Health Check Config

```yaml
health_checks:
  - timeout: 5s
    interval: 10s
    interval_jitter: 1s
    unhealthy_threshold: 3
    healthy_threshold: 1
    reuse_connection: true
    http_health_check:
      path: /health
      expected_statuses:
        - start: 200
          end: 299
    # TCP health check alternative:
    # tcp_health_check: {}
    # gRPC health check:
    # grpc_health_check:
    #   service_name: "grpc.health.v1.Health"
```

### Rate Limiting (Local)

```yaml
# In the http_connection_manager filter chain
http_filters:
  - name: envoy.filters.http.local_ratelimit
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
      stat_prefix: local_rate_limiter
      token_bucket:
        max_tokens: 100
        tokens_per_fill: 100
        fill_interval: 60s
      filter_enabled:
        runtime_key: local_rate_limit_enabled
        default_value:
          numerator: 100
          denominator: HUNDRED
      filter_enforced:
        runtime_key: local_rate_limit_enforced
        default_value:
          numerator: 100
          denominator: HUNDRED
      response_headers_to_add:
        - append: false
          header:
            key: x-local-rate-limit
            value: "true"
  - name: envoy.filters.http.router
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
```

## Nginx vs Envoy Comparison

| Feature | Nginx | Envoy |
|---------|-------|-------|
| Config reload | `nginx -s reload` (no downtime) | xDS API (fully dynamic) |
| Service discovery | Static upstream blocks | Dynamic via EDS |
| Observability | access_log, error_log | Prometheus metrics, distributed tracing |
| Protocol support | HTTP/1.1, HTTP/2, TCP, GRPC (basic) | HTTP/1.1, HTTP/2, HTTP/3, gRPC, Thrift |
| Use case | Edge proxy, static sites, traditional apps | Service mesh sidecar, dynamic microservices |

## Rules

- Set `proxy_http_version 1.1` and `Connection ""` in Nginx to enable keepalive to upstream
- Always set `proxy_read_timeout` for long-running requests (default 60s kills streaming endpoints)
- Use `limit_req burst=N nodelay` to allow short bursts without queuing
- In Envoy, set `max_ejection_percent` < 100 so the cluster never fully ejects all hosts
- Enable Envoy admin endpoint on localhost only — it exposes config, stats, and drain endpoints
- Use `STRICT_DNS` cluster type with service names for Docker/Kubernetes environments
