# Web PWA Skill Guide

## Web App Manifest

```json
// public/manifest.json
{
  "name": "My Application",
  "short_name": "MyApp",
  "description": "A Progressive Web Application",
  "start_url": "/",
  "display": "standalone",
  "orientation": "portrait-primary",
  "theme_color": "#3b82f6",
  "background_color": "#ffffff",
  "scope": "/",
  "lang": "en",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ],
  "screenshots": [
    {
      "src": "/screenshots/desktop.png",
      "sizes": "1280x720",
      "type": "image/png",
      "form_factor": "wide"
    }
  ],
  "shortcuts": [
    {
      "name": "New Item",
      "url": "/new",
      "icons": [{ "src": "/icons/new.png", "sizes": "96x96" }]
    }
  ]
}
```

```html
<!-- In <head> -->
<link rel="manifest" href="/manifest.json" />
<meta name="theme-color" content="#3b82f6" />
<meta name="apple-mobile-web-app-capable" content="yes" />
<meta name="apple-mobile-web-app-status-bar-style" content="default" />
<link rel="apple-touch-icon" href="/icons/icon-192.png" />
```

---

## Service Worker

### Registration

```ts
// app/layout.tsx or index.tsx
if ('serviceWorker' in navigator) {
  window.addEventListener('load', async () => {
    try {
      const registration = await navigator.serviceWorker.register('/sw.js', {
        scope: '/',
      })
      console.log('SW registered:', registration.scope)

      registration.addEventListener('updatefound', () => {
        const newWorker = registration.installing
        newWorker?.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            // New version available — prompt user to refresh
            showUpdateBanner()
          }
        })
      })
    } catch (err) {
      console.error('SW registration failed:', err)
    }
  })
}
```

### Service Worker Lifecycle

```ts
// public/sw.js
const CACHE_NAME = 'app-v1'
const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/static/js/main.js',
  '/static/css/main.css',
  '/offline.html',
]

// Install — cache static assets and take control immediately
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(STATIC_ASSETS))
  )
  self.skipWaiting()
})

// Activate — delete old caches and claim clients
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
    )
  )
  self.clients.claim()
})

// Fetch — route-based caching strategy
self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  // API calls: network-first
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirst(request))
    return
  }

  // Static assets: cache-first
  if (request.destination === 'script' || request.destination === 'style' || request.destination === 'image') {
    event.respondWith(cacheFirst(request))
    return
  }

  // HTML pages: stale-while-revalidate
  if (request.mode === 'navigate') {
    event.respondWith(staleWhileRevalidate(request))
    return
  }

  event.respondWith(cacheFirst(request))
})
```

### Caching Strategies

```ts
// Cache-first: return cached, fallback to network (great for static assets)
async function cacheFirst(request: Request): Promise<Response> {
  const cached = await caches.match(request)
  if (cached) return cached

  try {
    const response = await fetch(request)
    const cache = await caches.open(CACHE_NAME)
    cache.put(request, response.clone())
    return response
  } catch {
    return caches.match('/offline.html') as Promise<Response>
  }
}

// Network-first: try network, fallback to cache (great for API data)
async function networkFirst(request: Request): Promise<Response> {
  try {
    const response = await fetch(request)
    const cache = await caches.open(CACHE_NAME)
    cache.put(request, response.clone())
    return response
  } catch {
    const cached = await caches.match(request)
    return cached ?? new Response('{"error":"offline"}', {
      status: 503,
      headers: { 'Content-Type': 'application/json' },
    })
  }
}

// Stale-while-revalidate: return cached immediately, update in background
async function staleWhileRevalidate(request: Request): Promise<Response> {
  const cache = await caches.open(CACHE_NAME)
  const cached = await cache.match(request)

  const fetchPromise = fetch(request).then((response) => {
    cache.put(request, response.clone())
    return response
  }).catch(() => null)

  return cached ?? (await fetchPromise) ?? (await caches.match('/offline.html') as Response)
}
```

---

## Push Notifications

### Frontend

```ts
// Request permission and subscribe
async function subscribeToPush(vapidPublicKey: string) {
  if (!('Notification' in window) || !('serviceWorker' in navigator)) return null

  const permission = await Notification.requestPermission()
  if (permission !== 'granted') return null

  const registration = await navigator.serviceWorker.ready

  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
  })

  // Send subscription to server
  await fetch('/api/push/subscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subscription),
  })

  return subscription
}

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = atob(base64)
  return Uint8Array.from([...rawData].map((c) => c.charCodeAt(0)))
}
```

### Service Worker Push Handler

```ts
// In sw.js
self.addEventListener('push', (event) => {
  const data = event.data?.json() ?? { title: 'Notification', body: '' }

  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: '/icons/icon-192.png',
      badge: '/icons/badge-72.png',
      data: { url: data.url ?? '/' },
      actions: [
        { action: 'open', title: 'Open' },
        { action: 'dismiss', title: 'Dismiss' },
      ],
    })
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()

  if (event.action === 'dismiss') return

  event.waitUntil(
    clients.matchAll({ type: 'window' }).then((windowClients) => {
      const client = windowClients.find((c) => c.url === event.notification.data.url)
      if (client) return client.focus()
      return clients.openWindow(event.notification.data.url)
    })
  )
})
```

---

## vite-plugin-pwa

```ts
// vite.config.ts
import { defineConfig } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
        runtimeCaching: [
          {
            urlPattern: /^https:\/\/api\.example\.com\/.*/i,
            handler: 'NetworkFirst',
            options: { cacheName: 'api-cache', networkTimeoutSeconds: 10 },
          },
        ],
      },
      manifest: {
        name: 'My App',
        short_name: 'App',
        theme_color: '#3b82f6',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
        ],
      },
    }),
  ],
})
```

## next-pwa

```ts
// next.config.js
const withPWA = require('next-pwa')({
  dest: 'public',
  register: true,
  skipWaiting: true,
  disable: process.env.NODE_ENV === 'development',
  runtimeCaching: [
    {
      urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/i,
      handler: 'CacheFirst',
      options: { cacheName: 'google-fonts', expiration: { maxEntries: 10, maxAgeSeconds: 60 * 60 * 24 * 365 } },
    },
    {
      urlPattern: /\/api\/.*/i,
      handler: 'NetworkFirst',
      options: { cacheName: 'api-cache', networkTimeoutSeconds: 10 },
    },
  ],
})

module.exports = withPWA({ reactStrictMode: true })
```

---

## Key Rules

- Always provide both 192x192 and 512x512 icons; use `purpose: "maskable"` for Android adaptive icons.
- Call `self.skipWaiting()` in `install` and `self.clients.claim()` in `activate` to take control immediately.
- Use `staleWhileRevalidate` for HTML navigation routes to keep the app fast while still refreshing content.
- Always implement an offline fallback page (`/offline.html`) for navigation requests that fail.
- VAPID keys must be generated server-side and the public key passed to `applicationServerKey`.
- Test PWA behavior in Chrome DevTools Application tab — use Lighthouse for PWA audit.
