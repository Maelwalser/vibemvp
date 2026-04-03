# Observability: Error Tracking Skill Guide

## Overview

Sentry, Rollbar, and Datadog error tracking — SDK initialization, exception capture, performance transactions, source maps, and user/tag context.

## Sentry

### Browser / TypeScript

```typescript
import * as Sentry from '@sentry/browser';
import { browserTracingIntegration, replayIntegration } from '@sentry/browser';

Sentry.init({
  dsn: process.env.SENTRY_DSN,
  environment: process.env.NODE_ENV,           // 'production' | 'staging'
  release: process.env.APP_VERSION,            // e.g. '1.4.2'
  integrations: [
    browserTracingIntegration(),               // automatic page load / navigation tracing
    replayIntegration({
      maskAllText: true,
      blockAllMedia: true,
    }),
  ],
  tracesSampleRate: 0.1,                       // 10% of transactions
  replaysSessionSampleRate: 0.1,
  replaysOnErrorSampleRate: 1.0,              // always record on error
});
```

### Node.js / TypeScript Backend

```typescript
import * as Sentry from '@sentry/node';

Sentry.init({
  dsn: process.env.SENTRY_DSN,
  environment: process.env.NODE_ENV,
  release: process.env.APP_VERSION,
  tracesSampleRate: 0.1,
});

// Express error handler (must be last middleware)
app.use(Sentry.Handlers.requestHandler());
app.use(Sentry.Handlers.tracingHandler());
// ... routes ...
app.use(Sentry.Handlers.errorHandler());
```

### Capture Exception with Context

```typescript
try {
  await processOrder(order);
} catch (err) {
  Sentry.captureException(err, {
    tags: { component: 'order-processor', payment_provider: 'stripe' },
    extra: { order_id: order.id, amount: order.total },
    user: { id: userId, email: userEmail },
  });
  throw err;
}
```

### Set User / Tag Context

```typescript
// Set on login
Sentry.setUser({ id: userId, email: userEmail, plan: 'pro' });

// Set global tags
Sentry.setTag('region', 'us-east-1');
Sentry.setTag('tenant', tenantSlug);

// Clear on logout
Sentry.setUser(null);
```

### Performance Transaction with Custom Span

```typescript
const transaction = Sentry.startTransaction({
  name: 'processOrder',
  op: 'task',
});

Sentry.configureScope((scope) => scope.setSpan(transaction));

const span = transaction.startChild({ op: 'db.query', description: 'INSERT orders' });
try {
  await db.insertOrder(order);
} finally {
  span.finish();
}

transaction.finish();
```

### Source Maps Upload (webpack)

```javascript
// webpack.config.js
const SentryWebpackPlugin = require('@sentry/webpack-plugin');

module.exports = {
  devtool: 'source-map',
  plugins: [
    new SentryWebpackPlugin({
      org: 'my-org',
      project: 'my-project',
      authToken: process.env.SENTRY_AUTH_TOKEN,
      release: process.env.APP_VERSION,
      include: './dist',
      urlPrefix: '~/static/',
    }),
  ],
};
```

### Go

```go
import "github.com/getsentry/sentry-go"

func main() {
    sentry.Init(sentry.ClientOptions{
        Dsn:              os.Getenv("SENTRY_DSN"),
        Environment:      os.Getenv("ENV"),
        Release:          os.Getenv("APP_VERSION"),
        TracesSampleRate: 0.1,
    })
    defer sentry.Flush(2 * time.Second)
}

func handleRequest(ctx context.Context, req Request) error {
    hub := sentry.GetHubFromContext(ctx)
    hub.Scope().SetUser(sentry.User{ID: req.UserID})
    hub.Scope().SetTag("tenant", req.TenantSlug)

    if err := process(ctx, req); err != nil {
        hub.CaptureException(err)
        return err
    }
    return nil
}
```

### Python

```python
import sentry_sdk
from sentry_sdk.integrations.fastapi import FastApiIntegration
from sentry_sdk.integrations.sqlalchemy import SqlalchemyIntegration

sentry_sdk.init(
    dsn=os.environ["SENTRY_DSN"],
    environment=os.environ.get("ENV", "production"),
    release=os.environ.get("APP_VERSION"),
    traces_sample_rate=0.1,
    integrations=[FastApiIntegration(), SqlalchemyIntegration()],
)

# Capture with context
with sentry_sdk.push_scope() as scope:
    scope.set_user({"id": user_id, "email": user_email})
    scope.set_tag("order_id", order_id)
    sentry_sdk.capture_exception(err)
```

## Rollbar

```typescript
import Rollbar from 'rollbar';

const rollbar = new Rollbar({
  accessToken: process.env.ROLLBAR_TOKEN,
  environment: process.env.NODE_ENV,
  captureUncaught: true,
  captureUnhandledRejections: true,
  payload: {
    server: {
      root: '/app',
    },
  },
});

// Capture error with request context
rollbar.error('Payment processing failed', err, {
  request: { url: req.url, method: req.method, user_ip: req.ip },
  custom: { order_id: orderId, amount: amount },
});

// Express middleware
app.use(rollbar.errorHandler());
```

## Datadog Error Tracking

```typescript
// dd-trace auto-captures exceptions from APM spans
import tracer from 'dd-trace';

tracer.init({
  service: 'user-service',
  env: process.env.NODE_ENV,
  version: process.env.APP_VERSION,
  // errors are automatically reported via APM
});
```

```python
# dd-trace Python — automatic exception capture
from ddtrace import tracer
import ddtrace.auto  # auto-instrument popular frameworks

# Custom error fingerprint for grouping
from ddtrace import tracer as dd_tracer

with dd_tracer.trace("process_order") as span:
    span.set_tag("order.id", order_id)
    span.set_tag("error.fingerprint", "payment-gateway-timeout")
    try:
        process(order)
    except Exception as e:
        span.error = 1
        span.set_tag("error.msg", str(e))
        span.set_tag("error.type", type(e).__name__)
        raise
```

## Key Rules

- Initialize Sentry/Rollbar before any route handlers — missed errors during startup are invisible.
- Always pass `environment` and `release` — they are essential for grouping and regression detection.
- Capture exceptions at the boundary closest to the error source, not at the top-level handler only.
- Attach user context (`setUser`) immediately after authentication — before any routes run.
- Source maps must be uploaded at build time with the same `release` identifier used in `init()`.
- Use `captureException` rather than `captureMessage` for actual errors — preserves stack trace.
- Set `tracesSampleRate` to 0.1 or lower in production to control cost.
- Use custom `error.fingerprint` tags in Datadog to prevent unrelated errors grouping together.
