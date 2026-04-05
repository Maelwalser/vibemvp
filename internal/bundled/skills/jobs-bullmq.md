# BullMQ Skill Guide

## Overview

BullMQ is a Node.js Redis-backed job queue with priority, delayed jobs, repeatable (cron) jobs, DAG-style flow dependencies, and rate limiting. It uses three core classes: `Queue` (enqueue), `Worker` (process), `QueueEvents` (observe).

## Installation & Setup

```bash
npm install bullmq ioredis
```

```typescript
import { Queue, Worker, QueueEvents, FlowProducer } from 'bullmq';
import { Redis } from 'ioredis';

const connection = new Redis(process.env.REDIS_URL!, {
  maxRetriesPerRequest: null,  // required for BullMQ blocking calls
  enableReadyCheck: false,
});
```

## Queue — Enqueuing Jobs

```typescript
const orderQueue = new Queue('orders', { connection });

// Basic enqueue
await orderQueue.add('process-order', { orderId: '123', customerId: 'cust-456' });

// With job options
await orderQueue.add(
  'process-order',
  { orderId: '123' },
  {
    attempts: 3,                      // retry up to 3 times
    backoff: {
      type: 'exponential',
      delay: 1000,                    // 1s, 2s, 4s
    },
    priority: 10,                     // lower number = higher priority
    delay: 5000,                      // delay first attempt by 5s
    removeOnComplete: { count: 100 }, // keep last 100 completed jobs
    removeOnFail: { count: 500 },     // keep last 500 failed jobs for inspection
    jobId: `order-${orderId}`,        // deduplicate by stable ID
  },
);
```

## Worker — Processing Jobs

```typescript
const worker = new Worker(
  'orders',
  async (job) => {
    const { orderId, customerId } = job.data;

    // Report progress (visible in dashboards)
    await job.updateProgress(10);

    const result = await chargePayment(customerId, job.data.amount);
    await job.updateProgress(50);

    await createShipment(orderId);
    await job.updateProgress(100);

    return { confirmationId: result.id };
  },
  {
    connection,
    concurrency: 10,   // process up to 10 jobs in parallel
    limiter: {
      max: 100,        // rate limit: max 100 jobs
      duration: 1000,  // per 1000ms
    },
  },
);

// Graceful shutdown
worker.on('closed', () => console.log('Worker closed'));
process.on('SIGTERM', async () => {
  await worker.close();
});
```

## QueueEvents — Observability

```typescript
const queueEvents = new QueueEvents('orders', { connection });

queueEvents.on('completed', ({ jobId, returnvalue }) => {
  console.log(`Job ${jobId} completed:`, returnvalue);
});

queueEvents.on('failed', ({ jobId, failedReason }) => {
  console.error(`Job ${jobId} failed: ${failedReason}`);
});

queueEvents.on('progress', ({ jobId, data }) => {
  console.log(`Job ${jobId} progress: ${data}%`);
});
```

## Repeatable Jobs (Cron)

```typescript
// Add once — BullMQ deduplicates repeatable jobs by name+cron
await orderQueue.add(
  'daily-report',
  { reportType: 'summary' },
  {
    repeat: {
      pattern: '0 9 * * 1-5',  // 9am Mon-Fri (cron syntax)
      // or fixed interval:
      // every: 60000,          // every 60 seconds
    },
    removeOnComplete: true,
  },
);

// List repeatable jobs
const repeatableJobs = await orderQueue.getRepeatableJobs();

// Remove a repeatable job
await orderQueue.removeRepeatable('daily-report', { pattern: '0 9 * * 1-5' });
```

## FlowProducer — DAG-Style Dependencies

Jobs run in dependency order: children complete before parent.

```typescript
const flow = new FlowProducer({ connection });

const jobTree = await flow.add({
  name: 'fulfill-order',
  queueName: 'orders',
  data: { orderId: '123' },
  children: [
    {
      name: 'charge-payment',
      queueName: 'payments',
      data: { orderId: '123', amount: 99.99 },
    },
    {
      name: 'reserve-inventory',
      queueName: 'inventory',
      data: { orderId: '123', items: ['sku-1'] },
      children: [
        {
          name: 'check-stock',
          queueName: 'inventory',
          data: { sku: 'sku-1' },
        },
      ],
    },
  ],
});
// fulfill-order runs only after charge-payment and reserve-inventory complete
```

## Dead Letter Queue Pattern

BullMQ does not have a native DLQ — inspect failed jobs and handle them:

```typescript
// Get failed jobs (acts as DLQ)
const failedJobs = await orderQueue.getFailed(0, 99);

for (const job of failedJobs) {
  console.log(`Failed job ${job.id}:`, job.failedReason);

  if (isPermanentFailure(job.failedReason)) {
    // Archive to external DLQ or DB
    await archiveToDLQ(job);
    await job.remove();
  } else {
    // Re-add with reset attempts
    await job.retry('failed');
  }
}
```

Configure `maxStalledCount` to control stalled job handling:

```typescript
const worker = new Worker('orders', processor, {
  connection,
  stalledInterval: 30000,  // check for stalled jobs every 30s
  maxStalledCount: 2,      // move to failed after 2 stalled checks
});
```

## Typed Jobs

```typescript
interface OrderJobData {
  orderId: string;
  customerId: string;
  amount: number;
}

interface OrderJobResult {
  confirmationId: string;
}

const orderQueue = new Queue<OrderJobData, OrderJobResult>('orders', { connection });
const worker = new Worker<OrderJobData, OrderJobResult>('orders', async (job) => {
  // job.data is typed as OrderJobData
  return { confirmationId: 'conf-123' };
}, { connection });
```

## Key Rules

- Set `maxRetriesPerRequest: null` on the ioredis connection — BullMQ uses blocking commands that require it.
- Use `removeOnComplete` and `removeOnFail` with a count limit to prevent unbounded Redis memory growth.
- Set `concurrency` on Worker based on I/O vs CPU work — I/O-bound workers can handle high concurrency (50+).
- Use stable `jobId` for deduplication — if a job with the same ID exists, BullMQ skips adding a duplicate.
- FlowProducer DAGs are powerful but add complexity — prefer simple queues for independent tasks.
- Cron-based repeatable jobs must be added once (at startup) and are self-deduplicating by name + pattern.
- Rate limiter applies per-worker — set `max/duration` relative to external API limits to avoid throttling.
- Always implement graceful shutdown: `await worker.close()` on SIGTERM to drain in-flight jobs.
