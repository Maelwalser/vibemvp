# Modular Monolith Architecture Skill Guide

## Overview

A modular monolith is a single deployable unit where each module is a self-contained vertical slice with explicit public API boundaries. Modules communicate through interfaces, not direct package imports. The goal is microservice-level decoupling without the operational overhead of distributed systems.

## Project Layout

```
service/
├── cmd/
│   └── server/
│       └── main.go              # Wires module registry, starts server
├── internal/
│   ├── modules/
│   │   ├── users/
│   │   │   ├── module.go        # Module registration: routes, handlers, repos
│   │   │   ├── api.go           # Public interface — what other modules may call
│   │   │   ├── handler.go       # HTTP handlers (private)
│   │   │   ├── service.go       # Business logic (private)
│   │   │   ├── repository.go    # Data access (private)
│   │   │   └── domain.go        # Types exported through api.go only
│   │   ├── orders/
│   │   │   ├── module.go
│   │   │   ├── api.go
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   └── notifications/
│   │       ├── module.go
│   │       └── api.go
│   ├── shared/
│   │   ├── db.go                # Shared DB pool
│   │   └── eventbus/            # In-process event bus
│   │       └── bus.go
│   └── registry/
│       └── registry.go          # Module registry
└── Dockerfile
```

## Module Interface Definition

Each module exposes a narrow public API. Other modules depend only on the interface:

```go
// internal/modules/users/api.go  — the public contract
package users

import "context"

// UserAPI is the only thing other modules import from this package.
type UserAPI interface {
    GetUser(ctx context.Context, id string) (UserInfo, error)
    ValidateCredentials(ctx context.Context, email, password string) (UserInfo, error)
}

// UserInfo is the exported DTO — not the internal domain.User struct.
type UserInfo struct {
    ID    string
    Email string
    Role  string
}

// internal/modules/users/service.go — implements UserAPI
type Service struct { repo Repository }

func (s *Service) GetUser(ctx context.Context, id string) (UserInfo, error) { ... }
func (s *Service) ValidateCredentials(ctx context.Context, email, password string) (UserInfo, error) { ... }
```

## Module Registry Pattern

The registry wires all modules together at startup. Modules register themselves:

```go
// internal/registry/registry.go
package registry

type Module interface {
    Name() string
    RegisterRoutes(router Router)
}

type Registry struct {
    modules []Module
}

func (r *Registry) Register(m Module) {
    r.modules = append(r.modules, m)
}

func (r *Registry) MountAll(router Router) {
    for _, m := range r.modules {
        m.RegisterRoutes(router)
    }
}
```

```go
// internal/modules/orders/module.go
package orders

type Module struct {
    handler  *Handler
    userAPI  users.UserAPI       // injected interface — not the users package directly
}

func New(db *sql.DB, userAPI users.UserAPI) *Module {
    repo    := newRepository(db)
    service := newService(repo, userAPI)
    return &Module{handler: newHandler(service), userAPI: userAPI}
}

func (m *Module) Name() string { return "orders" }

func (m *Module) RegisterRoutes(r Router) {
    r.POST("/orders", m.handler.Create)
    r.GET("/orders/:id", m.handler.Get)
}
```

```go
// cmd/server/main.go
db := openDB(os.Getenv("DATABASE_URL"))

usersModule  := users.New(db)
ordersModule := orders.New(db, usersModule.Service())   // pass the interface
notifModule  := notifications.New(db, eventBus)

reg := registry.New()
reg.Register(usersModule)
reg.Register(ordersModule)
reg.Register(notifModule)
reg.MountAll(router)
```

## Inter-Module Communication

### Synchronous: Interface Injection

```go
// orders/service.go
type UserAPI interface {
    GetUser(ctx context.Context, id string) (users.UserInfo, error)
}

type OrderService struct {
    repo    Repository
    userAPI UserAPI         // depends on interface, not on users.Service concrete type
}

func (s *OrderService) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (Order, error) {
    user, err := s.userAPI.GetUser(ctx, req.UserID)   // in-process call via interface
    if err != nil {
        return Order{}, fmt.Errorf("place order: %w", err)
    }
    ...
}
```

### Asynchronous: In-Process Event Bus

```go
// internal/shared/eventbus/bus.go
type Event struct {
    Topic   string
    Payload any
}

type Handler func(ctx context.Context, e Event) error

type Bus struct {
    mu       sync.RWMutex
    handlers map[string][]Handler
}

func (b *Bus) Subscribe(topic string, h Handler) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[topic] = append(b.handlers[topic], h)
}

func (b *Bus) Publish(ctx context.Context, e Event) error {
    b.mu.RLock()
    hs := b.handlers[e.Topic]
    b.mu.RUnlock()
    for _, h := range hs {
        if err := h(ctx, e); err != nil {
            return err
        }
    }
    return nil
}
```

```go
// users/service.go — publish event after registration
func (s *Service) Register(ctx context.Context, req RegisterRequest) (UserInfo, error) {
    user, err := s.repo.Create(ctx, ...)
    if err != nil { return UserInfo{}, err }
    _ = s.bus.Publish(ctx, eventbus.Event{
        Topic:   "users.registered",
        Payload: user.ID,
    })
    return toUserInfo(user), nil
}

// notifications/module.go — subscribe at startup
func (m *Module) Init(bus *eventbus.Bus) {
    bus.Subscribe("users.registered", m.service.OnUserRegistered)
}
```

## Dependency Inversion Rules

```
CORRECT dependency direction:
  orders/service.go  →  users.UserAPI (interface defined in orders)
  NOT:
  orders/service.go  →  users.Service (concrete type from users package)

CORRECT event flow:
  users publishes event  →  bus  →  notifications subscribes
  NOT:
  users imports notifications
```

## No Circular Imports

Enforce at the module level:

```
Allowed:        orders → shared
                orders → users interface (defined locally in orders)
                users  → shared

Forbidden:      orders → users (concrete package import)
                users  → orders (circular)
                any module → another module's internal packages
```

Use Go's `internal/` directory: `modules/users/internal/` blocks external imports.

## Shared Database, Isolated Tables

All modules share one DB pool, but each module owns its tables:

```sql
-- users module owns:
CREATE TABLE users (...);
CREATE TABLE user_sessions (...);

-- orders module owns:
CREATE TABLE orders (...);
CREATE TABLE order_items (...);

-- Cross-module joins are done in the calling service, not in SQL joins across module boundaries
```

For cross-module data needs, fetch from each module's API, then join in application code.

## Rules

- Each module exposes one interface file (`api.go`) — all inter-module calls go through it.
- No module imports another module's concrete types or internal packages.
- Shared infrastructure (DB pool, event bus, logger) lives in `internal/shared/`.
- Interfaces are defined in the consuming module, not in the provider module.
- In-process event bus for async decoupling; no message broker required for intra-service events.
- Module tables are owned exclusively — no cross-module SQL joins.
- The module registry in `main.go` is the only place that knows all modules exist.
