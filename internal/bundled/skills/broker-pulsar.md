# Apache Pulsar Skill Guide

## Overview

Apache Pulsar is a cloud-native distributed messaging and streaming platform with multi-tenancy, geo-replication, tiered storage offload, and Pulsar Functions for serverless compute. Topics are hierarchical: `persistent://tenant/namespace/topic`.

## Topic Naming & Multi-Tenancy

```
persistent://my-org/production/orders          # persistent topic
non-persistent://my-org/dev/notifications      # in-memory, no durability

# Partitioned topic (parallel processing)
persistent://my-org/production/orders-partition
```

Tenants and namespaces isolate teams, environments, and retention policies without separate clusters.

## Client Setup

### Go (apache/pulsar-client-go)

```go
package pulsar

import (
    "fmt"
    "github.com/apache/pulsar-client-go/pulsar"
)

func NewClient(serviceURL string) (pulsar.Client, error) {
    client, err := pulsar.NewClient(pulsar.ClientOptions{
        URL:               serviceURL, // "pulsar://localhost:6650" or "pulsar+ssl://..."
        OperationTimeout:  30 * time.Second,
        ConnectionTimeout: 30 * time.Second,
    })
    if err != nil {
        return nil, fmt.Errorf("pulsar client: %w", err)
    }
    return client, nil
}
```

### Java

```java
PulsarClient client = PulsarClient.builder()
    .serviceUrl("pulsar://localhost:6650")
    .ioThreads(4)
    .listenerThreads(8)
    .build();
```

## Producer Pattern

### Go

```go
producer, err := client.CreateProducer(pulsar.ProducerOptions{
    Topic:           "persistent://my-org/production/orders",
    CompressionType: pulsar.LZ4,
    BatchingEnabled:  true,
    BatchingMaxMessages: 100,
    BatchingMaxPublishDelay: 10 * time.Millisecond,
    SendTimeout:     30 * time.Second,
})
defer producer.Close()

msgID, err := producer.Send(ctx, &pulsar.ProducerMessage{
    Payload: payload,
    Key:     order.CustomerID,   // key-based routing to partitions
    Properties: map[string]string{
        "eventType": "order.created",
        "version":   "1",
    },
})
if err != nil {
    return fmt.Errorf("pulsar send: %w", err)
}
```

## Subscription Types

| Type | Behavior | Use Case |
|------|----------|----------|
| `Exclusive` | One consumer only | Single ordered processor |
| `Shared` | Round-robin across consumers | Competing consumers (task queue) |
| `Failover` | Active + standby consumers | HA single-active |
| `Key_Shared` | Same key → same consumer | Per-entity ordering at scale |

## Consumer / Worker Pattern

### Go

```go
consumer, err := client.Subscribe(pulsar.ConsumerOptions{
    Topic:             "persistent://my-org/production/orders",
    SubscriptionName:  "order-processor",          // durable subscription name
    Type:              pulsar.Shared,               // competing consumers
    ReceiverQueueSize: 100,
    NackRedeliveryDelay: 30 * time.Second,
})
defer consumer.Close()

for {
    msg, err := consumer.Receive(ctx)
    if err != nil {
        return fmt.Errorf("receive: %w", err)
    }

    if err := processOrder(msg.Payload()); err != nil {
        consumer.Nack(msg)  // re-deliver after NackRedeliveryDelay
        continue
    }
    consumer.Ack(msg)
}
```

### Key_Shared Subscription (per-entity ordering)

```go
consumer, _ := client.Subscribe(pulsar.ConsumerOptions{
    Topic:            "persistent://my-org/production/orders",
    SubscriptionName: "order-processor",
    Type:             pulsar.KeyShared,  // same OrderID key → same consumer instance
})
```

## Schema Registry Integration

```go
// Produce with Avro schema — schema enforced at broker level
producer, _ := client.CreateProducer(pulsar.ProducerOptions{
    Topic:  "persistent://my-org/production/orders",
    Schema: pulsar.NewAvroSchema(orderSchema, nil),
})

producer.Send(ctx, &pulsar.ProducerMessage{
    Value: &Order{ID: "123", Amount: 99.99},
})

// Consume with schema validation
consumer, _ := client.Subscribe(pulsar.ConsumerOptions{
    Topic:            "persistent://my-org/production/orders",
    SubscriptionName: "processor",
    Schema:           pulsar.NewAvroSchema(orderSchema, nil),
})

msg, _ := consumer.Receive(ctx)
var order Order
msg.GetSchemaValue(&order)
```

## Tiered Storage Offload

Configure namespace-level offload to S3/GCS for cold data retention without disk cost:

```bash
# Configure S3 offload via CLI
pulsar-admin namespaces set-offload-threshold \
  --size 10G \
  my-org/production

pulsar-admin namespaces set-offload-policies \
  --driver s3 \
  --region us-east-1 \
  --bucket my-pulsar-offload \
  --endpoint https://s3.amazonaws.com \
  my-org/production
```

```java
// Trigger offload manually for a topic
pulsarAdmin.topics().triggerOffload(
    "persistent://my-org/production/orders",
    messageIdToOffloadFrom
);
```

## Geo-Replication

Enable replication between clusters at namespace level — producers publish to local cluster, Pulsar replicates automatically:

```bash
# Allow replication to us-west cluster
pulsar-admin namespaces set-clusters my-org/production \
  --clusters us-east,us-west
```

```go
// Produce with selective replication
producer.Send(ctx, &pulsar.ProducerMessage{
    Payload:             payload,
    ReplicationClusters: []string{"us-west"}, // override: replicate only to us-west
})
```

## Pulsar Functions (Serverless Compute)

Lightweight stateless processing without an external stream processor:

```go
// Go function: filter and transform
func OrderEnricher(ctx context.Context, input []byte) ([]byte, error) {
    var order Order
    json.Unmarshal(input, &order)
    order.EnrichedAt = time.Now()
    return json.Marshal(order)
}
```

```bash
# Deploy function
pulsar-admin functions create \
  --go ./order-enricher \
  --inputs persistent://my-org/production/orders \
  --output persistent://my-org/production/enriched-orders \
  --name order-enricher \
  --parallelism 3
```

## Dead-Letter Topic

```go
consumer, _ := client.Subscribe(pulsar.ConsumerOptions{
    Topic:            "persistent://my-org/production/orders",
    SubscriptionName: "order-processor",
    Type:             pulsar.Shared,
    DLQ: &pulsar.DLQPolicy{
        MaxDeliveries:   5,                             // after 5 nacks → DLQ
        DeadLetterTopic: "persistent://my-org/production/orders-dlq",
    },
})
```

## Key Rules

- Topic naming: `persistent://tenant/namespace/topic` — use namespaces to scope retention, quotas, and replication per environment.
- Use `Key_Shared` subscription for per-entity ordering with horizontal scale (e.g. per-customer order processing).
- Use `Shared` subscription for competing consumers (task queue pattern) — no ordering guarantees.
- Configure `NackRedeliveryDelay` to avoid tight retry loops on processing failures.
- Set DLQ policy at consumer level with `MaxDeliveries` — Pulsar handles routing to dead-letter topic automatically.
- Enable tiered storage offload for long-retention topics to reduce broker disk cost.
- Pulsar Functions are best for simple stateless transforms — use Flink/Spark for stateful aggregations.
- Schema registry prevents schema drift — enforce schemas in production namespaces.
