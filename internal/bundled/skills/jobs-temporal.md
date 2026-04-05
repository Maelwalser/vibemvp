# Temporal Skill Guide

## Overview

Temporal is a durable execution platform for long-running, fault-tolerant workflows. Workflow functions must be deterministic (no I/O, no randomness, no time.Now()). Activity functions perform all non-deterministic work. Temporal replays workflow history to recover from failures.

## Core Concepts

| Concept | Rule |
|---------|------|
| Workflow | Deterministic orchestrator — no I/O, no random, no time.Now() |
| Activity | Non-deterministic work — DB, HTTP, file I/O |
| Worker | Process that polls task queue and executes workflows + activities |
| Task Queue | Named routing channel connecting clients to workers |
| Signal | External input to a running workflow |
| Query | Read-only inspection of workflow state |

## Project Layout

```
service/
├── workflow/
│   ├── order_workflow.go     # Workflow definitions
│   └── order_workflow_test.go
├── activity/
│   ├── order_activity.go     # Activity implementations
│   └── order_activity_test.go
├── worker/
│   └── main.go               # Worker process entry point
└── client/
    └── starter.go            # Workflow client / starter
```

## Workflow Definition

```go
package workflow

import (
    "time"
    "go.temporal.io/sdk/workflow"
    "go.temporal.io/sdk/activity"
    "yourapp/activity"
)

const OrderTaskQueue = "order-processing"

type OrderInput struct {
    OrderID    string
    CustomerID string
    Amount     float64
}

type OrderResult struct {
    ConfirmationID string
}

func OrderWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {
    logger := workflow.GetLogger(ctx)
    logger.Info("OrderWorkflow started", "orderID", input.OrderID)

    // Activity options — configure retry and timeout per activity call
    actOpts := workflow.ActivityOptions{
        StartToCloseTimeout:    10 * time.Minute, // max duration for a single activity attempt
        ScheduleToCloseTimeout: 1 * time.Hour,    // max duration including all retries
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    100 * time.Second,
            MaximumAttempts:    5,
            NonRetryableErrorTypes: []string{"PaymentDeclinedError"},
        },
    }
    ctx = workflow.WithActivityOptions(ctx, actOpts)

    // Execute activities sequentially — Temporal tracks each step durably
    var paymentResult activity.PaymentResult
    if err := workflow.ExecuteActivity(ctx, activity.ChargePayment, input).Get(ctx, &paymentResult); err != nil {
        return OrderResult{}, fmt.Errorf("charge payment: %w", err)
    }

    var shipResult activity.ShipmentResult
    if err := workflow.ExecuteActivity(ctx, activity.CreateShipment, input, paymentResult).Get(ctx, &shipResult); err != nil {
        // Saga compensation: reverse previous steps on failure
        workflow.ExecuteActivity(ctx, activity.RefundPayment, paymentResult.ChargeID).Get(ctx, nil)
        return OrderResult{}, fmt.Errorf("create shipment: %w", err)
    }

    return OrderResult{ConfirmationID: shipResult.TrackingID}, nil
}
```

## Activity Definitions

```go
package activity

import (
    "context"
    "go.temporal.io/sdk/activity"
)

type Activities struct {
    db      *sql.DB
    payment PaymentClient
}

type PaymentResult struct {
    ChargeID string
}

func (a *Activities) ChargePayment(ctx context.Context, input workflow.OrderInput) (PaymentResult, error) {
    // Heartbeat for long-running activities
    activity.RecordHeartbeat(ctx, "charging payment")

    chargeID, err := a.payment.Charge(ctx, input.CustomerID, input.Amount)
    if err != nil {
        // Return a non-retryable error type
        return PaymentResult{}, temporal.NewNonRetryableApplicationError(
            "payment declined", "PaymentDeclinedError", err,
        )
    }
    return PaymentResult{ChargeID: chargeID}, nil
}
```

## Saga Pattern (Compensating Activities)

```go
func SagaWorkflow(ctx workflow.Context, input OrderInput) error {
    var compensations []func(workflow.Context)

    // Step 1
    var reserveResult ReserveResult
    if err := workflow.ExecuteActivity(ctx, ReserveInventory, input).Get(ctx, &reserveResult); err != nil {
        return err
    }
    compensations = append(compensations, func(ctx workflow.Context) {
        workflow.ExecuteActivity(ctx, ReleaseInventory, reserveResult.ReservationID).Get(ctx, nil)
    })

    // Step 2
    var chargeResult ChargeResult
    if err := workflow.ExecuteActivity(ctx, ChargePayment, input).Get(ctx, &chargeResult); err != nil {
        // Run compensations in reverse
        for i := len(compensations) - 1; i >= 0; i-- {
            compensations[i](ctx)
        }
        return err
    }

    return nil
}
```

## Signals (External Control)

```go
// In workflow: register signal channel
cancelCh := workflow.GetSignalChannel(ctx, "cancel-order")
workflow.Go(ctx, func(ctx workflow.Context) {
    cancelCh.Receive(ctx, nil)
    // handle cancellation
})

// From client: send signal to running workflow
err = temporalClient.SignalWorkflow(ctx, workflowID, runID, "cancel-order", nil)

// SignalWithStart: signal + start if not running
run, err := temporalClient.SignalWithStartWorkflow(ctx,
    workflowID,
    "cancel-order",
    nil,       // signal payload
    startOpts,
    workflow.OrderWorkflow,
    input,
)
```

## Queries (State Inspection)

```go
// In workflow: register query handler
workflow.SetQueryHandler(ctx, "get-status", func() (string, error) {
    return currentStatus, nil
})

// From client
val, err := temporalClient.QueryWorkflow(ctx, workflowID, runID, "get-status")
var status string
val.Get(&status)
```

## Worker Setup

```go
package main

import (
    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/worker"
    "yourapp/workflow"
    "yourapp/activity"
)

func main() {
    c, err := client.Dial(client.Options{
        HostPort: os.Getenv("TEMPORAL_HOST"), // "localhost:7233" or Temporal Cloud endpoint
    })
    if err != nil {
        log.Fatal("temporal dial:", err)
    }
    defer c.Close()

    acts := &activity.Activities{
        db:      connectDB(),
        payment: newPaymentClient(),
    }

    w := worker.New(c, workflow.OrderTaskQueue, worker.Options{
        MaxConcurrentActivityExecutionSize:      50,
        MaxConcurrentWorkflowTaskExecutionSize:  100,
    })
    w.RegisterWorkflow(workflow.OrderWorkflow)
    w.RegisterActivity(acts)

    if err := w.Run(worker.InterruptCh()); err != nil {
        log.Fatal("worker run:", err)
    }
}
```

## Workflow Client — Starting Workflows

```go
func StartOrder(ctx context.Context, c client.Client, input workflow.OrderInput) (string, error) {
    run, err := c.ExecuteWorkflow(ctx,
        client.StartWorkflowOptions{
            ID:        "order-" + input.OrderID,  // deterministic ID for deduplication
            TaskQueue: workflow.OrderTaskQueue,
        },
        workflow.OrderWorkflow,
        input,
    )
    if err != nil {
        return "", fmt.Errorf("start workflow: %w", err)
    }
    return run.GetID(), nil
}
```

## Timeout Reference

| Timeout | Scope | When to Set |
|---------|-------|-------------|
| `StartToCloseTimeout` | Single activity attempt | Always required |
| `ScheduleToCloseTimeout` | Activity total including retries | Long-running with many retries |
| `ScheduleToStartTimeout` | Time in queue before start | Detect stuck workers |
| `WorkflowRunTimeout` | Single workflow run | Max wall time for one run |
| `WorkflowExecutionTimeout` | Entire workflow lifecycle across runs | Absolute deadline |

## Key Rules

- Workflow code MUST be deterministic: no `time.Now()`, no random, no network calls — use `workflow.Now()` and activities instead.
- Always set `StartToCloseTimeout` on every activity — there is no default.
- Use `NonRetryableApplicationError` to prevent retrying unrecoverable errors (e.g. invalid input, payment declined).
- Heartbeat long-running activities with `activity.RecordHeartbeat` so Temporal detects worker crashes.
- Use a stable, deterministic workflow ID (e.g. `"order-" + orderID`) to enable deduplication and idempotent starts.
- Saga compensation: collect rollback functions during execution, run them in reverse order on failure.
- Test workflows with Temporal's `testsuite.WorkflowTestSuite` — it provides a simulated clock and activity mocking.
