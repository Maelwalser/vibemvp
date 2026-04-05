# RabbitMQ Skill Guide

## Overview

RabbitMQ is an AMQP-based message broker with flexible routing via exchanges. Producers publish to exchanges; exchanges route to queues via bindings and routing keys. Consumers pull from queues with explicit acknowledgment for reliability.

## Connection & Channel Management

### Go (amqp091-go)

```go
package rabbitmq

import (
    "fmt"
    amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

func NewClient(url string) (*Client, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, fmt.Errorf("amqp dial: %w", err)
    }
    ch, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("open channel: %w", err)
    }
    return &Client{conn: conn, channel: ch}, nil
}

func (c *Client) Close() {
    c.channel.Close()
    c.conn.Close()
}
```

### TypeScript (amqplib)

```typescript
import amqp from 'amqplib';

const conn = await amqp.connect(process.env.RABBITMQ_URL!);
const ch = await conn.createChannel();

process.on('SIGINT', async () => {
  await ch.close();
  await conn.close();
});
```

## Exchange Types

| Type | Routing | Use Case |
|------|---------|----------|
| `direct` | Exact routing key match | Task queues, RPC |
| `fanout` | Broadcast to all bound queues | Event broadcasting |
| `topic` | Pattern match (`*` one word, `#` many) | Multi-category events |
| `headers` | Message header attributes | Complex routing without routing key |

```go
// Declare a topic exchange
err = ch.ExchangeDeclare(
    "events",  // name
    "topic",   // kind
    true,      // durable
    false,     // auto-delete
    false,     // internal
    false,     // no-wait
    nil,
)
```

## Queue Declaration

```go
q, err := ch.QueueDeclare(
    "order.processing",  // name
    true,                // durable — survives broker restart
    false,               // auto-delete
    false,               // exclusive
    false,               // no-wait
    amqp.Table{
        "x-dead-letter-exchange":    "dlx",          // DLQ routing
        "x-dead-letter-routing-key": "order.failed",
        "x-message-ttl":             int32(86400000), // 24h TTL in ms
        "x-max-length":              int32(10000),    // max queue depth
    },
)
```

## Dead Letter Exchange (DLX) Setup

```go
// 1. Declare the DLX
ch.ExchangeDeclare("dlx", "direct", true, false, false, false, nil)

// 2. Declare the dead-letter queue
ch.QueueDeclare("order.failed", true, false, false, false, nil)

// 3. Bind DLQ to DLX
ch.QueueBind("order.failed", "order.failed", "dlx", false, nil)

// 4. Main queue points to DLX (done in x-dead-letter-exchange above)
// Messages go to DLQ when: rejected (nack+requeue=false), TTL expires, or queue length exceeded
```

## Producer Pattern

```go
func (c *Client) Publish(exchange, routingKey string, body []byte) error {
    return c.channel.PublishWithContext(ctx,
        exchange,
        routingKey,
        false, // mandatory
        false, // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp.Persistent, // survives broker restart
            Body:         body,
            MessageId:    uuid.NewString(),
            Timestamp:    time.Now(),
        },
    )
}
```

```typescript
// TypeScript publisher
await ch.assertExchange('events', 'topic', { durable: true });
ch.publish('events', 'order.created', Buffer.from(JSON.stringify(payload)), {
  persistent: true,
  contentType: 'application/json',
  messageId: crypto.randomUUID(),
});
```

## Consumer / Worker Pattern

### Prefetch Count (QoS)

Set `prefetch` to limit unacknowledged messages per consumer — prevents a slow consumer from being overwhelmed.

```go
// Process max 10 messages at a time before acking
ch.Qos(10, 0, false)

msgs, err := ch.Consume(
    "order.processing",
    "worker-1",  // consumer tag
    false,       // auto-ack = false (manual ack)
    false,       // exclusive
    false,       // no-local
    false,       // no-wait
    nil,
)

for d := range msgs {
    if err := processOrder(d.Body); err != nil {
        // nack + requeue=false → goes to DLX
        d.Nack(false, false)
        continue
    }
    d.Ack(false)
}
```

```typescript
// TypeScript consumer
await ch.prefetch(10);
await ch.consume('order.processing', async (msg) => {
  if (!msg) return;
  try {
    await processOrder(JSON.parse(msg.content.toString()));
    ch.ack(msg);
  } catch (err) {
    ch.nack(msg, false, false); // send to DLX
  }
});
```

## Acknowledgment Reference

| Method | Effect |
|--------|--------|
| `Ack(false)` | Acknowledge single message — removed from queue |
| `Nack(false, true)` | Reject + requeue — message returns to front of queue |
| `Nack(false, false)` | Reject without requeue — routes to DLX if configured |
| `Reject(false)` | Same as Nack single |

## Binding Pattern (Topic Exchange)

```go
// Bind queue to receive all order events
ch.QueueBind("order.processing", "order.*", "events", false, nil)

// Bind queue to receive all events
ch.QueueBind("audit.log", "#", "events", false, nil)
```

## Key Rules

- Always declare exchanges and queues as `durable: true` in production — non-durable state is lost on restart.
- Set `DeliveryMode: amqp.Persistent` on messages to survive broker restart.
- Use `x-dead-letter-exchange` to route failed/expired messages to a DLQ — never discard silently.
- Set `prefetch` (basic.qos) to a low value (5–20) to balance throughput and backpressure per consumer.
- Use manual ack (`autoAck: false`) and only ack after successful processing.
- Open one channel per goroutine/thread — channels are not thread-safe; connections can be shared.
- Implement reconnection logic — connections and channels can drop on network blips or broker restarts.
- Use `MessageId` and idempotency checks to handle at-least-once redelivery safely.
