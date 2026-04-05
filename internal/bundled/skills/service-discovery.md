# Service Discovery Skill Guide

## Overview

Service discovery lets services find each other's network addresses without hardcoding IPs. Strategies: DNS-based (Kubernetes built-in), registry-based (Consul, Eureka), and static config (simplest, for small deployments).

## Kubernetes DNS-Based Discovery

Kubernetes automatically creates DNS entries for every Service. Services find each other by name.

### DNS Format

```
<service-name>.<namespace>.svc.cluster.local
```

| Form | Resolves To | Usable From |
|------|------------|-------------|
| `user-service` | ClusterIP | Same namespace |
| `user-service.default` | ClusterIP | Any namespace |
| `user-service.default.svc.cluster.local` | ClusterIP | Any namespace (fully qualified) |

### ClusterIP Service (Load Balanced)

```yaml
# Standard service — DNS resolves to stable ClusterIP, kube-proxy load balances
apiVersion: v1
kind: Service
metadata:
  name: user-service
  namespace: default
spec:
  selector:
    app: user-service
  ports:
    - name: http
      port: 8080
      targetPort: 8080
  type: ClusterIP
```

```go
// Client just uses the service name
resp, err := http.Get("http://user-service:8080/users")

// Or with namespace:
resp, err := http.Get("http://user-service.payments.svc.cluster.local:8080/users")
```

### Headless Service (Direct Pod IPs)

Use when you need the actual pod IPs (e.g., for StatefulSets, databases with leader election, or gRPC client-side load balancing).

```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres-headless
spec:
  clusterIP: None   # headless — DNS returns all pod IPs
  selector:
    app: postgres
  ports:
    - port: 5432
```

```
# DNS query for headless service returns multiple A records (one per pod)
postgres-headless.default.svc.cluster.local → [10.0.1.5, 10.0.1.6, 10.0.1.7]

# StatefulSet pods also get stable DNS entries:
postgres-0.postgres-headless.default.svc.cluster.local → 10.0.1.5
postgres-1.postgres-headless.default.svc.cluster.local → 10.0.1.6
```

### Environment Variable Discovery (Legacy)

Kubernetes injects env vars for each service in the same namespace:

```bash
# Automatically injected for a service named "user-service"
USER_SERVICE_SERVICE_HOST=10.96.0.1
USER_SERVICE_SERVICE_PORT=8080
USER_SERVICE_PORT=tcp://10.96.0.1:8080
```

Prefer DNS over env vars — env vars are only set for services that existed when the pod started.

---

## Consul Service Registry

### Service Registration

```hcl
# agent/services/user-service.hcl (file-based config)
service {
  name = "user-service"
  id   = "user-service-1"
  port = 8080
  tags = ["v1", "api"]

  meta {
    version = "1.2.3"
    region  = "us-east-1"
  }

  check {
    id       = "user-service-health"
    name     = "HTTP health check"
    http     = "http://localhost:8080/health"
    interval = "10s"
    timeout  = "5s"
    deregister_critical_service_after = "30s"
  }
}
```

```bash
# Or register via API
curl -X PUT http://consul:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "user-service",
    "Port": 8080,
    "Check": {
      "HTTP": "http://localhost:8080/health",
      "Interval": "10s"
    }
  }'
```

### DNS Lookup via Consul

```bash
# Consul exposes DNS on port 8600
# Format: <service>.service.consul
dig @consul-dns:8600 user-service.service.consul

# Filter by tag
dig @consul-dns:8600 v1.user-service.service.consul

# SRV records include port
dig @consul-dns:8600 user-service.service.consul SRV
```

### Consul HTTP API Discovery

```go
import consul "github.com/hashicorp/consul/api"

client, _ := consul.NewClient(consul.DefaultConfig())

// Find healthy instances
services, _, err := client.Health().Service("user-service", "v1", true, nil)
if err != nil || len(services) == 0 {
    return fmt.Errorf("no healthy user-service instances")
}

// Pick one (simple round-robin or use a load balancer)
svc := services[0].Service
addr := fmt.Sprintf("http://%s:%d", svc.Address, svc.Port)
```

```typescript
// Node.js Consul client
import Consul from "consul";

const consul = new Consul({ host: "consul", port: 8500 });

async function discoverService(name: string): Promise<string> {
  const services = await consul.health.service({ service: name, passing: true });
  if (!services.length) throw new Error(`No healthy instances of ${name}`);
  const svc = services[Math.floor(Math.random() * services.length)].Service;
  return `http://${svc.Address}:${svc.Port}`;
}
```

---

## Eureka (Spring Cloud)

```xml
<!-- pom.xml -->
<dependency>
  <groupId>org.springframework.cloud</groupId>
  <artifactId>spring-cloud-starter-netflix-eureka-client</artifactId>
</dependency>
```

```yaml
# application.yml
spring:
  application:
    name: user-service

eureka:
  client:
    service-url:
      defaultZone: http://eureka-server:8761/eureka/
  instance:
    prefer-ip-address: true
    lease-renewal-interval-in-seconds: 10
    lease-expiration-duration-in-seconds: 30
```

```java
// Enable on main class
@SpringBootApplication
@EnableDiscoveryClient
public class UserServiceApplication {
    public static void main(String[] args) {
        SpringApplication.run(UserServiceApplication.class, args);
    }
}

// Call another service with load balancing via FeignClient
@FeignClient(name = "order-service")  // name matches spring.application.name
public interface OrderServiceClient {
    @GetMapping("/api/v1/orders/{orderId}")
    OrderDto getOrder(@PathVariable String orderId);
}

// Or inject LoadBalancerClient manually
@Autowired
private LoadBalancerClient loadBalancer;

public String getOrderServiceUrl() {
    ServiceInstance instance = loadBalancer.choose("order-service");
    return instance.getUri().toString();
}
```

---

## Static Configuration Patterns

For small deployments, local dev, or services that don't change often.

```yaml
# docker-compose.yml — Docker DNS handles service names
services:
  api:
    environment:
      USER_SERVICE_URL: http://user-service:8080
      ORDER_SERVICE_URL: http://order-service:8080

  user-service:
    image: user-service:latest
  order-service:
    image: order-service:latest
```

```go
// Go — read from environment
type Config struct {
    UserServiceURL  string
    OrderServiceURL string
}

func LoadConfig() Config {
    return Config{
        UserServiceURL:  mustEnv("USER_SERVICE_URL"),
        OrderServiceURL: mustEnv("ORDER_SERVICE_URL"),
    }
}

func mustEnv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("required env var %s not set", key)
    }
    return v
}
```

## Sidecar Proxy (Service Mesh)

In Istio/Linkerd, the application calls `localhost` — the sidecar proxy handles discovery, mTLS, and load balancing transparently.

```yaml
# No service discovery code needed in the app
# Just call the service by its Kubernetes Service name
http.Get("http://user-service:8080/users")

# Istio VirtualService controls routing rules
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: user-service
spec:
  hosts: [user-service]
  http:
    - route:
        - destination:
            host: user-service
            subset: v1
          weight: 90
        - destination:
            host: user-service
            subset: v2
          weight: 10       # canary 10% to v2
```

## Rules

- Prefer DNS-based discovery (Kubernetes built-in) for Kubernetes deployments — zero code required
- Use headless services with StatefulSets so gRPC clients can do client-side load balancing across pods
- Always register health check endpoints and use them in service registry configurations
- Deregister services on graceful shutdown — don't rely on TTL-based expiry alone
- Cache service endpoint lookups briefly (1–5s) to avoid DNS/registry overload on high-traffic services
- Never hardcode IP addresses in application config — always use service names that DNS resolves
