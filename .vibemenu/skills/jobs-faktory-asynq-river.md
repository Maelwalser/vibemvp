# Faktory, Asynq & River Skill Guide

## Overview

Three job queue systems with different trade-offs:
- **Faktory** — Language-agnostic job server with worker clients for Go, Ruby, Node, Python, and more.
- **Asynq** — Go + Redis job queue with priority queues, cron scheduling, and a web UI.
- **River** — Go + PostgreSQL job queue using transactional enqueue for exactly-once guarantees via advisory locks.

---

## Faktory

### Architecture

```
Producer → Faktory Server → Worker Client → perform job
```

Faktory is a standalone server (port 7419) that stores jobs in Redis. Workers connect over a TCP protocol. All job payloads are JSON.

### Job Payload Format

```json
{
  "jid":     "7d8d6e2d-a5a2-4a7a-9a0a-1f0d2e3a4b5c",
  "jobtype": "OrderProcessor",
  "args":    ["order-123", "cust-456"],
  "queue":   "orders",
  "retry":   25,
  "at":      "2026-01-01T09:00:00Z"
}
```

### Go Worker Client

```go
package main

import (
    "context"
    "fmt"
    faktory "github.com/contribsys/faktory/client"
    worker "github.com/contribsys/faktory_worker_go"
)

func main() {
    mgr := worker.NewManager()
    mgr.Concurrency = 20
    mgr.Queues = []string{"critical", "orders", "default"}

    // Register handlers
    mgr.Register("OrderProcessor", processOrder)
    mgr.Register("EmailNotification", sendEmail)

    if err := mgr.Run(); err != nil {
        log.Fatal("faktory worker:", err)
    }
}

func processOrder(ctx context.Context, args ...interface{}) error {
    orderID, ok := args[0].(string)
    if !ok {
        return fmt.Errorf("invalid arg type for orderID")
    }
    return fulfillOrder(ctx, orderID)
}

// Enqueue from producer
func EnqueueOrder(orderID, customerID string) error {
    cl, err := faktory.Open()
    if err != nil {
        return fmt.Errorf("faktory open: %w", err)
    }
    defer cl.Close()

    job := faktory.NewJob("OrderProcessor", orderID, customerID)
    job.Queue = "orders"
    job.ReserveFor = 600  // seconds before job considered abandoned

    return cl.Push(job)
}
```

### Retry & Dead Set

Faktory retries failed jobs with 25-step exponential backoff by default (up to ~21 days). After exhausting retries, jobs land in the Dead Set, visible in the Faktory web UI.

```go
// Custom retry count
job.Retry = 5

// Disable retry — move directly to dead on failure
job.Retry = 0
```

### Middleware Chain

```go
mgr.Use(func(ctx worker.PerformContext, job *faktory.Job, next func(worker.PerformContext) error) error {
    start := time.Now()
    err := next(ctx)
    duration := time.Since(start)
    log.Printf("job=%s duration=%v err=%v", job.Type, duration, err)
    return err
})
```

---

## Asynq (Go + Redis)

### Installation

```bash
go get github.com/hibiken/asynq
```

### Task Definition

```go
package tasks

import (
    "encoding/json"
    "fmt"
    "github.com/hibiken/asynq"
)

// Task type constants — avoids magic strings
const (
    TypeOrderProcess = "order:process"
    TypeEmailSend    = "email:send"
)

type OrderPayload struct {
    OrderID    string `json:"order_id"`
    CustomerID string `json:"customer_id"`
}

func NewOrderProcessTask(orderID, customerID string) (*asynq.Task, error) {
    payload, err := json.Marshal(OrderPayload{OrderID: orderID, CustomerID: customerID})
    if err != nil {
        return nil, fmt.Errorf("marshal payload: %w", err)
    }
    return asynq.NewTask(TypeOrderProcess, payload), nil
}
```

### Client — Enqueuing Tasks

```go
func EnqueueOrder(client *asynq.Client, orderID, customerID string) error {
    task, err := tasks.NewOrderProcessTask(orderID, customerID)
    if err != nil {
        return err
    }

    info, err := client.Enqueue(task,
        asynq.MaxRetry(5),
        asynq.ProcessIn(0),              // immediate
        // asynq.ProcessAt(time.Now().Add(5*time.Minute)),
        asynq.Queue("orders"),
        asynq.Timeout(10*time.Minute),
        asynq.Unique(24*time.Hour),      // deduplication window
    )
    if err != nil {
        return fmt.Errorf("enqueue: %w", err)
    }
    log.Printf("enqueued task id=%s queue=%s", info.ID, info.Queue)
    return nil
}
```

### Server — Processing Tasks

```go
func StartWorker(redisURL string) error {
    srv := asynq.NewServer(
        asynq.RedisClientOpt{Addr: redisURL},
        asynq.Config{
            Queues: map[string]int{
                "critical": 6,   // weighted priority
                "orders":   3,
                "default":  1,
            },
            Concurrency: 20,
            ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
                log.Printf("task failed type=%s err=%v", task.Type(), err)
            }),
        },
    )

    mux := asynq.NewServeMux()
    mux.HandleFunc(tasks.TypeOrderProcess, handleOrderProcess)
    mux.HandleFunc(tasks.TypeEmailSend, handleEmailSend)

    return srv.Run(mux)
}

func handleOrderProcess(ctx context.Context, task *asynq.Task) error {
    var payload tasks.OrderPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return fmt.Errorf("%w: %w", asynq.SkipRetry, err) // non-retryable
    }
    return fulfillOrder(ctx, payload.OrderID)
}
```

### Scheduler (Cron)

```go
scheduler := asynq.NewScheduler(
    asynq.RedisClientOpt{Addr: redisURL},
    &asynq.SchedulerOpts{},
)

// Register cron jobs
scheduler.Register("0 9 * * 1-5", asynq.NewTask("report:daily", nil))
scheduler.Register("*/30 * * * *", asynq.NewTask("health:ping", nil),
    asynq.Queue("default"),
)

if err := scheduler.Run(); err != nil {
    log.Fatal("scheduler:", err)
}
```

### Asynqmon Web UI

```bash
go install github.com/hibiken/asynqmon@latest
asynqmon --redis-addr localhost:6379 --port 8080
```

---

## River (Go + PostgreSQL)

### Why River

River uses PostgreSQL advisory locks and `INSERT ... RETURNING` for exactly-once job delivery without Redis. Jobs enqueued in the same database transaction as business data — commit atomically.

### Installation

```bash
go get github.com/riverqueue/river
go get github.com/riverqueue/river/riverdriver/riverpgxv5
```

### Job Args Definition

```go
package jobs

import "github.com/riverqueue/river"

// JobArgs must implement Kind() string
type OrderProcessArgs struct {
    OrderID    string `json:"order_id"`
    CustomerID string `json:"customer_id"`
}

func (OrderProcessArgs) Kind() string { return "order_process" }
```

### Worker Implementation

```go
package jobs

import (
    "context"
    "fmt"
    "github.com/riverqueue/river"
)

type OrderProcessWorker struct {
    river.WorkerDefaults[OrderProcessArgs]
    payment PaymentClient
    shipment ShipmentClient
}

func (w *OrderProcessWorker) Work(ctx context.Context, job *river.Job[OrderProcessArgs]) error {
    args := job.Args
    if err := w.payment.Charge(ctx, args.CustomerID, job.Args.Amount); err != nil {
        return fmt.Errorf("charge payment: %w", err)
    }
    return w.shipment.Create(ctx, args.OrderID)
}
```

### Client Setup & Worker

```go
func NewRiverClient(pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
    workers := river.NewWorkers()
    river.AddWorker(workers, &OrderProcessWorker{
        payment:  newPaymentClient(),
        shipment: newShipmentClient(),
    })

    return river.NewClient(riverpgxv5.New(pool), &river.Config{
        Queues: map[string]river.QueueConfig{
            river.QueueDefault: {MaxWorkers: 10},
            "orders":           {MaxWorkers: 50},
        },
        Workers: workers,
    })
}
```

### Transactional Enqueue (InsertTx)

The key River advantage — enqueue atomically with your business transaction:

```go
func CreateOrder(ctx context.Context, pool *pgxpool.Pool, riverClient *river.Client[pgx.Tx], orderData OrderData) error {
    tx, err := pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    // Insert order row
    var orderID string
    err = tx.QueryRow(ctx,
        "INSERT INTO orders (customer_id, amount) VALUES ($1, $2) RETURNING id",
        orderData.CustomerID, orderData.Amount,
    ).Scan(&orderID)
    if err != nil {
        return fmt.Errorf("insert order: %w", err)
    }

    // Enqueue job in the SAME transaction — atomically
    _, err = riverClient.InsertTx(ctx, tx, OrderProcessArgs{
        OrderID:    orderID,
        CustomerID: orderData.CustomerID,
    }, &river.InsertOpts{
        Queue:    "orders",
        Priority: 1,
    })
    if err != nil {
        return fmt.Errorf("insert job: %w", err)
    }

    return tx.Commit(ctx) // both order row and job are committed together
}
```

### Job Uniqueness

```go
// Advisory lock-based deduplication
_, err = riverClient.Insert(ctx, OrderProcessArgs{OrderID: orderID}, &river.InsertOpts{
    UniqueOpts: river.UniqueOpts{
        ByArgs:   true,         // unique per distinct args
        ByPeriod: 24 * time.Hour,
    },
})
```

## Key Rules

### Faktory
- Workers must call `Ack` (handled automatically by the SDK) only after successful completion.
- Use queue names for priority routing — Faktory dequeues from queues in the order listed.
- The Dead Set is inspectable via the web UI — plan a reprocessing workflow for dead jobs.

### Asynq
- Return `fmt.Errorf("%w: %w", asynq.SkipRetry, err)` to mark a task as non-retryable.
- Use `asynq.Unique(duration)` for deduplication — prevents duplicate enqueues within the window.
- Run Scheduler as a separate process from the Server to avoid single-process failures.

### River
- `InsertTx` is the primary advantage of River — use it whenever the job must be created atomically with a DB write.
- Jobs are stored in the `river_jobs` table — run `river migrate-up` to create schema.
- Discarded jobs (exhausted retries) remain in the DB for inspection — implement a cleanup job for old discarded rows.
