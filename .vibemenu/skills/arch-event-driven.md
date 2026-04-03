# Event-Driven Architecture Skill Guide

## Overview

Event-driven systems communicate via asynchronous events on a message broker (Kafka, RabbitMQ, NATS, etc.). Producers publish events without knowing consumers. Consumers process events independently. Key concerns: at-least-once delivery, idempotency, eventual consistency, and the outbox pattern for reliable publishing.

## Core Concepts

```
Producer → Broker (topic/queue) → Consumer(s)
                ↑
        At-least-once delivery guarantee
        (messages may be redelivered on failure)
                ↓
        Consumers MUST be idempotent
```

## Outbox Pattern (Reliable Event Publishing)

The outbox pattern guarantees events are published even if the broker is temporarily unavailable. Write the event to the database in the same transaction as your business data, then relay it to the broker asynchronously.

### Database Schema

```sql
CREATE TABLE outbox_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id VARCHAR(255) NOT NULL,      -- e.g. order ID
    event_type   VARCHAR(255) NOT NULL,      -- e.g. "order.placed"
    payload      JSONB        NOT NULL,
    published    BOOLEAN      NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox_events (published, created_at)
    WHERE published = false;
```

### Write Business Data + Outbox in One Transaction

```go
func (s *OrderService) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (Order, error) {
    var order Order
    err := s.db.BeginTxFunc(ctx, func(tx *sql.Tx) error {
        var err error
        order, err = s.orderRepo.CreateTx(ctx, tx, req)
        if err != nil {
            return err
        }
        payload, _ := json.Marshal(OrderPlacedEvent{
            OrderID:  order.ID,
            UserID:   order.UserID,
            TotalCents: order.TotalCents,
        })
        _, err = tx.ExecContext(ctx,
            `INSERT INTO outbox_events (aggregate_id, event_type, payload)
             VALUES ($1, $2, $3)`,
            order.ID, "order.placed", payload,
        )
        return err
    })
    return order, err
}
```

### Outbox Relay (Polling Approach)

```go
// Runs as a background goroutine or separate process
func (r *OutboxRelay) Run(ctx context.Context) {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := r.publishBatch(ctx); err != nil {
                log.Printf("outbox relay error: %v", err)
            }
        }
    }
}

func (r *OutboxRelay) publishBatch(ctx context.Context) error {
    rows, err := r.db.QueryContext(ctx,
        `SELECT id, aggregate_id, event_type, payload
         FROM outbox_events WHERE published = false
         ORDER BY created_at LIMIT 100
         FOR UPDATE SKIP LOCKED`,  // safe for multiple relay instances
    )
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var ev OutboxEvent
        if err := rows.Scan(&ev.ID, &ev.AggregateID, &ev.EventType, &ev.Payload); err != nil {
            return err
        }
        if err := r.broker.Publish(ctx, ev.EventType, ev.Payload); err != nil {
            return fmt.Errorf("publish %s: %w", ev.ID, err)
        }
        if _, err := r.db.ExecContext(ctx,
            `UPDATE outbox_events SET published = true, published_at = NOW() WHERE id = $1`,
            ev.ID,
        ); err != nil {
            return err
        }
    }
    return nil
}
```

## Idempotent Consumer

Because brokers guarantee at-least-once delivery, consumers must handle duplicate messages safely.

### Idempotency Key Table

```sql
CREATE TABLE processed_events (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Idempotent Handler Pattern

```go
func (c *OrderConsumer) HandleOrderPlaced(ctx context.Context, msg BrokerMessage) error {
    idempotencyKey := msg.MessageID  // unique per message from the broker

    // Check if already processed
    var exists bool
    err := c.db.QueryRowContext(ctx,
        `SELECT EXISTS(SELECT 1 FROM processed_events WHERE idempotency_key = $1)`,
        idempotencyKey,
    ).Scan(&exists)
    if err != nil {
        return fmt.Errorf("check idempotency: %w", err)
    }
    if exists {
        return nil  // already handled, safe to ack
    }

    var event OrderPlacedEvent
    if err := json.Unmarshal(msg.Body, &event); err != nil {
        return fmt.Errorf("unmarshal: %w", err)
    }

    return c.db.BeginTxFunc(ctx, func(tx *sql.Tx) error {
        // Business logic
        if err := c.inventoryRepo.ReserveTx(ctx, tx, event.ItemID, event.Quantity); err != nil {
            return err
        }
        // Mark as processed in same transaction
        _, err := tx.ExecContext(ctx,
            `INSERT INTO processed_events (idempotency_key) VALUES ($1) ON CONFLICT DO NOTHING`,
            idempotencyKey,
        )
        return err
    })
}
```

## Retry and Dead-Letter Queue

```
Message fails processing
        ↓
Retry (up to N times with backoff)
        ↓
Dead-Letter Queue (DLQ) — for manual inspection or automated reprocessing
```

```go
func (c *Consumer) processWithRetry(ctx context.Context, msg BrokerMessage) {
    var err error
    for attempt := 0; attempt < c.maxRetries; attempt++ {
        if attempt > 0 {
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            time.Sleep(backoff)
        }
        err = c.handler(ctx, msg)
        if err == nil {
            msg.Ack()
            return
        }
        log.Printf("attempt %d failed: %v", attempt+1, err)
    }
    // Send to DLQ after exhausting retries
    log.Printf("sending to DLQ: %v", err)
    c.dlq.Publish(ctx, msg)
    msg.Ack()  // Ack original to prevent infinite redelivery
}
```

Kafka: use a separate `<topic>.dlq` topic. RabbitMQ: use a dead-letter exchange. AWS SQS: configure `RedrivePolicy` with `maxReceiveCount`.

## Event Schema Versioning

Events are contracts — version them explicitly:

```go
// Include version in event payload
type OrderPlacedEvent struct {
    SchemaVersion int    `json:"schema_version"`  // increment on breaking changes
    OrderID       string `json:"order_id"`
    UserID        string `json:"user_id"`
    TotalCents    int64  `json:"total_cents"`
}

// Consumer: handle multiple versions
func (c *Consumer) handle(msg BrokerMessage) error {
    var envelope struct {
        SchemaVersion int             `json:"schema_version"`
        Rest          json.RawMessage `json:",inline"`
    }
    json.Unmarshal(msg.Body, &envelope)
    switch envelope.SchemaVersion {
    case 1:
        return c.handleV1(envelope.Rest)
    case 2:
        return c.handleV2(envelope.Rest)
    default:
        return fmt.Errorf("unknown schema version: %d", envelope.SchemaVersion)
    }
}
```

Use a schema registry (Confluent Schema Registry, AWS Glue) for Avro/Protobuf schemas.

## Event Topic / Routing Key Conventions

```
Kafka topics:
  orders.placed
  orders.shipped
  users.registered
  inventory.reserved

RabbitMQ routing keys (same convention):
  orders.placed
  orders.*          # wildcard: all order events
  #                 # all events

Naming: <domain>.<past-tense-verb>
```

## Eventual Consistency Handling

```go
// DO: Accept that reads may be stale; use saga or process manager for multi-step flows
// DO: Return 202 Accepted for async operations, not 200 OK
func (h *OrderHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
    order, err := h.service.PlaceOrder(r.Context(), req)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "order_id": order.ID,
        "status":   "pending",  // not yet confirmed — event processing is async
    })
}

// DO NOT: assume cross-service state is consistent immediately after publishing
```

## Rules

- Always use the outbox pattern for publishing events — never publish directly from a transaction.
- All consumers must be idempotent — duplicate delivery is expected and must be safe.
- Include `schema_version` in every event payload.
- Use past-tense event names: `order.placed`, not `place.order`.
- Failed messages go to a DLQ after exhausting retries — never silently discard them.
- Events are append-only facts — never mutate published events.
- Use `FOR UPDATE SKIP LOCKED` in the outbox relay to support multiple relay instances safely.
- Return 202 Accepted for operations that result in async event processing.
