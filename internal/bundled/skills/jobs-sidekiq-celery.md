# Sidekiq & Celery Skill Guide

## Overview

**Sidekiq** is a Redis-backed background job processor for Ruby with threads. **Celery** is a distributed task queue for Python supporting Redis, RabbitMQ, and SQS brokers. Both support scheduling, retries with backoff, and workflow primitives.

---

## Sidekiq (Ruby)

### Installation

```ruby
# Gemfile
gem 'sidekiq'
gem 'sidekiq-cron'  # for scheduled jobs
```

### Worker Definition

```ruby
class OrderProcessingWorker
  include Sidekiq::Worker

  sidekiq_options(
    queue: :orders,         # named queue for priority routing
    retry: 5,               # retry up to 5 times (false to disable)
    backtrace: true,        # capture backtrace in failed jobs
    tags: ['orders'],
  )

  def perform(order_id, customer_id)
    # args must be JSON-serializable (strings, numbers, arrays, hashes)
    order = Order.find(order_id)
    PaymentService.charge(order, customer_id)
    ShipmentService.create(order)
  end
end

# Enqueue
OrderProcessingWorker.perform_async(order.id, customer.id)

# Enqueue with delay
OrderProcessingWorker.perform_in(5.minutes, order.id, customer.id)

# Enqueue at specific time
OrderProcessingWorker.perform_at(2.hours.from_now, order.id, customer.id)
```

### Custom Retry Backoff

```ruby
class OrderProcessingWorker
  include Sidekiq::Worker

  sidekiq_options retry: 10

  # Custom exponential backoff — return seconds until next retry
  sidekiq_retry_in do |count, exception|
    case exception
    when PaymentGatewayError
      (count ** 4) + 15 + (rand(10) * (count + 1))  # exponential with jitter
    when Errno::ETIMEDOUT
      30 * (count + 1)  # linear backoff for timeouts
    else
      :kill  # move to dead set immediately
    end
  end

  def perform(order_id)
    # ...
  end
end
```

### Scheduled Jobs (sidekiq-cron)

```ruby
# config/initializers/sidekiq.rb
Sidekiq.configure_server do |config|
  config.on(:startup) do
    Sidekiq::Cron::Job.load_from_array([
      {
        name:  'Daily Summary Report',
        cron:  '0 9 * * 1-5',          # 9am Mon-Fri
        class: 'DailySummaryWorker',
      },
      {
        name:  'Hourly Cleanup',
        cron:  '0 * * * *',
        class: 'CleanupWorker',
        args:  ['stale_sessions'],
      },
    ])
  end
end
```

### Server & Client Middleware

```ruby
# Server middleware (runs around perform)
class AuditMiddleware
  def call(worker, job, queue)
    Rails.logger.info("Starting job #{job['class']} id=#{job['jid']}")
    yield
    Rails.logger.info("Completed job #{job['jid']}")
  rescue => e
    Rails.logger.error("Failed job #{job['jid']}: #{e.message}")
    raise
  end
end

Sidekiq.configure_server do |config|
  config.server_middleware do |chain|
    chain.add AuditMiddleware
  end
end
```

### Dead Set Inspection

```ruby
# Inspect dead jobs
dead = Sidekiq::DeadSet.new
dead.each do |job|
  puts "#{job.klass} args=#{job.args} failed_at=#{job.at}"
end

# Retry a dead job
dead.first&.retry

# Clear dead set
Sidekiq::DeadSet.new.clear
```

### Sidekiq Configuration

```yaml
# config/sidekiq.yml
concurrency: 10
queues:
  - [critical, 3]   # weight 3x
  - [orders, 2]
  - [default, 1]
```

---

## Celery (Python)

### Installation

```bash
pip install celery redis  # or celery[rabbitmq]
```

### App Setup

```python
# celery_app.py
from celery import Celery

app = Celery(
    'myapp',
    broker=os.environ['CELERY_BROKER_URL'],    # redis://localhost:6379/0
    backend=os.environ['CELERY_RESULT_BACKEND'],
    include=['myapp.tasks.orders', 'myapp.tasks.notifications'],
)

app.conf.update(
    task_serializer='json',
    result_serializer='json',
    accept_content=['json'],
    timezone='UTC',
    task_track_started=True,
    task_acks_late=True,       # ack after completion, not delivery
    worker_prefetch_multiplier=1,  # one task per worker at a time
)
```

### Task Definition

```python
# tasks/orders.py
from celery import shared_task
from myapp.celery_app import app

@app.task(
    bind=True,              # access self for retry/request context
    name='orders.process',
    queue='orders',
    max_retries=5,
    default_retry_delay=60,
    rate_limit='100/m',     # 100 tasks per minute per worker
    soft_time_limit=300,    # SoftTimeLimitExceeded after 5min
    time_limit=360,         # SIGKILL after 6min
)
def process_order(self, order_id: str, customer_id: str):
    try:
        order = Order.objects.get(id=order_id)
        PaymentService.charge(order, customer_id)
        ShipmentService.create(order)
        return {'confirmation_id': order.confirmation_id}
    except PaymentDeclinedError as exc:
        # non-retryable — do not retry
        raise
    except TemporaryError as exc:
        # retry with exponential backoff
        raise self.retry(
            exc=exc,
            countdown=2 ** self.request.retries * 60,  # 60s, 120s, 240s, ...
        )
```

### Enqueueing Tasks

```python
# Basic async call
process_order.delay(order_id, customer_id)

# With options
process_order.apply_async(
    args=[order_id, customer_id],
    countdown=300,           # delay 5 minutes
    # eta=datetime(2026, 1, 1, 9, 0),  # exact time
    queue='orders',
    priority=5,              # 0-9 (RabbitMQ only)
    expires=3600,            # discard if not started within 1h
)
```

### Canvas — Workflow Composition

```python
from celery import chain, group, chord

# chain: sequential pipeline
pipeline = chain(
    validate_order.s(order_id),
    charge_payment.s(),       # receives result of previous task
    create_shipment.s(),
    send_confirmation.s(),
)
pipeline.delay()

# group: parallel execution
parallel = group(
    send_email.s(customer_id),
    update_inventory.s(items),
    notify_warehouse.s(order_id),
)
parallel.delay()

# chord: parallel → callback when all done
result = chord(
    group(process_item.s(item) for item in items),
    aggregate_results.s(),    # runs after all group tasks complete
)()
```

### Beat Scheduler (Periodic Tasks)

```python
# celery_app.py
from celery.schedules import crontab

app.conf.beat_schedule = {
    'daily-summary-report': {
        'task': 'reports.generate_summary',
        'schedule': crontab(hour=9, minute=0, day_of_week='1-5'),
        'args': ('summary',),
    },
    'hourly-cleanup': {
        'task': 'maintenance.cleanup_sessions',
        'schedule': crontab(minute=0),  # every hour
    },
    'every-30-seconds': {
        'task': 'health.ping',
        'schedule': 30.0,  # timedelta seconds
    },
}
```

```bash
# Run beat scheduler (separate process)
celery -A celery_app beat --loglevel=info
# Run worker
celery -A celery_app worker --queues=orders,default --concurrency=4
```

## Key Rules

### Sidekiq
- Args must be JSON-serializable — use IDs, not ActiveRecord objects.
- Use `sidekiq_retry_in` for custom backoff; return `:kill` to skip retries and move to DeadSet.
- Set queue weights in `sidekiq.yml` — critical queues should have higher weight multipliers.
- Never share state between workers — Sidekiq runs threads concurrently; use thread-safe data structures.

### Celery
- Use `bind=True` to access `self.retry()` and `self.request.retries` in task methods.
- Set `task_acks_late=True` and `worker_prefetch_multiplier=1` for at-least-once processing with crash safety.
- Use `soft_time_limit` (raises exception) and `time_limit` (hard kill) to prevent hung tasks.
- Canvas workflows (`chain`/`group`/`chord`) must have JSON-serializable results between steps.
- Run Beat and Worker as separate processes — never combine them in production.
- Use named queues and route high-priority tasks to dedicated worker pools.
