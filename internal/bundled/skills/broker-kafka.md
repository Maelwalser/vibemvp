# Kafka Skill Guide

## Overview

Apache Kafka is a distributed event streaming platform built for high-throughput, fault-tolerant, ordered message delivery. Producers write to topics partitioned for parallelism; consumers read via consumer groups for horizontal scaling.

## Producer Pattern

### Go (confluent-kafka-go)

```go
package kafka

import (
    "fmt"
    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func NewProducer(brokers string) (*kafka.Producer, error) {
    return kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers":  brokers,
        "acks":               "all",        // wait for all ISR replicas
        "retries":            5,
        "retry.backoff.ms":   200,
        "enable.idempotence": true,         // exactly-once at producer level
    })
}

func Produce(p *kafka.Producer, topic, key string, value []byte) error {
    deliveryChan := make(chan kafka.Event, 1)
    err := p.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     &topic,
            Partition: kafka.PartitionAny,
        },
        Key:   []byte(key),   // key-based partitioning — same key → same partition
        Value: value,
    }, deliveryChan)
    if err != nil {
        return fmt.Errorf("produce enqueue: %w", err)
    }
    e := <-deliveryChan
    m := e.(*kafka.Message)
    if m.TopicPartition.Error != nil {
        return fmt.Errorf("produce delivery: %w", m.TopicPartition.Error)
    }
    return nil
}
```

### TypeScript (kafkajs)

```typescript
import { Kafka, CompressionTypes } from 'kafkajs';

const kafka = new Kafka({ brokers: ['localhost:9092'] });
const producer = kafka.producer({ idempotent: true });

await producer.connect();

await producer.send({
  topic: 'orders',
  messages: [
    {
      key: order.customerId,       // key-based partition routing
      value: JSON.stringify(order),
      headers: { source: 'api' },
    },
  ],
  acks: -1,                        // acks=all
  compression: CompressionTypes.GZIP,
});
```

## Consumer / Worker Pattern

### Go — Manual Offset Commit

```go
func NewConsumer(brokers, groupID, topic string) error {
    c, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers":        brokers,
        "group.id":                 groupID,
        "auto.offset.reset":        "earliest",
        "enable.auto.commit":       false,   // manual commit for at-least-once
        "max.poll.interval.ms":     300000,
    })
    if err != nil {
        return fmt.Errorf("new consumer: %w", err)
    }
    defer c.Close()

    if err := c.SubscribeTopics([]string{topic}, nil); err != nil {
        return fmt.Errorf("subscribe: %w", err)
    }

    for {
        msg, err := c.ReadMessage(-1)
        if err != nil {
            continue // poll errors are informational
        }

        if err := process(msg.Value); err != nil {
            // log and decide: retry, DLQ, or skip
            continue
        }

        // commitSync after successful processing
        if _, err := c.CommitMessage(msg); err != nil {
            return fmt.Errorf("commit offset: %w", err)
        }
    }
}
```

### TypeScript (kafkajs)

```typescript
const consumer = kafka.consumer({ groupId: 'order-processor' });
await consumer.connect();
await consumer.subscribe({ topic: 'orders', fromBeginning: false });

await consumer.run({
  autoCommit: false,
  eachMessage: async ({ topic, partition, message, heartbeat }) => {
    await processOrder(JSON.parse(message.value!.toString()));
    await consumer.commitOffsets([
      { topic, partition, offset: (Number(message.offset) + 1).toString() },
    ]);
  },
});
```

## Exactly-Once Semantics (Transactions)

```go
// Producer side: transactional producer
p, _ := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers":     brokers,
    "transactional.id":      "order-tx-1",  // unique per producer instance
    "enable.idempotence":    true,
})
p.InitTransactions(nil)
p.BeginTransaction()

// produce messages inside transaction
p.Produce(&kafka.Message{...}, nil)

// commit or abort
if err := p.CommitTransaction(nil); err != nil {
    p.AbortTransaction(nil)
}
```

## Kafka Streams DSL (Java)

```java
StreamsBuilder builder = new StreamsBuilder();

KStream<String, Order> orders = builder.stream("raw-orders");

orders
    .filter((key, order) -> order.getAmount() > 0)
    .mapValues(order -> enrich(order))
    .to("enriched-orders", Produced.with(Serdes.String(), orderSerde));

KafkaStreams streams = new KafkaStreams(builder.build(), streamsConfig);
streams.start();
Runtime.getRuntime().addShutdownHook(new Thread(streams::close));
```

## Error Handling & DLQ

```go
// Send to dead-letter topic on unrecoverable processing error
func sendToDLQ(p *kafka.Producer, dlqTopic string, msg *kafka.Message, reason string) error {
    return Produce(p, dlqTopic, string(msg.Key), msg.Value)
}

// Retry policy: attempt N times, then DLQ
const maxRetries = 3
for attempt := 0; attempt < maxRetries; attempt++ {
    if err := process(msg.Value); err == nil {
        break
    }
    if attempt == maxRetries-1 {
        sendToDLQ(dlqProducer, topic+".dlq", msg, err.Error())
    }
    time.Sleep(time.Duration(attempt+1) * time.Second)
}
```

## Key Rules

- Set `enable.auto.commit=false` and call `commitSync`/`CommitMessage` only after successful processing.
- Use `acks=all` + `enable.idempotence=true` on producers to prevent data loss and duplicates.
- Assign `group.id` per logical consumer group; different services must use different group IDs.
- Partition by a meaningful key (e.g. `customerId`) to preserve ordering within an entity.
- Use transactional producers for exactly-once semantics across produce + consume + produce pipelines.
- DLQ topic convention: `<topic>.dlq`; include original offset, partition, and error reason as headers.
- Never call blocking I/O inside `eachMessage` without a heartbeat — increase `max.poll.interval.ms` or offload to a worker pool.
