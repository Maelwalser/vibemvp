# Redis Streams Skill Guide

## Overview

Redis Streams is a log-like data structure built into Redis for append-only message streaming with consumer groups. It combines Kafka-style consumer groups with Redis's simplicity. Messages persist until explicitly deleted; consumer groups track per-consumer delivery and acknowledgment.

## Producing Messages (XADD)

### Go (go-redis/v9)

```go
package streams

import (
    "context"
    "fmt"
    "github.com/redis/go-redis/v9"
)

func Produce(ctx context.Context, rdb *redis.Client, stream string, fields map[string]any) (string, error) {
    id, err := rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: stream,
        ID:     "*",      // auto-generate: <milliseconds>-<seq>
        Values: fields,
        MaxLen: 10000,    // cap stream length (approximate trim)
        Approx: true,
    }).Result()
    if err != nil {
        return "", fmt.Errorf("xadd %s: %w", stream, err)
    }
    return id, nil
}

// Usage
Produce(ctx, rdb, "orders", map[string]any{
    "order_id":    "ord-123",
    "customer_id": "cust-456",
    "amount":      "99.99",
})
```

### TypeScript (ioredis)

```typescript
import Redis from 'ioredis';

const redis = new Redis(process.env.REDIS_URL!);

async function produce(stream: string, fields: Record<string, string>): Promise<string> {
  const id = await redis.xadd(stream, 'MAXLEN', '~', '10000', '*', ...Object.entries(fields).flat());
  return id as string;
}
```

## Consumer Groups

### Create Group

```go
// XGROUP CREATE — idempotent with $ for new-messages-only or 0 for all
err := rdb.XGroupCreateMkStream(ctx, "orders", "order-processor", "$").Err()
if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
    return fmt.Errorf("create group: %w", err)
}
```

### XREADGROUP — Competing Consumers

```go
func Consume(ctx context.Context, rdb *redis.Client, stream, group, consumer string) error {
    for {
        streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    group,
            Consumer: consumer,   // unique per worker instance (e.g. hostname + pid)
            Streams:  []string{stream, ">"},  // ">" = new undelivered messages
            Count:    10,
            Block:    5 * time.Second,
        }).Result()
        if err == redis.Nil {
            continue // timeout, no new messages
        }
        if err != nil {
            return fmt.Errorf("xreadgroup: %w", err)
        }

        for _, s := range streams {
            for _, msg := range s.Messages {
                if err := process(msg.Values); err != nil {
                    // leave in PEL for reprocessing; handle via XPENDING/XCLAIM
                    continue
                }
                // Acknowledge successful processing
                rdb.XAck(ctx, stream, group, msg.ID)
            }
        }
    }
}
```

## Acknowledgment (XACK)

```go
// Ack a single message
rdb.XAck(ctx, "orders", "order-processor", messageID)

// Ack multiple messages at once
rdb.XAck(ctx, "orders", "order-processor", id1, id2, id3)
```

## Inspecting Pending Messages (XPENDING)

```go
// Summary: how many pending per consumer
pending, err := rdb.XPending(ctx, "orders", "order-processor").Result()
fmt.Printf("pending=%d, consumers=%v\n", pending.Count, pending.Consumers)

// Detail: messages pending longer than 60s
pendingMsgs, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
    Stream:   "orders",
    Group:    "order-processor",
    Idle:     60 * time.Second,
    Start:    "-",
    Stop:     "+",
    Count:    100,
}).Result()

for _, p := range pendingMsgs {
    fmt.Printf("id=%s consumer=%s idle=%v deliveries=%d\n",
        p.ID, p.Consumer, p.Idle, p.RetryCount)
}
```

## Reclaiming Stuck Messages (XCLAIM)

Claim messages that have been idle too long (e.g. dead consumer) and re-deliver to a healthy worker.

```go
func ClaimStuck(ctx context.Context, rdb *redis.Client, stream, group, newConsumer string) error {
    pendingMsgs, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
        Stream: stream,
        Group:  group,
        Idle:   2 * time.Minute,
        Start:  "-",
        Stop:   "+",
        Count:  100,
    }).Result()
    if err != nil {
        return fmt.Errorf("xpending: %w", err)
    }

    ids := make([]string, 0, len(pendingMsgs))
    for _, p := range pendingMsgs {
        if p.RetryCount >= 5 {
            // move to DLQ stream instead of re-claiming
            sendToDLQ(ctx, rdb, stream+":dlq", p.ID)
            rdb.XAck(ctx, stream, group, p.ID)
            continue
        }
        ids = append(ids, p.ID)
    }

    if len(ids) > 0 {
        rdb.XClaim(ctx, &redis.XClaimArgs{
            Stream:   stream,
            Group:    group,
            Consumer: newConsumer,
            MinIdle:  2 * time.Minute,
            Messages: ids,
        })
    }
    return nil
}
```

## Reading Without a Group (XREAD)

```go
// Simple tail — start from latest
msgs, err := rdb.XRead(ctx, &redis.XReadArgs{
    Streams: []string{"orders", "$"},
    Count:   100,
    Block:   0,
}).Result()
```

## Dead-Letter Queue Pattern

```go
func sendToDLQ(ctx context.Context, rdb *redis.Client, dlqStream, originalID string) {
    rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: dlqStream,
        ID:     "*",
        Values: map[string]any{
            "original_id": originalID,
            "failed_at":   time.Now().Unix(),
        },
    })
}
```

## Key Rules

- Use `XADD ... MAXLEN ~ N` to cap stream size and prevent unbounded memory growth.
- Consumer names within a group must be unique per worker (use hostname + PID or UUID).
- Use `">"` as the ID in XREADGROUP to read only new, undelivered messages.
- Use `XPENDING` + `XCLAIM` in a background reconciler loop to reclaim idle messages from crashed consumers.
- After N delivery attempts (check `RetryCount`), move messages to a DLQ stream and XACK to clear from PEL.
- XACK removes the message from the Pending Entry List (PEL) — always ack after successful processing.
- Redis Streams persist until explicitly deleted with `XDEL` or trimmed via `MAXLEN` — plan retention accordingly.
