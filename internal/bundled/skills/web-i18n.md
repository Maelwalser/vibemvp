# Web Internationalization Skill Guide

## i18next (Framework-Agnostic)

### Setup

```ts
import i18next from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import HttpBackend from 'i18next-http-backend'

await i18next
  .use(LanguageDetector)
  .use(HttpBackend)
  .init({
    fallbackLng: 'en',
    supportedLngs: ['en', 'fr', 'de', 'es'],
    defaultNS: 'common',
    ns: ['common', 'auth', 'dashboard'],
    backend: {
      loadPath: '/locales/{{lng}}/{{ns}}.json',
    },
    detection: {
      order: ['querystring', 'cookie', 'localStorage', 'navigator'],
      caches: ['localStorage', 'cookie'],
    },
    interpolation: {
      escapeValue: false,   // React already escapes
    },
  })
```

### React: useTranslation Hook

```tsx
import { useTranslation } from 'react-i18next'

function Greeting({ name, count }: { name: string; count: number }) {
  const { t, i18n } = useTranslation('common')

  return (
    <div>
      {/* Interpolation */}
      <p>{t('greeting', { name })}</p>

      {/* Pluralization (uses count key automatically) */}
      <p>{t('items', { count })}</p>

      {/* Nested keys */}
      <p>{t('auth.login.title')}</p>

      {/* Context variant */}
      <p>{t('status', { context: 'pending' })}</p>

      {/* Language switcher */}
      <button onClick={() => i18n.changeLanguage('fr')}>FR</button>
    </div>
  )
}
```

### Translation JSON Structure

```json
// /locales/en/common.json
{
  "greeting": "Hello, {{name}}!",
  "items": "{{count}} item",
  "items_other": "{{count}} items",
  "items_zero": "No items",
  "status": "Status: active",
  "status_pending": "Status: pending",
  "status_inactive": "Status: inactive",
  "auth": {
    "login": { "title": "Sign in" }
  }
}
```

---

## next-intl

### [locale] Route Segment Setup

```
app/
├── [locale]/
│   ├── layout.tsx
│   ├── page.tsx
│   └── dashboard/
│       └── page.tsx
└── middleware.ts
```

```ts
// middleware.ts
import createMiddleware from 'next-intl/middleware'

export default createMiddleware({
  locales: ['en', 'fr', 'de'],
  defaultLocale: 'en',
  localePrefix: 'as-needed',   // /en → /, /fr → /fr
})

export const config = {
  matcher: ['/((?!api|_next|.*\\..*).*)'],
}
```

```tsx
// app/[locale]/layout.tsx
import { NextIntlClientProvider } from 'next-intl'
import { getMessages } from 'next-intl/server'

export default async function LocaleLayout({
  children,
  params: { locale },
}: {
  children: React.ReactNode
  params: { locale: string }
}) {
  const messages = await getMessages()

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
```

### Server Components

```tsx
// app/[locale]/page.tsx (Server Component)
import { getTranslations } from 'next-intl/server'

export async function generateMetadata({ params: { locale } }: { params: { locale: string } }) {
  const t = await getTranslations({ locale, namespace: 'Meta' })
  return { title: t('home.title'), description: t('home.description') }
}

export default async function HomePage() {
  const t = await getTranslations('Home')
  return <h1>{t('welcome')}</h1>
}
```

### Client Components

```tsx
'use client'
import { useTranslations, useLocale, useFormatter } from 'next-intl'

function PriceDisplay({ amount, currency }: { amount: number; currency: string }) {
  const t = useTranslations('Shop')
  const locale = useLocale()
  const format = useFormatter()

  const price = format.number(amount, { style: 'currency', currency })
  const date = format.dateTime(new Date(), { dateStyle: 'medium' })

  return (
    <div>
      <span>{t('price', { price })}</span>
      <span>{date}</span>
    </div>
  )
}
```

### Translation Files

```json
// messages/en.json
{
  "Home": {
    "welcome": "Welcome back!",
    "items": "{count, plural, =0 {No items} one {# item} other {# items}}"
  },
  "Meta": {
    "home": { "title": "Home | My App", "description": "..." }
  }
}
```

---

## vue-i18n

```ts
// i18n/index.ts
import { createI18n } from 'vue-i18n'

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  legacy: false,       // use Composition API
  messages: {},        // loaded lazily
})

// Lazy locale loading
export async function loadLocale(locale: string) {
  if (!i18n.global.availableLocales.includes(locale)) {
    const messages = await import(`./locales/${locale}.json`)
    i18n.global.setLocaleMessage(locale, messages.default)
  }
  i18n.global.locale.value = locale
}

export default i18n
```

```vue
<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

function switchLang(lang: string) {
  locale.value = lang
}
</script>

<template>
  <p>{{ t('greeting', { name: 'World' }) }}</p>
  <p>{{ t('items', { count: 5 }) }}</p>
  <button @click="switchLang('fr')">FR</button>
</template>
```

---

## Timezone Handling

### Core Principle

Always store dates as UTC in the database. Display in the user's local timezone.

### date-fns-tz

```ts
import { formatInTimeZone, toZonedTime, fromZonedTime } from 'date-fns-tz'

// Display UTC date in user timezone
function displayDate(utcDate: Date, userTimezone: string): string {
  return formatInTimeZone(utcDate, userTimezone, 'PPpp')
  // → "Apr 2, 2026, 3:45 PM" (in user's TZ)
}

// Convert local input to UTC for storage
function toUTC(localDate: Date, userTimezone: string): Date {
  return fromZonedTime(localDate, userTimezone)
}

// Get user timezone
const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone
// → 'America/New_York'
```

### Luxon

```ts
import { DateTime } from 'luxon'

// Parse UTC ISO string, convert to user TZ
const dt = DateTime.fromISO('2026-04-02T15:00:00Z', { zone: 'utc' })
  .setZone('America/New_York')

console.log(dt.toFormat('MMM d, yyyy h:mm a'))  // → "Apr 2, 2026 11:00 AM"

// User input in local TZ → UTC for storage
const userInput = DateTime.fromISO('2026-04-02T11:00:00', { zone: 'America/New_York' })
const utcForStorage = userInput.toUTC().toISO()
```

### Temporal API (Modern)

```ts
// Temporal is stage 3 — use polyfill in production
import { Temporal } from '@js-temporal/polyfill'

const utcInstant = Temporal.Instant.from('2026-04-02T15:00:00Z')
const userTZ = Temporal.Now.timeZoneId()  // 'America/New_York'

const zoned = utcInstant.toZonedDateTimeISO(userTZ)
console.log(zoned.toLocaleString('en-US'))

// Duration
const duration = Temporal.Duration.from({ hours: 1, minutes: 30 })
const later = zoned.add(duration)
```

### Intl.DateTimeFormat (Native, No Library)

```ts
function formatDate(date: Date, timezone: string, locale = 'en'): string {
  return new Intl.DateTimeFormat(locale, {
    timeZone: timezone,
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function getRelativeTime(date: Date, locale = 'en'): string {
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  const diffMs = date.getTime() - Date.now()
  const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24))

  if (Math.abs(diffDays) < 1) {
    const diffHours = Math.round(diffMs / (1000 * 60 * 60))
    return rtf.format(diffHours, 'hour')
  }
  return rtf.format(diffDays, 'day')
}
```

---

## Key Rules

- Store all dates in UTC; convert to user TZ only at display time.
- Use `Intl.DateTimeFormat().resolvedOptions().timeZone` to detect user timezone without libraries.
- next-intl: use `getTranslations` in Server Components and `useTranslations` in Client Components.
- Pluralization rules differ by language — always use the `count` variable with ICU message format.
- Lazy-load locale files by language code to avoid bundling all translations upfront.
- Never hardcode timezone offsets (e.g. `-05:00`) — named TZ IDs (e.g. `America/New_York`) handle DST automatically.
