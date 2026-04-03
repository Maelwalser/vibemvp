# Web Real User Monitoring (RUM) Skill Guide

## Sentry Browser

### Setup

```ts
// app/sentry.client.config.ts (Next.js) or src/main.ts
import * as Sentry from '@sentry/nextjs'
import { browserTracingIntegration, replayIntegration } from '@sentry/nextjs'

Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NODE_ENV,
  release: process.env.NEXT_PUBLIC_RELEASE_VERSION,   // e.g. git SHA

  // Performance tracing
  integrations: [
    browserTracingIntegration(),
    replayIntegration({
      maskAllText: false,
      blockAllMedia: false,
      maskAllInputs: true,    // always mask form inputs
    }),
  ],

  tracesSampleRate: process.env.NODE_ENV === 'production' ? 0.1 : 1.0,

  // Session replay
  replaysSessionSampleRate: 0.05,    // 5% of sessions
  replaysOnErrorSampleRate: 1.0,     // 100% of error sessions

  beforeSend(event) {
    // Filter out non-actionable errors
    if (event.exception?.values?.[0]?.value?.includes('ResizeObserver')) return null
    return event
  },
})
```

### Capture Errors

```ts
import * as Sentry from '@sentry/nextjs'

// Unhandled errors are captured automatically
// Manual capture:

try {
  await riskyOperation()
} catch (err) {
  Sentry.captureException(err, {
    tags: { component: 'PaymentFlow', step: 'charge' },
    extra: { orderId, amount },
    level: 'error',
  })
  throw err
}

// Capture a message (non-exception event)
Sentry.captureMessage('Deprecated API endpoint called', {
  level: 'warning',
  tags: { endpoint: '/api/v1/users' },
})

// Error boundary (React)
import { ErrorBoundary } from '@sentry/nextjs'

function App() {
  return (
    <ErrorBoundary
      fallback={({ error, resetError }) => (
        <div>
          <p>Something went wrong: {error.message}</p>
          <button onClick={resetError}>Retry</button>
        </div>
      )}
      onError={(error, componentStack) => console.error(error, componentStack)}
    >
      <Router />
    </ErrorBoundary>
  )
}
```

### User Context & Tags

```ts
import * as Sentry from '@sentry/nextjs'

// Set user context after login
Sentry.setUser({
  id: user.id,
  email: user.email,
  username: user.name,
})

// Set tags for all subsequent events
Sentry.setTag('plan', user.plan)
Sentry.setTag('region', user.region)

// Set extra context
Sentry.setExtra('featureFlags', enabledFlags)

// Scoped context (doesn't pollute global scope)
Sentry.withScope((scope) => {
  scope.setTag('operation', 'file-upload')
  scope.setExtra('fileSize', file.size)
  Sentry.captureException(err)
})

// Clear on logout
Sentry.setUser(null)
```

### Performance Tracing

```ts
import * as Sentry from '@sentry/nextjs'

// Custom span within an active transaction
async function processOrder(orderId: string) {
  return Sentry.startSpan(
    { name: 'processOrder', op: 'function' },
    async (span) => {
      span.setAttribute('order.id', orderId)

      const order = await Sentry.startSpan(
        { name: 'fetchOrder', op: 'db.query' },
        () => db.order.findUnique({ where: { id: orderId } })
      )

      await Sentry.startSpan(
        { name: 'chargePayment', op: 'http.client' },
        () => stripe.charges.create({ amount: order.total })
      )

      return order
    }
  )
}
```

### Source Map Upload

```ts
// next.config.js
const { withSentryConfig } = require('@sentry/nextjs')

module.exports = withSentryConfig(
  { /* next config */ },
  {
    org: 'my-org',
    project: 'my-project',
    authToken: process.env.SENTRY_AUTH_TOKEN,
    silent: true,
    hideSourceMaps: true,   // don't serve source maps publicly
  }
)
```

---

## Datadog RUM

### Setup

```ts
import { datadogRum } from '@datadog/browser-rum'

datadogRum.init({
  applicationId: process.env.NEXT_PUBLIC_DD_APPLICATION_ID!,
  clientToken: process.env.NEXT_PUBLIC_DD_CLIENT_TOKEN!,
  site: 'datadoghq.com',
  service: 'my-app',
  env: process.env.NODE_ENV,
  version: process.env.NEXT_PUBLIC_RELEASE_VERSION,
  sessionSampleRate: 100,
  sessionReplaySampleRate: 10,   // 10% get session replay
  trackUserInteractions: true,
  trackResources: true,
  trackLongTasks: true,
  defaultPrivacyLevel: 'mask-user-input',
})

datadogRum.startSessionReplayRecording()
```

### Custom Events

```ts
import { datadogRum } from '@datadog/browser-rum'

// Custom action (user interaction)
datadogRum.addAction('upgrade_button_clicked', {
  plan: 'pro',
  location: 'pricing_page',
})

// Custom error
try {
  await riskyOperation()
} catch (err) {
  datadogRum.addError(err, { source: 'PaymentFlow', orderId })
}

// Custom timing (for performance metrics)
const start = performance.now()
await loadData()
datadogRum.addTiming('data_load', performance.now() - start)

// Set user
datadogRum.setUser({
  id: user.id,
  email: user.email,
  name: user.name,
  plan: user.plan,
})

// Set global context
datadogRum.setGlobalContextProperty('feature_flags', enabledFlags)
```

---

## LogRocket

```ts
import LogRocket from 'logrocket'
import setupLogRocketReact from 'logrocket-react'

// Initialize
LogRocket.init('your-app/your-environment', {
  network: {
    isEnabled: true,
    requestSanitizer: (request) => {
      // Redact auth headers
      if (request.headers['Authorization']) {
        request.headers['Authorization'] = '[REDACTED]'
      }
      return request
    },
    responseSanitizer: (response) => {
      // Redact sensitive response fields
      if (response.body) {
        const body = JSON.parse(response.body)
        if (body.password) body.password = '[REDACTED]'
        response.body = JSON.stringify(body)
      }
      return response
    },
  },
})

setupLogRocketReact(LogRocket)

// Identify user
LogRocket.identify(user.id, {
  name: user.name,
  email: user.email,
  plan: user.plan,
})
```

### LogRocket + Sentry Integration

```ts
// Link LogRocket session URL to Sentry errors
import * as Sentry from '@sentry/nextjs'
import LogRocket from 'logrocket'

LogRocket.getSessionURL((sessionURL) => {
  Sentry.setContext('logrocket', { sessionURL })
})
```

---

## Key Rules

- Set `tracesSampleRate` to 0.1 (10%) in production to control costs — use 1.0 in development.
- Always set `release` version (git SHA or semver) so Sentry can track error regressions per deploy.
- Mask all form inputs in session replay (`maskAllInputs: true`) to protect user privacy.
- Call `Sentry.setUser(null)` on logout to avoid leaking user context across sessions.
- Source maps must be uploaded at deploy time — without them stack traces are useless minified bundles.
- Use `withScope` for contextual errors to avoid polluting the global Sentry scope.
- Link LogRocket session URLs to Sentry errors to enable replay-on-error debugging workflows.
