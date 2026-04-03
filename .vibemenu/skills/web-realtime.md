# Web Realtime Skill Guide

## WebSocket Client

### Basic Setup

```ts
class WebSocketClient {
  private ws: WebSocket | null = null
  private url: string
  private reconnectDelay = 1000
  private maxReconnectDelay = 30_000
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private handlers = new Map<string, (data: unknown) => void>()

  constructor(url: string) {
    this.url = url
  }

  connect() {
    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      console.log('WebSocket connected')
      this.reconnectDelay = 1000   // reset backoff on successful connect
    }

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data as string) as { type: string; data: unknown }
      this.handlers.get(message.type)?.(message.data)
    }

    this.ws.onerror = (event) => {
      console.error('WebSocket error:', event)
    }

    this.ws.onclose = (event) => {
      if (!event.wasClean) {
        this.scheduleReconnect()
      }
    }
  }

  private scheduleReconnect() {
    this.reconnectTimer = setTimeout(() => {
      console.log(`Reconnecting in ${this.reconnectDelay}ms...`)
      this.connect()
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay)
    }, this.reconnectDelay)
  }

  on(type: string, handler: (data: unknown) => void) {
    this.handlers.set(type, handler)
    return this
  }

  send(type: string, data: unknown) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, data }))
    }
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.ws?.close(1000, 'Client disconnect')
    this.ws = null
  }

  get state(): 'CONNECTING' | 'OPEN' | 'CLOSING' | 'CLOSED' | 'DISCONNECTED' {
    if (!this.ws) return 'DISCONNECTED'
    return (['CONNECTING', 'OPEN', 'CLOSING', 'CLOSED'] as const)[this.ws.readyState]
  }
}
```

### useWebSocket Hook (React)

```ts
import { useEffect, useRef, useState, useCallback } from 'react'

type ConnectionState = 'connecting' | 'open' | 'closing' | 'closed'

interface UseWebSocketOptions {
  onMessage?: (event: MessageEvent) => void
  onOpen?: () => void
  onClose?: () => void
  reconnect?: boolean
}

export function useWebSocket(url: string, options: UseWebSocketOptions = {}) {
  const [state, setState] = useState<ConnectionState>('connecting')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectDelayRef = useRef(1000)

  const connect = useCallback(() => {
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      setState('open')
      reconnectDelayRef.current = 1000
      options.onOpen?.()
    }

    ws.onmessage = (event) => options.onMessage?.(event)

    ws.onclose = () => {
      setState('closed')
      options.onClose?.()
      if (options.reconnect !== false) {
        setTimeout(connect, reconnectDelayRef.current)
        reconnectDelayRef.current = Math.min(reconnectDelayRef.current * 2, 30_000)
      }
    }

    ws.onerror = () => setState('closed')
  }, [url])

  useEffect(() => {
    connect()
    return () => {
      wsRef.current?.close(1000, 'Component unmounted')
    }
  }, [connect])

  const send = useCallback((data: string | object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(typeof data === 'string' ? data : JSON.stringify(data))
    }
  }, [])

  return { state, send }
}

// Usage
function Chat() {
  const { state, send } = useWebSocket('wss://api.example.com/ws', {
    onMessage: (event) => {
      const msg = JSON.parse(event.data)
      setMessages(prev => [...prev, msg])
    },
    reconnect: true,
  })

  return (
    <div>
      <span>Status: {state}</span>
      <button onClick={() => send({ type: 'chat', text: 'Hello' })}>Send</button>
    </div>
  )
}
```

---

## Server-Sent Events (SSE)

### Basic EventSource

```ts
const source = new EventSource('/api/events')

// Named events (server sends: event: price\ndata: {"value":42}\n\n)
source.addEventListener('price', (event) => {
  const data = JSON.parse(event.data)
  console.log('New price:', data.value)
})

// Default message event
source.onmessage = (event) => {
  console.log('Message:', event.data)
}

source.onerror = (event) => {
  if (source.readyState === EventSource.CLOSED) {
    console.log('SSE connection closed')
  }
}

// Close when done
source.close()
```

### Resumability with last-event-id

```ts
// Browser automatically sends Last-Event-ID header on reconnect
// Server must set event id: id: <event-id>\n\n

function createResumedSource(url: string, lastEventId?: string) {
  const fullUrl = lastEventId
    ? `${url}?lastEventId=${encodeURIComponent(lastEventId)}`
    : url
  return new EventSource(fullUrl)
}

// Track last id
let lastId: string | undefined
const source = new EventSource('/api/events')
source.onmessage = (event) => {
  if (event.lastEventId) lastId = event.lastEventId
}
```

### useEventSource Hook (React)

```ts
import { useEffect, useRef, useState } from 'react'

export function useEventSource<T>(url: string) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<Event | null>(null)
  const sourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    const source = new EventSource(url)
    sourceRef.current = source

    source.onmessage = (event) => {
      setData(JSON.parse(event.data) as T)
    }

    source.onerror = (err) => setError(err)

    return () => source.close()
  }, [url])

  return { data, error }
}

// Usage
function StockTicker() {
  const { data } = useEventSource<{ symbol: string; price: number }>('/api/stocks/stream')
  return <span>{data?.symbol}: ${data?.price}</span>
}
```

---

## Polling

```ts
import { useEffect, useRef, useCallback } from 'react'

export function usePolling(fn: () => Promise<void>, intervalMs: number, enabled = true) {
  const savedFn = useRef(fn)
  savedFn.current = fn

  useEffect(() => {
    if (!enabled) return

    const controller = new AbortController()
    let timeoutId: ReturnType<typeof setTimeout>

    async function tick() {
      if (controller.signal.aborted) return
      try {
        await savedFn.current()
      } catch (err) {
        if (!controller.signal.aborted) console.error('Polling error:', err)
      }
      if (!controller.signal.aborted) {
        timeoutId = setTimeout(tick, intervalMs)
      }
    }

    tick()

    return () => {
      controller.abort()
      clearTimeout(timeoutId)
    }
  }, [intervalMs, enabled])
}

// Usage
function DataPanel() {
  const fetchData = useCallback(async () => {
    const res = await fetch('/api/status')
    setStatus(await res.json())
  }, [])

  usePolling(fetchData, 5000)
}
```

### React Query with Polling

```ts
import { useQuery } from '@tanstack/react-query'

function useStatusPoll() {
  return useQuery({
    queryKey: ['status'],
    queryFn: () => fetch('/api/status').then(r => r.json()),
    refetchInterval: 5000,          // poll every 5s
    refetchIntervalInBackground: false,  // pause when tab hidden
  })
}
```

---

## Key Rules

- WebSocket: always implement exponential backoff (1s → 2s → 4s → ... → 30s max) before reconnecting.
- SSE: prefer EventSource over WebSocket for one-way server-to-client streaming — it auto-reconnects.
- Always clean up: close `WebSocket`/`EventSource` on component unmount via useEffect cleanup.
- Never reconnect immediately on close — a brief delay prevents thundering-herd on server restarts.
- Use `last-event-id` for SSE streams that must survive reconnects (e.g. notifications feed).
- React Query's `refetchInterval` is the simplest polling approach — prefer it over manual `setInterval`.
