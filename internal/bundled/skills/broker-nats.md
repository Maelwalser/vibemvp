# NATS Skill Guide

## Overview

NATS is a lightweight, high-performance cloud-native messaging system. Core NATS provides at-most-once pub/sub; JetStream adds persistence, at-least-once delivery, and exactly-once semantics. Use queue groups for competing consumers without a broker-level group concept.

## Connection Setup

### Go (nats.go)

```go
package messaging

import (
    "fmt"
    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
)

func Connect(url string) (*nats.Conn, error) {
    nc, err := nats.Connect(url,
        nats.RetryOnFailedConnect(true),
        nats.MaxReconnects(-1),
        nats.ReconnectWait(2*time.Second),
        nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
            log.Printf("NATS disconnected: %v", err)
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("nats connect: %w", err)
    }
    return nc, nil
}
```

### TypeScript (nats.ws / nats.js)

```typescript
import { connect, StringCodec } from 'nats';

const nc = await connect({ servers: 'nats://localhost:4222' });
const sc = StringCodec();
```

## Core Pub/Sub Pattern

```go
// Publisher
nc.Publish("orders.created", []byte(`{"id":"123"}`))

// Subscriber (at-most-once)
sub, err := nc.Subscribe("orders.*", func(msg *nats.Msg) {
    fmt.Printf("subject=%s data=%s\n", msg.Subject, msg.Data)
})
defer sub.Unsubscribe()
```

## Queue Groups (Load Balancing)

Queue groups distribute messages across subscribers — only one member receives each message.

```go
// Each worker joins the same queue group
sub, err := nc.QueueSubscribe("orders.created", "order-processor", func(msg *nats.Msg) {
    process(msg.Data)
})
```

```typescript
// TypeScript
await nc.subscribe('orders.created', {
  queue: 'order-processor',
  callback: (err, msg) => {
    if (!err) processOrder(sc.decode(msg.data));
  },
});
```

## JetStream — Persistent Streams

### Stream Creation

```go
js, err := jetstream.New(nc)
if err != nil {
    return fmt.Errorf("jetstream: %w", err)
}

stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
    Name:      "ORDERS",
    Subjects:  []string{"orders.>"},
    Retention: jetstream.LimitsPolicy,
    MaxAge:    24 * time.Hour,
    Storage:   jetstream.FileStorage,
    Replicas:  3,
})
```

### JetStream Publish

```go
ack, err := js.Publish(ctx, "orders.created", payload)
if err != nil {
    return fmt.Errorf("js publish: %w", err)
}
// ack.Sequence is the stream sequence number
```

### Durable Consumer

```go
consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
    Durable:       "order-processor",     // survives restarts
    AckPolicy:     jetstream.AckExplicitPolicy,
    FilterSubject: "orders.created",
    MaxDeliver:    5,                     // retry up to 5 times
    AckWait:       30 * time.Second,
})

// Pull-based consumption
msgs, _ := consumer.Messages()
for msg := range msgs {
    if err := process(msg.Data()); err != nil {
        msg.Nak()   // re-deliver
        continue
    }
    msg.Ack()
}
```

### Push-based Subscribe (simpler)

```go
sub, err := js.Subscribe(ctx, "orders.created", func(msg jetstream.Msg) {
    if err := process(msg.Data()); err != nil {
        msg.Nak()
        return
    }
    msg.Ack()
}, jetstream.Durable("processor-v1"), jetstream.AckExplicit())
```

## JetStream Key-Value Store

```go
kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
    Bucket: "sessions",
    TTL:    1 * time.Hour,
})

// Put
kv.Put(ctx, "sess:abc123", []byte(`{"userId":"u1"}`))

// Get
entry, err := kv.Get(ctx, "sess:abc123")
if err == jetstream.ErrKeyNotFound {
    // handle miss
}

// Delete
kv.Delete(ctx, "sess:abc123")

// Watch for changes
watcher, _ := kv.Watch(ctx, "sess.*")
for update := range watcher.Updates() {
    fmt.Printf("key=%s op=%v\n", update.Key(), update.Operation())
}
```

## Request-Reply Pattern

```go
// Server (responder)
nc.Subscribe("rpc.pricing", func(msg *nats.Msg) {
    result := computePrice(msg.Data)
    msg.Respond(result)
})

// Client (requestor) — blocks until reply or timeout
reply, err := nc.Request("rpc.pricing", payload, 5*time.Second)
if err != nil {
    return fmt.Errorf("rpc timeout: %w", err)
}
```

## Error Handling & DLQ

JetStream handles retries automatically via `MaxDeliver`. After exhausting retries, messages land in the `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.<stream>.<consumer>` advisory subject. Route these to a dead-letter stream:

```go
js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
    Name:     "ORDERS_DLQ",
    Subjects: []string{"orders.dlq.>"},
})

// In consumer handler, after max retries exceeded advisory:
js.Publish(ctx, "orders.dlq.created", originalPayload)
```

## Key Rules

- Core NATS is at-most-once; use JetStream for at-least-once or exactly-once guarantees.
- Queue groups require no server config — any number of subscribers with the same queue name form a group.
- Always use `AckExplicit` policy in JetStream; never rely on auto-ack in production.
- Set `MaxDeliver` and `AckWait` on consumers to control retry behavior.
- Durable consumers survive server restarts; ephemeral consumers are cleaned up after inactivity.
- KV buckets are JetStream streams under the hood — TTL and replication apply.
- For request-reply, set a sensible timeout (5s default) and handle `nats.ErrTimeout` at call sites.
