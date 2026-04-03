# Feature Flags Skill Guide

## Overview

LaunchDarkly, Unleash, Flagsmith, and simple env-var toggles — SDK initialization, flag evaluation, user targeting, gradual rollout, and multivariate flags.

## LaunchDarkly

### Node.js / TypeScript

```typescript
import * as ld from '@launchdarkly/node-server-sdk';

let ldClient: ld.LDClient;

export async function initLaunchDarkly(): Promise<void> {
  ldClient = ld.init(process.env.LAUNCHDARKLY_SDK_KEY!, {
    offline: process.env.NODE_ENV === 'test',
  });
  await ldClient.waitForInitialization({ timeout: 10 });
}

// User context
function buildContext(userId: string, email: string, plan: string): ld.LDContext {
  return {
    kind: 'user',
    key: userId,
    email,
    custom: { plan },          // custom attributes for targeting rules
    anonymous: false,
  };
}

// Boolean flag
export async function isFeatureEnabled(
  flagKey: string,
  userId: string,
  email: string,
  plan: string,
): Promise<boolean> {
  const context = buildContext(userId, email, plan);
  return ldClient.variation(flagKey, context, false);  // false = default
}

// String multivariate flag
export async function getExperimentVariant(
  userId: string,
  email: string,
): Promise<string> {
  const context = buildContext(userId, email, 'free');
  return ldClient.variation('checkout-flow', context, 'control');
  // returns 'control' | 'variant-a' | 'variant-b'
}

// JSON multivariate flag (remote config)
export async function getPaymentConfig(userId: string): Promise<PaymentConfig> {
  const context = { kind: 'user', key: userId };
  return ldClient.variation('payment-config', context, {
    provider: 'stripe',
    maxRetries: 3,
  }) as PaymentConfig;
}

// Graceful shutdown
export function closeLaunchDarkly(): Promise<void> {
  return ldClient.close();
}
```

### Go

```go
import (
    ld "github.com/launchdarkly/go-server-sdk/v7"
    "github.com/launchdarkly/go-server-sdk/v7/ldcontext"
)

var ldClient *ld.LDClient

func InitLD(sdkKey string) error {
    var err error
    ldClient, err = ld.MakeClient(sdkKey, 5*time.Second)
    return err
}

func IsEnabled(flagKey, userID, email, plan string) bool {
    ctx := ldcontext.NewBuilder(userID).
        SetString("email", email).
        SetString("plan", plan).
        Build()
    result, _ := ldClient.BoolVariation(flagKey, ctx, false)
    return result
}
```

### Targeting Rules in LaunchDarkly UI

```
Gradual rollout: percentage of users (e.g., 10% → 50% → 100%)
User segment: users where plan = "pro" OR email ends with "@beta.example.com"
Boolean flag rollout:
  - If user is in "early-access" segment → true
  - If user.plan = "enterprise" → true
  - Otherwise: 10% rollout → true, 90% → false
```

## Unleash

### Node.js / TypeScript

```typescript
import { UnleashClient } from 'unleash-proxy-client';

const unleash = new UnleashClient({
  url: process.env.UNLEASH_URL!,           // e.g. https://us.app.unleash-hosted.com/ushosted/api/proxy
  clientKey: process.env.UNLEASH_API_KEY!,
  appName: 'user-service',
  environment: process.env.NODE_ENV,
  refreshInterval: 15,    // seconds
  metricsInterval: 60,
});

await unleash.start();

// Boolean flag with user context
function isFeatureEnabled(flagName: string, userId: string): boolean {
  return unleash.isEnabled(flagName, {
    userId,
    properties: { plan: 'pro', region: 'us-east' },
  });
}

// Multi-variant flag
function getVariant(flagName: string, userId: string): string {
  const variant = unleash.getVariant(flagName, { userId });
  return variant.enabled ? variant.name : 'control';
}
```

### Go (Unleash)

```go
import unleash "github.com/Unleash/unleash-client-go/v4"

func Init() error {
    return unleash.Initialize(
        unleash.WithUrl(os.Getenv("UNLEASH_URL")),
        unleash.WithAppName("user-service"),
        unleash.WithCustomHeaders(http.Header{
            "Authorization": []string{os.Getenv("UNLEASH_API_KEY")},
        }),
    )
}

func IsEnabled(flagName, userID string) bool {
    return unleash.IsEnabled(flagName, unleash.WithContext(context.Context{
        UserId: userID,
        Properties: map[string]string{"plan": "pro"},
    }))
}
```

### Unleash Strategies

| Strategy | When to Use |
|----------|-------------|
| `gradualRolloutUserId` | Sticky rollout by user ID (consistent per user) |
| `gradualRolloutRandom` | Random percentage (not sticky) |
| `userWithId` | Explicit list of user IDs |
| `remoteAddress` | By client IP |
| `applicationHostname` | By server hostname |
| Custom strategy | Any business rule |

### Unleash Edge Proxy (Self-hosted)

```bash
# Run Unleash Edge for low-latency flag evaluation
docker run -p 3063:3063 \
  -e UNLEASH_SERVER_URL=https://your-unleash.example.com \
  -e UNLEASH_SERVER_API_TOKEN=your-token \
  unleashorg/unleash-edge:latest
```

## Flagsmith

```typescript
import Flagsmith from 'flagsmith-nodejs';

const flagsmith = new Flagsmith({
  environmentKey: process.env.FLAGSMITH_ENV_KEY!,
  enableLocalEvaluation: true,  // evaluates flags server-side without network call
  environmentRefreshIntervalSeconds: 60,
});

// Get all flags for an identity
const flags = await flagsmith.getIdentityFlags('user-123', {
  email: 'alice@example.com',
  plan: 'pro',
});

// Boolean flag
const isEnabled = flags.isFeatureEnabled('new-checkout-ui');

// Remote config (string/number/JSON value)
const limit = flags.getFeatureValue('api_rate_limit');  // returns '1000'
const config = JSON.parse(flags.getFeatureValue('payment_config') as string);

// Without identity (anonymous)
const anonFlags = await flagsmith.getEnvironmentFlags();
const isMaintenance = anonFlags.isFeatureEnabled('maintenance_mode');
```

## Simple Env-Var Flags

For teams without an external feature flag service:

```typescript
// flags.ts
const flags = {
  newCheckoutUi: process.env.FEATURE_NEW_CHECKOUT_UI === 'true',
  betaApi: process.env.FEATURE_BETA_API === 'true',
  darkMode: process.env.FEATURE_DARK_MODE === 'true',
} as const;

export function isEnabled(flag: keyof typeof flags): boolean {
  return flags[flag] ?? false;
}
```

```bash
# .env.production
FEATURE_NEW_CHECKOUT_UI=false
FEATURE_BETA_API=true
FEATURE_DARK_MODE=false
```

```go
// Go
func IsFeatureEnabled(name string) bool {
    return os.Getenv("FEATURE_"+strings.ToUpper(name)) == "true"
}

// Usage
if IsFeatureEnabled("NEW_CHECKOUT_UI") {
    return newCheckoutHandler(c)
}
return legacyCheckoutHandler(c)
```

## Key Rules

- Always provide a safe default value — flags must work if the flag service is unreachable.
- Use sticky rollout strategies (e.g., `gradualRolloutUserId`) for consistent user experience.
- Clean up old flags regularly — a flag that is 100% rolled out and never toggled is dead code.
- Track which code paths are behind flags — they must be cleaned up after full rollout.
- Use multivariate flags for A/B tests and remote config, not just boolean on/off.
- In tests, use `offline: true` (LaunchDarkly) or a mock client — never call the real service in unit tests.
- Log flag evaluations at DEBUG level with the flag key, user ID, and result for debugging.
