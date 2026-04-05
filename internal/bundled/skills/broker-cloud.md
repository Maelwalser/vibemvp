# Cloud Managed Brokers Skill Guide

## Overview

AWS SQS/SNS, GCP Pub/Sub, and Azure Service Bus are fully managed messaging services. No broker infrastructure to operate — configure via SDK and IAM/IAP policies. Prefer these over self-hosted brokers when running in a single cloud.

---

## AWS SQS

### Standard vs FIFO Queues

| Feature | Standard | FIFO |
|---------|----------|------|
| Ordering | Best-effort | Strict per `MessageGroupId` |
| Throughput | Unlimited | 3,000 msg/s with batching |
| Exactly-once | No | Yes (deduplication) |
| Use case | High-throughput tasks | Ordered workflows |

### Producer — SendMessage

```go
package sqs

import (
    "context"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/sqs"
)

func Send(ctx context.Context, client *sqs.Client, queueURL, body string) error {
    _, err := client.SendMessage(ctx, &sqs.SendMessageInput{
        QueueUrl:    aws.String(queueURL),
        MessageBody: aws.String(body),
        // FIFO only:
        // MessageGroupId:         aws.String("customer-123"),
        // MessageDeduplicationId: aws.String(idempotencyKey),
    })
    if err != nil {
        return fmt.Errorf("sqs send: %w", err)
    }
    return nil
}
```

```typescript
import { SQSClient, SendMessageCommand } from '@aws-sdk/client-sqs';

const sqs = new SQSClient({ region: process.env.AWS_REGION });

await sqs.send(new SendMessageCommand({
  QueueUrl: process.env.QUEUE_URL,
  MessageBody: JSON.stringify(payload),
  // MessageGroupId: 'orders',          // FIFO only
  // MessageDeduplicationId: idempKey,  // FIFO only
}));
```

### Consumer — ReceiveMessage + DeleteMessage

```go
func Poll(ctx context.Context, client *sqs.Client, queueURL string) error {
    for {
        out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
            QueueUrl:            aws.String(queueURL),
            MaxNumberOfMessages: 10,
            WaitTimeSeconds:     20,  // long polling — reduces empty responses and cost
            VisibilityTimeout:   30,  // seconds before message reappears for retry
        })
        if err != nil {
            return fmt.Errorf("sqs receive: %w", err)
        }

        for _, msg := range out.Messages {
            if err := process(*msg.Body); err != nil {
                // do NOT delete — visibility timeout expiry causes retry
                continue
            }
            // Delete only after successful processing
            client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
                QueueUrl:      aws.String(queueURL),
                ReceiptHandle: msg.ReceiptHandle,
            })
        }
    }
}
```

### DLQ Redrive Policy

Configure in Terraform or CloudFormation — not in application code:

```hcl
resource "aws_sqs_queue" "main" {
  name                      = "orders"
  visibility_timeout_seconds = 30
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 5   # after 5 failed deliveries → DLQ
  })
}

resource "aws_sqs_queue" "dlq" {
  name = "orders-dlq"
}
```

---

## AWS SNS — Fan-out to Multiple Destinations

```go
import "github.com/aws/aws-sdk-go-v2/service/sns"

// Publish to topic — SNS fans out to all subscriptions (SQS, Lambda, HTTP, email)
_, err := snsClient.Publish(ctx, &sns.PublishInput{
    TopicArn: aws.String(topicARN),
    Message:  aws.String(string(payload)),
    MessageAttributes: map[string]snstypes.MessageAttributeValue{
        "eventType": {
            DataType:    aws.String("String"),
            StringValue: aws.String("order.created"),
        },
    },
})
```

SNS → SQS subscription with filter policy (console/Terraform): only deliver messages where `eventType = order.created` to specific queues.

---

## GCP Pub/Sub

### Publisher

```go
import "cloud.google.com/go/pubsub"

client, _ := pubsub.NewClient(ctx, projectID)
topic := client.Topic("orders")
topic.PublishSettings.DelayThreshold = 10 * time.Millisecond

result := topic.Publish(ctx, &pubsub.Message{
    Data: payload,
    Attributes: map[string]string{"eventType": "order.created"},
})
// Block to confirm delivery
if _, err := result.Get(ctx); err != nil {
    return fmt.Errorf("pubsub publish: %w", err)
}
```

### Subscriber

```go
sub := client.Subscription("orders-sub")
sub.ReceiveSettings.MaxOutstandingMessages = 50
sub.ReceiveSettings.NumGoroutines = 4

err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
    if err := process(msg.Data); err != nil {
        msg.Nack()  // re-deliver; ack deadline configurable (10s–600s)
        return
    }
    msg.Ack()
})
```

```typescript
// TypeScript
import { PubSub } from '@google-cloud/pubsub';

const pubsub = new PubSub({ projectId: process.env.GCP_PROJECT });
const subscription = pubsub.subscription('orders-sub');

subscription.on('message', async (msg) => {
  try {
    await processOrder(JSON.parse(msg.data.toString()));
    msg.ack();
  } catch {
    msg.nack();
  }
});
```

---

## Azure Service Bus

### Setup

```go
import "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"

client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
sender, err := client.NewSender("orders", nil)
receiver, err := client.NewReceiverForQueue("orders", nil)
```

### Producer

```go
err = sender.SendMessage(ctx, &azservicebus.Message{
    Body:        payload,
    ContentType: to.Ptr("application/json"),
    MessageID:   to.Ptr(idempotencyKey), // deduplication within 5-minute window
    // SessionID: to.Ptr("customer-123")  // sessions for ordered processing
}, nil)
```

### Consumer

```go
messages, err := receiver.ReceiveMessages(ctx, 10, nil)
for _, msg := range messages {
    if err := process(msg.Body); err != nil {
        receiver.AbandonMessage(ctx, msg, nil)  // re-enqueue
        continue
    }
    receiver.CompleteMessage(ctx, msg, nil)     // remove from queue
}
```

### Sessions (Ordered Processing)

```go
// Sessions guarantee FIFO ordering per SessionId group
sessionReceiver, _ := client.AcceptNextSessionForQueue(ctx, "orders", nil)
// Only messages with the same SessionId are delivered to this receiver in order
```

## Key Rules

- Always use **long polling** on SQS (`WaitTimeSeconds=20`) — reduces API calls and cost by up to 90%.
- Set `VisibilityTimeout` longer than your processing time to avoid duplicate delivery during retries.
- Delete SQS messages only after successful processing — failure leaves the message for retry until DLQ.
- Use FIFO queues with `MessageGroupId` when strict per-entity ordering is required.
- SNS + SQS fan-out pattern: one SNS topic distributes events to multiple SQS queues with different filter policies.
- GCP Pub/Sub ack deadline defaults to 10s — extend with `ModifyAckDeadline` for long-running processing.
- Azure Service Bus sessions enable ordered, FIFO processing per `SessionId` — use for per-customer workflows.
- Set DLQ/dead-lettering policies at infrastructure level (Terraform/Pulumi), not application code.
