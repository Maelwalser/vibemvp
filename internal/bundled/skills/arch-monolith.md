# Monolith Architecture Skill Guide

## Overview

A monolith deploys as a single binary or container. All features share one process, one database connection pool, and one deployment unit. Keep internal modules decoupled through strict package/namespace boundaries — the goal is a modular codebase that happens to ship as one artifact.

## Project Layout

```
service/
├── cmd/
│   └── server/
│       └── main.go          # Entry point: wire dependencies, start server
├── internal/
│   ├── handler/             # HTTP layer — parse requests, call services, return responses
│   ├── service/             # Business logic — orchestrates repositories, enforces rules
│   ├── repository/          # Data access — SQL queries, ORM calls
│   ├── middleware/          # Auth, logging, rate limiting
│   ├── domain/              # Shared domain types (structs, enums, errors)
│   └── config/              # Config loading and validation
├── migrations/              # Database migration files
└── Dockerfile
```

Organize by feature when the codebase grows:
```
internal/
├── users/
│   ├── handler.go
│   ├── service.go
│   └── repository.go
├── orders/
│   ├── handler.go
│   ├── service.go
│   └── repository.go
└── shared/
    └── db.go                # Single shared DB connection pool
```

## Layered Request Flow

```
HTTP Request
    ↓
Handler         – validate input, parse path/query params, call service
    ↓
Service         – business rules, cross-domain coordination
    ↓
Repository      – database queries only, no business logic
    ↓
Database        – single shared connection pool
```

## Single DB Connection Pool

Initialize once at startup and inject everywhere:

```go
// cmd/server/main.go
db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatalf("db open: %v", err)
}
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)

userRepo    := repository.NewUserRepo(db)
orderRepo   := repository.NewOrderRepo(db)
userService := service.NewUserService(userRepo, orderRepo)
userHandler := handler.NewUserHandler(userService)
```

## Repository Interface Pattern

Define interfaces in the service layer; implement in repository:

```go
// internal/service/user_service.go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (domain.User, error)
    Create(ctx context.Context, u domain.User) (domain.User, error)
    Update(ctx context.Context, u domain.User) (domain.User, error)
}

// internal/repository/user_repo.go
type UserRepo struct { db *sql.DB }

func (r *UserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
    // SQL query — no business logic here
}
```

## Service Layer Pattern

Services orchestrate — they do not hold state or mutate inputs:

```go
func (s *UserService) Register(ctx context.Context, req RegisterRequest) (domain.User, error) {
    if err := validateEmail(req.Email); err != nil {
        return domain.User{}, fmt.Errorf("register: %w", err)
    }
    hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return domain.User{}, fmt.Errorf("register hash: %w", err)
    }
    user := domain.User{
        Email:        req.Email,
        PasswordHash: string(hashed),
        CreatedAt:    time.Now(),
    }
    return s.repo.Create(ctx, user)
}
```

## Intra-Process Communication

Never make HTTP calls between features in a monolith. Call services directly:

```go
// CORRECT — direct in-process call
func (s *OrderService) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (domain.Order, error) {
    user, err := s.userService.GetUser(ctx, req.UserID)  // in-process
    ...
}

// WRONG — do not do inter-service HTTP in a monolith
resp, err := http.Get("http://localhost:8080/api/users/" + req.UserID)
```

## Transaction Coordination

Transactions span across repositories by passing the tx object:

```go
func (s *OrderService) PlaceOrder(ctx context.Context, req PlaceOrderRequest) error {
    return s.db.BeginTxFunc(ctx, func(tx *sql.Tx) error {
        order, err := s.orderRepo.CreateTx(ctx, tx, req.ToOrder())
        if err != nil {
            return err
        }
        return s.inventoryRepo.DeductTx(ctx, tx, order.ItemID, order.Quantity)
    })
}
```

## Health Check

```go
app.GET("/health", func(c echo.Context) error {
    if err := db.PingContext(c.Request().Context()); err != nil {
        return c.JSON(503, map[string]string{"status": "error", "db": err.Error()})
    }
    return c.JSON(200, map[string]string{"status": "ok"})
})
```

## Graceful Shutdown

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := server.Shutdown(ctx); err != nil {
    log.Printf("shutdown error: %v", err)
}
```

## Dockerfile (Single Binary)

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates && adduser -D -u 1001 appuser
USER appuser
COPY --from=builder /server /server
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["/server"]
```

## Rules

- One database connection pool, shared across all repositories.
- No HTTP calls between features — use in-process service calls.
- Interfaces defined in the service layer, not the repository layer.
- No business logic in handlers or repositories.
- All configuration via environment variables validated at startup.
- Migrations are versioned files applied on startup or by a separate migration step.
- Feature packages must not import each other — shared types live in `domain/` or `shared/`.
