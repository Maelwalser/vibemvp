# Web Analytics Skill Guide

## PostHog

### Setup

```ts
// lib/posthog.ts
import posthog from 'posthog-js'

export function initPostHog() {
  if (typeof window === 'undefined') return

  posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY!, {
    api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST ?? 'https://app.posthog.com',
    loaded: (ph) => {
      if (process.env.NODE_ENV === 'development') ph.opt_out_capturing()
    },
    capture_pageview: false,     // handle manually with router events
    session_recording: {
      mask_all_inputs: true,     // privacy: mask form inputs in recordings
    },
  })
}
```

```tsx
// app/providers.tsx
'use client'
import posthog from 'posthog-js'
import { PostHogProvider, usePostHog } from 'posthog-js/react'
import { useEffect } from 'react'
import { usePathname } from 'next/navigation'

function PageviewTracker() {
  const pathname = usePathname()
  const ph = usePostHog()
  useEffect(() => {
    ph?.capture('$pageview')
  }, [pathname, ph])
  return null
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <PostHogProvider client={posthog}>
      <PageviewTracker />
      {children}
    </PostHogProvider>
  )
}
```

### Events, Identify, Feature Flags

```ts
import posthog from 'posthog-js'

// Track custom events
posthog.capture('button_clicked', {
  button_name: 'upgrade',
  plan: 'pro',
  $current_url: window.location.href,
})

posthog.capture('purchase_completed', {
  revenue: 29.99,
  currency: 'USD',
  product_id: 'plan-pro',
})

// Identify user (call after login)
posthog.identify(user.id, {
  email: user.email,
  name: user.name,
  plan: user.plan,
  created_at: user.createdAt,
})

// Group analytics (org/team)
posthog.group('company', orgId, { name: orgName, plan: orgPlan })

// Feature flags
if (posthog.isFeatureEnabled('new-dashboard')) {
  // Show new dashboard
}

const flagPayload = posthog.getFeatureFlagPayload('experiment-variant')

// Reset on logout
posthog.reset()
```

---

## Google Analytics 4 (GA4)

### Setup with next/script

```tsx
// app/layout.tsx
import Script from 'next/script'

const GA_ID = process.env.NEXT_PUBLIC_GA_ID!  // 'G-XXXXXXXXXX'

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html>
      <head>
        <Script src={`https://www.googletagmanager.com/gtag/js?id=${GA_ID}`} strategy="afterInteractive" />
        <Script id="ga-init" strategy="afterInteractive">{`
          window.dataLayer = window.dataLayer || [];
          function gtag(){dataLayer.push(arguments);}
          gtag('js', new Date());
          gtag('config', '${GA_ID}', { page_path: window.location.pathname });
        `}</Script>
      </head>
      <body>{children}</body>
    </html>
  )
}
```

### Events

```ts
// lib/gtag.ts
declare global {
  interface Window { gtag: (...args: unknown[]) => void }
}

export function pageview(url: string) {
  window.gtag('config', process.env.NEXT_PUBLIC_GA_ID!, { page_path: url })
}

export function event(action: string, params: Record<string, unknown> = {}) {
  window.gtag('event', action, params)
}

// Usage
import * as gtag from '@/lib/gtag'

gtag.event('purchase', {
  transaction_id: 'txn-123',
  value: 29.99,
  currency: 'USD',
  items: [{ item_id: 'plan-pro', item_name: 'Pro Plan', price: 29.99 }],
})

gtag.event('sign_up', { method: 'google' })
gtag.event('search', { search_term: 'react hooks' })
```

---

## Plausible (GDPR-Friendly)

```html
<!-- Simple script include — no cookie consent needed -->
<script defer data-domain="myapp.com" src="https://plausible.io/js/script.js"></script>
```

```ts
// Custom events
function trackEvent(name: string, props?: Record<string, string>) {
  if (typeof window !== 'undefined' && window.plausible) {
    window.plausible(name, { props })
  }
}

// Usage
trackEvent('Upgrade Click', { plan: 'pro', location: 'pricing-page' })
trackEvent('File Download', { format: 'pdf' })
```

```tsx
// Next.js with next-plausible
import { usePlausible } from 'next-plausible'

function UpgradeButton() {
  const plausible = usePlausible()
  return (
    <button onClick={() => plausible('UpgradeClick', { props: { plan: 'pro' } })}>
      Upgrade
    </button>
  )
}
```

---

## Mixpanel

```ts
import mixpanel from 'mixpanel-browser'

// Initialize
mixpanel.init(process.env.NEXT_PUBLIC_MIXPANEL_TOKEN!, {
  debug: process.env.NODE_ENV === 'development',
  track_pageview: true,
  persistence: 'localStorage',
})

// Track events
mixpanel.track('Button Clicked', {
  button_name: 'upgrade',
  plan: 'pro',
  page: '/pricing',
})

// Identify user
mixpanel.identify(user.id)
mixpanel.people.set({
  $email: user.email,
  $name: user.name,
  plan: user.plan,
})

// Track revenue
mixpanel.people.track_charge(29.99, { plan: 'pro' })

// Reset on logout
mixpanel.reset()
```

---

## Segment

```ts
import { Analytics } from '@segment/analytics-node'

// Server-side (Node.js)
const analytics = new Analytics({ writeKey: process.env.SEGMENT_WRITE_KEY! })

analytics.identify({
  userId: user.id,
  traits: { email: user.email, name: user.name, plan: user.plan },
})

analytics.track({
  userId: user.id,
  event: 'Purchase Completed',
  properties: { revenue: 29.99, currency: 'USD', product: 'Pro Plan' },
})

analytics.page({
  userId: user.id,
  name: 'Pricing',
  properties: { url: 'https://myapp.com/pricing' },
})

// Client-side (browser)
// analytics.js snippet loads and routes to destinations (GA4, Mixpanel, etc.)
// configured in Segment dashboard — no code changes needed per destination
```

---

## Consent Management

```ts
// lib/consent.ts
type ConsentStatus = 'pending' | 'granted' | 'denied'

const CONSENT_KEY = 'analytics_consent'

export function getConsent(): ConsentStatus {
  return (localStorage.getItem(CONSENT_KEY) as ConsentStatus) ?? 'pending'
}

export function grantConsent() {
  localStorage.setItem(CONSENT_KEY, 'granted')
  initAnalytics()
}

export function denyConsent() {
  localStorage.setItem(CONSENT_KEY, 'denied')
}

function initAnalytics() {
  // Initialize only after consent
  posthog.opt_in_capturing()
  gtag.event('consent_update', { analytics_storage: 'granted' })
}

// components/ConsentBanner.tsx
function ConsentBanner() {
  const [consent, setConsent] = useState(getConsent())

  if (consent !== 'pending') return null

  return (
    <div className="fixed bottom-0 inset-x-0 bg-white border-t p-4 flex gap-4">
      <p>We use analytics to improve your experience.</p>
      <button onClick={() => { grantConsent(); setConsent('granted') }}>Accept</button>
      <button onClick={() => { denyConsent(); setConsent('denied') }}>Decline</button>
    </div>
  )
}
```

---

## Key Rules

- Never fire analytics events before user consent if required by GDPR/CCPA.
- Identify users by server-generated ID — never use email as the primary identifier.
- Use `posthog.reset()` / `mixpanel.reset()` on logout to unlink the anonymous identity.
- PostHog and Plausible are GDPR-friendly defaults; GA4 requires explicit consent in the EU.
- Track meaningful actions (not just pageviews): sign-up, upgrade, feature usage, churn events.
- Batch analytics calls where possible — avoid firing per-keystroke events.
