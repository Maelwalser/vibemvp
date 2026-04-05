# CORS Configuration Skill Guide

## Overview

CORS (Cross-Origin Resource Sharing) controls which origins can make cross-origin HTTP requests from browsers. The browser enforces CORS — not the server. Misconfiguration either blocks legitimate requests or opens security holes.

## Key Concepts

- **Simple requests**: GET/POST with safe headers — browser sends directly, checks `Access-Control-Allow-Origin` in response
- **Preflight**: Non-simple requests (PUT/DELETE, custom headers, JSON content-type) — browser sends OPTIONS first
- **Credentials**: Cookies and `Authorization` header require `withCredentials: true` on client AND `Access-Control-Allow-Credentials: true` on server — wildcard origin `*` cannot be used with credentials

## Headers Reference

```
# Server response headers
Access-Control-Allow-Origin: https://app.example.com   # or * (never use * with credentials)
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID
Access-Control-Expose-Headers: X-Request-ID, X-Rate-Limit-Remaining
Access-Control-Allow-Credentials: true                 # only with specific origin, not *
Access-Control-Max-Age: 600                            # preflight cache in seconds (10 min)

# Browser request headers (set automatically)
Origin: https://app.example.com
Access-Control-Request-Method: PUT           # in preflight
Access-Control-Request-Headers: Authorization # in preflight
```

## Origin Allowlist vs Wildcard

```
# Wildcard — allows any origin (NEVER use with credentials)
Access-Control-Allow-Origin: *

# Specific origin — required for credentialed requests
Access-Control-Allow-Origin: https://app.example.com

# Multiple origins — validate dynamically and echo back the matched origin
if origin in ALLOWED_ORIGINS:
    Access-Control-Allow-Origin: <origin>
    Vary: Origin          # CRITICAL: tells caches this response varies by origin
```

## Per-Framework Middleware

### Express (TypeScript) — cors package

```typescript
import cors from "cors";
import express from "express";

const ALLOWED_ORIGINS = [
  "https://app.example.com",
  "https://admin.example.com",
  process.env.NODE_ENV === "development" ? "http://localhost:3000" : "",
].filter(Boolean);

const app = express();

app.use(cors({
  origin: (origin, callback) => {
    // Allow non-browser requests (curl, server-to-server) — origin is undefined
    if (!origin || ALLOWED_ORIGINS.includes(origin)) {
      callback(null, true);
    } else {
      callback(new Error(`CORS: origin ${origin} not allowed`));
    }
  },
  methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
  allowedHeaders: ["Authorization", "Content-Type", "X-Request-ID"],
  exposedHeaders: ["X-Request-ID", "X-Rate-Limit-Remaining"],
  credentials: true,         // allow cookies / Authorization header
  maxAge: 600,               // preflight cache 10 minutes
  optionsSuccessStatus: 204, // some legacy browsers expect 204 for OPTIONS
}));

// Explicit preflight handler (cors() handles this, but explicit is clearer)
app.options("*", cors());
```

### FastAPI (Python) — CORSMiddleware

```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import os

app = FastAPI()

ALLOWED_ORIGINS = [
    "https://app.example.com",
    "https://admin.example.com",
]
if os.getenv("ENVIRONMENT") == "development":
    ALLOWED_ORIGINS.append("http://localhost:3000")

app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
    allow_headers=["Authorization", "Content-Type", "X-Request-ID"],
    expose_headers=["X-Request-ID", "X-Rate-Limit-Remaining"],
    max_age=600,
)
```

### Go (net/http) — rs/cors

```go
import "github.com/rs/cors"

allowedOrigins := []string{
    "https://app.example.com",
    "https://admin.example.com",
}
if os.Getenv("ENVIRONMENT") == "development" {
    allowedOrigins = append(allowedOrigins, "http://localhost:3000")
}

c := cors.New(cors.Options{
    AllowedOrigins:   allowedOrigins,
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
    ExposedHeaders:   []string{"X-Request-ID", "X-Rate-Limit-Remaining"},
    AllowCredentials: true,
    MaxAge:           600,
    Debug:            os.Getenv("ENVIRONMENT") == "development",
})

handler := c.Handler(router)
http.ListenAndServe(":8080", handler)
```

### Spring Boot (Java)

```java
@Configuration
public class CorsConfig {

    @Value("${cors.allowed-origins}")
    private List<String> allowedOrigins;

    @Bean
    public CorsConfigurationSource corsConfigurationSource() {
        CorsConfiguration config = new CorsConfiguration();
        config.setAllowedOrigins(allowedOrigins);
        config.setAllowedMethods(List.of("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"));
        config.setAllowedHeaders(List.of("Authorization", "Content-Type", "X-Request-ID"));
        config.setExposedHeaders(List.of("X-Request-ID", "X-Rate-Limit-Remaining"));
        config.setAllowCredentials(true);
        config.setMaxAge(600L);

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", config);
        return source;
    }

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http.cors(cors -> cors.configurationSource(corsConfigurationSource()));
        // ... rest of security config
        return http.build();
    }
}
```

### Go Fiber

```go
import "github.com/gofiber/fiber/v2/middleware/cors"

app.Use(cors.New(cors.Config{
    AllowOrigins:     "https://app.example.com,https://admin.example.com",
    AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
    AllowHeaders:     "Authorization,Content-Type,X-Request-ID",
    ExposeHeaders:    "X-Request-ID,X-Rate-Limit-Remaining",
    AllowCredentials: true,
    MaxAge:           600,
}))
```

## Nginx CORS Headers

```nginx
# nginx.conf — CORS for API proxy
location /api/ {
    # Handle preflight
    if ($request_method = 'OPTIONS') {
        add_header 'Access-Control-Allow-Origin' '$http_origin' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, X-Request-ID' always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Max-Age' '600' always;
        add_header 'Vary' 'Origin' always;
        return 204;
    }

    # Add CORS headers to all responses
    add_header 'Access-Control-Allow-Origin' '$http_origin' always;
    add_header 'Access-Control-Allow-Credentials' 'true' always;
    add_header 'Access-Control-Expose-Headers' 'X-Request-ID, X-Rate-Limit-Remaining' always;
    add_header 'Vary' 'Origin' always;

    proxy_pass http://api_servers;
}
```

## Common CORS Mistakes

```
# MISTAKE 1: Wildcard with credentials
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
→ Browser rejects this combination. Use specific origin.

# MISTAKE 2: Missing Vary: Origin header
Access-Control-Allow-Origin: https://app.example.com
# (no Vary header)
→ Shared caches may return wrong origin response to different clients.
Always add: Vary: Origin

# MISTAKE 3: Trusting the Origin header alone for security
if request.headers.origin in allowed_origins:
    allow()
→ CORS is not an authentication mechanism. It only controls browser behavior.
   Server-to-server requests bypass CORS entirely.

# MISTAKE 4: Broad wildcard on allowed headers
Access-Control-Allow-Headers: *
→ Not supported in all browsers for credentialed requests. List headers explicitly.
```

## Rules

- Always add `Vary: Origin` when echoing back a dynamic allowed origin to prevent cache poisoning
- Never use `Access-Control-Allow-Origin: *` with `Access-Control-Allow-Credentials: true` — browsers block it
- CORS is enforced by the browser only — it does not protect server-to-server calls or `curl`
- Return 204 (not 200) for preflight OPTIONS responses to minimize response size
- Configure `maxAge`/`max-age` to cache preflights for 5–10 minutes to reduce OPTIONS requests
- Maintain an explicit allowlist of origins from environment config — never derive allowed origins from request headers dynamically without validation
- In development, only add `localhost` origins when `NODE_ENV === "development"` — never ship it to production
